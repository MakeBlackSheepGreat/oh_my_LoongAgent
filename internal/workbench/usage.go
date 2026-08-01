package workbench

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"slim-agent/internal/harness"
)

// ---- 预编译正则 ----

var (
	reEventID  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
	reWindow   = regexp.MustCompile(`^(today|week|month|all)$`)
)

// ---- 枚举 ----

// UsageWindow 聚合时间窗口。
type UsageWindow string

const (
	WindowToday  UsageWindow = "today"
	WindowWeek   UsageWindow = "week"
	WindowMonth  UsageWindow = "month"
	WindowAll    UsageWindow = "all"
)

// ---- 结构体 ----

// UsageEvent 单次计量事件；EventID 为 ULID，AccountID 标识用量归属。
// 成功调用：填充 InputTokens/OutputTokens/EstimatedCostUSD/LatencyMs，ErrorCode 留空。
// 失败调用：仅填充 ErrorCode 与 LatencyMs；token/成本保持零值。
type UsageEvent struct {
	EventID          string    `json:"event_id"`
	AccountID        string    `json:"account_id"`
	ProviderID       string    `json:"provider_id"`
	ModelID          string    `json:"model_id"`
	RunID            string    `json:"run_id,omitempty"`
	ConversationID   string    `json:"conversation_id,omitempty"`
	InputTokens      int       `json:"input_tokens"`
	OutputTokens     int       `json:"output_tokens"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd"`
	LatencyMs        int       `json:"latency_ms"`
	ErrorCode        string    `json:"error_code,omitempty"`
	OccurredAt       time.Time `json:"occurred_at"`
}

// Validate 校验计量事件字段。
func (e *UsageEvent) Validate() error {
	if !reEventID.MatchString(e.EventID) {
		return fmt.Errorf("event_id invalid: %s", e.EventID)
	}
	if e.AccountID == "" {
		return errors.New("account_id must be non-empty")
	}
	if e.ProviderID == "" {
		return errors.New("provider_id must be non-empty")
	}
	if e.ModelID == "" {
		return errors.New("model_id must be non-empty")
	}
	if e.InputTokens < 0 || e.OutputTokens < 0 {
		return errors.New("token counts must be >= 0")
	}
	if e.LatencyMs < 0 {
		return errors.New("latency_ms must be >= 0")
	}
	if e.EstimatedCostUSD < 0 {
		return errors.New("estimated_cost_usd must be >= 0")
	}
	return nil
}

// UsageAggregate 聚合结果：总量 + 按 provider × model 拆分。
type UsageAggregate struct {
	AccountID        string          `json:"account_id"`
	Window           UsageWindow     `json:"window"`
	TotalInputTokens  int            `json:"total_input_tokens"`
	TotalOutputTokens int            `json:"total_output_tokens"`
	TotalTokens       int            `json:"total_tokens"`
	TotalCostUSD      float64        `json:"total_cost_usd"`
	CallCount         int            `json:"call_count"`
	ByProvider        []ProviderUsage `json:"by_provider"`
}

// ProviderUsage 单个 provider × model 的用量汇总。
type ProviderUsage struct {
	ProviderID       string  `json:"provider_id"`
	ModelID          string  `json:"model_id"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	CallCount        int     `json:"call_count"`
}

// InitUsageTable 创建 usage_events 表（幂等）。
// account_id + occurred_at 复合索引支持按账户×时间窗口聚合（O(log n)）；provider_id 索引支持公共池筛选。
func (s *WorkbenchStore) InitUsageTable(ctx context.Context) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS usage_events (
			event_id           TEXT PRIMARY KEY,
			account_id         TEXT NOT NULL,
			provider_id        TEXT NOT NULL,
			model_id           TEXT NOT NULL,
			run_id             TEXT NOT NULL DEFAULT '',
			conversation_id    TEXT NOT NULL DEFAULT '',
			input_tokens       INTEGER NOT NULL DEFAULT 0,
			output_tokens      INTEGER NOT NULL DEFAULT 0,
			estimated_cost_usd REAL NOT NULL DEFAULT 0,
			latency_ms         INTEGER NOT NULL DEFAULT 0,
			error_code         TEXT NOT NULL DEFAULT '',
			occurred_at        TEXT NOT NULL,
			FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_usage_account_time
			ON usage_events(account_id, occurred_at);
		CREATE INDEX IF NOT EXISTS idx_usage_provider
			ON usage_events(provider_id);
	`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return harness.NewHarnessError(
			harness.ErrCodeInternal,
			"failed to initialize usage_events table",
			err,
		)
	}
	return nil
}

// MeterRecorder 计量记录器；单次 INSERT 时间复杂度 O(1)。
type MeterRecorder struct {
	store *WorkbenchStore
}

// NewMeterRecorder 构造 MeterRecorder。
func NewMeterRecorder(store *WorkbenchStore) *MeterRecorder {
	return &MeterRecorder{store: store}
}

