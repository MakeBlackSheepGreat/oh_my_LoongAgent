package providers

import (
	"testing"
)

// newTestDeepSeekProvider 构造测试用 DeepSeek Provider。
func newTestDeepSeekProvider(t *testing.T) *DeepSeekProvider {
	t.Helper()
	t.Setenv("HARNESS_TEST_API_KEY", "test-secret")
	config := ProviderConfig{
		ProviderID:  "deepseek",
		BaseURL:     "http://localhost:1234/v1",
		ModelID:     "deepseek-chat",
		APIKeyEnv:   "HARNESS_TEST_API_KEY",
		DisplayName: "DeepSeek",
		Preset:      string(PresetDeepSeek),
	}
	p, err := NewDeepSeekProvider(config)
	if err != nil {
		t.Fatalf("NewDeepSeekProvider: %v", err)
	}
	return p
}

// TestDeepSeekProvider_BuildMessages 验证 stablePrefix → contextSummary → dynamicMessages 顺序。
func TestDeepSeekProvider_BuildMessages(t *testing.T) {
	p := newTestDeepSeekProvider(t)
	p.SetStablePrefix([]ChatMessage{
		{Role: "system", Content: "stable-prefix"},
	})
	p.SetContextSummary("context-summary")

	dynamic := []ChatMessage{
		{Role: "user", Content: "dynamic-1"},
		{Role: "assistant", Content: "dynamic-2"},
	}
	out := p.BuildMessages(dynamic)

	// 预期顺序：stablePrefix(1) + contextSummary(1) + dynamic(2) = 4
	if len(out) != 4 {
		t.Fatalf("期望 4 条消息，得到 %d", len(out))
	}
	if out[0].Content != "stable-prefix" {
		t.Fatalf("out[0]: 期望 'stable-prefix'，得到 %q", out[0].Content)
	}
	if out[1].Role != "system" || out[1].Content != "context-summary" {
		t.Fatalf("out[1]: 期望 system/context-summary，得到 %s/%q", out[1].Role, out[1].Content)
	}
	if out[2].Content != "dynamic-1" {
		t.Fatalf("out[2]: 期望 'dynamic-1'，得到 %q", out[2].Content)
	}
	if out[3].Content != "dynamic-2" {
		t.Fatalf("out[3]: 期望 'dynamic-2'，得到 %q", out[3].Content)
	}
}

// TestDeepSeekProvider_BuildMessages_Stability 验证同一 stablePrefix 两次调用前缀顺序不变。
func TestDeepSeekProvider_BuildMessages_Stability(t *testing.T) {
	p := newTestDeepSeekProvider(t)
	prefix := []ChatMessage{
		{Role: "system", Content: "prefix-a"},
		{Role: "system", Content: "prefix-b"},
	}
	p.SetStablePrefix(prefix)
	p.SetContextSummary("summary")

	dynamic := []ChatMessage{{Role: "user", Content: "dynamic"}}

	first := p.BuildMessages(dynamic)
	second := p.BuildMessages(dynamic)

	// 两次调用长度一致
	if len(first) != len(second) {
		t.Fatalf("长度不一致: %d vs %d", len(first), len(second))
	}
	// 逐条比较角色与内容
	for i := 0; i < len(first); i++ {
		if first[i].Role != second[i].Role || first[i].Content != second[i].Content {
			t.Fatalf("message[%d] 不一致: first=%v second=%v", i, first[i], second[i])
		}
	}
	// 验证前缀在最前
	if first[0].Content != "prefix-a" || first[1].Content != "prefix-b" {
		t.Fatalf("前缀顺序错误: %v %v", first[0], first[1])
	}
}

// TestDeepSeekProvider_PruneStale 验证 tool 消息被后续 assistant 覆盖时被剪枝。
func TestDeepSeekProvider_PruneStale(t *testing.T) {
	p := newTestDeepSeekProvider(t)
	messages := []ChatMessage{
		{Role: "user", Content: "question"},
		{Role: "assistant", Content: "calling tool"},
		{Role: "tool", Content: "tool-result-1", ToolCallID: "call-1"},
		{Role: "tool", Content: "tool-result-2", ToolCallID: "call-2"},
		{Role: "assistant", Content: "final answer"},
	}
	out := p.PruneStale(messages)
	// tool 消息被后续 assistant 覆盖，应被剪枝
	// 预期：user, assistant(calling tool), assistant(final answer) = 3
	if len(out) != 3 {
		t.Fatalf("期望剪枝后 3 条消息，得到 %d: %v", len(out), out)
	}
	if out[0].Content != "question" {
		t.Fatalf("out[0]: 期望 'question'，得到 %q", out[0].Content)
	}
	if out[1].Content != "calling tool" {
		t.Fatalf("out[1]: 期望 'calling tool'，得到 %q", out[1].Content)
	}
	if out[2].Content != "final answer" {
		t.Fatalf("out[2]: 期望 'final answer'，得到 %q", out[2].Content)
	}
	// 验证不含 tool 消息
	for _, m := range out {
		if m.Role == "tool" {
			t.Fatalf("tool 消息应被剪枝: %v", m)
		}
	}
}

// TestDeepSeekProvider_InjectContextSummary 验证环境摘要注入在 dynamicMessages 之前。
func TestDeepSeekProvider_InjectContextSummary(t *testing.T) {
	p := newTestDeepSeekProvider(t)
	p.SetContextSummary("env-summary")

	dynamic := []ChatMessage{
		{Role: "user", Content: "dynamic-1"},
		{Role: "assistant", Content: "dynamic-2"},
	}
	out := p.InjectContextSummary(dynamic)
	// 预期：summary(1) + dynamic(2) = 3
	if len(out) != 3 {
		t.Fatalf("期望 3 条消息，得到 %d", len(out))
	}
	if out[0].Role != "system" || out[0].Content != "env-summary" {
		t.Fatalf("out[0]: 期望 system/env-summary，得到 %s/%q", out[0].Role, out[0].Content)
	}
	if out[1].Content != "dynamic-1" {
		t.Fatalf("out[1]: 期望 'dynamic-1'，得到 %q", out[1].Content)
	}
	if out[2].Content != "dynamic-2" {
		t.Fatalf("out[2]: 期望 'dynamic-2'，得到 %q", out[2].Content)
	}
}

// TestDeepSeekProvider_SetStablePrefix 验证设置后 BuildMessages 包含前缀。
func TestDeepSeekProvider_SetStablePrefix(t *testing.T) {
	p := newTestDeepSeekProvider(t)
	prefix := []ChatMessage{
		{Role: "system", Content: "you are helpful"},
	}
	p.SetStablePrefix(prefix)

	dynamic := []ChatMessage{{Role: "user", Content: "hi"}}
	out := p.BuildMessages(dynamic)

	// 预期：prefix(1) + dynamic(1) = 2
	if len(out) != 2 {
		t.Fatalf("期望 2 条消息，得到 %d", len(out))
	}
	if out[0].Content != "you are helpful" {
		t.Fatalf("out[0]: 期望 'you are helpful'，得到 %q", out[0].Content)
	}
	if out[1].Content != "hi" {
		t.Fatalf("out[1]: 期望 'hi'，得到 %q", out[1].Content)
	}
}
