<script setup lang="ts">
// 任务草案面板：创建/审批/拒绝/删除。
import { ref, onMounted } from 'vue'
import { useWorkbench } from '../stores/workbench'
import { t } from '../i18n'

const wb = useWorkbench()
const showCreate = ref(false)
const objective = ref('')
const skillId = ref('')

onMounted(() => void wb.loadDrafts())

async function handleCreate(): Promise<void> {
  if (!objective.value.trim()) return
  const ok = await wb.createDraft(objective.value.trim(), skillId.value || 'generic')
  if (ok) { showCreate.value = false; objective.value = ''; skillId.value = '' }
}

function statusBadge(status: string): string {
  switch (status) {
    case 'draft': return 'st-wait'
    case 'approved': return 'st-ok'
    case 'rejected': return 'st-fail'
    default: return 'st-info'
  }
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <span class="panel-title">{{ t('nav.drafts') }}</span>
      <button class="btn btn-sm btn-primary" @click="showCreate = true">
        {{ t('common.create') }}
      </button>
    </div>

    <div class="panel-body">
      <div v-if="wb.state.loading.drafts" class="empty">
        <span class="spinner"></span>
      </div>
      <div v-else-if="wb.state.drafts.length === 0" class="empty">
        <div class="empty-mark">📋</div>
        <span>{{ t('drafts.empty') }}</span>
      </div>
      <div v-else class="list">
        <div
          v-for="d in wb.state.drafts"
          :key="d.draft_id"
          class="list-item"
        >
          <div class="list-item-main">
            <span class="list-item-title">{{ d.objective }}</span>
            <span class="list-item-sub">
              <span class="mono">{{ d.skill_id }}</span>
              <span v-if="d.run_id"> · {{ t('drafts.runId') }}: {{ d.run_id.slice(0, 10) }}…</span>
            </span>
          </div>
          <span class="badge" :class="statusBadge(d.status)">
            {{ t('drafts.status.' + d.status) }}
          </span>
          <div class="list-item-actions">
            <button
              v-if="d.status === 'draft'"
              class="btn btn-sm btn-primary"
              @click="void wb.approveDraft(d.draft_id)"
            >
              {{ t('drafts.approve') }}
            </button>
            <button
              v-if="d.status === 'draft'"
              class="btn btn-sm btn-ghost btn-danger"
              @click="void wb.rejectDraft(d.draft_id)"
            >
              {{ t('drafts.reject') }}
            </button>
            <button
              class="btn btn-sm btn-ghost btn-danger"
              @click="void wb.deleteDraft(d.draft_id)"
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
          <h3 class="dialog-title">{{ t('drafts.create') }}</h3>
          <textarea
            v-model="objective"
            class="field"
            :placeholder="t('drafts.objective')"
            rows="3"
          ></textarea>
          <select v-model="skillId" class="field">
            <option value="generic">{{ t('drafts.skillPlaceholder') }}</option>
            <option
              v-for="s in wb.state.skills"
              :key="s.skill_id"
              :value="s.skill_id"
            >
              {{ s.title }}
            </option>
          </select>
          <div class="dialog-actions">
            <button class="btn" @click="showCreate = false">{{ t('common.cancel') }}</button>
            <button class="btn btn-primary" :disabled="!objective.trim()" @click="handleCreate">
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
  max-width: 420px;
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