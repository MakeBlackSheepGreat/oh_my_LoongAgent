package harness

import (
	"context"
	"strings"
	"testing"
	"time"
)

// inMemArtifact 构造内存工件（绕过存储校验，供验证器测试）。
func inMemArtifact(id, kind string, content []byte, refs []any) *Artifact {
	md := map[string]any{"_bytes": content}
	if refs != nil {
		md["references"] = refs
	}
	return &Artifact{
		ArtifactID: id,
		RunID:      "run_x",
		Kind:       kind,
		SHA256:     strings.Repeat("a", 64),
		SizeBytes:  len(content),
		Revision:   1,
		StorageURI: "mem://test",
		Metadata:   md,
		CreatedAt:  time.Now().UTC(),
	}
}

func newValidatedRun(t *testing.T, taskID string, artifactIDs []string, usage map[string]float64) *RunState {
	t.Helper()
	task, err := NewTaskContract(taskID, "lit_search", "find papers about MoE")
	if err != nil {
		t.Fatalf("NewTaskContract: %v", err)
	}
	if usage != nil {
		task.Budget = Budget{MaxModelCalls: 10, MaxToolCalls: 10, MaxRuntimeSeconds: 60, MaxArtifactBytes: 1000}
	}
	rs, err := NewRunState(task)
	if err != nil {
		t.Fatalf("NewRunState: %v", err)
	}
	rs.ArtifactIDs = artifactIDs
	if usage != nil {
		rs.Usage = usage
	}
	return rs
}

func TestValidatorRegistry_RegisterLookupList(t *testing.T) {
	reg := NewValidatorRegistry()
	if err := reg.Register(NewArtifactExistsValidator()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := reg.Register(NewJSONSchemaValidator()); err != nil {
		t.Fatalf("Register json_schema: %v", err)
	}
	v, err := reg.Lookup("artifact_exists")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if v.ID() != "artifact_exists" || v.Version() != "v1" {
		t.Fatalf("unexpected validator identity: %s/%s", v.ID(), v.Version())
	}
	if len(reg.List()) != 2 {
		t.Fatalf("expected 2 registered, got %d", len(reg.List()))
	}
}

func TestValidatorRegistry_DuplicateConflict(t *testing.T) {
	reg := NewValidatorRegistry()
	if err := reg.Register(NewArtifactExistsValidator()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	err := reg.Register(NewArtifactExistsValidator())
	he, ok := err.(*HarnessError)
	if !ok || he.Code != ErrCodeConflict {
		t.Fatalf("expected CONFLICT, got %v", err)
	}
}

func TestValidatorRegistry_LookupUnknown(t *testing.T) {
	reg := NewValidatorRegistry()
	_, err := reg.Lookup("nope")
	he, ok := err.(*HarnessError)
	if !ok || he.Code != ErrCodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestValidatorRegistry_RunAllFailsFirst(t *testing.T) {
	reg := NewValidatorRegistry()
	reg.Register(NewArtifactExistsValidator())
	// 注册一个始终失败的验证器
	reg.Register(failingValidator{id: "always_fail"})
	run := newValidatedRun(t, "run_v1", nil, nil)
	results, err := reg.RunAll(context.Background(), run, map[string]*Artifact{})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ValidatorID != "always_fail" {
		t.Fatalf("failed validator should come first, got %s", results[0].ValidatorID)
	}
}

func TestArtifactExistsValidator(t *testing.T) {
	v := NewArtifactExistsValidator()
	run := newValidatedRun(t, "run_v2", []string{"art_a"}, nil)
	// 缺失 → 失败
	vr, err := v.Validate(context.Background(), run, map[string]*Artifact{})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if vr.Passed {
		t.Fatal("expected failure for missing artifact")
	}
	// 齐备 → 通过
	artifacts := map[string]*Artifact{"art_a": inMemArtifact("art_a", "note", []byte("x"), nil)}
	vr, err = v.Validate(context.Background(), run, artifacts)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !vr.Passed {
		t.Fatalf("expected pass, got findings %v", vr.Findings)
	}
}

func TestJSONSchemaValidator(t *testing.T) {
	v := NewJSONSchemaValidator()
	run := newValidatedRun(t, "run_v3", []string{"good.json", "bad.json"}, nil)
	artifacts := map[string]*Artifact{
		"good.json": inMemArtifact("good.json", "analysis.json", []byte(`{"ok": 1}`), nil),
		"bad.json":  inMemArtifact("bad.json", "analysis.json", []byte(`{"ok":`), nil),
	}
	vr, err := v.Validate(context.Background(), run, artifacts)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if vr.Passed {
		t.Fatal("expected failure for invalid JSON artifact")
	}
	if len(vr.Findings) != 1 || !strings.Contains(vr.Findings[0], "bad.json") {
		t.Fatalf("expected finding on bad.json, got %v", vr.Findings)
	}
}

func TestReferenceIntegrityValidator(t *testing.T) {
	v := NewReferenceIntegrityValidator()
	run := newValidatedRun(t, "run_v4", []string{"a", "b"}, nil)
	artifacts := map[string]*Artifact{
		"a": inMemArtifact("a", "note", []byte("x"), []any{"b", "ghost"}),
		"b": inMemArtifact("b", "note", []byte("y"), nil),
	}
	vr, err := v.Validate(context.Background(), run, artifacts)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if vr.Passed {
		t.Fatal("expected failure for dangling reference")
	}
	if len(vr.Findings) != 1 || !strings.Contains(vr.Findings[0], "ghost") {
		t.Fatalf("expected finding on ghost reference, got %v", vr.Findings)
	}
}

func TestBudgetValidator(t *testing.T) {
	v := NewBudgetValidator()
	// 超限 → 失败
	over := newValidatedRun(t, "run_v5", nil, map[string]float64{"model_calls": 11})
	vr, err := v.Validate(context.Background(), over, nil)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if vr.Passed {
		t.Fatal("expected failure when model_calls exceeds budget")
	}
	// 未超限 → 通过
	under := newValidatedRun(t, "run_v5", nil, map[string]float64{"model_calls": 2, "tool_calls": 1})
	vr, err = v.Validate(context.Background(), under, nil)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !vr.Passed {
		t.Fatalf("expected pass, got findings %v", vr.Findings)
	}
}

func TestAggregateValidation(t *testing.T) {
	passed := &ValidatorResult{Passed: true}
	failed := &ValidatorResult{Passed: false, Findings: []string{"f1"}}
	agg := AggregateValidation([]*ValidatorResult{passed, failed})
	if agg.Passed {
		t.Fatal("aggregate should fail when any validator fails")
	}
	if len(agg.Findings) != 1 {
		t.Fatalf("expected 1 aggregated finding, got %d", len(agg.Findings))
	}
}

// failingValidator 始终失败的测试验证器。
type failingValidator struct{ id string }

func (f failingValidator) ID() string      { return f.id }
func (f failingValidator) Version() string { return "v1" }
func (f failingValidator) Validate(_ context.Context, _ *RunState, _ map[string]*Artifact) (*ValidatorResult, error) {
	return &ValidatorResult{ValidatorID: f.id, Passed: false, Findings: []string{"intentional failure"}}, nil
}
