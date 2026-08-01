package harness

import (
	"context"
	"fmt"
	"strings"

	"slim-agent/internal/providers"
)

// ---- 决策 ----

// DecisionKind 策略决策类型。
type DecisionKind string

const (
	DecisionCallModel     DecisionKind = "call_model"
	DecisionExecuteTool   DecisionKind = "execute_tool"
	DecisionWaitApproval  DecisionKind = "wait_approval"
	DecisionRecover       DecisionKind = "recover"
	DecisionFinish        DecisionKind = "finish"
)

// Decision 策略单步输出。
type Decision struct {
	Kind      DecisionKind `json:"kind"`
	ToolName  string       `json:"tool_name,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Messages  []providers.ChatMessage `json:"-"`
	Reasoning string       `json:"reasoning,omitempty"`
}

// NewCallModelDecision 构造模型调用决策。
func NewCallModelDecision(messages []providers.ChatMessage, reasoning string) *Decision {
	return &Decision{Kind: DecisionCallModel, Messages: messages, Reasoning: reasoning}
}

// NewToolDecision 构造工具执行决策。
func NewToolDecision(toolName string, args map[string]any, reasoning string) *Decision {
	return &Decision{Kind: DecisionExecuteTool, ToolName: toolName, Arguments: args, Reasoning: reasoning}
}

// Strategy 计算分配策略接口；B0-B4 共享（对照公平）。
type Strategy interface {
	// Name 返回策略标识（B0/B1/B2/B3/B4）。
	Name() string
	// SelectNext 依据状态、预算与验证结果选择下一步动作。
	SelectNext(ctx context.Context, state *RunState, budget *BudgetTracker, validations []*ValidatorResult) (*Decision, error)
}

// validationsPassed 判断全部验证是否通过。
func validationsPassed(validations []*ValidatorResult) bool {
	for _, vr := range validations {
		if !vr.Passed {
			return false
		}
	}
	return true
}

// validationFindings 汇总验证失败 findings（修复提示用）。
func validationFindings(validations []*ValidatorResult) string {
	var parts []string
	for _, vr := range validations {
		for _, f := range vr.Findings {
			parts = append(parts, fmt.Sprintf("%s: %s", vr.ValidatorID, f))
		}
	}
	return strings.Join(parts, "; ")
}

// objectiveMessage 构造面向任务目标的 user 消息。
func objectiveMessage(state *RunState) providers.ChatMessage {
	return providers.ChatMessage{Role: "user", Content: state.Task.Objective}
}

// ---- B0 固定单流程 ----

// B0Strategy 固定单流程：一次模型调用后结束，无自由路由、无修复。
type B0Strategy struct{}

// NewB0Strategy 构造 B0 策略。
func NewB0Strategy() Strategy { return B0Strategy{} }

// Name 返回策略标识。
func (B0Strategy) Name() string { return "B0" }

// SelectNext 单次模型调用后 Finish。
func (B0Strategy) SelectNext(_ context.Context, state *RunState, budget *BudgetTracker, _ []*ValidatorResult) (*Decision, error) {
	if budget.ModelCalls() == 0 {
		return NewCallModelDecision([]providers.ChatMessage{objectiveMessage(state)}, "B0: single fixed call"), nil
	}
	return &Decision{Kind: DecisionFinish, Reasoning: "B0: fixed single flow complete"}, nil
}

// ---- B1 单 Agent 闭环 ----

// B1Strategy 单 Agent 闭环：验证失败可一次定向修复（AGI 缺口修复）。
type B1Strategy struct {
	repairUsed bool
}

// NewB1Strategy 构造 B1 策略。
func NewB1Strategy() Strategy { return &B1Strategy{} }

// Name 返回策略标识。
func (s *B1Strategy) Name() string { return "B1" }

// SelectNext 验证失败且未修复过 → 带修复提示再调用一次。
func (s *B1Strategy) SelectNext(_ context.Context, state *RunState, budget *BudgetTracker, validations []*ValidatorResult) (*Decision, error) {
	if budget.ModelCalls() == 0 {
		return NewCallModelDecision([]providers.ChatMessage{objectiveMessage(state)}, "B1: initial call"), nil
	}
	if !s.repairUsed && !validationsPassed(validations) {
		s.repairUsed = true
		msg := "你的上一步产出未通过验证，请修复以下问题并重新输出：\n" + validationFindings(validations)
		return NewCallModelDecision([]providers.ChatMessage{
			{Role: "user", Content: state.Task.Objective},
			{Role: "user", Content: msg},
		}, "B1: one directed repair (AGI gap)"), nil
	}
	return &Decision{Kind: DecisionFinish, Reasoning: "B1: closed loop complete"}, nil
}

// ---- B2 串行角色复用 ----

// b2Roles 固定角色序列（单卡顺序复用同一模型）。
var b2Roles = []string{"router", "planner", "executor", "reviewer"}

// B2Strategy 串行角色复用：按固定顺序推进角色。
type B2Strategy struct {
	roleIdx int
}

// NewB2Strategy 构造 B2 策略。
func NewB2Strategy() Strategy { return &B2Strategy{} }

// Name 返回策略标识。
func (s *B2Strategy) Name() string { return "B2" }

// SelectNext 顺序调用各角色；全部角色完成后 Finish。
func (s *B2Strategy) SelectNext(_ context.Context, state *RunState, budget *BudgetTracker, _ []*ValidatorResult) (*Decision, error) {
	if s.roleIdx >= len(b2Roles) {
		return &Decision{Kind: DecisionFinish, Reasoning: "B2: all roles executed in fixed order"}, nil
	}
	role := b2Roles[s.roleIdx]
	s.roleIdx++
	content := fmt.Sprintf("[role: %s]\n%s", role, state.Task.Objective)
	return NewCallModelDecision([]providers.ChatMessage{{Role: "user", Content: content}},
		fmt.Sprintf("B2: serial role reuse step %d/%d (%s)", s.roleIdx, len(b2Roles), role)), nil
}

// ---- B3 固定分支候选 ----

// B3Strategy 固定分支候选：生成固定数量候选（多采样）后按验证结果择优结束。
type B3Strategy struct {
	candidateCount int
	calls          int
}

// NewB3Strategy 构造 B3 策略；candidateCount 固定（默认 3）。
func NewB3Strategy(candidateCount int) Strategy {
	if candidateCount < 1 {
		candidateCount = 3
	}
	return &B3Strategy{candidateCount: candidateCount}
}

// Name 返回策略标识。
func (s *B3Strategy) Name() string { return "B3" }

// SelectNext 生成固定候选；达到候选数后 Finish（选择规则固定：验证全过即可）。
func (s *B3Strategy) SelectNext(_ context.Context, state *RunState, budget *BudgetTracker, validations []*ValidatorResult) (*Decision, error) {
	if s.calls < s.candidateCount {
		s.calls++
		return NewCallModelDecision([]providers.ChatMessage{
			{Role: "user", Content: fmt.Sprintf("生成第 %d/%d 个候选方案：%s", s.calls, s.candidateCount, state.Task.Objective)},
		}, fmt.Sprintf("B3: fixed branch candidate %d/%d", s.calls, s.candidateCount)), nil
	}
	if validationsPassed(validations) {
		return &Decision{Kind: DecisionFinish, Reasoning: "B3: fixed candidates complete, validation passed"}, nil
	}
	return &Decision{Kind: DecisionRecover, Reasoning: "B3: candidates complete but validation failed"}, nil
}

// ---- B4 BVAR 预算感知验证器路由 ----

// B4Strategy BVAR：依据验证置信度、候选分歧与剩余预算动态选择动作。
// DRA→AAR→AGI→verifier→BVAR 的决策链收敛为 SelectNext 的确定性分支。
type B4Strategy struct {
	statefulCalls int
}

// NewB4Strategy 构造 B4 策略。
func NewB4Strategy() Strategy { return &B4Strategy{} }

// Name 返回策略标识。
func (s *B4Strategy) Name() string { return "B4" }

// SelectNext BVAR 决策：
//   - 预算仅够 1 次调用 → 降级 Finish（确定性收尾）
//   - 验证失败且剩余预算 ≥ 2 → Recover（repair 语义）
//   - 尚未产生候选 → 生成候选（分歧采样）
//   - 验证全过 → Finish
func (s *B4Strategy) SelectNext(_ context.Context, state *RunState, budget *BudgetTracker, validations []*ValidatorResult) (*Decision, error) {
	remaining := budget.budget.MaxModelCalls - budget.ModelCalls()
	if remaining <= 1 {
		return &Decision{Kind: DecisionFinish, Reasoning: "B4: budget tight, deterministic finish"}, nil
	}
	if s.statefulCalls == 0 {
		s.statefulCalls++
		return NewCallModelDecision([]providers.ChatMessage{objectiveMessage(state)},
			"B4: initial candidate generation"), nil
	}
	if !validationsPassed(validations) {
		return &Decision{Kind: DecisionRecover, Reasoning: "B4: verification gap, budget-aware repair: " + validationFindings(validations)}, nil
	}
	return &Decision{Kind: DecisionFinish, Reasoning: "B4: validation passed, finish"}, nil
}
