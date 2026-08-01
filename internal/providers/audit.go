package providers

import (
	"context"
	"database/sql"
	"time"

	"slim-agent/internal/harness/errs"
)

// AuditRecord 单次 Provider 调用的审计记录。
// 成功调用：填充 InputTokens、OutputTokens、EstimatedCostUSD、LatencyMs、ModelID，ErrorCode 留空。
// 失败调用：仅填充 ErrorCode；InputTokens、OutputTokens、EstimatedCostUSD 保持零值。
type AuditRecord struct {
	ProviderID       string
	ModelID          string
	RunID            string
	InputTokens      int
	OutputTokens     int
	EstimatedCostUSD float64
	LatencyMs        int
	ErrorCode        string
	OccurredAt       time.Time
}

// Auditor 审计记录器接口；实现方负责持久化 AuditRecord。
type Auditor interface {
	Record(ctx context.Context, rec *AuditRecord) error
}

// SQLiteAuditor 基于 SQLite 的审计记录器；单次 INSERT 时间复杂度 O(1)。
type SQLiteAuditor struct {
	db *sql.DB
}

// NewSQLiteAuditor 构造 SQLiteAuditor；不持有 db 的所有权，调用方负责关闭。
func NewSQLiteAuditor(db *sql.DB) *SQLiteAuditor {
	return &SQLiteAuditor{db: db}
}

// Init 创建 provider_call_log 表（如不存在）；字段对齐 AuditRecord。
func (a *SQLiteAuditor) Init(ctx context.Context) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS provider_call_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id TEXT NOT NULL,
			model_id TEXT NOT NULL,
			run_id TEXT NOT NULL DEFAULT '',
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			estimated_cost_usd REAL NOT NULL DEFAULT 0,
			latency_ms INTEGER NOT NULL DEFAULT 0,
			error_code TEXT NOT NULL DEFAULT '',
			occurred_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_provider_call_log_occurred_at
			ON provider_call_log(occurred_at);
	`
	if _, err := a.db.ExecContext(ctx, schema); err != nil {
		return errs.NewHarnessError(
			errs.ErrCodeInternal,
			"failed to initialize provider_call_log table",
			err,
		)
	}
	return nil
}

// Record 插入一条审计记录；使用 ExecContext 避免阻塞，时间复杂度 O(1)。
func (a *SQLiteAuditor) Record(ctx context.Context, rec *AuditRecord) error {
	_, err := a.db.ExecContext(
		ctx,
		`INSERT INTO provider_call_log
		 (provider_id, model_id, run_id, input_tokens, output_tokens,
		  estimated_cost_usd, latency_ms, error_code, occurred_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ProviderID, rec.ModelID, rec.RunID,
		rec.InputTokens, rec.OutputTokens,
		rec.EstimatedCostUSD, rec.LatencyMs, rec.ErrorCode,
		rec.OccurredAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return errs.NewHarnessError(
			errs.ErrCodeInternal,
			"failed to record provider call audit",
			err,
		)
	}
	return nil
}
