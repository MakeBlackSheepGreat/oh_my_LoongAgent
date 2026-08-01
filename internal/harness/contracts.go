// Package harness 定义领域无关的 Agent Harness 核心契约。
// 契约从 Python/Pydantic 迁移而来，保持领域语义不变；验证逻辑在构造函数中执行。
package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// utcNow 返回当前 UTC 时间。
func utcNow() time.Time { return time.Now().UTC() }

// ---- 枚举 ----

// RunStatus 运行状态。
type RunStatus string

const (
	StatusCreated   RunStatus = "created"
	StatusRunning   RunStatus = "running"
	StatusWaiting   RunStatus = "waiting"
	StatusCompleted RunStatus = "completed"
	StatusFailed    RunStatus = "failed"
	StatusCancelled RunStatus = "cancelled"
)

// RiskLevel 风险等级。
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// RiskOrder 风险等级排序值。
var RiskOrder = map[RiskLevel]int{
	RiskLow: 0, RiskMedium: 1, RiskHigh: 2, RiskCritical: 3,
}

// RecoveryLabel 机器可读的失败后续步骤分类。
type RecoveryLabel string

const (
	RecoveryRetry        RecoveryLabel = "retry"
	RecoveryRepair       RecoveryLabel = "repair"
	RecoveryHumanReview  RecoveryLabel = "human_review"
	RecoveryStop         RecoveryLabel = "stop"
)

// Permission 权限枚举。
type Permission string

const (
	PermReadWorkspace      Permission = "read_workspace"
	PermWriteWorkspace     Permission = "write_workspace"
	PermNetwork            Permission = "network"
	PermProcess            Permission = "process"
	PermExternalSideEffect Permission = "external_side_effect"
)

// ---- 预编译正则 ----

