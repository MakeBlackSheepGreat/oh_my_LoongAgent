package workbench

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"time"

	"slim-agent/internal/harness"
)

// ---- 预编译正则 ----

var (
	reDraftID    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
	reDraftSkill = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
)

// ---- 枚举 ----

// TaskDraftStatus 任务草案状态。
type TaskDraftStatus string

const (
	DraftStatusPending  TaskDraftStatus = "draft"
	DraftStatusApproved TaskDraftStatus = "approved"
	DraftStatusRejected TaskDraftStatus = "rejected"
)

// TaskDraft 任务草案契约；approved 后调用 HarnessStore.CreateRun（D-011）。
type TaskDraft struct {
	DraftID        string          `json:"draft_id"`
	AccountID      string          `json:"account_id"`
	ConversationID string          `json:"conversation_id,omitempty"`
	Objective      string          `json:"objective"`
	SkillID        string          `json:"skill_id"`
	Status         TaskDraftStatus `json:"status"`
	RunID          string          `json:"run_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Validate 校验草案字段。
func (d *TaskDraft) Validate() error {
	if !reDraftID.MatchString(d.DraftID) {
		return errors.New("draft_id invalid")
	}
	if d.AccountID == "" {
		return errors.New("account_id must be non-empty")
	}
	if d.Objective == "" || len(d.Objective) > 10000 {
		return errors.New("objective length out of range [1,10000]")
	}
	if d.SkillID != "" && !reDraftSkill.MatchString(d.SkillID) {
		return errors.New("skill_id invalid")
	}
	switch d.Status {
	case DraftStatusPending, DraftStatusApproved, DraftStatusRejected:
	default:
		return errors.New("status invalid")
	}
	return nil
}

// CreateDraft 在当前账户下创建任务草案；skill_id 为空时默认 generic。
func (a *AccountScoped) CreateDraft(ctx context.Context, draftID, conversationID, objective, skillID string) (*TaskDraft, error) {
	if skillID == "" {
		skillID = "generic"
	}
	d := &TaskDraft{
		DraftID:        draftID,
		AccountID:      a.accountID,
		ConversationID: conversationID,
		Objective:      objective,
		SkillID:        skillID,
		Status:         DraftStatusPending,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := d.Validate(); err != nil {
		return nil, harness.ErrValidation("draft validation failed", err)
	}
	_, err := a.store.db.ExecContext(ctx,
		`INSERT INTO task_drafts(draft_id, account_id, conversation_id, objective, skill_id, status, run_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.DraftID, d.AccountID, d.ConversationID, d.Objective, d.SkillID, string(d.Status), "",
		d.CreatedAt.Format(time.RFC3339Nano), d.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to create draft", err)
	}
	return d, nil
}

// GetDraft 按 ID 查询草案；跨账户返回 nil, nil（对应 404）。
func (a *AccountScoped) GetDraft(ctx context.Context, draftID string) (*TaskDraft, error) {
	var (
		d         TaskDraft
		status    string
		createdAt string
		updatedAt string
	)
	err := a.store.db.QueryRowContext(ctx,
		`SELECT draft_id, account_id, conversation_id, objective, skill_id, status, run_id, created_at, updated_at
		 FROM task_drafts WHERE draft_id = ? AND account_id = ?`,
		draftID, a.accountID,
	).Scan(&d.DraftID, &d.AccountID, &d.ConversationID, &d.Objective, &d.SkillID, &status, &d.RunID, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to get draft", err)
	}
	d.Status = TaskDraftStatus(status)
	d.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	d.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return &d, nil
}

// ListDrafts 列出当前账户的草案，按 created_at 降序。
func (a *AccountScoped) ListDrafts(ctx context.Context) ([]*TaskDraft, error) {
	rows, err := a.store.db.QueryContext(ctx,
		`SELECT draft_id, account_id, conversation_id, objective, skill_id, status, run_id, created_at, updated_at
		 FROM task_drafts WHERE account_id = ? ORDER BY created_at DESC`,
		a.accountID,
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to list drafts", err)
	}
	defer rows.Close()
	var result []*TaskDraft
	for rows.Next() {
		var (
			d         TaskDraft
			status    string
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(&d.DraftID, &d.AccountID, &d.ConversationID, &d.Objective, &d.SkillID,
			&status, &d.RunID, &createdAt, &updatedAt); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan draft", err)
		}
		d.Status = TaskDraftStatus(status)
		d.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		d.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		result = append(result, &d)
	}
	if result == nil {
		result = []*TaskDraft{}
	}
	return result, rows.Err()
}