// RecordUsage 落账一条计量事件。
// AccountID 优先级：rec.AccountID 非空则直接用；否则从 ctx 读取（AccountFromContext）。
func (m *MeterRecorder) RecordUsage(ctx context.Context, rec *UsageEvent) error {
	if rec.AccountID == "" {
		accountID, err := AccountFromContext(ctx)
		if err != nil {
			return err
		}
		rec.AccountID = accountID
	}
	if rec.EventID == "" {
		rec.EventID = newULID()
	}
	if rec.OccurredAt.IsZero() {
		rec.OccurredAt = time.Now().UTC()
	}
	if err := rec.Validate(); err != nil {
		return harness.ErrValidation("usage event validation failed", err)
	}
	_, err := m.store.db.ExecContext(ctx,
		`INSERT INTO usage_events
		 (event_id, account_id, provider_id, model_id, run_id, conversation_id,
		  input_tokens, output_tokens, estimated_cost_usd, latency_ms, error_code, occurred_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.EventID, rec.AccountID, rec.ProviderID, rec.ModelID, rec.RunID, rec.ConversationID,
		rec.InputTokens, rec.OutputTokens, rec.EstimatedCostUSD, rec.LatencyMs, rec.ErrorCode,
		rec.OccurredAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to record usage event", err)
	}
	return nil
}

// windowStart 返回时间窗口起点（UTC）；all 返回零值表示不过滤。
func windowStart(window UsageWindow, now time.Time) (time.Time, bool) {
	switch window {
	case WindowToday:
		y, mo, d := now.Date()
		return time.Date(y, mo, d, 0, 0, 0, 0, time.UTC), true
	case WindowWeek:
		// 自然周：周一 00:00 UTC
		y, mo, d := now.Date()
		dayStart := time.Date(y, mo, d, 0, 0, 0, 0, time.UTC)
		offset := (int(dayStart.Weekday()) + 6) % 7 // 周一=0
		return dayStart.AddDate(0, 0, -offset), true
	case WindowMonth:
		y, mo, _ := now.Date()
		return time.Date(y, mo, 1, 0, 0, 0, 0, time.UTC), true
	default:
		return time.Time{}, false
	}
}

// parseUsageWindow 解析时间窗口参数；非法值返回错误。
func parseUsageWindow(s string) (UsageWindow, error) {
	if !reWindow.MatchString(s) {
		return "", harness.ErrValidation("invalid usage window, must be today|week|month|all", nil)
	}
	return UsageWindow(s), nil
}

// Aggregate 按账户 × 时间窗口聚合用量；只统计成功调用（error_code = ''）。
// 查询走 idx_usage_account_time 索引，O(log n) 起。
func (m *MeterRecorder) Aggregate(ctx context.Context, accountID string, window UsageWindow) (*UsageAggregate, error) {
	start, filterTime := windowStart(window, time.Now().UTC())
	query := `
		SELECT provider_id, model_id,
		       SUM(input_tokens), SUM(output_tokens),
		       SUM(estimated_cost_usd), COUNT(*)
		FROM usage_events
		WHERE account_id = ? AND error_code = ''`
	args := []any{accountID}
	if filterTime {
		query += ` AND occurred_at >= ?`
		args = append(args, start.Format(time.RFC3339Nano))
	}
	query += ` GROUP BY provider_id, model_id ORDER BY provider_id, model_id`

	rows, err := m.store.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to aggregate usage", err)
	}
	defer rows.Close()

	agg := &UsageAggregate{AccountID: accountID, Window: window}
	for rows.Next() {
		var (
			pu         ProviderUsage
			callCount  int
		)
		if err := rows.Scan(&pu.ProviderID, &pu.ModelID,
			&pu.InputTokens, &pu.OutputTokens, &pu.EstimatedCostUSD, &callCount); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan usage aggregate", err)
		}
		pu.TotalTokens = pu.InputTokens + pu.OutputTokens
		pu.CallCount = callCount
		agg.ByProvider = append(agg.ByProvider, pu)
		agg.TotalInputTokens += pu.InputTokens
		agg.TotalOutputTokens += pu.OutputTokens
		agg.TotalTokens += pu.TotalTokens
		agg.TotalCostUSD += pu.EstimatedCostUSD
		agg.CallCount += callCount
	}
	return agg, rows.Err()
}

// PublicPoolSummary 汇总当前账户在 system scope 供应商上的用量。
// 用量归属调用方账户（WHERE account_id = ?），而非档案属主——"公共模型池显示各账号 token 消耗"语义。
func (m *MeterRecorder) PublicPoolSummary(ctx context.Context, accountID string) ([]ProviderUsage, error) {
	// 取 system scope 档案的 provider_id 列表
	rows, err := m.store.db.QueryContext(ctx,
		`SELECT DISTINCT provider_id FROM provider_profiles WHERE scope = ?`,
		string(ScopeSystem),
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to list system providers", err)
	}
	defer rows.Close()
	var providerIDs []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan system provider", err)
		}
		providerIDs = append(providerIDs, pid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(providerIDs) == 0 {
		return []ProviderUsage{}, nil
	}

	placeholders := make([]string, len(providerIDs))
	args := make([]any, 0, len(providerIDs)+1)
	args = append(args, accountID)
	for i, pid := range providerIDs {
		placeholders[i] = "?"
		args = append(args, pid)
	}
	query := `
		SELECT provider_id, model_id,
		       SUM(input_tokens), SUM(output_tokens),
		       SUM(estimated_cost_usd), COUNT(*)
		FROM usage_events
		WHERE account_id = ? AND error_code = '' AND provider_id IN (` + strings.Join(placeholders, ",") + `)
		GROUP BY provider_id, model_id ORDER BY provider_id, model_id`

	aggRows, err := m.store.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to summarize public pool", err)
	}
	defer aggRows.Close()

	var result []ProviderUsage
	for aggRows.Next() {
		var (
			pu        ProviderUsage
			callCount int
		)
		if err := aggRows.Scan(&pu.ProviderID, &pu.ModelID,
			&pu.InputTokens, &pu.OutputTokens, &pu.EstimatedCostUSD, &callCount); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan public pool usage", err)
		}
		pu.TotalTokens = pu.InputTokens + pu.OutputTokens
		pu.CallCount = callCount
		result = append(result, pu)
	}
	return result, aggRows.Err()
}
