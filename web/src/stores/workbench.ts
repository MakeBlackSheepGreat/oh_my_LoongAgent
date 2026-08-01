// 工作台数据 store：项目/会话/消息/草案/供应商/用量/技能 + SSE 实时事件 + 轻量通知。
import { reactive } from 'vue'
import * as api from '../api/client'
import { t } from '../i18n'
import type {
  Conversation,
  HealthResult,
  Message,
  PresetProvider,
  Project,
  ProviderProfile,
  PublicPool,
  SkillInfo,
  TaskDraft,
  UsageAggregate,
  UsageWindow,
} from '../api/types'

export type NoticeType = 'ok' | 'info' | 'warn' | 'error'
export interface Notice {
  id: number
  type: NoticeType
  text: string
}

interface LoadingFlags {
  projects: boolean
  conversations: boolean
  messages: boolean
  sendMessage: boolean
  drafts: boolean
  providers: boolean
  skills: boolean
  usage: boolean
}

interface WorkbenchState {
  projects: Project[]
  conversations: Conversation[]
  currentConversationId: string | null
  messages: Message[]
  drafts: TaskDraft[]
  providers: ProviderProfile[]
  presets: PresetProvider[]
  health: Record<string, HealthResult>
  skills: SkillInfo[]
  usage: UsageAggregate | null
  pool: PublicPool | null
  window: UsageWindow
  loading: LoadingFlags
  sse: 'connected' | 'reconnecting' | 'closed'
  notices: Notice[]
}

const state = reactive<WorkbenchState>({
  projects: [],
  conversations: [],
  currentConversationId: null,
  messages: [],
  drafts: [],
  providers: [],
  presets: [],
  health: {},
  skills: [],
  usage: null,
  pool: null,
  window: 'today',
  loading: {
    projects: false,
    conversations: false,
    messages: false,
    sendMessage: false,
    drafts: false,
    providers: false,
    skills: false,
    usage: false,
  },
  sse: 'closed',
  notices: [],
})

let noticeSeq = 0
let eventSource: EventSource | null = null
let reconnectTimer: number | undefined

function notify(type: NoticeType, text: string): void {
  const id = ++noticeSeq
  state.notices.push({ id, type, text })
  window.setTimeout(() => {
    const idx = state.notices.findIndex((n) => n.id === id)
    if (idx >= 0) state.notices.splice(idx, 1)
  }, 4200)
}

function errorText(err: unknown): string {
  return err instanceof Error ? err.message : t('common.error')
}

