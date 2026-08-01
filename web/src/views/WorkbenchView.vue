<script setup lang="ts">
// 工作台主视图：三栏布局，左导航切换、中会话+聊天、右详情面板。
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useWorkbench } from '../stores/workbench'
import { useSession } from '../stores/session'
import { useRouter } from '../router'
import { t } from '../i18n'
import Header from '../components/Header.vue'
import ProjectList from '../components/ProjectList.vue'
import ConversationList from '../components/ConversationList.vue'
import ChatPanel from '../components/ChatPanel.vue'
import DraftPanel from '../components/DraftPanel.vue'
import ProviderPanel from '../components/ProviderPanel.vue'
import UsagePanel from '../components/UsagePanel.vue'
import SkillList from '../components/SkillList.vue'
import { UNAUTHORIZED_EVENT } from '../api/client'

type NavTab = 'projects' | 'conversations' | 'drafts' | 'providers' | 'usage' | 'skills'
type RightTab = 'drafts' | 'providers' | 'usage' | 'skills'

const wb = useWorkbench()
const session = useSession()
const router = useRouter()

const leftNav = ref<NavTab>('conversations')
const rightTab = ref<RightTab>('usage')

// 右栏可见性：窄屏时折叠
const rightVisible = ref(true)

const navItems: { key: NavTab; label: string }[] = [
  { key: 'projects', label: t('nav.projects') },
  { key: 'conversations', label: t('nav.conversations') },
  { key: 'drafts', label: t('nav.drafts') },
  { key: 'providers', label: t('nav.providers') },
  { key: 'usage', label: t('nav.usage') },
  { key: 'skills', label: t('nav.skills') },
]

const rightItems: { key: RightTab; label: string }[] = [
  { key: 'drafts', label: t('nav.drafts') },
  { key: 'providers', label: t('nav.providers') },
  { key: 'usage', label: t('nav.usage') },
  { key: 'skills', label: t('nav.skills') },
]

// 401 → 登录页
function onUnauthorized(): void {
  session.state.me = null
  wb.disconnectEvents()
  router.navigate('login')
}

onMounted(async () => {
  if (!session.state.me) {
    await session.boot()
    if (!session.state.me) {
      router.navigate('login')
      return
    }
  }
  window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized)
  wb.connectEvents()
  // 初始加载
  void wb.loadProjects()
  void wb.loadConversations()
  void wb.loadDrafts()
  void wb.loadProviders()
  void wb.loadUsage()
  void wb.loadSkills()
})

onUnmounted(() => {
  window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized)
  wb.disconnectEvents()
})

// 当前左栏组件
const leftComponent = computed(() => {
  switch (leftNav.value) {
    case 'projects': return ProjectList
    case 'conversations': return ConversationList
    case 'drafts': return DraftPanel
    case 'providers': return ProviderPanel
    case 'usage': return UsagePanel
    case 'skills': return SkillList
    default: return ConversationList
  }
})

// 当前右栏组件
const rightComponent = computed(() => {
  switch (rightTab.value) {
    case 'drafts': return DraftPanel
    case 'providers': return ProviderPanel
    case 'usage': return UsagePanel
    case 'skills': return SkillList
    default: return UsagePanel
  }
})
</script>

<template>
  <div class="workbench">
    <!-- toast 通知栈 -->
    <div class="toast-stack">
      <div
        v-for="n in wb.state.notices"
        :key="n.id"
        class="toast"
        :class="'toast-' + n.type"
      >
        {{ n.text }}
      </div>
    </div>

    <Header />

    <div class="layout">
      <!-- 左导航条 -->
      <nav class="nav-bar">
        <button
          v-for="item in navItems"
          :key="item.key"
          class="nav-btn"
          :class="{ active: leftNav === item.key }"
          @click="leftNav = item.key"
        >
          {{ item.label }}
        </button>
      </nav>

      <!-- 左栏 -->
      <div class="col-left">
        <component :is="leftComponent" />
      </div>

      <!-- 中栏：聊天 -->
      <div class="col-center">
        <ChatPanel />
      </div>

      <!-- 右栏切换按钮（窄屏可见） -->
      <button
        class="right-toggle"
        :class="{ open: rightVisible }"
        @click="rightVisible = !rightVisible"
      >
        {{ rightVisible ? '›' : '‹' }}
      </button>

      <!-- 右栏 -->
      <div class="col-right" :class="{ collapsed: !rightVisible }">
        <div class="right-tabs">
          <button
            v-for="item in rightItems"
            :key="item.key"
            class="right-tab"
            :class="{ active: rightTab === item.key }"
            @click="rightTab = item.key"
          >
            {{ item.label }}
          </button>
        </div>
        <div class="right-content">
          <component :is="rightComponent" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.workbench {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.layout {
  display: flex;
  flex: 1;
  min-height: 0;
  gap: 0;
}

/* 左导航条 */
.nav-bar {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 48px;
  flex: none;
  padding: 8px 4px;
  background: var(--surface-2);
  border-right: 1px solid var(--border);
}
.nav-btn {
  writing-mode: vertical-lr;
  letter-spacing: 0.06em;
  padding: 10px 6px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-3);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.14s ease;
  text-orientation: mixed;
}
.nav-btn:hover {
  color: var(--text-2);
  background: var(--surface-3);
}
.nav-btn.active {
  color: var(--teal);
  background: var(--teal-weak);
  font-weight: 600;
}

/* 左栏 */
.col-left {
  width: 240px;
  flex: none;
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-right: 1px solid var(--border);
  background: var(--surface);
}

/* 中栏 */
.col-center {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 12px;
  background: var(--bg);
}

/* 右栏切换 */
.right-toggle {
  display: none;
  position: absolute;
  right: 280px;
  top: 50%;
  z-index: 10;
  width: 22px;
  height: 44px;
  border: 1px solid var(--border);
  border-right: none;
  border-radius: 8px 0 0 8px;
  background: var(--surface);
  color: var(--text-3);
  font-size: 16px;
  cursor: pointer;
  transform: translateY(-50%);
  transition: right 0.22s ease, color 0.14s ease;
}
.right-toggle.open {
  right: 0;
}
.right-toggle:hover {
  color: var(--teal);
}

/* 右栏 */
.col-right {
  width: 280px;
  flex: none;
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-left: 1px solid var(--border);
  background: var(--surface);
  transition: width 0.22s ease, padding 0.22s ease;
  overflow: hidden;
}
.col-right.collapsed {
  width: 0;
  padding: 0;
  border: none;
}

.right-tabs {
  display: flex;
  gap: 2px;
  padding: 8px 10px 0;
  flex: none;
  border-bottom: 1px solid var(--border);
}
.right-tab {
  flex: 1;
  padding: 5px 6px;
  border: none;
  border-radius: 6px 6px 0 0;
  background: transparent;
  color: var(--text-3);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.14s ease;
}
.right-tab.active {
  background: var(--surface-2);
  color: var(--text);
  font-weight: 600;
}
.right-tab:hover:not(.active) {
  color: var(--text-2);
}

.right-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 8px;
}

/* 窄屏响应式 */
@media (max-width: 900px) {
  .col-left {
    width: 180px;
  }
  .col-right {
    width: 220px;
  }
  .right-toggle {
    display: block;
  }
}
@media (max-width: 640px) {
  .nav-bar {
    width: 40px;
  }
  .nav-btn {
    font-size: 10px;
    padding: 8px 4px;
  }
  .col-left {
    width: 0;
    border: none;
    overflow: hidden;
  }
  .col-center {
    padding: 8px;
  }
}
</style>