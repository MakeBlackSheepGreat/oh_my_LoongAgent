// 极简 i18n：无第三方依赖（npm 受限，spec task8.0 允许手写）。
// 组合式响应式 locale；登录后按账户偏好设置，未登录用 localStorage 回退。
import { reactive } from 'vue'
import zhCN from './zh-CN'
import en from './en'

export type Locale = 'zh-CN' | 'en'

export const LOCALES: Locale[] = ['zh-CN', 'en']

const messages: Record<Locale, Record<string, unknown>> = { 'zh-CN': zhCN, en }

const STORAGE_KEY = 'workbench.locale'

function initialLocale(): Locale {
  const saved = localStorage.getItem(STORAGE_KEY)
  return saved === 'zh-CN' || saved === 'en' ? saved : 'zh-CN'
}

const state = reactive<{ locale: Locale }>({ locale: initialLocale() })

export function getLocale(): Locale {
  return state.locale
}

export function setLocale(locale: Locale): void {
  state.locale = locale
  localStorage.setItem(STORAGE_KEY, locale)
}

// t 按点路径取文案，支持 {param} 插值；缺键时回退路径本身。
export function t(path: string, params?: Record<string, string | number>): string {
  const keys = path.split('.')
  let node: unknown = messages[state.locale]
  for (const key of keys) {
    if (node && typeof node === 'object' && key in (node as Record<string, unknown>)) {
      node = (node as Record<string, unknown>)[key]
    } else {
      node = undefined
      break
    }
  }
  let out = typeof node === 'string' ? node : path
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      out = out.replaceAll(`{${k}}`, String(v))
    }
  }
  return out
}