// UpdateDraftObjective 更新草案目标；仅 draft 状态可更新（approve 后不可再变更）。
func (a *AccountScoped) UpdateDraftObjective(ctx context.Context, draftID, objective string) error {
	if objective == "" || len(objective) > 10000 {
		return harness.ErrValidation("objective length out of range [1,10000]", nil)
	}
	res, err := a.store.db.ExecContext(ctx,
		`UPDATE task_drafts SET objective = ?, updated_at = ?
		 WHERE draft_id = ? AND account_id = ? AND status = ?`,
		objective, time.Now().UTC().Format(time.RFC3339Nano), draftID, a.accountID, string(DraftStatusPending),
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to update draft", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return harness.ErrNotFound("task_draft", draftID)
	}
	return nil
}

// DeleteDraft 删除草案；仅 draft 状态可删。
func (a *AccountScoped) DeleteDraft(ctx context.Context, draftID string) error {
	res, err := a.store.db.ExecContext(ctx,
		`DELETE FROM task_drafts WHERE draft_id = ? AND account_id = ? AND status = ?`,
		draftID, a.accountID, string(DraftStatusPending),
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to delete draft", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return harness.ErrNotFound("task_draft", draftID)
	}
	return nil
}

// DraftService 草案审批工作流；组合 WorkbenchStore、HarnessStore 与 HarnessRuntime。
// ApproveDraft 先创建 Run，再标记 draft approved，然后异步启动 HarnessRuntime.Run。
type DraftService struct {
	wb      *WorkbenchStore
	harness *harness.HarnessStore
	runtime *harness.HarnessRuntime
}

// NewDraftService 构造 DraftService。
func NewDraftService(wb *WorkbenchStore, h *harness.HarnessStore, runtime *harness.HarnessRuntime) *DraftService {
	return &DraftService{wb: wb, harness: h, runtime: runtime}
}

// ApproveDraft 批准草案：draft→approved，创建 TaskContract 并调用 HarnessStore.CreateRun，
// 然后异步启动 HarnessRuntime.Run 执行 Agent 循环。
func (s *DraftService) ApproveDraft(ctx context.Context, accountID, draftID string) (*harness.RunState, error) {
	scoped := NewAccountScoped(s.wb, accountID)
	d, err := scoped.GetDraft(ctx, draftID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, harness.ErrNotFound("task_draft", draftID)
	}
	if d.Status != DraftStatusPending {
		return nil, harness.NewHarnessError(harness.ErrCodeConflict,
			"draft is not in draft status; cannot approve", nil)
	}
	tc, err := harness.NewTaskContract(d.DraftID, d.SkillID, d.Objective)
	if err != nil {
		return nil, harness.ErrValidation("task contract invalid", err)
	}
	state, err := s.harness.CreateRun(tc)
	if err != nil {
		return nil, err
	}
	// CreateRun 成功后标记 approved
	_, err = s.wb.db.ExecContext(ctx,
		`UPDATE task_drafts SET status = ?, run_id = ?, updated_at = ?
		 WHERE draft_id = ? AND account_id = ? AND status = ?`,
		string(DraftStatusApproved), state.RunID, time.Now().UTC().Format(time.RFC3339Nano),
		draftID, accountID, string(DraftStatusPending),
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to approve draft", err)
	}
	// 异步启动运行时执行（P0 修复：管线接通）
	if s.runtime != nil {
		go s.startRun(state.RunID)
	}
	return state, nil
}

// startRun 在后台 goroutine 中执行 HarnessRuntime.Run 直至终态。
// 策略使用 B4（BVAR 预算感知验证器路由），超时 30 分钟防止泄漏。
func (s *DraftService) startRun(runID string) {
	runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	_, _ = s.runtime.Run(runCtx, runID, harness.NewB4Strategy())
}

// RejectDraft 拒绝草案：draft→rejected；不创建运行。
func (s *DraftService) RejectDraft(ctx context.Context, accountID, draftID string) error {
	res, err := s.wb.db.ExecContext(ctx,
		`UPDATE task_drafts SET status = ?, updated_at = ?
		 WHERE draft_id = ? AND account_id = ? AND status = ?`,
		string(DraftStatusRejected), time.Now().UTC().Format(time.RFC3339Nano),
		draftID, accountID, string(DraftStatusPending),
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to reject draft", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return harness.NewHarnessError(harness.ErrCodeConflict,
			"draft is not in draft status; cannot reject", nil)
	}
	return nil
}
