package providers

import (
	"sync"
)

// DeepSeekProvider DeepSeek 定制化 Provider。
// 嵌入 OpenAICompatibleProvider 复用 HTTP 调用能力；
// 额外支持 prefix cache 稳定前缀、stale 输出剪枝与 context maintenance。
// stablePrefix 与 contextSummary 可通过 Setter 更新，内部使用 sync.RWMutex 保证并发安全。
type DeepSeekProvider struct {
	*OpenAICompatibleProvider
	mu             sync.RWMutex
	stablePrefix   []ChatMessage
	contextSummary string
}

// NewDeepSeekProvider 构造 DeepSeek Provider；使用 DeepSeek preset 配置。
func NewDeepSeekProvider(config ProviderConfig) (*DeepSeekProvider, error) {
	base, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		return nil, err
	}
	return &DeepSeekProvider{
		OpenAICompatibleProvider: base,
	}, nil
}

// BuildMessages 构造完整消息列表：stablePrefix + contextSummary(system) + dynamicMessages。
// 保证同一会话连续调用的前缀顺序不变，命中 DeepSeek prefix cache。
func (p *DeepSeekProvider) BuildMessages(dynamicMessages []ChatMessage) []ChatMessage {
	p.mu.RLock()
	defer p.mu.RUnlock()
	needSummary := p.contextSummary != ""
	total := len(p.stablePrefix) + len(dynamicMessages)
	if needSummary {
		total++
	}
	out := make([]ChatMessage, 0, total)
	out = append(out, p.stablePrefix...)
	if needSummary {
		out = append(out, ChatMessage{
			Role:    "system",
			Content: p.contextSummary,
		})
	}
	out = append(out, dynamicMessages...)
	return out
}

// SetStablePrefix 设置稳定前缀段（system prompt 等）。
func (p *DeepSeekProvider) SetStablePrefix(messages []ChatMessage) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stablePrefix = messages
}

// PruneStale 剪枝 stale 消息。
// tool 角色消息若被后续 assistant 消息覆盖（tool 结果已被模型消费），则剪枝。
// 单次遍历 O(n)：缓冲连续的 tool 消息，遇到 assistant 时丢弃缓冲。
func (p *DeepSeekProvider) PruneStale(messages []ChatMessage) []ChatMessage {
	out := make([]ChatMessage, 0, len(messages))
	var toolBuffer []ChatMessage
	for _, m := range messages {
		switch m.Role {
		case "tool":
			toolBuffer = append(toolBuffer, m)
		case "assistant":
			// 后续 assistant 消息覆盖了缓冲的 tool 消息，丢弃
			toolBuffer = nil
			out = append(out, m)
		default:
			// system/user 等消息不覆盖 tool 消息，先刷新缓冲
			out = append(out, toolBuffer...)
			toolBuffer = nil
			out = append(out, m)
		}
	}
	// 刷新剩余未被覆盖的 tool 消息
	out = append(out, toolBuffer...)
	return out
}

// InjectContextSummary 在 stablePrefix 之后、dynamicMessages 之前注入 system 角色的环境摘要消息。
// 摘要内容为 contextSummary 字段；若摘要为空则原样返回。
func (p *DeepSeekProvider) InjectContextSummary(messages []ChatMessage) []ChatMessage {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.contextSummary == "" {
		return messages
	}
	out := make([]ChatMessage, 0, len(messages)+1)
	out = append(out, ChatMessage{
		Role:    "system",
		Content: p.contextSummary,
	})
	out = append(out, messages...)
	return out
}

// SetContextSummary 更新环境摘要。
func (p *DeepSeekProvider) SetContextSummary(summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.contextSummary = summary
}
