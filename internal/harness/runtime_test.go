package harness

import (
	"context"
	"sync"
	"testing"

	"slim-agent/internal/providers"
)

// fakeProvider 可编程的测试 Provider。
type fakeProvider struct {
	mu    sync.Mutex
	calls int
	resp  *providers.ChatResponse
	err   error
}

func (f *fakeProvider) Chat(_ context.Context, _ *providers.ChatRequest) (*providers.ChatResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func (f *fakeProvider) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// recordingSink 记录计量回调的 UsageSink。
type recordingSink struct {
	mu    sync.Mutex
	calls []ModelUsage
}

func (s *recordingSink) RecordModelCall(_ context.Context, u ModelUsage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, u)
	return nil
}

func (s *recordingSink) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *recordingSink) Last() ModelUsage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[len(s.calls)-1]
}

// alwaysCallStrategy 始终调用模型的策略（触发预算停止用）。
type alwaysCallStrategy struct{}

func (alwaysCallStrategy) Name() string { return "TEST_ALWAYS_CALL" }
func (alwaysCallStrategy) SelectNext(_ context.Context, state *RunState, _ *BudgetTracker, _ []*ValidatorResult) (*Decision, error) {
	return NewCallModelDecision([]providers.ChatMessage{{Role: "user", Content: "keep going"}}, "test"), nil
}

// waitThenFinishStrategy 先等待审批再结束。
type waitThenFinishStrategy struct{ waited bool }

func (s *waitThenFinishStrategy) Name() string { return "TEST_WAIT" }
func (s *waitThenFinishStrategy) SelectNext(_ context.Context, _ *RunState, _ *BudgetTracker, _ []*ValidatorResult) (*Decision, error) {
	if !s.waited {
		s.waited = true
		return &Decision{Kind: DecisionWaitApproval, Reasoning: "needs approval"}, nil
	}
	return &Decision{Kind: DecisionFinish, Reasoning: "approved"}, nil
}

// toolThenFinishStrategy 先执行工具再结束。
type toolThenFinishStrategy struct{ usedTool bool }

func (s *toolThenFinishStrategy) Name() string { return "TEST_TOOL" }
func (s *toolThenFinishStrategy) SelectNext(_ context.Context, _ *RunState, _ *BudgetTracker, _ []*ValidatorResult) (*Decision, error) {
	if !s.usedTool {
		s.usedTool = true
		return NewToolDecision("simulate.fs", map[string]any{"op": "put", "path": "a.txt", "content": "hi"}, "test tool"), nil
	}
	return &Decision{Kind: DecisionFinish, Reasoning: "done"}, nil
}

func newRuntimeStore(t *testing.T, taskID string) (*HarnessStore, *RunState) {
	t.Helper()
	s := newTestStore(t)
	task, err := NewTaskContract(taskID, "lit_search", "organize papers into categories")
	if err != nil {
		t.Fatalf("NewTaskContract: %v", err)
	}
	task.Budget = Budget{MaxModelCalls: 5, MaxToolCalls: 5, MaxRuntimeSeconds: 60}
	state, err := s.CreateRun(task)
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	return s, state
}

func newRuntimeOptions(provider providers.Provider) RuntimeOptions {
	return RuntimeOptions{
		Provider:   provider,
		ProviderID: "test-provider",
		ModelID:    "test-model",
	}
}

func TestRuntimeRun_B0Completes(t *testing.T) {
	s, state := newRuntimeStore(t, "run_rt1")
	fp := &fakeProvider{resp: &providers.ChatResponse{
		Content: "summary",
		Usage:   providers.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		Model:   "test-model",
	}}
	sink := &recordingSink{}
	opts := newRuntimeOptions(fp)
	opts.UsageSink = sink
	r, err := NewHarnessRuntime(s, opts)
	if err != nil {
		t.Fatalf("NewHarnessRuntime: %v", err)
	}
	final, err := r.Run(context.Background(), state.RunID, NewB0Strategy())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", final.Status)
	}
	if final.Usage["model_calls"] != 1 {
		t.Fatalf("expected 1 model call, got %v", final.Usage["model_calls"])
	}
	if fp.CallCount() != 1 {
		t.Fatalf("expected 1 provider call, got %d", fp.CallCount())
	}
	if sink.Count() != 1 {
		t.Fatalf("expected 1 metering record, got %d", sink.Count())
	}
	last := sink.Last()
	if last.InputTokens != 10 || last.OutputTokens != 5 {
		t.Fatalf("unexpected usage: %+v", last)
	}
	// 终态事件已写库
	events, err := s.EventsAfter(state.RunID, 0)
	if err != nil {
		t.Fatalf("EventsAfter: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected events recorded")
	}
}

