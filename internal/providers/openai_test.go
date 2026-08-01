package providers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"slim-agent/internal/harness/errs"
)

// newTestOpenAIProvider 构造测试用 OpenAI 兼容 Provider。
// 使用 t.Setenv 设置 HARNESS_TEST_API_KEY 让 ResolveAPIKey 通过。
func newTestOpenAIProvider(t *testing.T, baseURL string) *OpenAICompatibleProvider {
	t.Helper()
	t.Setenv("HARNESS_TEST_API_KEY", "test-secret")
	config := ProviderConfig{
		ProviderID:  "test",
		BaseURL:     baseURL,
		ModelID:     "test-model",
		APIKeyEnv:   "HARNESS_TEST_API_KEY",
		DisplayName: "Test",
		Preset:      "test",
	}
	p, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		t.Fatalf("NewOpenAICompatibleProvider: %v", err)
	}
	return p
}

// TestOpenAICompatibleProvider_Chat_Success 验证标准 OpenAI 格式响应的 Content/Usage/Model 提取。
func TestOpenAICompatibleProvider_Chat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if r.URL.Path != "/chat/completions" {
			t.Errorf("期望路径 /chat/completions，得到 %s", r.URL.Path)
		}
		// 验证 Authorization header
		if got := r.Header.Get("Authorization"); got != "Bearer test-secret" {
			t.Errorf("期望 Bearer test-secret，得到 %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"model": "test-model",
			"choices": [{"index": 0, "message": {"role": "assistant", "content": "hello world"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer srv.Close()

	provider := newTestOpenAIProvider(t, srv.URL)
	req := &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}
	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "hello world" {
		t.Fatalf("Content: 期望 'hello world'，得到 %q", resp.Content)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Fatalf("PromptTokens: 期望 10，得到 %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Fatalf("CompletionTokens: 期望 5，得到 %d", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("TotalTokens: 期望 15，得到 %d", resp.Usage.TotalTokens)
	}
	if resp.Model != "test-model" {
		t.Fatalf("Model: 期望 'test-model'，得到 %q", resp.Model)
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("FinishReason: 期望 'stop'，得到 %q", resp.FinishReason)
	}
}

// TestOpenAICompatibleProvider_Chat_Timeout 验证 context 超时后返回错误。
func TestOpenAICompatibleProvider_Chat_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(1 * time.Second):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
			return
		}
	}))
	defer srv.Close()

	provider := newTestOpenAIProvider(t, srv.URL)
	req := &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := provider.Chat(ctx, req)
	if err == nil {
		t.Fatal("期望超时错误")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("期望 context.DeadlineExceeded，得到: %v", err)
	}
}

// TestOpenAICompatibleProvider_Chat_RateLimit 验证 429 返回 ProviderUnavailable 错误。
func TestOpenAICompatibleProvider_Chat_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer srv.Close()

	provider := newTestOpenAIProvider(t, srv.URL)
	req := &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}
	_, err := provider.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("期望错误")
	}
	hErr, ok := err.(*errs.HarnessError)
	if !ok {
		t.Fatalf("期望 *HarnessError，得到 %T: %v", err, err)
	}
	if hErr.Code != errs.ErrCodeProviderUnavailable {
		t.Fatalf("期望 %s，得到 %s", errs.ErrCodeProviderUnavailable, hErr.Code)
	}
}

// TestOpenAICompatibleProvider_Chat_InvalidJSON 验证非 JSON 响应返回 Validation 错误。
func TestOpenAICompatibleProvider_Chat_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("this is not json"))
	}))
	defer srv.Close()

	provider := newTestOpenAIProvider(t, srv.URL)
	req := &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}
	_, err := provider.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("期望错误")
	}
	hErr, ok := err.(*errs.HarnessError)
	if !ok {
		t.Fatalf("期望 *HarnessError，得到 %T: %v", err, err)
	}
	if hErr.Code != errs.ErrCodeValidation {
		t.Fatalf("期望 %s，得到 %s", errs.ErrCodeValidation, hErr.Code)
	}
}

// TestOpenAICompatibleProvider_Chat_HTTPError 验证 500 返回错误。
func TestOpenAICompatibleProvider_Chat_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	}))
	defer srv.Close()

	provider := newTestOpenAIProvider(t, srv.URL)
	req := &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}
	_, err := provider.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("期望错误")
	}
	hErr, ok := err.(*errs.HarnessError)
	if !ok {
		t.Fatalf("期望 *HarnessError，得到 %T: %v", err, err)
	}
	if hErr.Code != errs.ErrCodeInternal {
		t.Fatalf("期望 %s，得到 %s", errs.ErrCodeInternal, hErr.Code)
	}
	if !strings.Contains(hErr.Error(), "500") {
		t.Fatalf("期望错误包含 '500'，得到: %s", hErr.Error())
	}
}
