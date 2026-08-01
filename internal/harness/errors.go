// Package harness 错误定义。
// 类型与错误码已下沉到 internal/harness/errs（避免 harness ↔ providers 导入循环）；
// 本文件保留 alias 与 wrapper，维持 harness.HarnessError 等现有 API 不变。
package harness

import "slim-agent/internal/harness/errs"

// HarnessError 错误类型 alias。
type HarnessError = errs.HarnessError

// ErrorCode 错误码常量 alias。
const (
	ErrCodeValidation        = errs.ErrCodeValidation
	ErrCodeNotFound         = errs.ErrCodeNotFound
	ErrCodeBudgetExceeded   = errs.ErrCodeBudgetExceeded
	ErrCodePermissionDenied = errs.ErrCodePermissionDenied
	ErrCodeConflict         = errs.ErrCodeConflict
	ErrCodeProviderUnavailable = errs.ErrCodeProviderUnavailable
	ErrCodeInternal         = errs.ErrCodeInternal
)

// NewHarnessError 构造 Harness 错误。
func NewHarnessError(code, message string, cause error) *HarnessError {
	return errs.NewHarnessError(code, message, cause)
}

// ErrValidation 校验错误快捷构造。
func ErrValidation(message string, cause error) *HarnessError {
	return errs.ErrValidation(message, cause)
}

// ErrNotFound 未找到错误快捷构造。
func ErrNotFound(resource, id string) *HarnessError {
	return errs.ErrNotFound(resource, id)
}

// ErrBudgetExceeded 预算超限快捷构造。
func ErrBudgetExceeded(detail string) *HarnessError {
	return errs.ErrBudgetExceeded(detail)
}

// ErrPermissionDenied 权限拒绝快捷构造。
func ErrPermissionDenied(detail string) *HarnessError {
	return errs.ErrPermissionDenied(detail)
}