func TestRuntimeRun_BudgetExhausted(t *testing.T) {
	s := newTestStore(t)
	task, err := NewTaskContract("run_rt2", "lit_search", "organize papers")
	if err != nil {
		t.Fatalf("NewTaskContract: %v", err)
	}
	task.Budget = Budget{MaxModelCalls: 1, MaxToolCalls: 5, MaxRuntimeSeconds: 60}
	state, err := s.CreateRun(task)
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	fp := &fakeProvider{resp: &providers.ChatResponse{Content: "x"}}
	r, err := NewHarnessRuntime(s, newRuntimeOptions(fp))
	if err != nil {
		t.Fatalf("NewHarnessRuntime: %v", err)
	}
	final, err := r.Run(context.Background(), state.RunID, alwaysCallStrategy{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", final.Status)
	}
	if final.LastError == nil || final.LastError.Code != ErrCodeBudgetExceeded {
		t.Fatalf("expected BUDGET_EXCEEDED error, got %+v", final.LastError)
	}
}

func TestRuntimeRun_ProviderFailure(t *testing.T) {
	s, state := newRuntimeStore(t, "run_rt3")
	fp := &fakeProvider{err: context.DeadlineExceeded}
	r, err := NewHarnessRuntime(s, newRuntimeOptions(fp))
	if err != nil {
		t.Fatalf("NewHarnessRuntime: %v", err)
	}
	final, err := r.Run(context.Background(), state.RunID, NewB0Strategy())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", final.Status)
	}
	if final.LastError == nil || final.LastError.Code != ErrCodeProviderUnavailable {
		t.Fatalf("expected PROVIDER_UNAVAILABLE error, got %+v", final.LastError)
	}
	if !final.LastError.Recoverable {
		t.Fatal("provider failure should be marked recoverable")
	}
}

func TestRuntimeRun_WaitApproval(t *testing.T) {
	s, state := newRuntimeStore(t, "run_rt4")
	fp := &fakeProvider{resp: &providers.ChatResponse{Content: "x"}}
	r, err := NewHarnessRuntime(s, newRuntimeOptions(fp))
	if err != nil {
		t.Fatalf("NewHarnessRuntime: %v", err)
	}
	final, err := r.Run(context.Background(), state.RunID, &waitThenFinishStrategy{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Status != StatusWaiting {
		t.Fatalf("expected waiting, got %s", final.Status)
	}
}

func TestRuntimeRun_ContextCancelled(t *testing.T) {
	s, state := newRuntimeStore(t, "run_rt5")
	fp := &fakeProvider{resp: &providers.ChatResponse{Content: "x"}}
	r, err := NewHarnessRuntime(s, newRuntimeOptions(fp))
	if err != nil {
		t.Fatalf("NewHarnessRuntime: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	final, err := r.Run(ctx, state.RunID, alwaysCallStrategy{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", final.Status)
	}
	if final.LastError == nil || final.LastError.Code != "CONTEXT_CANCELLED" {
		t.Fatalf("expected CONTEXT_CANCELLED error, got %+v", final.LastError)
	}
}

func TestRuntimeRun_ToolExecution(t *testing.T) {
	s, state := newRuntimeStore(t, "run_rt6")
	fp := &fakeProvider{resp: &providers.ChatResponse{Content: "x"}}
	opts := newRuntimeOptions(fp)
	opts.Tools = NewToolGovernor(Policy{
		AllowedToolNames:   []string{"simulate.fs"},
		AllowedPermissions: []Permission{PermWriteWorkspace},
	})
	opts.Tools.Register(NewSimulateFSTool())
	r, err := NewHarnessRuntime(s, opts)
	if err != nil {
		t.Fatalf("NewHarnessRuntime: %v", err)
	}
	final, err := r.Run(context.Background(), state.RunID, &toolThenFinishStrategy{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", final.Status)
	}
	if final.Usage["tool_calls"] != 1 {
		t.Fatalf("expected 1 tool call, got %v", final.Usage["tool_calls"])
	}
}

func TestRuntimeRun_ValidationWiring(t *testing.T) {
	s, state := newRuntimeStore(t, "run_rt7")
	fp := &fakeProvider{resp: &providers.ChatResponse{Content: "x"}}
	opts := newRuntimeOptions(fp)
	reg := NewValidatorRegistry()
	reg.Register(NewArtifactExistsValidator())
	opts.Validators = reg
	r, err := NewHarnessRuntime(s, opts)
	if err != nil {
		t.Fatalf("NewHarnessRuntime: %v", err)
	}
	final, err := r.Run(context.Background(), state.RunID, NewB0Strategy())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if final.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", final.Status)
	}
	results, err := s.ValidatorResults(state.RunID)
	if err != nil {
		t.Fatalf("ValidatorResults: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected validator results persisted")
	}
}
