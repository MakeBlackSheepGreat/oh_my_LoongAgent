package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HealthResult 健康检查结果。
type HealthResult struct {
	OK        bool
	LatencyMs int
	Error     string
}

// HealthCheck 对 Provider 配置执行一次健康检查。
// 发送 GET 请求到 {BaseURL}/models，携带 Authorization header；
// 3 秒超时（context.WithTimeout），不重试。
// 密钥缺失直接返回 OK=false。
func HealthCheck(ctx context.Context, config ProviderConfig) HealthResult {
	apiKey, err := ResolveAPIKey(config.APIKeyEnv)
	if err != nil {
		return HealthResult{OK: false, Error: "api key unavailable"}
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	url := strings.TrimSuffix(config.BaseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return HealthResult{OK: false, Error: "connection failed: " + err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	latencyMs := int(time.Since(start) / time.Millisecond)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return HealthResult{OK: false, Error: "timeout"}
		}
		return HealthResult{OK: false, Error: "connection failed: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return HealthResult{
			OK:        false,
			LatencyMs: latencyMs,
			Error:     fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	}
	return HealthResult{OK: true, LatencyMs: latencyMs}
}
