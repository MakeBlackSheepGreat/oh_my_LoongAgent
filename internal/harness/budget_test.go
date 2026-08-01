package harness

import (
	"strings"
	"testing"
	"time"
)

func TestBudgetTracker_ModelCallLimit(t *testing.T) {
	now := time.Now()
	b := NewBudgetTracker(Budget{MaxModelCalls: 2, MaxToolCalls: 5, MaxRuntimeSeconds: 60}, now)
	if err := b.CheckModelCall(); err != nil {
		t.Fatalf("first call should pass: %v", err)
	}
	b.RecordModelCall()
	if err := b.CheckModelCall(); err != nil {
		t.Fatalf("second call should pass: %v", err)
	}
	b.RecordModelCall()
	err := b.CheckModelCall()
	he, ok := err.(*HarnessError)
	if !ok || he.Code != ErrCodeBudgetExceeded {
		t.Fatalf("expected BUDGET_EXCEEDED, got %v", err)
	}
	if b.StopReason() == "" || !strings.Contains(b.StopReason(), "model_calls") {
		t.Fatalf("expected auditable stop reason, got %q", b.StopReason())
	}
}

func TestBudgetTracker_ToolCallLimit(t *testing.T) {
	now := time.Now()
	b := NewBudgetTracker(Budget{MaxModelCalls: 10, MaxToolCalls: 1, MaxRuntimeSeconds: 60}, now)
	if err := b.CheckToolCall(); err != nil {
		t.Fatalf("first tool call should pass: %v", err)
	}
	b.RecordToolCall()
	err := b.CheckToolCall()
	if err == nil || !strings.Contains(err.Error(), "tool_calls") {
		t.Fatalf("expected tool_calls budget error, got %v", err)
	}
}

func TestBudgetTracker_RuntimeLimit(t *testing.T) {
	now := time.Now()
	b := NewBudgetTracker(Budget{MaxModelCalls: 10, MaxToolCalls: 10, MaxRuntimeSeconds: 5}, now)
	if err := b.CheckRuntime(now.Add(4 * time.Second)); err != nil {
		t.Fatalf("within limit should pass: %v", err)
	}
	err := b.CheckRuntime(now.Add(6 * time.Second))
	if err == nil || !strings.Contains(err.Error(), "runtime_seconds") {
		t.Fatalf("expected runtime budget error, got %v", err)
	}
}

func TestBudgetTracker_CostLimit(t *testing.T) {
	now := time.Now()
	maxCost := 0.001
	b := NewBudgetTracker(Budget{
		MaxModelCalls: 10, MaxToolCalls: 10, MaxRuntimeSeconds: 60, MaxCostUSD: &maxCost,
	}, now)
	if err := b.AddCost(0.0009); err != nil {
		t.Fatalf("within cost should pass: %v", err)
	}
	err := b.AddCost(0.0002)
	if err == nil || !strings.Contains(err.Error(), "cost_usd") {
		t.Fatalf("expected cost budget error, got %v", err)
	}
}

func TestBudgetTracker_StopIsTerminal(t *testing.T) {
	now := time.Now()
	b := NewBudgetTracker(Budget{MaxModelCalls: 1, MaxToolCalls: 1, MaxRuntimeSeconds: 60}, now)
	_ = b.CheckModelCall()
	b.RecordModelCall()
	_ = b.CheckModelCall() // 触发停止
	// 停止后所有检查立即失败，且不改变停止原因
	if err := b.CheckRuntime(now); err == nil {
		t.Fatal("runtime check should fail after stop")
	}
	if err := b.CheckToolCall(); err == nil {
		t.Fatal("tool check should fail after stop")
	}
	if err := b.AddCost(0.01); err == nil {
		t.Fatal("cost add should fail after stop")
	}
	if b.ModelCalls() != 1 {
		t.Fatalf("model calls should remain 1, got %d", b.ModelCalls())
	}
}

func TestBudgetTracker_Counters(t *testing.T) {
	now := time.Now()
	b := NewBudgetTracker(Budget{MaxModelCalls: 10, MaxToolCalls: 10, MaxRuntimeSeconds: 60}, now)
	b.RecordModelCall()
	b.RecordModelCall()
	b.RecordToolCall()
	if b.ModelCalls() != 2 || b.ToolCalls() != 1 {
		t.Fatalf("counters wrong: model=%d tool=%d", b.ModelCalls(), b.ToolCalls())
	}
	if !b.StartedAt().Equal(now) {
		t.Fatal("started_at mismatch")
	}
}
