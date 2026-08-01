package harness

import (
	"strings"
	"testing"
)

func TestTaskContractValidation(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		skillID string
		obj     string
		wantErr string
	}{
		{"valid", "task_001", "lit_search", "find papers", ""},
		{"empty task_id", "", "lit_search", "obj", "task_id invalid"},
		{"bad skill_id", "task_1", "LitSearch", "obj", "skill_id invalid"},
		{"empty objective", "task_1", "lit", "", "objective length"},
		{"too long objective", "task_1", "lit", strings.Repeat("x", 10001), "objective length"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTaskContract(tt.taskID, tt.skillID, tt.obj)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestBudgetDefaults(t *testing.T) {
	b := DefaultBudget()
	if b.MaxModelCalls != 12 || b.MaxToolCalls != 50 || b.MaxRuntimeSeconds != 900 {
		t.Fatalf("unexpected defaults: %+v", b)
	}
	if b.MaxCostUSD != nil {
		t.Fatal("default MaxCostUSD should be nil")
	}
	if err := b.Validate(); err != nil {
		t.Fatalf("default budget invalid: %v", err)
	}
}

func TestPolicyDefaults(t *testing.T) {
	p := DefaultPolicy()
	if len(p.ApprovalRequiredFor) != 2 {
		t.Fatalf("expected 2 approval levels, got %d", len(p.ApprovalRequiredFor))
	}
	seen := map[RiskLevel]bool{}
	for _, r := range p.ApprovalRequiredFor {
		seen[r] = true
	}
	if !seen[RiskHigh] || !seen[RiskCritical] {
		t.Fatal("default policy should require approval for high and critical")
	}
}

func TestArtifactFromInput(t *testing.T) {
	content := []byte("hello world")
	ai, err := NewArtifactInput("art_1", "report", content)
	if err != nil {
		t.Fatalf("NewArtifactInput: %v", err)
	}
	art, err := ArtifactFromInput("run_1", ai, "store://art_1")
	if err != nil {
		t.Fatalf("ArtifactFromInput: %v", err)
	}
	if art.SizeBytes != len(content) {
		t.Fatalf("size mismatch: %d vs %d", art.SizeBytes, len(content))
	}
	if len(art.SHA256) != 64 {
		t.Fatalf("sha256 length: %d", len(art.SHA256))
	}
	// 验证 sha256 正确性
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if art.SHA256 != expected {
		t.Fatalf("sha256 mismatch: got %s, want %s", art.SHA256, expected)
	}
}

func TestErrorRecordRecoverable(t *testing.T) {
	er, err := NewErrorRecord("TIMEOUT_ERROR", "request timed out", true)
	if err != nil {
		t.Fatalf("NewErrorRecord: %v", err)
	}
	if !er.Recoverable {
		t.Fatal("should be recoverable")
	}
	if len(er.RecoveryLabels) != 1 || er.RecoveryLabels[0] != RecoveryRetry {
		t.Fatalf("recoverable error should auto-add retry label, got %v", er.RecoveryLabels)
	}
}

func TestRunStateTaskIDMatch(t *testing.T) {
	tc, _ := NewTaskContract("run_001", "skill_a", "do something")
	rs, err := NewRunState(tc)
	if err != nil {
		t.Fatalf("NewRunState: %v", err)
	}
	if rs.RunID != "run_001" {
		t.Fatalf("run_id mismatch: %s", rs.RunID)
	}
	if rs.Status != StatusCreated {
		t.Fatalf("expected created, got %s", rs.Status)
	}
	if rs.StateVersion != 1 {
		t.Fatalf("expected version 1, got %d", rs.StateVersion)
	}
}

func TestEventValidation(t *testing.T) {
	_, err := NewEvent(0, "run_1", "info", "msg")
	if err == nil || !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("expected sequence error, got %v", err)
	}
	_, err = NewEvent(1, "run_1", "INFO", "msg")
	if err == nil || !strings.Contains(err.Error(), "kind invalid") {
		t.Fatalf("expected kind invalid, got %v", err)
	}
	_, err = NewEvent(1, "run_1", "info", "")
	if err == nil || !strings.Contains(err.Error(), "message length") {
		t.Fatalf("expected message length error, got %v", err)
	}
}

func TestSkillManifestValidation(t *testing.T) {
	_, err := NewSkillManifest("lit_search", "v1", "Literature Search", "search papers")
	if err != nil {
		t.Fatalf("valid manifest failed: %v", err)
	}
	_, err = NewSkillManifest("LitSearch", "v1", "t", "d")
	if err == nil || !strings.Contains(err.Error(), "skill_id invalid") {
		t.Fatalf("expected skill_id invalid, got %v", err)
	}
}

func TestHarnessError(t *testing.T) {
	he := ErrNotFound("task", "123")
	if he.Code != ErrCodeNotFound {
		t.Fatalf("code mismatch: %s", he.Code)
	}
	if !strings.Contains(he.Error(), "task not found: 123") {
		t.Fatalf("unexpected message: %s", he.Error())
	}
}
