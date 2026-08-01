package providers

// PresetProvider 预设供应商信息，供前端展示与一键添加。
type PresetProvider struct {
	ProviderID  string `json:"provider_id"`
	DisplayName string `json:"display_name"`
	BaseURL     string `json:"base_url"`
	ModelID     string `json:"model_id"`
	APIKeyEnv   string `json:"api_key_env"`
	Category    string `json:"category"` // official / cn_official / aggregator / third_party / local
	WebsiteURL  string `json:"website_url,omitempty"`
	IsOfficial  bool   `json:"is_official"`
}

// presetProviders 完整预设列表，来源 CC-Switch（farion1231/cc-switch）的 codex 供应商预设。
// 所有供应商均为 OpenAI Chat Completions 兼容 API。
var presetProviders = []PresetProvider{
	// ---- 官方 ----
	{
		ProviderID: "deepseek", DisplayName: "DeepSeek",
		BaseURL: "https://api.deepseek.com/v1", ModelID: "deepseek-chat",
		APIKeyEnv: "HARNESS_DEEPSEEK_API_KEY", Category: "official", IsOfficial: true,
		WebsiteURL: "https://platform.deepseek.com",
	},
	{
		ProviderID: "siliconflow", DisplayName: "SiliconFlow",
		BaseURL: "https://api.siliconflow.cn/v1", ModelID: "deepseek-ai/DeepSeek-V3",
		APIKeyEnv: "HARNESS_SILICONFLOW_API_KEY", Category: "aggregator",
		WebsiteURL: "https://siliconflow.cn",
	},
	{
		ProviderID: "modelscope", DisplayName: "ModelScope",
		BaseURL: "https://api-inference.modelscope.cn/v1", ModelID: "deepseek-ai/DeepSeek-V3",
		APIKeyEnv: "HARNESS_MODELSCOPE_API_KEY", Category: "aggregator",
		WebsiteURL: "https://modelscope.cn",
	},
	{
		ProviderID: "local", DisplayName: "Local",
		BaseURL: "http://127.0.0.1:8080/v1", ModelID: "local-model",
		APIKeyEnv: "HARNESS_LOCAL_API_KEY", Category: "local",
	},

	// ---- 国内官方 ----
	{
		ProviderID: "kimi", DisplayName: "Kimi（月之暗面）",
		BaseURL: "https://api.moonshot.cn/v1", ModelID: "kimi-k2.7-code",
		APIKeyEnv: "HARNESS_KIMI_API_KEY", Category: "cn_official",
		WebsiteURL: "https://platform.kimi.com",
	},
	{
		ProviderID: "kimi-coding", DisplayName: "Kimi For Coding",
		BaseURL: "https://api.kimi.com/coding/v1", ModelID: "kimi-for-coding",
		APIKeyEnv: "HARNESS_KIMI_CODING_API_KEY", Category: "cn_official",
		WebsiteURL: "https://www.kimi.com/code/",
	},
	{
		ProviderID: "volc-agentplan", DisplayName: "火山 Agentplan（豆包）",
		BaseURL: "https://ark.cn-beijing.volces.com/api/coding/v3", ModelID: "ark-code-latest",
		APIKeyEnv: "HARNESS_VOLC_API_KEY", Category: "cn_official",
		WebsiteURL: "https://www.volcengine.com/activity/codingplan",
	},
	{
		ProviderID: "doubaoseed", DisplayName: "DouBaoSeed（豆包）",
		BaseURL: "https://ark.cn-beijing.volces.com/api/compatible", ModelID: "doubao-seed-2-1-pro-260628",
		APIKeyEnv: "HARNESS_DOUBAO_API_KEY", Category: "cn_official",
		WebsiteURL: "https://console.volcengine.com/ark/",
	},

	// ---- 聚合/第三方 ----
	{
		ProviderID: "zetaapi", DisplayName: "ZetaAPI",
		BaseURL: "https://api.zetaapi.ai/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_ZETAAPI_API_KEY", Category: "aggregator",
		WebsiteURL: "https://zetaapi.ai",
	},
	{
		ProviderID: "packycode", DisplayName: "PackyCode",
		BaseURL: "https://www.packyapi.ai/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_PACKYCODE_API_KEY", Category: "third_party",
		WebsiteURL: "https://www.packyapi.ai",
	},
	{
		ProviderID: "apinebula", DisplayName: "APINebula",
		BaseURL: "https://apinebula.ai/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_APINEBULA_API_KEY", Category: "third_party",
		WebsiteURL: "https://apinebula.ai",
	},
	{
		ProviderID: "aicodemirror", DisplayName: "AICodeMirror",
		BaseURL: "https://api.aicodemirror.com/api/codex/backend-api/codex", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_AICODEMIRROR_API_KEY", Category: "third_party",
		WebsiteURL: "https://www.aicodemirror.ai",
	},
	{
		ProviderID: "patewayai", DisplayName: "PatewayAI",
		BaseURL: "https://api.pateway.ai/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_PATEWAYAI_API_KEY", Category: "third_party",
		WebsiteURL: "https://pateway.ai",
	},
	{
		ProviderID: "fennoai", DisplayName: "FennoAI",
		BaseURL: "https://api.fenno.ai", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_FENNOAI_API_KEY", Category: "aggregator",
		WebsiteURL: "https://api.fenno.ai",
	},
	{
		ProviderID: "runapi", DisplayName: "RunAPI",
		BaseURL: "https://runapi.co/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_RUNAPI_API_KEY", Category: "aggregator",
		WebsiteURL: "https://runapi.co",
	},
	{
		ProviderID: "unity2", DisplayName: "Unity2.ai",
		BaseURL: "https://api.unity2.ai", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_UNITY2_API_KEY", Category: "aggregator",
		WebsiteURL: "https://unity2.ai",
	},
	{
		ProviderID: "shengsuanyun", DisplayName: "Shengsuanyun（算云）",
		BaseURL: "https://router.shengsuanyun.com/api/v1", ModelID: "openai/gpt-5.6-sol",
		APIKeyEnv: "HARNESS_SHENGSUANYUN_API_KEY", Category: "aggregator",
		WebsiteURL: "https://www.shengsuanyun.com",
	},
	{
		ProviderID: "aigocode", DisplayName: "AIGoCode",
		BaseURL: "https://api.aigocode.com", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_AIGOCODE_API_KEY", Category: "third_party",
		WebsiteURL: "https://aigocode.com",
	},
	{
		ProviderID: "aicoding", DisplayName: "AICoding",
		BaseURL: "https://api.aicoding.sh", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_AICODING_API_KEY", Category: "third_party",
		WebsiteURL: "https://aicoding.sh",
	},
	{
		ProviderID: "subrouter", DisplayName: "SubRouter",
		BaseURL: "https://subrouter.ai/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_SUBROUTER_API_KEY", Category: "aggregator",
		WebsiteURL: "https://subrouter.ai",
	},
	{
		ProviderID: "apikeyfun", DisplayName: "APIKEY.FUN",
		BaseURL: "https://api.apikey.fun/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_APIKEYFUN_API_KEY", Category: "third_party",
		WebsiteURL: "https://apikey.fun",
	},
	{
		ProviderID: "code0", DisplayName: "Code0",
		BaseURL: "https://code0.ai/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_CODE0_API_KEY", Category: "aggregator",
		WebsiteURL: "https://code0.ai",
	},
	{
		ProviderID: "teamorouter", DisplayName: "TeamoRouter",
		BaseURL: "https://api.teamorouter.com/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_TEAMOROUTER_API_KEY", Category: "aggregator",
		WebsiteURL: "https://teamorouter.com",
	},
	{
		ProviderID: "claudecn", DisplayName: "ClaudeCN",
		BaseURL: "https://claudecn.top/v1", ModelID: "gpt-5.6-sol",
		APIKeyEnv: "HARNESS_CLAUDECN_API_KEY", Category: "third_party",
		WebsiteURL: "https://claudecn.top",
	},
}

