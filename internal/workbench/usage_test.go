package workbench

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newUsageStore 构造带 usage_events 表的测试 store。
func newUsageStore(t *testing.T) *WorkbenchStore {
	t.Helper()
	store := newTestStore(t)
	if err := store.InitUsageTable(context.Background()); err != nil {
		t.Fatalf("InitUsageTable: %v", err)
	}
	return store
}

// insertUsage 直接插入一条计量事件（绕过 RecordUsage，便于控制时间）。
func insertUsage(t *testing.T, store *WorkbenchStore, e *UsageEvent) {
	t.Helper()
	if e.EventID == "" {
		e.EventID = newULID()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	_, err := store.db.ExecContext(context.Background(),
		`INSERT INTO usage_events
		 (event_id, account_id, provider_id, model_id, run_id, conversation_id,
		  input_tokens, output_tokens, estimated_cost_usd, latency_ms, error_code, occurred_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.EventID, e.AccountID, e.ProviderID, e.ModelID, e.RunID, e.ConversationID,
		e.InputTokens, e.OutputTokens, e.EstimatedCostUSD, e.LatencyMs, e.ErrorCode,
		e.OccurredAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatalf("insert usage: %v", err)
	}
}

// TestUsageEvent_Validate 校验计量事件字段。
func TestUsageEvent_Validate(t *testing.T) {
	valid := UsageEvent{
		EventID: "01HXY9KP1ZQR4C8N7T3V6RWYBJ", AccountID: "01HXY9KP1ZQR4C8N7T3V6RWYBJ",
		ProviderID: "deepseek", ModelID: "deepseek-chat",
		InputTokens: 100, OutputTokens: 50, EstimatedCostUSD: 0.001, LatencyMs: 200,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid event: %v", err)
	}
	if err := valid.Validate(); err != nil { // 重复校验无副作用
		t.Fatalf("valid event re-check: %v", err)
	}
	// 非法字段
	bad := valid
	bad.AccountID = ""
	if err := bad.Validate(); err == nil {
		t.Fatal("expected error for empty account_id")
	}
	bad = valid
	bad.InputTokens = -1
	if err := bad.Validate(); err == nil {
		t.Fatal("expected error for negative input_tokens")
	}
}

// TestRecordUsage_Success 验证成功落账。
func TestRecordUsage_Success(t *testing.T) {
	store := newUsageStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	recorder := NewMeterRecorder(store)

	rec := &UsageEvent{
		AccountID: acc.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat",
		RunID: "run_001", InputTokens: 100, OutputTokens: 50,
		EstimatedCostUSD: 0.001, LatencyMs: 200,
	}
	if err := recorder.RecordUsage(context.Background(), rec); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
	if rec.EventID == "" {
		t.Fatal("event_id should be auto-generated")
	}

	var (
		accountID string
		input     int
		output    int
	)
	err := store.db.QueryRowContext(context.Background(),
		`SELECT account_id, input_tokens, output_tokens FROM usage_events WHERE event_id = ?`,
		rec.EventID,
	).Scan(&accountID, &input, &output)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if accountID != acc.AccountID {
		t.Fatalf("account_id: got %q", accountID)
	}
	if input != 100 || output != 50 {
		t.Fatalf("tokens: got %d/%d", input, output)
	}
}

// TestRecordUsage_AccountFromContext 验证 rec 无 AccountID 时从 ctx 读取。
func TestRecordUsage_AccountFromContext(t *testing.T) {
	store := newUsageStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	recorder := NewMeterRecorder(store)

	ctx := context.WithValue(context.Background(), accountContextKey, acc.AccountID)
	rec := &UsageEvent{ProviderID: "deepseek", ModelID: "deepseek-chat", InputTokens: 10, OutputTokens: 5}
	if err := recorder.RecordUsage(ctx, rec); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
	if rec.AccountID != acc.AccountID {
		t.Fatalf("account_id from ctx: got %q, want %q", rec.AccountID, acc.AccountID)
	}
}

// TestRecordUsage_NoAccount 验证 ctx 无 account_id 且 rec 无 → 错误。
func TestRecordUsage_NoAccount(t *testing.T) {
	store := newUsageStore(t)
	recorder := NewMeterRecorder(store)
	rec := &UsageEvent{ProviderID: "deepseek", ModelID: "deepseek-chat"}
	if err := recorder.RecordUsage(context.Background(), rec); err == nil {
		t.Fatal("expected error when no account_id available")
	}
}

// TestRecordUsage_FailureCall 验证失败调用只记 ErrorCode，不计 token。
func TestRecordUsage_FailureCall(t *testing.T) {
	store := newUsageStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	recorder := NewMeterRecorder(store)
	rec := &UsageEvent{
		AccountID: acc.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat",
		ErrorCode: "PROVIDER_UNAVAILABLE", LatencyMs: 1500,
	}
	if err := recorder.RecordUsage(context.Background(), rec); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	// 失败调用不计入聚合（error_code = '' 过滤）
	agg, err := recorder.Aggregate(context.Background(), acc.AccountID, WindowAll)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if agg.CallCount != 0 {
		t.Fatalf("failed call should not count: got %d", agg.CallCount)
	}
}

// TestAggregate_All 验证 all 窗口包含全部成功调用并按 provider 拆分。
func TestAggregate_All(t *testing.T) {
	store := newUsageStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	recorder := NewMeterRecorder(store)

	insertUsage(t, store, &UsageEvent{AccountID: acc.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat", InputTokens: 100, OutputTokens: 50, EstimatedCostUSD: 0.001, OccurredAt: time.Now()})
	insertUsage(t, store, &UsageEvent{AccountID: acc.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat", InputTokens: 10, OutputTokens: 20, EstimatedCostUSD: 0.0005, OccurredAt: time.Now().Add(-48 * time.Hour)})
	insertUsage(t, store, &UsageEvent{AccountID: acc.AccountID, ProviderID: "siliconflow", ModelID: "Qwen2.5-7B", InputTokens: 5, OutputTokens: 5, EstimatedCostUSD: 0.0001, OccurredAt: time.Now()})

	agg, err := recorder.Aggregate(context.Background(), acc.AccountID, WindowAll)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if agg.TotalInputTokens != 115 || agg.TotalOutputTokens != 75 {
		t.Fatalf("totals: got %d/%d, want 115/75", agg.TotalInputTokens, agg.TotalOutputTokens)
	}
	if agg.TotalTokens != 190 {
		t.Fatalf("total_tokens: got %d, want 190", agg.TotalTokens)
	}
	if agg.CallCount != 3 {
		t.Fatalf("call_count: got %d, want 3", agg.CallCount)
	}
	if len(agg.ByProvider) != 2 {
		t.Fatalf("by_provider: got %d entries, want 2", len(agg.ByProvider))
	}
}

// TestAggregate_Today 验证 today 窗口只含今天事件。
func TestAggregate_Today(t *testing.T) {
	store := newUsageStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")

	insertUsage(t, store, &UsageEvent{AccountID: acc.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat", InputTokens: 100, OutputTokens: 50, OccurredAt: time.Now()})
	insertUsage(t, store, &UsageEvent{AccountID: acc.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat", InputTokens: 999, OutputTokens: 999, OccurredAt: time.Now().Add(-24 * time.Hour)})

	recorder := NewMeterRecorder(store)
	agg, err := recorder.Aggregate(context.Background(), acc.AccountID, WindowToday)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if agg.CallCount != 1 {
		t.Fatalf("today call_count: got %d, want 1", agg.CallCount)
	}
	if agg.TotalInputTokens != 100 {
		t.Fatalf("today input: got %d, want 100", agg.TotalInputTokens)
	}
}

// TestWindowStart 验证窗口起点计算。
func TestWindowStart(t *testing.T) {
	now := time.Date(2026, 8, 5, 15, 30, 0, 0, time.UTC) // 周三

	// today：当天 00:00
	start, ok := windowStart(WindowToday, now)
	if !ok || start != time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("today start: %v ok=%v", start, ok)
	}

	// week：本周一（2026-08-03 是周一）
	start, ok = windowStart(WindowWeek, now)
	if !ok || start != time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("week start: %v ok=%v", start, ok)
	}

	// month：本月 1 日
	start, ok = windowStart(WindowMonth, now)
	if !ok || start != time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("month start: %v ok=%v", start, ok)
	}

	// all：不过滤
	_, ok = windowStart(WindowAll, now)
	if ok {
		t.Fatal("all window should not filter")
	}
}

// TestPublicPoolSummary 验证 system scope 用量归属调用方账户。
func TestPublicPoolSummary(t *testing.T) {
	store := newUsageStore(t)
	accA, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	accB, _ := store.CreateAccount(context.Background(), "bob", "bob", "zh-CN", "test-hash")
	recorder := NewMeterRecorder(store)

	// 系统管理员创建 system scope 档案（属主为 accB，模拟 built_in）
	_, err := store.db.ExecContext(context.Background(),
		`INSERT INTO provider_profiles(profile_id, account_id, provider_id, display_name, base_url, model_id, api_key_env, scope, is_active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"prof_sys1", accB.AccountID, "deepseek", "Public DS",
		"https://api.deepseek.com/v1", "deepseek-chat", "KEY_DS",
		"system", 0, "2026-08-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert system profile: %v", err)
	}

	// alice 调用 system scope 供应商：用量归属 alice
	insertUsage(t, store, &UsageEvent{AccountID: accA.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat", InputTokens: 100, OutputTokens: 50, EstimatedCostUSD: 0.001, OccurredAt: time.Now()})
	// bob 调用私有 provider（非 system scope）：不应出现在公共池
	insertUsage(t, store, &UsageEvent{AccountID: accB.AccountID, ProviderID: "local", ModelID: "qwen-7b", InputTokens: 10, OutputTokens: 10, OccurredAt: time.Now()})

	summary, err := recorder.PublicPoolSummary(context.Background(), accA.AccountID)
	if err != nil {
		t.Fatalf("PublicPoolSummary: %v", err)
	}
	if len(summary) != 1 {
		t.Fatalf("expected 1 provider in pool, got %d", len(summary))
	}
	if summary[0].ProviderID != "deepseek" {
		t.Fatalf("provider_id: got %q", summary[0].ProviderID)
	}
	if summary[0].InputTokens != 100 || summary[0].OutputTokens != 50 {
		t.Fatalf("tokens: got %d/%d, want 100/50", summary[0].InputTokens, summary[0].OutputTokens)
	}
	if summary[0].CallCount != 1 {
		t.Fatalf("call_count: got %d, want 1", summary[0].CallCount)
	}

	// bob 的公共池不包含 local（非 system scope）
	summaryB, err := recorder.PublicPoolSummary(context.Background(), accB.AccountID)
	if err != nil {
		t.Fatalf("PublicPoolSummary B: %v", err)
	}
	if len(summaryB) != 0 {
		t.Fatalf("expected 0 providers for bob, got %d", len(summaryB))
	}
}

// TestPublicPoolSummary_NoSystemProfiles 验证无 system scope 档案时返回空。
func TestPublicPoolSummary_NoSystemProfiles(t *testing.T) {
	store := newUsageStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	recorder := NewMeterRecorder(store)
	insertUsage(t, store, &UsageEvent{AccountID: acc.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat", InputTokens: 1, OutputTokens: 1, OccurredAt: time.Now()})

	summary, err := recorder.PublicPoolSummary(context.Background(), acc.AccountID)
	if err != nil {
		t.Fatalf("PublicPoolSummary: %v", err)
	}
	if len(summary) != 0 {
		t.Fatalf("expected empty summary, got %d entries", len(summary))
	}
}

// usageTestCtx 构造带 account_id 的请求 context。
func usageTestCtx(accountID string) context.Context {
	return context.WithValue(context.Background(), accountContextKey, accountID)
}

// TestUsageHandler_Aggregate 验证聚合端点端到端。
func TestUsageHandler_Aggregate(t *testing.T) {
	store := newUsageStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	insertUsage(t, store, &UsageEvent{AccountID: acc.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat", InputTokens: 100, OutputTokens: 50, OccurredAt: time.Now()})

	mux := http.NewServeMux()
	NewUsageHandler(store).Routes(mux)

	req := httptest.NewRequest("GET", "/api/usage/aggregate?window=all", nil).WithContext(usageTestCtx(acc.AccountID))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	var agg UsageAggregate
	if err := json.Unmarshal(w.Body.Bytes(), &agg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if agg.CallCount != 1 {
		t.Fatalf("call_count: got %d, want 1", agg.CallCount)
	}
}

// TestUsageHandler_Aggregate_Unauthorized 验证未登录返回 401。
func TestUsageHandler_Aggregate_Unauthorized(t *testing.T) {
	store := newUsageStore(t)
	mux := http.NewServeMux()
	NewUsageHandler(store).Routes(mux)

	req := httptest.NewRequest("GET", "/api/usage/aggregate", nil) // 无 account context
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", w.Code)
	}
}

// TestUsageHandler_Aggregate_CrossAccount 验证跨账户查询返回 404。
func TestUsageHandler_Aggregate_CrossAccount(t *testing.T) {
	store := newUsageStore(t)
	accA, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	accB, _ := store.CreateAccount(context.Background(), "bob", "bob", "zh-CN", "test-hash")

	mux := http.NewServeMux()
	NewUsageHandler(store).Routes(mux)

	// alice 的 ctx，但 query 指定 bob → 404
	req := httptest.NewRequest("GET", "/api/usage/aggregate?account_id="+accB.AccountID, nil).
		WithContext(usageTestCtx(accA.AccountID))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", w.Code)
	}
}

// TestUsageHandler_Aggregate_InvalidWindow 验证非法 window 返回 400。
func TestUsageHandler_Aggregate_InvalidWindow(t *testing.T) {
	store := newUsageStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")

	mux := http.NewServeMux()
	NewUsageHandler(store).Routes(mux)

	req := httptest.NewRequest("GET", "/api/usage/aggregate?window=year", nil).
		WithContext(usageTestCtx(acc.AccountID))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", w.Code)
	}
}

// TestUsageHandler_PublicPool 验证公共池端点端到端。
func TestUsageHandler_PublicPool(t *testing.T) {
	store := newUsageStore(t)
	accA, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	accB, _ := store.CreateAccount(context.Background(), "bob", "bob", "zh-CN", "test-hash")

	// system scope 档案
	_, err := store.db.ExecContext(context.Background(),
		`INSERT INTO provider_profiles(profile_id, account_id, provider_id, display_name, base_url, model_id, api_key_env, scope, is_active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"prof_sys1", accB.AccountID, "deepseek", "Public DS",
		"https://api.deepseek.com/v1", "deepseek-chat", "KEY_DS",
		"system", 0, "2026-08-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert system profile: %v", err)
	}
	insertUsage(t, store, &UsageEvent{AccountID: accA.AccountID, ProviderID: "deepseek", ModelID: "deepseek-chat", InputTokens: 100, OutputTokens: 50, OccurredAt: time.Now()})

	mux := http.NewServeMux()
	NewUsageHandler(store).Routes(mux)

	req := httptest.NewRequest("GET", "/api/usage/public-pool", nil).
		WithContext(usageTestCtx(accA.AccountID))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	var resp struct {
		AccountID string          `json:"account_id"`
		Providers []ProviderUsage `json:"providers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.AccountID != accA.AccountID {
		t.Fatalf("account_id: got %q", resp.AccountID)
	}
	if len(resp.Providers) != 1 || resp.Providers[0].ProviderID != "deepseek" {
		t.Fatalf("providers: got %+v", resp.Providers)
	}
}

// TestHttpStatusFor 验证错误码映射无遗漏。
func TestHttpStatusFor(t *testing.T) {
	tests := []struct {
		code string
		want int
	}{
		{"VALIDATION_ERROR", http.StatusBadRequest},
		{"NOT_FOUND", http.StatusNotFound},
		{"PERMISSION_DENIED", http.StatusForbidden},
		{"BUDGET_EXCEEDED", http.StatusPaymentRequired},
		{"CONFLICT", http.StatusConflict},
		{"PROVIDER_UNAVAILABLE", http.StatusServiceUnavailable},
		{"INTERNAL_ERROR", http.StatusInternalServerError},
		{"UNKNOWN_CODE", http.StatusInternalServerError},
	}
	for _, tt := range tests {
		if got := httpStatusFor(tt.code); got != tt.want {
			t.Fatalf("httpStatusFor(%q): got %d, want %d", tt.code, got, tt.want)
		}
	}
}
