<script setup lang="ts">
// 聊天面板：消息流渲染 + 发送框。
import { ref, nextTick, watch } from 'vue'
import { useWorkbench } from '../stores/workbench'
import { t } from '../i18n'

const wb = useWorkbench()
const input = ref('')
const chatBody = ref<HTMLElement | null>(null)

watch(
  () => wb.state.messages.length,
  async () => {
    await nextTick()
    if (chatBody.value) chatBody.value.scrollTop = chatBody.value.scrollHeight
  },
)

function roleLabel(role: string): string {
  switch (role) {
    case 'user': return 'You'
    case 'assistant': return 'AI'
    case 'system': return 'System'
    case 'tool': return 'Tool'
    default: return role
  }
}

async function send(): Promise<void> {
  const text = input.value
  if (!text.trim() || wb.state.loading.sendMessage) return
  input.value = ''
  await wb.sendMessage(text)
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    void send()
  }
}
</script>

<template>
  <div class="panel chat-panel">
    <div class="panel-header">
      <span class="panel-title">
        {{ t('nav.conversations') }}
        <span v-if="wb.state.currentConversationId" class="meta mono">
          {{ wb.state.currentConversationId.slice(0, 10) }}…
        </span>
      </span>
    </div>

    <div ref="chatBody" class="panel-body chat-body">
      <div v-if="wb.state.loading.messages" class="empty">
        <span class="spinner"></span>
      </div>
      <div v-else-if="wb.state.messages.length === 0" class="empty">
        <div class="empty-mark">✎</div>
        <span>{{ t('conversations.noMessages') }}</span>
      </div>
      <div v-else class="msg-list">
        <div
          v-for="m in wb.state.messages"
          :key="m.message_id"
          class="msg"
          :class="`msg-${m.role}`"
        >
          <div class="msg-label">{{ roleLabel(m.role) }}</div>
          <div class="msg-content">{{ m.content }}</div>
          <div class="msg-meta">
            <span class="mono">{{ m.message_id.slice(0, 10) }}…</span>
            <span>{{ new Date(m.created_at).toLocaleTimeString() }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="panel-footer">
      <input
        v-model="input"
        class="field chat-input"
        :placeholder="t('conversations.sendPlaceholder')"
        :disabled="!wb.state.currentConversationId"
        @keydown="onKeydown"
      />
      <button
        class="btn btn-primary"
        :disabled="!input.trim() || !wb.state.currentConversationId || wb.state.loading.sendMessage"
        @click="send"
      >
        <span v-if="wb.state.loading.sendMessage" class="spinner"></span>
        <span>{{ wb.state.loading.sendMessage ? t('conversations.thinking') : t('conversations.send') }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.chat-panel {
  flex: 1;
  min-height: 0;
}
.chat-body {
  display: flex;
  flex-direction: column;
}
.msg-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-bottom: 8px;
}
.msg {
  padding: 10px 14px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--surface-2);
  max-width: 88%;
}
.msg-user {
  align-self: flex-end;
  background: linear-gradient(180deg, #e8f0fc, #dde7f9);
  border-color: rgba(47, 111, 219, 0.2);
  border-bottom-right-radius: 4px;
}
.msg-assistant {
  align-self: flex-start;
  border-bottom-left-radius: 4px;
  background: var(--surface);
}
.msg-system {
  align-self: center;
  max-width: 70%;
  background: var(--amber-weak);
  border-color: rgba(180, 122, 18, 0.15);
  font-size: 12.5px;
}
.msg-tool {
  align-self: flex-start;
  border-color: rgba(14, 147, 132, 0.15);
}
.msg-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 4px;
}
.msg-user .msg-label {
  text-align: right;
  color: var(--cobalt-deep);
}
.msg-content {
  font-size: 13.5px;
  line-height: 1.55;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-word;
}
.msg-meta {
  display: flex;
  gap: 10px;
  margin-top: 6px;
  font-size: 11px;
  color: var(--text-3);
}
.chat-input {
  flex: 1;
  min-width: 0;
}
</style>