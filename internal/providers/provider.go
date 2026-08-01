// Package providers 实现 OpenAI 兼容的模型 Provider 层。
// 契约风格对齐 internal/harness/contracts.go：校验在 Validate 方法中执行。
package providers

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)

// ---- 预编译正则 ----

var (
	reRole       = regexp.MustCompile(`^(system|user|assistant|tool)$`)
	reProviderID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
)

// ---- 枚举 ----

// ProviderPreset 预设 Provider 配置标识。
type ProviderPreset string

const (
	PresetSiliconFlow ProviderPreset = "siliconflow"
	PresetModelScope  ProviderPreset = "modelscope"
	PresetLocal       ProviderPreset = "local"
	PresetDeepSeek    ProviderPreset = "deepseek"
)

// ---- 接口与结构体 ----

// Provider 模型 Provider 抽象接口。
type Provider interface {
	// Chat 执行一次对话补全。
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

// ProviderConfig Provider 配置；APIKey 仅从环境变量读取，不落库、不进日志。
type ProviderConfig struct {
	ProviderID  string `json:"provider_id"`
	BaseURL     string `json:"base_url"`
	ModelID     string `json:"model_id"`
	APIKeyEnv   string `json:"api_key_env"`
	DisplayName string `json:"display_name"`
	Preset      string `json:"preset"`
}

// Validate 校验 Provider 配置；BaseURL、ModelID、APIKeyEnv 必须非空。
func (c ProviderConfig) Validate() error {
	if !reProviderID.MatchString(c.ProviderID) {
		return fmt.Errorf("provider_id invalid: %s", c.ProviderID)
	}
	if c.BaseURL == "" {
		return errors.New("base_url must be non-empty")
	}
	if c.ModelID == "" {
		return errors.New("model_id must be non-empty")
	}
	if c.APIKeyEnv == "" {
		return errors.New("api_key_env must be non-empty")
	}
	if len(c.DisplayName) > 256 {
		return errors.New("display_name too long")
	}
	return nil
}

// ChatMessage 对话消息；Role 取值 system/user/assistant/tool。
type ChatMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// Validate 校验消息。
func (m ChatMessage) Validate() error {
	if !reRole.MatchString(m.Role) {
		return fmt.Errorf("role invalid: %s", m.Role)
	}
	if m.Role == "tool" && m.ToolCallID == "" {
		return errors.New("tool message requires tool_call_id")
	}
	return nil
}

// ChatRequest 对话补全请求。
type ChatRequest struct {
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
}

// Validate 校验请求。
func (r *ChatRequest) Validate() error {
	if len(r.Messages) == 0 {
		return errors.New("messages must be non-empty")
	}
	if len(r.Messages) > 1000 {
		return errors.New("too many messages")
	}
	for i, m := range r.Messages {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("messages[%d]: %w", i, err)
		}
	}
	if r.Temperature != nil {
		if *r.Temperature < 0 || *r.Temperature > 2 {
			return errors.New("temperature out of range [0,2]")
		}
	}
	if r.MaxTokens != nil {
		if *r.MaxTokens < 1 || *r.MaxTokens > 1000000 {
			return errors.New("max_tokens out of range [1,1000000]")
		}
	}
	if len(r.Stop) > 16 {
		return errors.New("too many stop sequences")
	}
	return nil
}

// Usage token 用量统计。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse 对话补全响应。
type ChatResponse struct {
	Content      string         `json:"content"`
	Usage        Usage          `json:"usage"`
	Model        string         `json:"model"`
	FinishReason string         `json:"finish_reason"`
	Raw          map[string]any `json:"raw,omitempty"`
}