var (
	reID    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
	reSkill = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	reTool  = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)
	reSHA   = regexp.MustCompile(`^[a-f0-9]{64}$`)
	reVer   = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,32}$`)
	reCode  = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)
	reKind  = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)
)

// ---- 结构体 ----

// Budget 运行硬上限。
type Budget struct {
	MaxModelCalls    int     `json:"max_model_calls"`
	MaxToolCalls     int     `json:"max_tool_calls"`
	MaxRuntimeSeconds int    `json:"max_runtime_seconds"`
	MaxCostUSD       *float64 `json:"max_cost_usd,omitempty"`
	MaxArtifactBytes int     `json:"max_artifact_bytes"`
}

// DefaultBudget 返回默认预算。
func DefaultBudget() Budget {
	return Budget{
		MaxModelCalls:    12,
		MaxToolCalls:     50,
		MaxRuntimeSeconds: 900,
		MaxArtifactBytes: 100_000_000,
	}
}

// Validate 校验预算边界。
func (b Budget) Validate() error {
	if b.MaxModelCalls < 0 || b.MaxModelCalls > 10000 {
		return errors.New("max_model_calls out of range [0,10000]")
	}
	if b.MaxToolCalls < 0 || b.MaxToolCalls > 10000 {
		return errors.New("max_tool_calls out of range [0,10000]")
	}
	if b.MaxRuntimeSeconds < 1 || b.MaxRuntimeSeconds > 86400 {
		return errors.New("max_runtime_seconds out of range [1,86400]")
	}
	if b.MaxCostUSD != nil {
		if *b.MaxCostUSD < 0 || *b.MaxCostUSD > 100000 {
			return errors.New("max_cost_usd out of range [0,100000]")
		}
	}
	if b.MaxArtifactBytes < 0 || b.MaxArtifactBytes > 10_000_000_000 {
		return errors.New("max_artifact_bytes out of range")
	}
	return nil
}

// Policy 最小权限策略；空白名单默认拒绝。
type Policy struct {
	AllowedTools         []Permission  `json:"-"`
	AllowedToolNames     []string      `json:"allowed_tools"`
	AllowedPermissions   []Permission  `json:"allowed_permissions"`
	ApprovalRequiredFor  []RiskLevel   `json:"approval_required_for"`
	WorkspaceRoot        string        `json:"workspace_root,omitempty"`
	AllowNetworkDomains  []string      `json:"allow_network_domains"`
}

// DefaultPolicy 返回默认策略（高危需审批）。
func DefaultPolicy() Policy {
	return Policy{
		ApprovalRequiredFor: []RiskLevel{RiskHigh, RiskCritical},
	}
}

// Validate 校验策略。
func (p Policy) Validate() error {
	for _, t := range p.AllowedToolNames {
		if t == "" || t != trimSpace(t) {
			return errors.New("allowed tool names must be non-empty and trimmed")
		}
	}
	if len(p.WorkspaceRoot) > 1024 {
		return errors.New("workspace_root too long")
	}
	return nil
}

func trimSpace(s string) string {
	// 简单 trim，避免引入 strings 包的歧义
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// TaskContract 用户可见的任务契约。
type TaskContract struct {
	TaskID               string                 `json:"task_id"`
	SkillID              string                 `json:"skill_id"`
	Objective            string                 `json:"objective"`
	Inputs               map[string]any         `json:"inputs"`
	ExpectedArtifactKinds []string              `json:"expected_artifact_kinds"`
	Budget               Budget                 `json:"budget"`
	Policy               Policy                 `json:"policy"`
	Metadata             map[string]any         `json:"metadata"`
	ContractVersion      string                 `json:"contract_version"`
}

// NewTaskContract 构造并校验任务契约。
func NewTaskContract(taskID, skillID, objective string) (*TaskContract, error) {
	tc := &TaskContract{
		TaskID:          taskID,
		SkillID:         skillID,
		Objective:       objective,
		Inputs:          map[string]any{},
		Budget:          DefaultBudget(),
		Policy:          DefaultPolicy(),
		Metadata:        map[string]any{},
		ContractVersion: "v1",
	}
	if err := tc.Validate(); err != nil {
		return nil, err
	}
	return tc, nil
}

// Validate 校验任务契约。
func (tc *TaskContract) Validate() error {
	if !reID.MatchString(tc.TaskID) {
		return fmt.Errorf("task_id invalid: %s", tc.TaskID)
	}
	if !reSkill.MatchString(tc.SkillID) {
		return fmt.Errorf("skill_id invalid: %s", tc.SkillID)
	}
	if len(tc.Objective) < 1 || len(tc.Objective) > 10000 {
		return errors.New("objective length out of range [1,10000]")
	}
	if len(tc.ExpectedArtifactKinds) > 100 {
		return errors.New("too many expected_artifact_kinds")
	}
	seen := map[string]bool{}
	for _, k := range tc.ExpectedArtifactKinds {
		if k == "" || k != trimSpace(k) {
			return errors.New("artifact kind must be non-empty and trimmed")
		}
		if seen[k] {
			return errors.New("expected artifact kinds must be unique")
		}
		seen[k] = true
	}
	if !reVer.MatchString(tc.ContractVersion) {
		return fmt.Errorf("contract_version invalid: %s", tc.ContractVersion)
	}
	if err := tc.Budget.Validate(); err != nil {
		return err
	}
	if err := tc.Policy.Validate(); err != nil {
		return err
	}
	return nil
}

// ActionContract 模型提议、需授权后执行的动作。
type ActionContract struct {
	ActionID            string         `json:"action_id"`
	RunID               string         `json:"run_id"`
	ToolName            string         `json:"tool_name"`
	Arguments           map[string]any `json:"arguments"`
	ArgumentSchema      map[string]any `json:"argument_schema"`
	InputArtifactIDs    []string       `json:"input_artifact_ids"`
	OutputArtifactKinds []string       `json:"output_artifact_kinds"`
	Preconditions       []string       `json:"preconditions"`
	ImpactPreview       map[string]any `json:"impact_preview"`
	RollbackSummary     string         `json:"rollback_summary,omitempty"`
	RequiresApproval    bool           `json:"requires_approval"`
	Rationale           string         `json:"rationale,omitempty"`
	EstimatedCostUSD    *float64       `json:"estimated_cost_usd,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
}

// NewActionContract 构造并校验动作契约。
func NewActionContract(actionID, runID, toolName string) (*ActionContract, error) {
	a := &ActionContract{
		ActionID:  actionID,
		RunID:     runID,
		ToolName:  toolName,
		Arguments: map[string]any{},
		ArgumentSchema: map[string]any{},
		InputArtifactIDs: []string{},
		OutputArtifactKinds: []string{},
		Preconditions: []string{},
		ImpactPreview: map[string]any{},
		CreatedAt: utcNow(),
	}
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return a, nil
}

