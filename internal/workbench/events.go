package workbench

import (
	"sync"

	"slim-agent/internal/harness"
)

// hubBufferSize 每个 subscriber 的缓冲通道容量。
// 慢消费者超过缓冲时事件被丢弃并记录 event_dropped 警告。
const hubBufferSize = 16

// EventHub 运行事件广播中心；订阅者通过 chan 接收事件。
// 用 sync.RWMutex 保护 subscribers map，广播用 RLock 支持并发安全。
type EventHub struct {
	mu          sync.RWMutex
	subscribers map[chan *harness.Event]struct{}
}

// NewEventHub 构造 EventHub。
func NewEventHub() *EventHub {
	return &EventHub{subscribers: make(map[chan *harness.Event]struct{})}
}

// Subscribe 注册新订阅者并返回事件通道。
// 调用方负责在结束时调用 Unsubscribe。
func (h *EventHub) Subscribe() chan *harness.Event {
	ch := make(chan *harness.Event, hubBufferSize)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe 注销订阅者并关闭通道。
func (h *EventHub) Unsubscribe(ch chan *harness.Event) {
	h.mu.Lock()
	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Broadcast 广播事件到所有订阅者；慢消费者（缓冲满）丢弃事件并记录警告。
// 时间复杂度 O(n)，n 为订阅者数。
func (h *EventHub) Broadcast(event *harness.Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers {
		select {
		case ch <- event:
		default:
			// 缓冲满：丢弃并发送占位警告事件
			warn := &harness.Event{
				Sequence:  -1,
				RunID:     event.RunID,
				Kind:      "event_dropped",
				Message:   "subscriber too slow; event dropped",
				Payload:   map[string]any{"dropped_kind": event.Kind},
				Timestamp: event.Timestamp,
			}
			select {
			case ch <- warn:
			default:
				// 占位也满，彻底放弃
			}
		}
	}
}

// Len 返回当前订阅者数量（测试与诊断用）。
func (h *EventHub) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}
