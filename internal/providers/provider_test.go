package providers

import "testing"

// TestProviderConfig_Validate 验证 ProviderConfig.Validate 行为。
func TestProviderConfig_Validate(t *testing.T) {
	// 有效配置通过
	valid := ProviderConfig{
		ProviderID:  "openai",
		BaseURL:     "https://api.openai.com/v1",
		ModelID:     "gpt-4",
		APIKeyEnv:   "OPENAI_API_KEY",
		DisplayName: "OpenAI",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("有效配置应通过校验，得到: %v", err)
	}

	// BaseURL 为空失败
	noBase := valid
	noBase.BaseURL = ""
	if err := noBase.Validate(); err == nil {
		t.Fatal("BaseURL 为空应失败")
	}

	// ModelID 为空失败
	noModel := valid
	noModel.ModelID = ""
	if err := noModel.Validate(); err == nil {
		t.Fatal("ModelID 为空应失败")
	}

	// APIKeyEnv 为空失败
	noKey := valid
	noKey.APIKeyEnv = ""
	if err := noKey.Validate(); err == nil {
		t.Fatal("APIKeyEnv 为空应失败")
	}
}

// TestChatRequest_Validate 验证 ChatRequest.Validate 行为。
func TestChatRequest_Validate(t *testing.T) {
	// Messages 为空失败
	empty := &ChatRequest{}
	if err := empty.Validate(); err == nil {
		t.Fatal("Messages 为空应失败")
	}

	// Role 非法失败
	badRole := &ChatRequest{
		Messages: []ChatMessage{{Role: "invalid", Content: "hi"}},
	}
	if err := badRole.Validate(); err == nil {
		t.Fatal("Role 非法应失败")
	}

	// Temperature 越界（小于 0）失败
	tempLow := -1.0
	tempLowReq := &ChatRequest{
		Messages:    []ChatMessage{{Role: "user", Content: "hi"}},
		Temperature: &tempLow,
	}
	if err := tempLowReq.Validate(); err == nil {
		t.Fatal("Temperature < 0 应失败")
	}

	// Temperature 越界（大于 2）失败
	tempHigh := 3.0
	tempHighReq := &ChatRequest{
		Messages:    []ChatMessage{{Role: "user", Content: "hi"}},
		Temperature: &tempHigh,
	}
	if err := tempHighReq.Validate(); err == nil {
		t.Fatal("Temperature > 2 应失败")
	}

	// 正常 Temperature 通过
	tempOK := 1.0
	okReq := &ChatRequest{
		Messages:    []ChatMessage{{Role: "user", Content: "hi"}},
		Temperature: &tempOK,
	}
	if err := okReq.Validate(); err != nil {
		t.Fatalf("合法请求应通过校验，得到: %v", err)
	}
}

// TestPresetConfig 验证四个预设都返回有效配置。
func TestPresetConfig(t *testing.T) {
	presets := []ProviderPreset{
		PresetSiliconFlow,
		PresetModelScope,
		PresetLocal,
		PresetDeepSeek,
	}
	for _, p := range presets {
		cfg := PresetConfig(p)
		if err := cfg.Validate(); err != nil {
			t.Errorf("预设 %s 应通过校验，得到: %v", p, err)
		}
		if cfg.Preset != string(p) {
			t.Errorf("预设 %s 的 Preset 字段 = %s", p, cfg.Preset)
		}
		if cfg.ProviderID == "" {
			t.Errorf("预设 %s 的 ProviderID 为空", p)
		}
	}
}

// TestResolveAPIKey 验证 ResolveAPIKey 行为。
func TestResolveAPIKey(t *testing.T) {
	// 环境变量存在时返回值
	t.Setenv("HARNESS_TEST_API_KEY", "secret-value")
	val, err := ResolveAPIKey("HARNESS_TEST_API_KEY")
	if err != nil {
		t.Fatalf("环境变量存在时应无错误，得到: %v", err)
	}
	if val != "secret-value" {
		t.Fatalf("期望 'secret-value'，得到 %q", val)
	}

	// 环境变量不存在时返回错误
	_, err = ResolveAPIKey("HARNESS_TEST_DEFINITELY_NOT_SET")
	if err == nil {
		t.Fatal("环境变量不存在时应返回错误")
	}
}