// Validate 校验动作契约。
func (a *ActionContract) Validate() error {
	if !reID.MatchString(a.ActionID) {
		return fmt.Errorf("action_id invalid: %s", a.ActionID)
	}
	if !reID.MatchString(a.RunID) {
		return fmt.Errorf("run_id invalid: %s", a.RunID)
	}
	if !reTool.MatchString(a.ToolName) {
		return fmt.Errorf("tool_name invalid: %s", a.ToolName)
	}
	if len(a.InputArtifactIDs) > 100 || len(a.OutputArtifactKinds) > 100 || len(a.Preconditions) > 100 {
		return errors.New("list length exceeds 100")
	}
	if len(a.RollbackSummary) > 2000 || len(a.Rationale) > 2000 {
		return errors.New("rollback_summary or rationale too long")
	}
	if a.EstimatedCostUSD != nil {
		if *a.EstimatedCostUSD < 0 || *a.EstimatedCostUSD > 100000 {
			return errors.New("estimated_cost_usd out of range")
		}
	}
	return nil
}

// ArtifactInput 持久化不可变工件所需的输入。
type ArtifactInput struct {
	ArtifactID        string         `json:"artifact_id"`
	Kind              string         `json:"kind"`
	ContentType       string         `json:"content_type"`
	Content           []byte         `json:"-"`
	Revision          int            `json:"revision"`
	ParentArtifactIDs []string       `json:"parent_artifact_ids"`
	Metadata          map[string]any `json:"metadata"`
}

// NewArtifactInput 构造并校验工件输入。
func NewArtifactInput(artifactID, kind string, content []byte) (*ArtifactInput, error) {
	ai := &ArtifactInput{
		ArtifactID:  artifactID,
		Kind:        kind,
		ContentType: "application/octet-stream",
		Content:     content,
		Revision:    1,
		Metadata:    map[string]any{},
	}
	if err := ai.Validate(); err != nil {
		return nil, err
	}
	return ai, nil
}

// Validate 校验工件输入。
func (ai *ArtifactInput) Validate() error {
	if !reID.MatchString(ai.ArtifactID) {
		return fmt.Errorf("artifact_id invalid: %s", ai.ArtifactID)
	}
	if len(ai.Kind) < 1 || len(ai.Kind) > 128 {
		return errors.New("kind length out of range [1,128]")
	}
	if len(ai.ContentType) < 1 || len(ai.ContentType) > 255 {
		return errors.New("content_type length out of range [1,255]")
	}
	if ai.Revision < 1 {
		return errors.New("revision must be >= 1")
	}
	if len(ai.ParentArtifactIDs) > 100 {
		return errors.New("too many parent_artifact_ids")
	}
	return nil
}

