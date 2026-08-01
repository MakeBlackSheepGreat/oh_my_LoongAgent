package harness

import (
	"context"
	"strings"
	"testing"
	"time"
)

func strategyRun(t *testing.T, taskID string, maxModelCalls int) (*RunState, *BudgetTracker) {
	t.Helper()
	task, err := NewTaskContract(taskID, "lit_search", "organize papers into categories")
	if err != nil {
		t.Fatalf("NewTaskContract: %v", err)
	}
	task.Budget = Budget{MaxModelCalls: maxModelCalls, MaxToolCalls: 20, MaxRuntimeSeconds: 60}
	state, err := NewRunState(task)
	if err != nil {
		t.Fatalf("NewRunState: %v", err)
	}
	budget := NewBudgetTracker(state.Task.Budget, time.Now())
	return state, budget
}

func passedResults() []*ValidatorResult {
	return []*ValidatorResult{{ValidatorID: "artifact_exists", Passed: true}}
}

func failedResults() []*ValidatorResult {
	return []*ValidatorResult{{ValidatorID: "artifact_exists", Passed: false, Findings: []string{"artifact missing: x"}}}
}

func TestB0Strategy_FixedSingleFlow(t *testing.T) {
	s := NewB0Strategy()
	state, budget := strategyRun(t, "run_b0", 5)
	d1, err := s.SelectNext(context.Background(), state, budget, nil)
	if err != nil || d1.Kind != DecisionCallModel {
		t.Fatalf("first decision should be call_model, got %v err=%v", d1.Kind, err)
	}
	budget.RecordModelCall()
	d2, err := s.SelectNext(context.Background(), state, budget, passedResults())
	if err != nil || d2.Kind != DecisionFinish {
		t.Fatalf("second decision should be finish, got %v err=%v", d2.Kind, err)
	}
	if s.Name() != "B0" {
		t.Fatalf("name: %s", s.Name())
	}
}

func TestB1Strategy_OneDirectedRepair(t *testing.T) {
	s := NewB1Strategy()
	state, budget := strategyRun(t, "run_b1", 5)
	// 首次调用
	d1, _ := s.SelectNext(context.Background(), state, budget, nil)
	if d1.Kind != DecisionCallModel {
		t.Fatalf("first should be call_model, got %s", d1.Kind)
	}
	budget.RecordModelCall()
	// 验证失败 → 一次定向修复
	d2, _ := s.SelectNext(context.Background(), state, budget, failedResults())
	if d2.Kind != DecisionCallModel {
		t.Fatalf("second should be repair call, got %s", d2.Kind)
	}
	if len(d2.Messages) != 2 || len(d2.Messages[1].Content) == 0 {
		t.Fatalf("repair message should include findings, got %+v", d2.Messages)
	}
	budget.RecordModelCall()
	// 修复后仍未过验证 → 不再修复，直接结束
	d3, _ := s.SelectNext(context.Background(), state, budget, failedResults())
	if d3.Kind != DecisionFinish {
		t.Fatalf("third should be finish (repair used), got %s", d3.Kind)
	}
}

func TestB2Strategy_SerialRoleReuse(t *testing.T) {
	s := NewB2Strategy()
	state, budget := strategyRun(t, "run_b2", 10)
	for i := 0; i < len(b2Roles); i++ {
		d, err := s.SelectNext(context.Background(), state, budget, nil)
		if err != nil || d.Kind != DecisionCallModel {
			t.Fatalf("step %d should be call_model, got %v err=%v", i, d.Kind, err)
		}
		budget.RecordModelCall()
	}
	d, _ := s.SelectNext(context.Background(), state, budget, passedResults())
	if d.Kind != DecisionFinish {
		t.Fatalf("after all roles should finish, got %s", d.Kind)
	}
	if budget.ModelCalls() != len(b2Roles) {
		t.Fatalf("expected %d model calls, got %d", len(b2Roles), budget.ModelCalls())
	}
}

func TestB3Strategy_FixedCandidates(t *testing.T) {
	s := NewB3Strategy(3)
	state, budget := strategyRun(t, "run_b3", 10)
	for i := 0; i < 3; i++ {
		d, _ := s.SelectNext(context.Background(), state, budget, nil)
		if d.Kind != DecisionCallModel {
			t.Fatalf("candidate %d should be call_model, got %s", i, d.Kind)
		}
		budget.RecordModelCall()
	}
	// 候选齐备且验证通过 → 结束
	d, _ := s.SelectNext(context.Background(), state, budget, passedResults())
	if d.Kind != DecisionFinish {
		t.Fatalf("expected finish on passed validation, got %s", d.Kind)
	}
	// 候选齐备但验证失败 → recover
	d, _ = s.SelectNext(context.Background(), state, budget, failedResults())
	if d.Kind != DecisionRecover {
		t.Fatalf("expected recover on failed validation, got %s", d.Kind)
	}
}

func TestB4Strategy_BudgetAware(t *testing.T) {
	s := NewB4Strategy()
	state, budget := strategyRun(t, "run_b4", 2)
	// 预算充足（剩余 2）→ 生成候选
	d, _ := s.SelectNext(context.Background(), state, budget, nil)
	if d.Kind != DecisionCallModel {
		t.Fatalf("first should be call_model, got %s", d.Kind)
	}
	budget.RecordModelCall()
	// 剩余 1 → 预算紧张，确定性收尾
	d, _ = s.SelectNext(context.Background(), state, budget, failedResults())
	if d.Kind != DecisionFinish {
		t.Fatalf("expected deterministic finish when budget tight, got %s", d.Kind)
	}
}

func TestB4Strategy_VerificationGapRecover(t *testing.T) {
	s := NewB4Strategy()
	state, budget := strategyRun(t, "run_b4b", 5)
	d1, _ := s.SelectNext(context.Background(), state, budget, nil)
	if d1.Kind != DecisionCallModel {
		t.Fatalf("first should be call_model, got %s", d1.Kind)
	}
	budget.RecordModelCall()
	// 验证失败且预算充足（剩余 4）→ recover（repair 语义）
	d, _ := s.SelectNext(context.Background(), state, budget, failedResults())
	if d.Kind != DecisionRecover {
		t.Fatalf("expected recover on gap, got %s", d.Kind)
	}
	if !strings.Contains(d.Reasoning, "verification gap") {
		t.Fatalf("expected verification gap reasoning, got %s", d.Reasoning)
	}
}