// ListPresets 返回所有预设供应商列表。
func ListPresets() []PresetProvider {
	result := make([]PresetProvider, len(presetProviders))
	copy(result, presetProviders)
	return result
}

// PresetConfig 返回指定预设的 Provider 配置。
// 未知预设返回零值 ProviderConfig（无法通过 Validate）。
func PresetConfig(preset ProviderPreset) ProviderConfig {
	switch preset {
	case PresetSiliconFlow:
		return ProviderConfig{
			ProviderID:  "siliconflow",
			BaseURL:     "https://api.siliconflow.cn/v1",
			ModelID:     "deepseek-ai/DeepSeek-V3",
			APIKeyEnv:   "HARNESS_SILICONFLOW_API_KEY",
			DisplayName: "SiliconFlow",
			Preset:      string(PresetSiliconFlow),
		}
	case PresetModelScope:
		return ProviderConfig{
			ProviderID:  "modelscope",
			BaseURL:     "https://api-inference.modelscope.cn/v1",
			ModelID:     "deepseek-ai/DeepSeek-V3",
			APIKeyEnv:   "HARNESS_MODELSCOPE_API_KEY",
			DisplayName: "ModelScope",
			Preset:      string(PresetModelScope),
		}
	case PresetLocal:
		return ProviderConfig{
			ProviderID:  "local",
			BaseURL:     "http://127.0.0.1:8080/v1",
			ModelID:     "local-model",
			APIKeyEnv:   "HARNESS_LOCAL_API_KEY",
			DisplayName: "Local",
			Preset:      string(PresetLocal),
		}
	case PresetDeepSeek:
		return ProviderConfig{
			ProviderID:  "deepseek",
			BaseURL:     "https://api.deepseek.com/v1",
			ModelID:     "deepseek-chat",
			APIKeyEnv:   "HARNESS_DEEPSEEK_API_KEY",
			DisplayName: "DeepSeek",
			Preset:      string(PresetDeepSeek),
		}
	default:
		return ProviderConfig{}
	}
}