package harness

import (
	"errors"
	"testing"
)

func newErrorRecord(t *testing.T, code, message string, recoverable bool, labels ...RecoveryLabel) *ErrorRecord {
	t.Helper()
	er, err := NewErrorRecord(code, message, recoverable)
	if err != nil {
		t.Fatalf("NewErrorRecord: %v", err)
	}
	if len(labels) > 0 {
		er.RecoveryLabels = labels
	}
	return er
}

func TestRecoveryResolver_BudgetStop(t *testing.T) {
	r := NewRecoveryResolver(2)
	er := newErrorRecord(t, ErrCodeBudgetExceeded, "model_calls budget exceeded", false)
	plan := r.Resolve(er, 0)
	if plan.Action != RecoverStop {
		t.Fatalf("expected stop, got %s", plan.Action)
	}
}

func TestRecoveryResolver_ProviderRetryThenHumanReview(t *testing.T) {
	r := NewRecoveryResolver(2)
	er := newErrorRecord(t, ErrCodeProviderUnavailable, "provider rate limited", true)
	// 前两次重试
	if p := r.Resolve(er, 0); p.Action != RecoverRetry {
		t.Fatalf("expected retry at count 0, got %s", p.Action)
	}
	if p := r.Resolve(er, 1); p.Action != RecoverRetry {
		t.Fatalf("expected retry at count 1, got %s", p.Action)
	}
	// 重试耗尽 → 人工审查
	if p := r.Resolve(er, 2); p.Action != RecoverHumanReview {
		t.Fatalf("expected human_review at count 2, got %s", p.Action)
	}
	if p := r.Resolve(er, 3); p.Action != RecoverHumanReview {
		t.Fatalf("expected human_review at count 3, got %s", p.Action)
	}
}

func TestRecoveryResolver_RepairLabel(t *testing.T) {
	r := NewRecoveryResolver(2)
	er := newErrorRecord(t, ErrCodeValidation, "artifact missing", false, RecoveryRepair)
	plan := r.Resolve(er, 0)
	if plan.Action != RecoverRepair {
		t.Fatalf("expected repair, got %s", plan.Action)
	}
}

func TestRecoveryResolver_RetryLabelExhausted(t *testing.T) {
	r := NewRecoveryResolver(1)
	er := newErrorRecord(t, "TOOL_EXECUTION_FAILED", "retryable failure", true, RecoveryRetry)
	if p := r.Resolve(er, 0); p.Action != RecoverRetry {
		t.Fatalf("expected retry at count 0, got %s", p.Action)
	}
	if p := r.Resolve(er, 1); p.Action != RecoverHumanReview {
		t.Fatalf("expected human_review at count 1, got %s", p.Action)
	}
}

func TestRecoveryResolver_UnrecoverableStop(t *testing.T) {
	r := NewRecoveryResolver(2)
	er := newErrorRecord(t, ErrCodeInternal, "unknown failure", false)
	plan := r.Resolve(er, 0)
	if plan.Action != RecoverStop {
		t.Fatalf("expected stop, got %s", plan.Action)
	}
}

func TestRecoveryLabelFromError(t *testing.T) {
	cases := []struct {
		err  error
		want RecoveryLabel
	}{
		{nil, RecoveryStop},
		{errors.New("upstream timeout"), RecoveryRetry},
		{errors.New("provider unavailable"), RecoveryRetry},
		{errors.New("status=429 rate limited"), RecoveryRetry},
		{errors.New("invalid json response"), RecoveryRepair},
		{errors.New("boom"), RecoveryStop},
	}
	for _, c := range cases {
		if got := RecoveryLabelFromError(c.err); got != c.want {
			t.Fatalf("RecoveryLabelFromError(%v) = %s, want %s", c.err, got, c.want)
		}
	}
}
