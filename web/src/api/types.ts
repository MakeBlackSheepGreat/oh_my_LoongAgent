// 后端 API 类型化契约（字段对齐 internal/workbench 的 json tag）。
export interface Account {
  account_id: string
  username: string
  display_name: string
  status: 'active' | 'disabled'
  locale: string
  created_at: string
}

export interface Project {
  project_id: string
  account_id: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

export interface Conversation {
  conversation_id: string
  account_id: string
  project_id?: string
  title: string
  created_at: string
  updated_at: string
}

export interface Message {
  message_id: string
  account_id: string
  conversation_id: string
  role: 'system' | 'user' | 'assistant' | 'tool'
  content: string
  created_at: string
}

export type DraftStatus = 'draft' | 'approved' | 'rejected'

export interface TaskDraft {
  draft_id: string
  account_id: string
  conversation_id?: string
  objective: string
  skill_id: string
  status: DraftStatus
  run_id?: string
  created_at: string
  updated_at: string
}

export type ProfileScope = 'account' | 'system'

export interface ProviderProfile {
  profile_id: string
  account_id: string
  provider_id: string
  display_name: string
  base_url: string
  model_id: string
  api_key_env: string
  scope: ProfileScope
  is_active: boolean
  created_at: string
}

export interface HealthResult {
  profile_id: string
  ok: boolean
  latency_ms: number
  error: string
}

export interface ProviderUsage {
  provider_id: string
  model_id: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  estimated_cost_usd: number
  call_count: number
}

export type UsageWindow = 'today' | 'week' | 'month' | 'all'

export interface UsageAggregate {
  account_id: string
  window: UsageWindow
  total_input_tokens: number
  total_output_tokens: number
  total_tokens: number
  total_cost_usd: number
  call_count: number
  by_provider: ProviderUsage[]
}

export interface PublicPool {
  account_id: string
  providers: ProviderUsage[]
}

export interface SkillInfo {
  skill_id: string
  version: string
  title: string
  description: string
  output_artifact_kinds: string[]
  required_validators: string[]
}

// PresetProvider 预设供应商（对齐后端 providers.PresetProvider）。
export interface PresetProvider {
  provider_id: string
  display_name: string
  base_url: string
  model_id: string
  api_key_env: string
  category: string
  website_url?: string
  is_official: boolean
}

// SSE 事件（对齐 harness.Event 的 kind/message/payload）。
export interface RunEvent {
  sequence: number
  run_id: string
  kind: string
  message: string
  payload: Record<string, unknown>
  timestamp: string
}