// Artifact 不可变运行工件；哈希始终指向原始存储字节。
type Artifact struct {
	ArtifactID        string         `json:"artifact_id"`
	RunID             string         `json:"run_id"`
	Kind              string         `json:"kind"`
	ContentType       string         `json:"content_type"`
	SHA256            string         `json:"sha256"`
	SizeBytes         int            `json:"size_bytes"`
	Revision          int            `json:"revision"`
	StorageURI        string         `json:"storage_uri"`
	ParentArtifactIDs []string       `json:"parent_artifact_ids"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
}

// ArtifactFromInput 从输入构造工件记录。
func ArtifactFromInput(runID string, in *ArtifactInput, storageURI string) (*Artifact, error) {
	if !reID.MatchString(runID) {
		return nil, fmt.Errorf("run_id invalid: %s", runID)
	}
	sum := sha256.Sum256(in.Content)
	art := &Artifact{
		ArtifactID:        in.ArtifactID,
		RunID:             runID,
		Kind:              in.Kind,
		ContentType:       in.ContentType,
		SHA256:            hex.EncodeToString(sum[:]),
		SizeBytes:         len(in.Content),
		Revision:          in.Revision,
		StorageURI:        storageURI,
		ParentArtifactIDs: in.ParentArtifactIDs,
		Metadata:          in.Metadata,
		CreatedAt:         utcNow(),
	}
	if err := art.Validate(); err != nil {
		return nil, err
	}
	return art, nil
}

// Validate 校验工件。
func (a *Artifact) Validate() error {
	if !reID.MatchString(a.ArtifactID) {
		return fmt.Errorf("artifact_id invalid: %s", a.ArtifactID)
	}
	if !reID.MatchString(a.RunID) {
		return fmt.Errorf("run_id invalid: %s", a.RunID)
	}
	if len(a.Kind) < 1 || len(a.Kind) > 128 {
		return errors.New("kind length out of range")
	}
	if len(a.ContentType) < 1 || len(a.ContentType) > 255 {
		return errors.New("content_type length out of range")
	}
	if !reSHA.MatchString(a.SHA256) {
		return fmt.Errorf("sha256 invalid: %s", a.SHA256)
	}
	if a.SizeBytes < 0 {
		return errors.New("size_bytes must be >= 0")
	}
	if a.Revision < 1 {
		return errors.New("revision must be >= 1")
	}
	if len(a.StorageURI) < 1 || len(a.StorageURI) > 2048 {
		return errors.New("storage_uri length out of range [1,2048]")
	}
	if len(a.ParentArtifactIDs) > 100 {
		return errors.New("too many parent_artifact_ids")
	}
	return nil
}

// ErrorRecord 错误记录。
type ErrorRecord struct {
	Code            string           `json:"code"`
	Message         string           `json:"message"`
	Recoverable     bool             `json:"recoverable"`
	RecoveryLabels  []RecoveryLabel  `json:"recovery_labels"`
	Details         map[string]any   `json:"details"`
	OccurredAt      time.Time        `json:"occurred_at"`
}

// NewErrorRecord 构造并校验错误记录；可恢复错误自动补 retry 标签。
func NewErrorRecord(code, message string, recoverable bool) (*ErrorRecord, error) {
	er := &ErrorRecord{
		Code:        code,
		Message:     message,
		Recoverable: recoverable,
		Details:     map[string]any{},
		OccurredAt:  utcNow(),
	}
	if recoverable {
		er.RecoveryLabels = []RecoveryLabel{RecoveryRetry}
	}
	if err := er.Validate(); err != nil {
		return nil, err
	}
	return er, nil
}

// Validate 校验错误记录。
func (er *ErrorRecord) Validate() error {
	if !reCode.MatchString(er.Code) {
		return fmt.Errorf("code invalid: %s", er.Code)
	}
	if len(er.Message) < 1 || len(er.Message) > 2000 {
		return errors.New("message length out of range [1,2000]")
	}
	return nil
}

// RunState 完整的版本化状态快照；永不在原地修改。
type RunState struct {
	RunID          string            `json:"run_id"`
	Task           TaskContract      `json:"task"`
	Status         RunStatus         `json:"status"`
	StateVersion   int               `json:"state_version"`
	ActiveActionID string            `json:"active_action_id,omitempty"`
	ArtifactIDs    []string          `json:"artifact_ids"`
	Usage          map[string]float64 `json:"usage"`
	LastError      *ErrorRecord      `json:"last_error,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// NewRunState 构造并校验运行状态。
func NewRunState(task *TaskContract) (*RunState, error) {
	rs := &RunState{
		RunID:        task.TaskID,
		Task:         *task,
		Status:       StatusCreated,
		StateVersion: 1,
		ArtifactIDs:  []string{},
		Usage:        map[string]float64{},
		CreatedAt:    utcNow(),
		UpdatedAt:    utcNow(),
	}
	if err := rs.Validate(); err != nil {
		return nil, err
	}
	return rs, nil
}

// Validate 校验运行状态。
func (rs *RunState) Validate() error {
	if rs.RunID != rs.Task.TaskID {
		return errors.New("run_id must match task.task_id")
	}
	if !reID.MatchString(rs.RunID) {
		return fmt.Errorf("run_id invalid: %s", rs.RunID)
	}
	if rs.StateVersion < 1 {
		return errors.New("state_version must be >= 1")
	}
	if len(rs.ArtifactIDs) > 10000 {
		return errors.New("too many artifact_ids")
	}
	seen := map[string]bool{}
	for _, id := range rs.ArtifactIDs {
		if seen[id] {
			return errors.New("artifact_ids must be unique")
		}
		seen[id] = true
	}
	if len(rs.ActiveActionID) > 128 {
		return errors.New("active_action_id too long")
	}
	return nil
}

// ValidatorResult 验证器结果。
type ValidatorResult struct {
	ValidatorID    string         `json:"validator_id"`
	ValidatorVersion string       `json:"validator_version"`
	RunID          string         `json:"run_id"`
	ArtifactIDs    []string       `json:"artifact_ids"`
	Passed         bool           `json:"passed"`
	Confidence     float64        `json:"confidence"`
	Findings        []string       `json:"findings"`
	Details        map[string]any `json:"details"`
	CreatedAt      time.Time      `json:"created_at"`
}

// NewValidatorResult 构造并校验验证器结果。
func NewValidatorResult(validatorID, runID string, passed bool) (*ValidatorResult, error) {
	vr := &ValidatorResult{
		ValidatorID:      validatorID,
		ValidatorVersion: "v1",
		RunID:            runID,
		ArtifactIDs:      []string{},
		Passed:           passed,
		Confidence:       1.0,
		Findings:         []string{},
		Details:          map[string]any{},
		CreatedAt:        utcNow(),
	}
	if err := vr.Validate(); err != nil {
		return nil, err
	}
	return vr, nil
}

// Validate 校验验证器结果。
func (vr *ValidatorResult) Validate() error {
	if !reKind.MatchString(vr.ValidatorID) {
		return fmt.Errorf("validator_id invalid: %s", vr.ValidatorID)
	}
	if !reVer.MatchString(vr.ValidatorVersion) {
		return fmt.Errorf("validator_version invalid: %s", vr.ValidatorVersion)
	}
	if !reID.MatchString(vr.RunID) {
		return fmt.Errorf("run_id invalid: %s", vr.RunID)
	}
	if len(vr.ArtifactIDs) > 100 || len(vr.Findings) > 100 {
		return errors.New("list length exceeds 100")
	}
	if vr.Confidence < 0 || vr.Confidence > 1 {
		return errors.New("confidence out of range [0,1]")
	}
	return nil
}

// SkillManifest 已安装 Skill 的声明；领域逻辑留在核心运行时之外。
type SkillManifest struct {
	SkillID            string         `json:"skill_id"`
	Version            string         `json:"version"`
	Title              string         `json:"title"`
	Description        string         `json:"description"`
	InputSchema        map[string]any `json:"input_schema"`
	OutputArtifactKinds []string      `json:"output_artifact_kinds"`
	RequiredTools      []string       `json:"required_tools"`
	RequiredValidators []string       `json:"required_validators"`
	DefaultBudget      Budget         `json:"default_budget"`
	Metadata           map[string]any `json:"metadata"`
}

// NewSkillManifest 构造并校验 Skill 声明。
func NewSkillManifest(skillID, version, title, description string) (*SkillManifest, error) {
	sm := &SkillManifest{
		SkillID:     skillID,
		Version:     version,
		Title:       title,
		Description: description,
		InputSchema: map[string]any{},
		DefaultBudget: DefaultBudget(),
		Metadata:     map[string]any{},
	}
	if err := sm.Validate(); err != nil {
		return nil, err
	}
	return sm, nil
}

// Validate 校验 Skill 声明。
func (sm *SkillManifest) Validate() error {
	if !reSkill.MatchString(sm.SkillID) {
		return fmt.Errorf("skill_id invalid: %s", sm.SkillID)
	}
	if !reVer.MatchString(sm.Version) {
		return fmt.Errorf("version invalid: %s", sm.Version)
	}
	if len(sm.Title) < 1 || len(sm.Title) > 120 {
		return errors.New("title length out of range [1,120]")
	}
	if len(sm.Description) < 1 || len(sm.Description) > 2000 {
		return errors.New("description length out of range [1,2000]")
	}
	if len(sm.OutputArtifactKinds) > 100 || len(sm.RequiredTools) > 100 || len(sm.RequiredValidators) > 100 {
		return errors.New("list length exceeds 100")
	}
	return sm.DefaultBudget.Validate()
}

// Event 运行事件。
type Event struct {
	Sequence  int            `json:"sequence"`
	RunID     string         `json:"run_id"`
	Kind      string         `json:"kind"`
	Message   string         `json:"message"`
	Payload   map[string]any `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
}

// NewEvent 构造并校验事件。
func NewEvent(sequence int, runID, kind, message string) (*Event, error) {
	e := &Event{
		Sequence:  sequence,
		RunID:     runID,
		Kind:      kind,
		Message:   message,
		Payload:   map[string]any{},
		Timestamp: utcNow(),
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return e, nil
}

// Validate 校验事件。
func (e *Event) Validate() error {
	if e.Sequence < 1 {
		return errors.New("sequence must be >= 1")
	}
	if !reID.MatchString(e.RunID) {
		return fmt.Errorf("run_id invalid: %s", e.RunID)
	}
	if !reKind.MatchString(e.Kind) {
		return fmt.Errorf("kind invalid: %s", e.Kind)
	}
	if len(e.Message) < 1 || len(e.Message) > 2000 {
		return errors.New("message length out of range [1,2000]")
	}
	return nil
}
