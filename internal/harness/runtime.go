package harness

import (
	"context"
	"errors"
	"fmt"
	"time"

	"slim-agent/internal/providers"
)

// ModelUsage 单次模型调用计量（供 UsageSink 转发到 task4 MeterRecorder）。
type ModelUsage struct {
	RunID            string
	ProviderID       string
	ModelID          string
	InputTokens      int
	OutputTokens     int
	LatencyMs        int
	EstimatedCostUSD float64
}

// UsageSink 计量回调接口；由 workbench 端适配（nil 时跳过计量）。
type UsageSink interface {
	RecordModelCall(ctx context.Context, usage ModelUsage) error
}

// RuntimeOptions 运行时装配项。
type RuntimeOptions struct {
	// Provider 模型 Provider（必填）。
	Provider providers.Provider
	// ProviderID Provider 标识（计量用）。
	ProviderID string
	// ModelID 模型标识（计量用）。
	ModelID string
	// Tools 受限工具执行器（可 nil，跳过工具）。
	Tools *ToolGovernor
	// Validators 验证器注册表（可 nil，跳过验证）。
	Validators *ValidatorRegistry
	// UsageSink 计量回调（可选）。
	UsageSink UsageSink
	// OnEvent 事件转发回调（可选，如 workbench EventHub）。
	OnEvent func(*Event)
	// Now 可注入时钟（测试用）；nil 用 time.Now().UTC()。
	Now func() time.Time
}

// HarnessRuntime 执行循环：策略选动作→预算检查→执行→事件→状态转换→验证。
// 状态转换受 store.TransitionRun 版本守卫保护。
type HarnessRuntime struct {
	store *HarnessStore
	opts  RuntimeOptions
}

