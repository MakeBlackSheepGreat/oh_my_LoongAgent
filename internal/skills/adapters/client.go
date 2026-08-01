// Package adapters 提供 Skill 共用的外部访问基础设施。
// 目前只包含带超时/取消/大小上限的 HTTP 客户端，供四类文献来源共用。
package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"slim-agent/internal/harness/errs"
)

// defaultTimeout 默认请求超时（10 秒，spec task7.3）。
const defaultTimeout = 10 * time.Second

// defaultMaxBytes 默认响应大小上限（防恶意负载）。
const defaultMaxBytes = 5 << 20 // 5 MiB

// Client 带超时、取消与大小上限的 HTTP 客户端。
// 线程安全：http.Client 支持并发调用。
type Client struct {
	httpClient *http.Client
	timeout    time.Duration
	maxBytes   int64
}

// NewClient 构造客户端；timeout<=0 用默认 10 秒，maxBytes<=0 用默认 5 MiB。
func NewClient(timeout time.Duration, maxBytes int64) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
		maxBytes:   maxBytes,
	}
}

// Do 执行一次 GET/POST 请求并返回响应体（大小上限为客户端默认值）。
func (c *Client) Do(ctx context.Context, method, url string, headers map[string]string) ([]byte, error) {
	body, _, err := c.DoWithLimit(ctx, method, url, headers, c.maxBytes)
	return body, err
}

// DoWithLimit 执行请求并返回响应体与 Content-Type；maxBytes 指定响应上限。
// 错误标准化：网络/超时/非 2xx 映射 PROVIDER_UNAVAILABLE；响应超限映射 VALIDATION_ERROR。
func (c *Client) DoWithLimit(ctx context.Context, method, url string, headers map[string]string, maxBytes int64) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, "", errs.NewHarnessError(errs.ErrCodeValidation,
			fmt.Sprintf("build request failed: %s", url), err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "slim-agent/1.0")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, "", ctxErr
		}
		return nil, "", errs.NewHarnessError(errs.ErrCodeProviderUnavailable,
			fmt.Sprintf("http request failed: %s", url), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", errs.NewHarnessError(errs.ErrCodeProviderUnavailable,
			fmt.Sprintf("provider returned non-2xx: status=%d url=%s", resp.StatusCode, url), nil)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, "", ctxErr
		}
		return nil, "", errs.NewHarnessError(errs.ErrCodeInternal, "read response body failed", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, "", errs.NewHarnessError(errs.ErrCodeValidation,
			fmt.Sprintf("response exceeds size limit %d bytes", maxBytes), nil)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

// DoJSON 执行请求并将响应解码到 out。
func (c *Client) DoJSON(ctx context.Context, method, url string, headers map[string]string, out any) error {
	body, err := c.Do(ctx, method, url, headers)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return errs.NewHarnessError(errs.ErrCodeValidation,
			fmt.Sprintf("decode json response failed: %s", truncate(body, 500)), err)
	}
	return nil
}

// truncate 截取响应片段用于错误信息。
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n])
}
