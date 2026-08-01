<script setup lang="ts">
// 顶栏：账户信息、语言切换、SSE 状态。
import { useSession } from '../stores/session'
import { useWorkbench } from '../stores/workbench'
import { useRouter } from '../router'
import { t, type Locale } from '../i18n'

const session = useSession()
const wb = useWorkbench()
const router = useRouter()

function cycleLocale(): void {
  const next: Locale = session.state.me?.locale === 'en' ? 'zh-CN' : 'en'
  void session.updateLocale(next)
}
</script>

<template>
  <header class="header">
    <div class="header-left">
      <div class="header-brand">
        <div class="header-brand-mark">A</div>
        <span class="header-title">{{ t('app.title') }}</span>
      </div>
      <span v-if="session.state.me" class="header-account">
        {{ session.state.me.display_name }}
      </span>
    </div>

    <div class="header-right">
      <!-- SSE 状态 -->
      <span v-if="wb.state.sse === 'reconnecting'" class="sse-warn badge st-wait">
        {{ t('sse.reconnecting') }}
      </span>

      <!-- 语言切换 -->
      <button class="btn btn-ghost btn-sm" title="Language" @click="cycleLocale">
        {{ session.state.me?.locale === 'en' ? '中文' : 'EN' }}
      </button>

      <!-- 登出 -->
      <button
        class="btn btn-ghost btn-sm"
        @click="session.logout(); router.navigate('login')"
      >
        {{ t('auth.logout') }}
      </button>
    </div>
  </header>
</template>

<style scoped>
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 50px;
  padding: 0 18px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--border);
  flex: none;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.header-brand {
  display: flex;
  align-items: center;
  gap: 9px;
}
.header-brand-mark {
  width: 26px;
  height: 26px;
  border-radius: 8px;
  background: linear-gradient(135deg, #14a596, #0a7569);
  color: #fff;
  font-size: 13px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 6px rgba(14, 147, 132, 0.3);
}
.header-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.header-account {
  font-size: 12px;
  color: var(--text-3);
  padding: 2px 10px;
  border-radius: 999px;
  background: var(--surface-2);
  border: 1px solid var(--border);
}
.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.sse-warn {
  font-size: 11px;
}
</style>