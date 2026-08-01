<script setup lang="ts">
// 项目列表：CRUD 对话框。
import { ref, onMounted } from 'vue'
import { useWorkbench } from '../stores/workbench'
import { t } from '../i18n'

const wb = useWorkbench()
const showCreate = ref(false)
const name = ref('')
const desc = ref('')

onMounted(() => void wb.loadProjects())

async function handleCreate(): Promise<void> {
  if (!name.value.trim()) return
  const ok = await wb.createProject(name.value.trim(), desc.value.trim())
  if (ok) { showCreate.value = false; name.value = ''; desc.value = '' }
}

function confirmDelete(projectId: string, projectName: string): void {
  if (window.confirm(t('projects.deleteConfirm', { name: projectName }))) {
    void wb.deleteProject(projectId)
  }
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <span class="panel-title">{{ t('nav.projects') }}</span>
      <button class="btn btn-sm btn-primary" @click="showCreate = true">
        {{ t('common.create') }}
      </button>
    </div>

    <div class="panel-body">
      <div v-if="wb.state.loading.projects" class="empty">
        <span class="spinner"></span>
      </div>
      <div v-else-if="wb.state.projects.length === 0" class="empty">
        <div class="empty-mark">📁</div>
        <span>{{ t('projects.empty') }}</span>
      </div>
      <div v-else class="list">
        <div
          v-for="p in wb.state.projects"
          :key="p.project_id"
          class="list-item"
        >
          <div class="list-item-main">
            <span class="list-item-title">{{ p.name }}</span>
            <span v-if="p.description" class="list-item-sub">{{ p.description }}</span>
          </div>
          <div class="list-item-actions">
            <button
              class="btn btn-ghost btn-sm btn-danger"
              title="Delete"
              @click="confirmDelete(p.project_id, p.name)"
            >
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建对话框 -->
    <Teleport to="body">
      <div v-if="showCreate" class="overlay" @click.self="showCreate = false">
        <div class="dialog">
          <h3 class="dialog-title">{{ t('projects.create') }}</h3>
          <p class="dialog-hint">{{ t('projects.createHint') }}</p>
          <input
            v-model="name"
            class="field"
            :placeholder="t('projects.name')"
            @keyup.enter="handleCreate"
          />
          <textarea
            v-model="desc"
            class="field"
            :placeholder="t('projects.description')"
            rows="2"
          ></textarea>
          <div class="dialog-actions">
            <button class="btn" @click="showCreate = false">{{ t('common.cancel') }}</button>
            <button class="btn btn-primary" :disabled="!name.trim()" @click="handleCreate">
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
.dialog-hint {
  font-size: 12.5px;
  color: var(--text-3);
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
</style>