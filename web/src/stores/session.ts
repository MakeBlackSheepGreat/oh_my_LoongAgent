// 会话 store：账户、登录状态与 locale 偏好（组合式单例）。
import { reactive } from 'vue'
import * as api from '../api/client'
import { getLocale, setLocale, type Locale } from '../i18n'
import type { Account } from '../api/types'

interface SessionState {
  me: Account | null
  accounts: Account[]
  booted: boolean
  busy: boolean
}

const state = reactive<SessionState>({ me: null, accounts: [], booted: false, busy: false })

export function useSession() {
  // boot 启动探测：未登录或会话失效时 me 为 null（401 静默）。
  async function boot(): Promise<void> {
    try {
      const me = await api.me()
      state.me = me
      if (me) {
        setLocale(me.locale as Locale)
      }
    } catch {
      state.me = null
    }
    state.booted = true
  }

  async function login(username: string, password: string): Promise<void> {
    state.busy = true
    try {
      state.me = await api.login(username, password)
      setLocale(state.me.locale as Locale)
    } finally {
      state.busy = false
    }
  }

  async function logout(): Promise<void> {
    await api.logout().catch(() => undefined)
    state.me = null
  }

  async function updateLocale(locale: Locale): Promise<void> {
    setLocale(locale)
    if (state.me) {
      try {
        state.me = await api.updateMe(locale)
      } catch {
        // 离线/失败时保持 localStorage 回退
      }
    }
  }

  return { state, boot, login, logout, updateLocale, locale: () => getLocale() }
}