// NewHarnessRuntime 构造运行时。
func NewHarnessRuntime(store *HarnessStore, opts RuntimeOptions) (*HarnessRuntime, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	if opts.Provider == nil {
		return nil, errors.New("provider is required")
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	return &HarnessRuntime{
		store: store,
		opts:  opts,
	}, nil
}

// Run 对 runID 执行策略直至终态（completed/failed/cancelled/waiting）。
func (r *HarnessRuntime) Run(ctx context.Context, runID string, strategy Strategy) (*RunState, error) {
	state, err := r.store.GetRun(runID)
	if err != nil {
		return nil, err
	}
	// created → running
	state, err = r.transition(ctx, state, StatusRunning, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	budget := NewBudgetTracker(state.Task.Budget, r.opts.Now())
	// 循环迭代上限：模型+工具预算的常数倍，防止策略死循环。
	maxIterations := state.Task.Budget.MaxModelCalls + state.Task.Budget.MaxToolCalls + 10
	for i := 0; i < maxIterations; i++ {
		if err := ctx.Err(); err != nil {
			return r.fail(ctx, state, "CONTEXT_CANCELLED", "run cancelled by context", false, budget.StopReason())
		}
		if err := budget.CheckRuntime(r.opts.Now()); err != nil {
			return r.fail(ctx, state, ErrCodeBudgetExceeded, budget.StopReason(), false, "")
		}

		validations, vErr := r.validate(ctx, state)
		if vErr != nil {
			return r.fail(ctx, state, ErrCodeInternal, fmt.Sprintf("validation failed: %v", vErr), false, "")
		}

		decision, dErr := strategy.SelectNext(ctx, state, budget, validations)
		if dErr != nil {
			return r.fail(ctx, state, ErrCodeInternal, fmt.Sprintf("strategy error: %v", dErr), false, "")
		}

		switch decision.Kind {
		case DecisionCallModel:
			state, err = r.callModel(ctx, state, budget, decision)
		case DecisionExecuteTool:
			state, err = r.executeTool(ctx, state, budget, decision)
		case DecisionWaitApproval:
			state, err = r.waitApproval(ctx, state)
			if err == nil {
				return state, nil // 等待人工审批，退出循环（外部批准后重新 Run）
			}
		case DecisionRecover:
			state, err = r.handleRecover(ctx, state, budget, decision)
		case DecisionFinish:
			state, err = r.transition(ctx, state, StatusCompleted, nil, nil, nil)
			if err == nil {
				r.emit(&Event{Sequence: state.StateVersion, RunID: state.RunID, Kind: "run_completed",
				Message: "run completed", Payload: map[string]any{"strategy": strategy.Name()}, Timestamp: r.opts.Now()})
				return state, nil
			}
		default:
			err = NewHarnessError(ErrCodeInternal, fmt.Sprintf("unknown decision kind: %s", decision.Kind), nil)
		}
		if err != nil {
			return r.failFromError(ctx, state, err)
		}
	}
	return r.fail(ctx, state, ErrCodeBudgetExceeded, "max iterations reached without completion", true, "")
}

// failFromError 按错误码终止运行：预算/权限/Provider 错误保留原错误码与可恢复标记。
func (r *HarnessRuntime) failFromError(ctx context.Context, state *RunState, err error) (*RunState, error) {
	var he *HarnessError
	code := ErrCodeInternal
	if errors.As(err, &he) {
		code = he.Code
	}
	recoverable := code == ErrCodeProviderUnavailable
	return r.fail(ctx, state, code, err.Error(), recoverable, "")
}

// callModel 执行一次模型调用：预算检查→Chat→计量→状态转换。
func (r *HarnessRuntime) callModel(ctx context.Context, state *RunState, budget *BudgetTracker, decision *Decision) (*RunState, error) {
	if err := budget.CheckModelCall(); err != nil {
		return state, err
	}
	req := &providers.ChatRequest{Messages: decision.Messages}
	if err := req.Validate(); err != nil {
		return state, err
	}
	start := r.opts.Now()
	resp, err := r.opts.Provider.Chat(ctx, req)
	latency := int(r.opts.Now().Sub(start).Milliseconds())
	if err != nil {
		er, _ := NewErrorRecord("PROVIDER_CALL_FAILED", err.Error(), true)
		_, _ = r.store.RecordError(state.RunID, er)
		r.emitEvent(state, "model_call_failed", "provider call failed", map[string]any{"error": err.Error()})
		return state, NewHarnessError(ErrCodeProviderUnavailable, "provider call failed", err)
	}

	budget.RecordModelCall()
	estCost := estimateCost(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	_ = budget.AddCost(estCost)

	next := *state
	next.Usage = copyUsage(state.Usage)
	next.Usage["model_calls"] = float64(budget.ModelCalls())
	next.Usage["tool_calls"] = float64(budget.ToolCalls())
	next.Usage["cost_usd"] = budget.CostUSD()
	next.Usage["prompt_tokens"] += float64(resp.Usage.PromptTokens)
	next.Usage["completion_tokens"] += float64(resp.Usage.CompletionTokens)

	// 计量回调（task4 接入点）
	if r.opts.UsageSink != nil {
		_ = r.opts.UsageSink.RecordModelCall(ctx, ModelUsage{
			RunID: state.RunID, ProviderID: r.opts.ProviderID, ModelID: r.opts.ModelID,
			InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens,
			LatencyMs: latency, EstimatedCostUSD: estCost,
		})
	}

	newState, err := r.transition(ctx, &next, StatusRunning, nil, nil, nil)
	if err != nil {
		return state, err
	}
	r.emitEvent(newState, "model_call", "model call completed", map[string]any{
		"model": r.opts.ModelID, "prompt_tokens": resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens, "latency_ms": latency,
	})
	return newState, nil
}

// executeTool 执行一次受限工具调用：预算→审批→Governor→事件。
func (r *HarnessRuntime) executeTool(ctx context.Context, state *RunState, budget *BudgetTracker, decision *Decision) (*RunState, error) {
	if r.opts.Tools == nil {
		return state, errors.New("tools not configured")
	}
	if err := budget.CheckToolCall(); err != nil {
		return state, err
	}
	action, err := NewActionContract("act_"+fmt.Sprint(budget.ToolCalls()+1), state.RunID, decision.ToolName)
	if err != nil {
		return state, err
	}
	action.Arguments = decision.Arguments
	if r.opts.Tools.RequiresApproval(action) {
		return state, errors.New("approval required but not granted")
	}
	result, err := r.opts.Tools.Execute(ctx, action, decision.Arguments)
	if err != nil {
		er, _ := NewErrorRecord("TOOL_EXECUTION_FAILED", err.Error(), true)
		_, _ = r.store.RecordError(state.RunID, er)
		return state, err
	}
	budget.RecordToolCall()
	next := *state
	next.Usage = copyUsage(state.Usage)
	next.Usage["tool_calls"] = float64(budget.ToolCalls())
	next.Usage["cost_usd"] = budget.CostUSD()
	newState, err := r.transition(ctx, &next, StatusRunning, nil, nil, nil)
	if err != nil {
		return state, err
	}
	kind := "tool_call"
	message := "tool call completed"
	if !result.OK {
		kind = "tool_call_failed"
		message = result.Error
	}
	r.emitEvent(newState, kind, message, map[string]any{
		"tool": decision.ToolName, "ok": result.OK, "output": result.Output,
		"side_effects": result.SideEffects, "rollback": result.RollbackSummary,
	})
	return newState, nil
}

// waitApproval 状态转为 waiting（等待人工审批）。
func (r *HarnessRuntime) waitApproval(ctx context.Context, state *RunState) (*RunState, error) {
	newState, err := r.transition(ctx, state, StatusWaiting, nil, nil, nil)
	if err != nil {
		return state, err
	}
	r.emitEvent(newState, "waiting_approval", "run waiting for human approval", nil)
	return newState, nil
}

// handleRecover 处理恢复决策：repair 语义通过重新调用模型实现（带修复提示）。
func (r *HarnessRuntime) handleRecover(ctx context.Context, state *RunState, budget *BudgetTracker, decision *Decision) (*RunState, error) {
	if decision.Messages != nil && len(decision.Messages) > 0 {
		return r.callModel(ctx, state, budget, decision)
	}
	// 无修复消息则视为不可恢复 → failed
	return r.fail(ctx, state, ErrCodeInternal, "recovery requested without repair strategy", false, "")
}

// validate 运行全部验证器；将结果落库并返回。
func (r *HarnessRuntime) validate(ctx context.Context, state *RunState) ([]*ValidatorResult, error) {
	if r.opts.Validators == nil {
		return nil, nil
	}
	artifacts, err := r.loadArtifacts(ctx, state)
	if err != nil {
		return nil, err
	}
	results, err := r.opts.Validators.RunAll(ctx, state, artifacts)
	if err != nil {
		return nil, err
	}
	for _, vr := range results {
		_, _ = r.store.RecordValidatorResult(vr)
	}
	return results, nil
}

// loadArtifacts 加载运行工件并注入内容（供验证器内存校验）。
func (r *HarnessRuntime) loadArtifacts(ctx context.Context, state *RunState) (map[string]*Artifact, error) {
	artifacts := make(map[string]*Artifact, len(state.ArtifactIDs))
	for _, id := range state.ArtifactIDs {
		art, err := r.store.GetArtifact(state.RunID, id)
		if err != nil {
			return nil, err
		}
		content, err := r.store.ReadArtifact(state.RunID, id)
		if err != nil {
			return nil, err
		}
		art.Metadata = map[string]any{"_bytes": content}
		artifacts[id] = art
	}
	return artifacts, nil
}

// fail 终止运行（failed）并记录错误。
func (r *HarnessRuntime) fail(ctx context.Context, state *RunState, code, message string, recoverable bool, detail string) (*RunState, error) {
	er, _ := NewErrorRecord(code, message, recoverable)
	if detail != "" {
		er.Details["stop_reason"] = detail
	}
	_, _ = r.store.RecordError(state.RunID, er)
	newState, err := r.transition(ctx, state, StatusFailed, nil, nil, er)
	if err != nil {
		return state, err
	}
	r.emitEvent(newState, "run_failed", message, map[string]any{"code": code})
	return newState, nil
}

// transition 带版本守卫的状态转换。
func (r *HarnessRuntime) transition(ctx context.Context, state *RunState, status RunStatus,
	activeActionID *string, artifactIDs []string, lastError *ErrorRecord) (*RunState, error) {
	expected := state.StateVersion
	return r.store.TransitionRun(state.RunID, status, &expected, activeActionID, artifactIDs, state.Usage, lastError)
}

// emitEvent 事件转发（持久化由 store 内部 AppendEvent 负责；此处通知外部）。
func (r *HarnessRuntime) emitEvent(state *RunState, kind, message string, payload map[string]any) {
	r.emit(&Event{Sequence: state.StateVersion, RunID: state.RunID, Kind: kind, Message: message, Payload: payload, Timestamp: r.opts.Now()})
}

// emit 通过回调转发事件。
func (r *HarnessRuntime) emit(ev *Event) {
	if r.opts.OnEvent != nil {
		r.opts.OnEvent(ev)
	}
}

// copyUsage 深拷贝 usage map。
func copyUsage(src map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// estimateCost 简易成本估算：输入 0.1 美元/百万 token、输出 0.3 美元/百万 token。
func estimateCost(inputTokens, outputTokens int) float64 {
	return float64(inputTokens)*0.1/1e6 + float64(outputTokens)*0.3/1e6
}
