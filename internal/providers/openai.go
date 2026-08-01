package providers

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

// 默认 HTTP 超时。
const defaultHTTPTimeout = 120 * time.Second

// OpenAICompatibleProvider 通用 OpenAI 兼容 Provider。
// config 与 apiKey 在构造后不可变；httpClient 仅读不写，支持并发安全调用。
type OpenAICompatibleProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	apiKey     string
}

// NewOpenAICompatibleProvider 构造 OpenAI 兼容 Provider。
// 调用 ResolveAPIKey 获取密钥；创建带默认超时的 http.Client。
func NewOpenAICompatibleProvider(config ProviderConfig) (*OpenAICompatibleProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, errs.NewHarnessError(
			errs.ErrCodeValidation,
			"provider config invalid",
			err,
		)
	}
	apiKey, err := ResolveAPIKey(config.APIKeyEnv)
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
		apiKey: apiKey,
	}, nil
}

// chatRequestPayload 发送给 OpenAI 兼容服务的 JSON 请求体。
type chatRequestPayload struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
}

// openAIChoice OpenAI 响应中的单项选择。
type openAIChoice struct {
	Index int `json:"index"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

// openAIResponse OpenAI 兼容响应体。
type openAIResponse struct {
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat 执行一次对话补全。
func (p *OpenAICompatibleProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errs.NewHarnessError(
			errs.ErrCodeValidation,
			"chat request invalid",
			err,
		)
	}
	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		// context 超时/取消直接返回 ctx.Err()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, errs.NewHarnessError(
			errs.ErrCodeInternal,
			"http request failed",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errs.NewHarnessError(
			errs.ErrCodeInternal,
			"read response body failed",
			err,
		)
	}

	// HTTP 429：限流，返回 ProviderUnavailable，不自动重试
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errs.NewHarnessError(
			errs.ErrCodeProviderUnavailable,
			fmt.Sprintf("provider rate limited: status=%d body=%s", resp.StatusCode, truncateBody(body)),
			nil,
		)
	}
	// 其他非 200：内部错误
	if resp.StatusCode != http.StatusOK {
		return nil, errs.NewHarnessError(
			errs.ErrCodeInternal,
			fmt.Sprintf("provider returned non-200: status=%d body=%s", resp.StatusCode, truncateBody(body)),
			nil,
		)
	}

	return p.parseResponse(body)
}

// buildRequest 构造 POST 请求到 {BaseURL}/chat/completions。
// 使用 bytes.Buffer + json.Encoder 减少内存分配。
func (p *OpenAICompatibleProvider) buildRequest(ctx context.Context, req *ChatRequest) (*http.Request, error) {
	payload := chatRequestPayload{
		Model:       p.config.ModelID,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stop:        req.Stop,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return nil, errs.NewHarnessError(
			errs.ErrCodeInternal,
			"encode request body failed",
			err,
		)
	}
	url := p.config.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, errs.NewHarnessError(
			errs.ErrCodeInternal,
			"build http request failed",
			err,
		)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq, nil
}

// parseResponse 解析 OpenAI 格式响应 JSON 为 ChatResponse。
// 提取 choices[0].message.content、usage、model、finish_reason。
func (p *OpenAICompatibleProvider) parseResponse(body []byte) (*ChatResponse, error) {
	var raw openAIResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		// 响应非合法 JSON：记录前 500 字节，返回结构化错误
		return nil, errs.NewHarnessError(
			errs.ErrCodeValidation,
			fmt.Sprintf("invalid json response: %s", truncateBody(body)),
			err,
		)
	}
	if len(raw.Choices) == 0 {
		return nil, errs.NewHarnessError(
			errs.ErrCodeValidation,
			fmt.Sprintf("response has no choices: %s", truncateBody(body)),
			nil,
		)
	}
	choice := raw.Choices[0]
	// 保留原始响应映射，便于上游追溯
	var rawMap map[string]any
	if err := json.Unmarshal(body, &rawMap); err != nil {
		rawMap = nil
	}
	return &ChatResponse{
		Content: choice.Message.Content,
		Usage: Usage{
			PromptTokens:     raw.Usage.PromptTokens,
			CompletionTokens: raw.Usage.CompletionTokens,
			TotalTokens:      raw.Usage.TotalTokens,
		},
		Model:        raw.Model,
		FinishReason: choice.FinishReason,
		Raw:          rawMap,
	}, nil
}

// truncateBody 截取响应体前 500 字节用于错误日志。
func truncateBody(body []byte) string {
	if len(body) <= 500 {
		return string(body)
	}
	return string(body[:500])
}
