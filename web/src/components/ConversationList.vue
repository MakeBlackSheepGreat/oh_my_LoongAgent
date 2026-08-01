<script setup lang="ts">
// 会话列表：创建/切换/删除。
import { ref, onMounted } from 'vue'
import { useWorkbench } from '../stores/workbench'
import { t } from '../i18n'

const wb = useWorkbench()
const showCreate = ref(false)
const title = ref('')
const projectId = ref('')

onMounted(() => void wb.loadConversations())

async function handleCreate(): Promise<void> {
  if (!title.value.trim()) return
  const ok = await wb.createConversation(
    title.value.trim(),
    projectId.value || undefined,
  )
  if (ok) { showCreate.value = false; title.value = ''; projectId.value = '' }
}

function confirmDelete(c: { conversation_id: string; title: string }): void {
  if (window.confirm(t('conversations.deleteConfirm', { title: c.title }))) {
    void wb.deleteConversation(c.conversation_id)
  }
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <span class="panel-title">{{ t('nav.conversations') }}</span>
      <button class="btn btn-sm btn-primary" @click="showCreate = true">
        {{ t('common.create') }}
      </button>
    </div>

    <div class="panel-body">
      <div v-if="wb.state.loading.conversations" class="empty">
        <span class="spinner"></span>
      </div>
      <div v-else-if="wb.state.conversations.length === 0" class="empty">
        <div class="empty-mark">💬</div>
        <span>{{ t('conversations.empty') }}</span>
      </div>
      <div v-else class="list">
        <div
          v-for="c in wb.state.conversations"
          :key="c.conversation_id"
          class="list-item"
          :class="{ active: c.conversation_id === wb.state.currentConversationId }"
          @click="void wb.selectConversation(c.conversation_id)"
        >
          <div class="list-item-main">
            <span class="list-item-title">{{ c.title }}</span>
            <span class="list-item-sub mono">{{ c.conversation_id.slice(0, 10) }}…</span>
          </div>
          <div class="list-item-actions">
            <button
              class="btn btn-ghost btn-sm btn-danger"
              title="Delete"
              @click.stop="confirmDelete(c)"
            >
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showCreate" class="overlay" @click.self="showCreate = false">
        <div class="dialog">
          <h3 class="dialog-title">{{ t('conversations.create') }}</h3>
          <input
            v-model="title"
            class="field"
            :placeholder="t('conversations.title')"
            @keyup.enter="handleCreate"
          />
          <select v-model="projectId" class="field">
            <option value="">{{ t('projects.empty') }}</option>
            <option
              v-for="p in wb.state.projects"
              :key="p.project_id"
              :value="p.project_id"
            >
              {{ p.name }}
            </option>
          </select>
          <div class="dialog-actions">
            <button class="btn" @click="showCreate = false">{{ t('common.cancel') }}</button>
            <button class="btn btn-primary" :disabled="!title.trim()" @click="handleCreate">
              {{ t('common.save') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 500;
  background: rgba(33, 42, 55, 0.38);
  backdrop-filter: blur(3px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  animation: fade-in 0.16s ease;
}
@keyframes fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}
.dialog {
  width: 100%;
  max-width: 400px;
  background: var(--surface);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  padding: 26px 28px 22px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  animation: rise 0.2s ease;
}
@keyframes rise {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}
.dialog-title {
  font-size: 16px;
  color: var(--text);
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
</style>