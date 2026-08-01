// fetch API client：credentials include 携带 HttpOnly cookie、统一错误映射、401 广播。
// 后端错误格式：{"error": code, "detail": message}。
import type {
  Account,
  Conversation,
  DraftStatus,
  HealthResult,
  Message,
  PresetProvider,
  ProfileScope,
  Project,
  ProviderProfile,
  PublicPool,
  RunEvent,
  SkillInfo,
  TaskDraft,
  UsageAggregate,
  UsageWindow,
} from './types'

export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(code: string, message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

// UNAUTHORIZED 事件：任何请求 401 时广播，App 据此回到登录页。
export const UNAUTHORIZED_EVENT = 'auth:unauthorized'

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response
  try {
    res = await fetch(path, {
      method,
      credentials: 'include',
      headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError('NETWORK_ERROR', 'network request failed', 0)
  }
  if (res.status === 401) {
    window.dispatchEvent(new Event(UNAUTHORIZED_EVENT))
  }
  const data = (await res.json().catch(() => null)) as Record<string, unknown> | null
  if (!res.ok) {
    const code = typeof data?.error === 'string' ? data.error : 'UNKNOWN'
    const detail = typeof data?.detail === 'string' ? data.detail : res.statusText
    throw new ApiError(code, detail, res.status)
  }
  return data as T
}

// ---- 认证 ----
export function login(username: string, password: string): Promise<Account> {
  return request('POST', '/api/auth/login', { username, password })
}
export function register(
  username: string,
  displayName: string,
  password: string,
  locale: string,
): Promise<Account> {
  return request('POST', '/api/auth/register', { username, display_name: displayName, password, locale })
}
export function logout(): Promise<{ status: string }> {
  return request('POST', '/api/auth/logout')
}
export function me(): Promise<Account> {
  return request('GET', '/api/auth/me')
}
export function updateMe(locale: string): Promise<Account> {
  return request('PATCH', '/api/auth/me', { locale })
}

// ---- 项目 ----
export function listProjects(): Promise<{ projects: Project[] }> {
  return request('GET', '/api/projects')
}
export function createProject(name: string, description: string): Promise<Project> {
  return request('POST', '/api/projects', { name, description })
}
export function deleteProject(projectId: string): Promise<{ status: string }> {
  return request('DELETE', `/api/projects/${projectId}`)
}

// ---- 会话与消息 ----
export function listConversations(): Promise<{ conversations: Conversation[] }> {
  return request('GET', '/api/conversations')
}
export function createConversation(title: string, projectId?: string): Promise<Conversation> {
  return request('POST', '/api/conversations', { title, project_id: projectId })
}
export function deleteConversation(conversationId: string): Promise<{ status: string }> {
  return request('DELETE', `/api/conversations/${conversationId}`)
}
export function listMessages(conversationId: string): Promise<{ messages: Message[] }> {
  return request('GET', `/api/conversations/${conversationId}/messages`)
}
export function appendMessage(
  conversationId: string,
  content: string,
): Promise<Message> {
  return request('POST', `/api/conversations/${conversationId}/messages`, { role: 'user', content })
}

// ---- 任务草案 ----
export function listDrafts(): Promise<{ task_drafts: TaskDraft[] }> {
  return request('GET', '/api/task-drafts')
}
export function createDraft(objective: string, skillId: string): Promise<TaskDraft> {
  return request('POST', '/api/task-drafts', { objective, skill_id: skillId })
}
export function approveDraft(draftId: string): Promise<TaskDraft> {
  return request('POST', `/api/task-drafts/${draftId}/approve`)
}
export function rejectDraft(draftId: string): Promise<TaskDraft> {
  return request('POST', `/api/task-drafts/${draftId}/reject`)
}
export function deleteDraft(draftId: string): Promise<{ status: string }> {
  return request('DELETE', `/api/task-drafts/${draftId}`)
}

// ---- 供应商 ----
export function listPresetProviders(): Promise<{ presets: PresetProvider[] }> {
  return request('GET', '/api/providers/presets')
}
export function listProviders(): Promise<{ providers: ProviderProfile[] }> {
  return request('GET', '/api/providers')
}
export function createProvider(input: {
  provider_id: string
  display_name: string
  base_url: string
  model_id: string
  api_key_env: string
}): Promise<ProviderProfile> {
  return request('POST', '/api/providers', input)
}
export function deleteProvider(profileId: string): Promise<{ status: string }> {
  return request('DELETE', `/api/providers/${profileId}`)
}
export function activateProvider(profileId: string): Promise<ProviderProfile> {
  return request('POST', `/api/providers/${profileId}/activate`)
}
export function healthCheck(profileId: string): Promise<HealthResult> {
  return request('GET', `/api/providers/${profileId}/health`)
}

// ---- 用量 ----
export function usageAggregate(window: UsageWindow): Promise<UsageAggregate> {
  return request('GET', `/api/usage/aggregate?window=${window}`)
}
export function usagePublicPool(): Promise<PublicPool> {
  return request('GET', '/api/usage/public-pool')
}

// ---- 技能 ----
export function listSkills(): Promise<{ skills: SkillInfo[] }> {
  return request('GET', '/api/skills')
}

// 导出供 store 使用的类型。
export type { DraftStatus, ProfileScope, RunEvent }
