package providers

import (
	"os"

	"slim-agent/internal/harness/errs"
)

// ResolveAPIKey 从环境变量读取 API 密钥；密钥不落库、不进日志。
// 使用 os.LookupEnv 区分"未设置"与"设置为空"两种情况。
func ResolveAPIKey(envVar string) (string, error) {
	if envVar == "" {
		return "", errs.NewHarnessError(
			errs.ErrCodeProviderUnavailable,
			"api_key_env must be non-empty",
			nil,
		)
	}
	val, ok := os.LookupEnv(envVar)
	if !ok {
		return "", errs.NewHarnessError(
			errs.ErrCodeProviderUnavailable,
			"environment variable not set: "+envVar,
			nil,
		)
	}
	if val == "" {
		return "", errs.NewHarnessError(
			errs.ErrCodeProviderUnavailable,
			"environment variable is empty: "+envVar,
			nil,
		)
	}
	return val, nil
}
