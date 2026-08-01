package workbench

import (
	"encoding/json"
	"net/http"

	"slim-agent/internal/harness"
)

// UsageHandler 用量 API handler 集合。
// 路由由 Routes 注册到外部 ServeMux（cmd/server 在 task5 建立后挂载）。
// 所有 handler 从 context 读取 account_id（由 AuthMiddleware 注入）。
type UsageHandler struct {
	recorder *MeterRecorder
}

// NewUsageHandler 构造 UsageHandler。
func NewUsageHandler(store *WorkbenchStore) *UsageHandler {
	return &UsageHandler{recorder: NewMeterRecorder(store)}
}

// Routes 注册用量路由到 mux。
func (h *UsageHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/usage/aggregate", h.handleAggregate)
	mux.HandleFunc("GET /api/usage/public-pool", h.handlePublicPool)
}

// handleAggregate 返回当前账户的用量聚合。
// GET /api/usage/aggregate?window=today|week|month|all
func (h *UsageHandler) handleAggregate(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	// 跨账户查询返回 404（不泄露其他账户数据存在性）
	if other := r.URL.Query().Get("account_id"); other != "" && other != accountID {
		writeError(w, harness.ErrNotFound("usage aggregate", other))
		return
	}
	window, err := parseUsageWindow(r.URL.Query().Get("window"))
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.recorder.Aggregate(r.Context(), accountID, window)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agg)
}

// handlePublicPool 返回当前账户在 system scope 供应商上的用量。
// GET /api/usage/public-pool
func (h *UsageHandler) handlePublicPool(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if other := r.URL.Query().Get("account_id"); other != "" && other != accountID {
		writeError(w, harness.ErrNotFound("public pool summary", other))
		return
	}
	summary, err := h.recorder.PublicPoolSummary(r.Context(), accountID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account_id": accountID, "providers": summary})
}

// writeJSON 写入 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 将 HarnessError 映射为 HTTP 状态码并写入统一错误格式。
func writeError(w http.ResponseWriter, err error) {
	he, ok := err.(*harness.HarnessError)
	if !ok {
		he = harness.NewHarnessError(harness.ErrCodeInternal, err.Error(), nil)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFor(he.Code))
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":  he.Code,
		"detail": he.Message,
	})
}

// httpStatusFor HarnessError.Code → HTTP 状态码映射表。
func httpStatusFor(code string) int {
	switch code {
	case harness.ErrCodeValidation:
		return http.StatusBadRequest
	case harness.ErrCodeNotFound:
		return http.StatusNotFound
	case harness.ErrCodePermissionDenied:
		return http.StatusForbidden
	case harness.ErrCodeBudgetExceeded:
		return http.StatusPaymentRequired
	case harness.ErrCodeConflict:
		return http.StatusConflict
	case harness.ErrCodeProviderUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
