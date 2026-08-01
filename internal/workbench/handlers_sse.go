package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"slim-agent/internal/harness"
)

// handleEvents SSE 全局事件流：GET /api/events
// 受 AuthMiddleware 保护；按 account 过滤（仅转发当前账户 run 的事件）。
// 客户端断开（r.Context().Done()）自动清理订阅者；SSE goroutine 不泄漏。
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, harness.NewHarnessError(harness.ErrCodeInternal, "streaming unsupported", nil))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch := s.hub.Subscribe()
	defer s.hub.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			// 按账户过滤：仅转发当前账户 run 的事件
			if !s.runBelongsTo(r.Context(), ev.RunID, accountID) {
				continue
			}
			data, _ := json.Marshal(ev)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Kind, data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// runBelongsTo 判断 run 是否属于指定账户。
// 通过 task_drafts.run_id 反查草案属主（ApproveDraft 建立 run_id → account_id 归属）。
func (s *Server) runBelongsTo(ctx context.Context, runID, accountID string) bool {
	if runID == "" {
		return false
	}
	var count int
	err := s.store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM task_drafts WHERE run_id = ? AND account_id = ?`,
		runID, accountID,
	).Scan(&count)
	return err == nil && count > 0
}
