package harness

import (
	"fmt"
	"time"
)

// BudgetTracker 预算跟踪器：在运行时检查模型调用/工具调用/运行时长/成本四项硬上限。
// 时间复杂度 O(1) 每次检查。
type BudgetTracker struct {
	budget     Budget
	modelCalls int
	toolCalls  int
	cost       float64
	startedAt  time.Time
	stopped    bool
	stopReason string
}

// NewBudgetTracker 构造预算跟踪器。
func NewBudgetTracker(budget Budget, now time.Time) *BudgetTracker {
	return &BudgetTracker{budget: budget, startedAt: now}
}

// StartedAt 返回运行开始时间。
func (b *BudgetTracker) StartedAt() time.Time { return b.startedAt }

// ModelCalls 返回当前模型调用次数。
func (b *BudgetTracker) ModelCalls() int { return b.modelCalls }

// ToolCalls 返回当前工具调用次数。
func (b *BudgetTracker) ToolCalls() int { return b.toolCalls }

// CostUSD 返回当前累计成本。
func (b *BudgetTracker) CostUSD() float64 { return b.cost }

// StopReason 返回停止原因；未停止时为空。
func (b *BudgetTracker) StopReason() string { return b.stopReason }

// CheckModelCall 检查模型调用预算；超限标记停止并返回 BUDGET_EXCEEDED。
func (b *BudgetTracker) CheckModelCall() error {
	if b.stopped {
		return ErrBudgetExceeded(b.stopReason)
	}
	if b.modelCalls+1 > b.budget.MaxModelCalls {
		return b.stop(ErrBudgetExceeded(
			fmt.Sprintf("model_calls budget exceeded: %d/%d", b.modelCalls+1, b.budget.MaxModelCalls)))
	}
	return nil
}

// CheckToolCall 检查工具调用预算。
func (b *BudgetTracker) CheckToolCall() error {
	if b.stopped {
		return ErrBudgetExceeded(b.stopReason)
	}
	if b.toolCalls+1 > b.budget.MaxToolCalls {
		return b.stop(ErrBudgetExceeded(
			fmt.Sprintf("tool_calls budget exceeded: %d/%d", b.toolCalls+1, b.budget.MaxToolCalls)))
	}
	return nil
}

// CheckRuntime 检查运行时长预算；now 由调用方注入（可测试）。
func (b *BudgetTracker) CheckRuntime(now time.Time) error {
	if b.stopped {
		return ErrBudgetExceeded(b.stopReason)
	}
	elapsed := int(now.Sub(b.startedAt).Seconds())
	if elapsed > b.budget.MaxRuntimeSeconds {
		return b.stop(ErrBudgetExceeded(
			fmt.Sprintf("runtime_seconds budget exceeded: %d/%d", elapsed, b.budget.MaxRuntimeSeconds)))
	}
	return nil
}

// AddCost 累加成本并检查成本上限。
func (b *BudgetTracker) AddCost(cost float64) error {
	if b.stopped {
		return ErrBudgetExceeded(b.stopReason)
	}
	b.cost += cost
	if b.budget.MaxCostUSD != nil && b.cost > *b.budget.MaxCostUSD {
		return b.stop(ErrBudgetExceeded(
			fmt.Sprintf("cost_usd budget exceeded: %.6f/%.6f", b.cost, *b.budget.MaxCostUSD)))
	}
	return nil
}

// RecordModelCall 记录一次模型调用。
func (b *BudgetTracker) RecordModelCall() { b.modelCalls++ }

// RecordToolCall 记录一次工具调用。
func (b *BudgetTracker) RecordToolCall() { b.toolCalls++ }

func (b *BudgetTracker) stop(err error) error {
	b.stopped = true
	b.stopReason = err.Error()
	return err
}
