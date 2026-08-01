package harness

import "strings"

// RecoveryAction 恢复动作分类（对齐 RecoveryLabel 语义）。
type RecoveryAction string

const (
	RecoverRetry       RecoveryAction = "retry"
	RecoverRepair      RecoveryAction = "repair"
	RecoverHumanReview RecoveryAction = "human_review"
	RecoverStop        RecoveryAction = "stop"
)

// RecoveryPlan 恢复决策。
type RecoveryPlan struct {
	Action     RecoveryAction `json:"action"`
	RetryCount int            `json:"retry_count"`
	MaxRetries int            `json:"max_retries"`
	Reason     string         `json:"reason"`
}

// RecoveryResolver 定向恢复：依据错误码与恢复标签分类导向 retry/repair/human_review/stop。
type RecoveryResolver struct {
	maxRetries int
}

// NewRecoveryResolver 构造恢复解析器。
func NewRecoveryResolver(maxRetries int) *RecoveryResolver {
	if maxRetries < 0 {
		maxRetries = 0
	}
	return &RecoveryResolver{maxRetries: maxRetries}
}

// Resolve 依据错误记录与已重试次数返回恢复计划。
// 规则：
//   - BUDGET_EXCEEDED → stop（预算不可恢复）
//   - PROVIDER_UNAVAILABLE 且可恢复 → retry（限次）
//   - 带 repair 标签 → repair（AGI 定向修复）
//   - 带 retry 标签 → retry（限次）
//   - 可恢复但超过重试上限 → human_review
//   - 不可恢复未知错误 → stop
func (r *RecoveryResolver) Resolve(er *ErrorRecord, retryCount int) *RecoveryPlan {
	switch er.Code {
	case ErrCodeBudgetExceeded:
		return &RecoveryPlan{Action: RecoverStop, RetryCount: retryCount, MaxRetries: r.maxRetries,
			Reason: "budget exhausted, not recoverable"}
	case ErrCodeProviderUnavailable:
		if r.canRetry(retryCount) {
			return &RecoveryPlan{Action: RecoverRetry, RetryCount: retryCount, MaxRetries: r.maxRetries,
				Reason: "provider temporarily unavailable"}
		}
		return &RecoveryPlan{Action: RecoverHumanReview, RetryCount: retryCount, MaxRetries: r.maxRetries,
			Reason: "provider unavailable and retries exhausted"}
	}

	hasRepair := hasLabel(er, RecoveryRepair)
	hasRetry := hasLabel(er, RecoveryRetry)
	if hasRepair {
		return &RecoveryPlan{Action: RecoverRepair, RetryCount: retryCount, MaxRetries: r.maxRetries,
			Reason: "verification gap, directed repair available"}
	}
	if hasRetry && r.canRetry(retryCount) {
		return &RecoveryPlan{Action: RecoverRetry, RetryCount: retryCount, MaxRetries: r.maxRetries,
			Reason: "retryable failure"}
	}
	if er.Recoverable {
		return &RecoveryPlan{Action: RecoverHumanReview, RetryCount: retryCount, MaxRetries: r.maxRetries,
			Reason: "recoverable but retries exhausted, escalate to human"}
	}
	return &RecoveryPlan{Action: RecoverStop, RetryCount: retryCount, MaxRetries: r.maxRetries,
		Reason: "unrecoverable failure"}
}

// canRetry 判断是否仍可重试。
func (r *RecoveryResolver) canRetry(retryCount int) bool {
	return retryCount < r.maxRetries
}

// hasLabel 判断错误记录是否包含指定恢复标签。
func hasLabel(er *ErrorRecord, label RecoveryLabel) bool {
	for _, l := range er.RecoveryLabels {
		if l == label {
			return true
		}
	}
	return false
}

// RecoveryLabelFromError 从错误字符串推断 RecoveryLabel（防御：错误未显式标注时）。
func RecoveryLabelFromError(err error) RecoveryLabel {
	if err == nil {
		return RecoveryStop
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "unavailable"),
		strings.Contains(msg, "429"), strings.Contains(msg, "503"):
		return RecoveryRetry
	case strings.Contains(msg, "validation"), strings.Contains(msg, "invalid"):
		return RecoveryRepair
	default:
		return RecoveryStop
	}
}
