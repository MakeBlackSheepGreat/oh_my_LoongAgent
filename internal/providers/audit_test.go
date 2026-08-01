package providers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newTestAuditor 构造测试用内存 SQLite Auditor。
// 参考 internal/harness/storage_test.go 的内存 SQLite 测试设置方式。
func newTestAuditor(t *testing.T) (*SQLiteAuditor, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	auditor := NewSQLiteAuditor(db)
	if err := auditor.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return auditor, db
}

// TestSQLiteAuditor_Record_Success 验证成功调用记录的字段。
func TestSQLiteAuditor_Record_Success(t *testing.T) {
	auditor, db := newTestAuditor(t)
	rec := &AuditRecord{
		ProviderID:       "openai",
		ModelID:          "gpt-4",
		RunID:            "run_001",
		InputTokens:      100,
		OutputTokens:     50,
		EstimatedCostUSD: 0.002,
		LatencyMs:        200,
		ErrorCode:        "",
		OccurredAt:       time.Now(),
	}
	if err := auditor.Record(context.Background(), rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// 验证字段正确持久化
	var (
		providerID    string
		modelID       string
		inputTokens   int
		outputTokens  int
		estimatedCost float64
		latencyMs     int
		errorCode     string
	)
	err := db.QueryRow(
		`SELECT provider_id, model_id, input_tokens, output_tokens,
		        estimated_cost_usd, latency_ms, error_code
		 FROM provider_call_log WHERE provider_id = ?`,
		rec.ProviderID,
	).Scan(&providerID, &modelID, &inputTokens, &outputTokens, &estimatedCost, &latencyMs, &errorCode)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if providerID != "openai" {
		t.Fatalf("provider_id: 期望 'openai'，得到 %q", providerID)
	}
	if modelID != "gpt-4" {
		t.Fatalf("model_id: 期望 'gpt-4'，得到 %q", modelID)
	}
	if inputTokens != 100 {
		t.Fatalf("input_tokens: 期望 100，得到 %d", inputTokens)
	}
	if outputTokens != 50 {
		t.Fatalf("output_tokens: 期望 50，得到 %d", outputTokens)
	}
	if estimatedCost != 0.002 {
		t.Fatalf("estimated_cost_usd: 期望 0.002，得到 %f", estimatedCost)
	}
	if latencyMs != 200 {
		t.Fatalf("latency_ms: 期望 200，得到 %d", latencyMs)
	}
	if errorCode != "" {
		t.Fatalf("error_code: 期望空，得到 %q", errorCode)
	}
}

// TestSQLiteAuditor_Record_Failure 验证失败调用记录的 ErrorCode 非空、InputTokens=0。
func TestSQLiteAuditor_Record_Failure(t *testing.T) {
	auditor, db := newTestAuditor(t)
	rec := &AuditRecord{
		ProviderID: "openai",
		ModelID:    "gpt-4",
		RunID:      "run_002",
		ErrorCode:  "PROVIDER_UNAVAILABLE",
		OccurredAt: time.Now(),
	}
	if err := auditor.Record(context.Background(), rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	var (
		inputTokens int
		errorCode   string
	)
	err := db.QueryRow(
		`SELECT input_tokens, error_code FROM provider_call_log WHERE run_id = ?`,
		rec.RunID,
	).Scan(&inputTokens, &errorCode)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if errorCode == "" {
		t.Fatal("error_code: 期望非空")
	}
	if errorCode != "PROVIDER_UNAVAILABLE" {
		t.Fatalf("error_code: 期望 'PROVIDER_UNAVAILABLE'，得到 %q", errorCode)
	}
	if inputTokens != 0 {
		t.Fatalf("input_tokens: 期望 0，得到 %d", inputTokens)
	}
}

// TestSQLiteAuditor_Init 验证表创建成功，重复 Init 不报错。
func TestSQLiteAuditor_Init(t *testing.T) {
	auditor, _ := newTestAuditor(t) // Init 已在 helper 中调用一次
	// 重复 Init 不报错（CREATE TABLE IF NOT EXISTS 幂等）
	if err := auditor.Init(context.Background()); err != nil {
		t.Fatalf("重复 Init 不应报错，得到: %v", err)
	}
}