export function useWorkbench() {
  // ---- 项目 ----
  async function loadProjects(): Promise<void> {
    state.loading.projects = true
    try {
      const { projects } = await api.listProjects()
      state.projects = projects ?? []
    } catch (err) {
      notify('error', errorText(err))
    } finally {
      state.loading.projects = false
    }
  }

  async function createProject(name: string, description: string): Promise<boolean> {
    try {
      const project = await api.createProject(name, description)
      state.projects.unshift(project)
      return true
    } catch (err) {
      notify('error', errorText(err))
      return false
    }
  }

  async function deleteProject(projectId: string): Promise<void> {
    try {
      await api.deleteProject(projectId)
      state.projects = state.projects.filter((p) => p.project_id !== projectId)
      notify('ok', t('common.delete'))
    } catch (err) {
      notify('error', errorText(err))
    }
  }

  // ---- 会话与消息 ----
  async function loadConversations(): Promise<void> {
    state.loading.conversations = true
    try {
      const { conversations } = await api.listConversations()
      state.conversations = conversations ?? []
    } catch (err) {
      notify('error', errorText(err))
    } finally {
      state.loading.conversations = false
    }
  }

  async function createConversation(title: string, projectId?: string): Promise<boolean> {
    try {
      const conversation = await api.createConversation(title, projectId)
      state.conversations.unshift(conversation)
      state.currentConversationId = conversation.conversation_id
      state.messages = []
      return true
    } catch (err) {
      notify('error', errorText(err))
      return false
    }
  }

  async function deleteConversation(conversationId: string): Promise<void> {
    try {
      await api.deleteConversation(conversationId)
      state.conversations = state.conversations.filter((c) => c.conversation_id !== conversationId)
      if (state.currentConversationId === conversationId) {
        state.currentConversationId = null
        state.messages = []
      }
      notify('ok', t('common.delete'))
    } catch (err) {
      notify('error', errorText(err))
    }
  }

  async function selectConversation(conversationId: string): Promise<void> {
    state.currentConversationId = conversationId
    state.messages = []
    state.loading.messages = true
    try {
      const { messages } = await api.listMessages(conversationId)
      state.messages = messages ?? []
    } catch (err) {
      notify('error', errorText(err))
      state.currentConversationId = null
    } finally {
      state.loading.messages = false
    }
  }

  async function sendMessage(content: string): Promise<boolean> {
    const conversationId = state.currentConversationId
    if (!conversationId || !content.trim()) return false
    state.loading.sendMessage = true
    try {
      const message = await api.appendMessage(conversationId, content.trim())
      state.messages.push(message)
      return true
    } catch (err) {
      notify('error', errorText(err))
      return false
    } finally {
      state.loading.sendMessage = false
    }
  }

  // ---- 任务草案 ----
  async function loadDrafts(): Promise<void> {
    state.loading.drafts = true
    try {
      const { task_drafts } = await api.listDrafts()
      state.drafts = task_drafts ?? []
    } catch (err) {
      notify('error', errorText(err))
    } finally {
      state.loading.drafts = false
    }
  }

  async function createDraft(objective: string, skillId: string): Promise<boolean> {
    try {
      const draft = await api.createDraft(objective, skillId)
      state.drafts.unshift(draft)
      return true
    } catch (err) {
      notify('error', errorText(err))
      return false
    }
  }

  async function approveDraft(draftId: string): Promise<boolean> {
    try {
      const draft = await api.approveDraft(draftId)
      replaceDraft(draft)
      notify('ok', t('status.running'))
      return true
    } catch (err) {
      notify('error', errorText(err))
      return false
    }
  }

  async function rejectDraft(draftId: string): Promise<boolean> {
    try {
      const draft = await api.rejectDraft(draftId)
      replaceDraft(draft)
      return true
    } catch (err) {
      notify('error', errorText(err))
      return false
    }
  }

  function replaceDraft(draft: TaskDraft): void {
    const idx = state.drafts.findIndex((d) => d.draft_id === draft.draft_id)
    if (idx >= 0) state.drafts[idx] = draft
  }

  async function deleteDraft(draftId: string): Promise<void> {
    try {
      await api.deleteDraft(draftId)
      state.drafts = state.drafts.filter((d) => d.draft_id !== draftId)
    } catch (err) {
      notify('error', errorText(err))
    }
  }

  // ---- 供应商 ----
  async function loadPresets(): Promise<void> {
    try {
      const { presets } = await api.listPresetProviders()
      state.presets = presets ?? []
    } catch {
      state.presets = []
    }
  }

  async function loadProviders(): Promise<void> {
    state.loading.providers = true
    try {
      const { providers } = await api.listProviders()
      state.providers = providers ?? []
    } catch (err) {
      notify('error', errorText(err))
    } finally {
      state.loading.providers = false
    }
  }

  async function createProvider(input: {
    provider_id: string
    display_name: string
    base_url: string
    model_id: string
    api_key_env: string
  }): Promise<boolean> {
    try {
      const profile = await api.createProvider(input)
      state.providers.unshift(profile)
      return true
    } catch (err) {
      notify('error', errorText(err))
      return false
    }
  }

  async function createFromPreset(preset: PresetProvider): Promise<boolean> {
    return createProvider({
      provider_id: preset.provider_id,
      display_name: preset.display_name,
      base_url: preset.base_url,
      model_id: preset.model_id,
      api_key_env: preset.api_key_env,
    })
  }

  async function deleteProvider(profileId: string): Promise<void> {
    try {
      await api.deleteProvider(profileId)
      state.providers = state.providers.filter((p) => p.profile_id !== profileId)
      delete state.health[profileId]
      notify('ok', t('common.delete'))
    } catch (err) {
      notify('error', errorText(err))
    }
  }

  async function activateProvider(profileId: string): Promise<void> {
    try {
      const profile = await api.activateProvider(profileId)
      const idx = state.providers.findIndex((p) => p.profile_id === profileId)
      if (idx >= 0) state.providers[idx] = profile
    } catch (err) {
      notify('error', errorText(err))
    }
  }

  async function checkHealth(profileId: string): Promise<void> {
    try {
      state.health[profileId] = await api.healthCheck(profileId)
    } catch (err) {
      notify('error', errorText(err))
    }
  }

  // ---- 用量 ----
  async function loadUsage(window: UsageWindow = state.window): Promise<void> {
    state.window = window
    state.loading.usage = true
    try {
      const [usage, pool] = await Promise.all([api.usageAggregate(window), api.usagePublicPool()])
      state.usage = usage
      state.pool = pool
    } catch (err) {
      notify('error', errorText(err))
    } finally {
      state.loading.usage = false
    }
  }

  // ---- 技能 ----
  async function loadSkills(): Promise<void> {
    state.loading.skills = true
    try {
      const { skills } = await api.listSkills()
      state.skills = skills ?? []
    } catch (err) {
      notify('error', errorText(err))
    } finally {
      state.loading.skills = false
    }
  }

  // ---- SSE 实时事件 ----
  function connectEvents(): void {
    if (eventSource) return
    eventSource = new EventSource('/api/events')
    eventSource.onopen = () => {
      state.sse = 'connected'
    }
    eventSource.onmessage = (ev) => {
      state.sse = 'connected'
      try {
        const event = JSON.parse(ev.data) as { kind?: string; message?: string; run_id?: string }
        // 运行状态事件到达：刷新草案列表，让审批结果即时可见。
        if (event.kind || event.run_id) {
          void loadDrafts()
          void loadUsage()
        }
      } catch {
        // 非 JSON 心跳忽略
      }
    }
    eventSource.onerror = () => {
      state.sse = 'reconnecting'
      // 检查会话是否仍然有效：如果 401 则停止重连。
      fetch('/api/auth/me', { credentials: 'include' })
        .then((res) => {
          if (res.status === 401) {
            state.sse = 'closed'
            disconnectEvents()
            window.dispatchEvent(new Event('auth:unauthorized'))
            return
          }
          // 会话有效，启动重连
          if (reconnectTimer === undefined) {
            reconnectTimer = window.setTimeout(() => {
              reconnectTimer = undefined
              if (eventSource) {
                eventSource.close()
                eventSource = null
                connectEvents()
              }
            }, 3000)
          }
        })
        .catch(() => {
          // 网络错误，也尝试重连
          if (reconnectTimer === undefined) {
            reconnectTimer = window.setTimeout(() => {
              reconnectTimer = undefined
              if (eventSource) {
                eventSource.close()
                eventSource = null
                connectEvents()
              }
            }, 3000)
          }
        })
    }
  }

  function disconnectEvents(): void {
    if (reconnectTimer !== undefined) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = undefined
    }
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    state.sse = 'closed'
  }

  return {
    state,
    notify,
    loadProjects,
    createProject,
    deleteProject,
    loadConversations,
    createConversation,
    deleteConversation,
    selectConversation,
    sendMessage,
    loadDrafts,
    createDraft,
    approveDraft,
    rejectDraft,
    deleteDraft,
    loadProviders,
    createProvider,
    createFromPreset,
    loadPresets,
    deleteProvider,
    activateProvider,
    checkHealth,
    loadUsage,
    loadSkills,
    connectEvents,
    disconnectEvents,
  }
}
