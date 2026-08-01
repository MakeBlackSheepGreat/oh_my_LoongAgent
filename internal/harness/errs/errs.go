// Package errs 定义领域无关的错误类型与错误码。
// 独立子包避免 harness ↔ providers 的导入循环：错误类型是共享契约的最底层。
package errs

import "fmt"

// ErrorCode 错误码常量。
const (
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeBudgetExceeded   = "BUDGET_EXCEEDED"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeProviderUnavailable = "PROVIDER_UNAVAILABLE"
	ErrCodeInternal         = "INTERNAL_ERROR"
)

// HarnessError 带错误码的错误。
type HarnessError struct {
	Code    string
	Message string
	Cause   error
}

func (e *HarnessError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *HarnessError) Unwrap() error { return e.Cause }

// NewHarnessError 构造错误。
func NewHarnessError(code, message string, cause error) *HarnessError {
	return &HarnessError{Code: code, Message: message, Cause: cause}
}

// ErrValidation 校验错误快捷构造。
func ErrValidation(message string, cause error) *HarnessError {
	return NewHarnessError(ErrCodeValidation, message, cause)
}

// ErrNotFound 未找到错误快捷构造。
func ErrNotFound(resource, id string) *HarnessError {
	return NewHarnessError(ErrCodeNotFound, fmt.Sprintf("%s not found: %s", resource, id), nil)
}

// ErrBudgetExceeded 预算超限快捷构造。
func ErrBudgetExceeded(detail string) *HarnessError {
	return NewHarnessError(ErrCodeBudgetExceeded, detail, nil)
}

// ErrPermissionDenied 权限拒绝快捷构造。
func ErrPermissionDenied(detail string) *HarnessError {
	return NewHarnessError(ErrCodePermissionDenied, detail, nil)
}
