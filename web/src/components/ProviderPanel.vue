<script setup lang="ts">
// 供应商档案面板：预设列表一键添加 + 自定义创建 + 激活/健康检查/删除。
import { ref, computed, onMounted } from 'vue'
import { useWorkbench } from '../stores/workbench'
import { t } from '../i18n'
import type { PresetProvider } from '../api/types'

const wb = useWorkbench()

const showCreate = ref(false)
const showPresets = ref(false)
const searchQuery = ref('')
const addingIds = ref<Set<string>>(new Set())

const form = ref({ provider_id: '', display_name: '', base_url: '', model_id: '', api_key_env: '' })

onMounted(() => {
  void wb.loadProviders()
  void wb.loadPresets()
})

// 已添加的 provider_id 集合，用于标记预设列表中哪些已添加。
const addedProviderIds = computed(() => new Set(wb.state.providers.map((p) => p.provider_id)))

// 按搜索词过滤预设列表。
const filteredPresets = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return wb.state.presets
  return wb.state.presets.filter(
    (p) =>
      p.display_name.toLowerCase().includes(q) ||
      p.provider_id.toLowerCase().includes(q),
  )
})

// 按 category 分组。
const presetsByCategory = computed(() => {
  const map = new Map<string, PresetProvider[]>()
  for (const p of filteredPresets.value) {
    const list = map.get(p.category) ?? []
    list.push(p)
    map.set(p.category, list)
  }
  return map
})

async function handleCreate(): Promise<void> {
  const f = form.value
  if (!f.provider_id.trim() || !f.display_name.trim()) return
  const ok = await wb.createProvider({
    provider_id: f.provider_id.trim(),
    display_name: f.display_name.trim(),
    base_url: f.base_url.trim(),
    model_id: f.model_id.trim(),
    api_key_env: f.api_key_env.trim() || 'API_KEY',
  })
  if (ok) {
    showCreate.value = false
    form.value = { provider_id: '', display_name: '', base_url: '', model_id: '', api_key_env: '' }
  }
}

async function handleAddPreset(preset: PresetProvider): Promise<void> {
  if (addingIds.value.has(preset.provider_id)) return
  addingIds.value = new Set(addingIds.value).add(preset.provider_id)
  const ok = await wb.createFromPreset(preset)
  if (ok) {
    wb.notify('ok', t('providers.addSuccess'))
  }
  const next = new Set(addingIds.value)
  next.delete(preset.provider_id)
  addingIds.value = next
}

function openPresets(): void {
  showPresets.value = true
  searchQuery.value = ''
  void wb.loadPresets()
}

function categoryLabel(cat: string): string {
  const key = `providers.category.${cat}` as keyof typeof t
  const label = t(key)
  return typeof label === 'string' ? label : cat
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <span class="panel-title">{{ t('nav.providers') }}</span>
      <div class="header-actions">
        <button class="btn btn-sm btn-ghost" @click="openPresets">
          {{ t('providers.presets') }}
        </button>
        <button class="btn btn-sm btn-primary" @click="showCreate = true">
          {{ t('common.create') }}
        </button>
      </div>
    </div>

    <div class="panel-body">
      <div v-if="wb.state.loading.providers" class="empty">
        <span class="spinner"></span>
      </div>
      <div v-else-if="wb.state.providers.length === 0" class="empty">
        <div class="empty-mark">⚙️</div>
        <span>{{ t('providers.empty') }}</span>
      </div>
      <div v-else class="list">
        <div
          v-for="p in wb.state.providers"
          :key="p.profile_id"
          class="list-item"
        >
          <div class="list-item-main">
            <span class="list-item-title">{{ p.display_name }}</span>
            <span class="list-item-sub">
              <span class="mono">{{ p.model_id }}</span>
              <span v-if="p.is_active" class="badge st-ok" style="margin-left:6px">
                {{ t('providers.active') }}
              </span>
            </span>
          </div>
          <span class="badge st-info" style="font-size:11px">
            {{ p.scope === 'system' ? t('providers.scope.system') : t('providers.scope.account') }}
          </span>
          <div class="list-item-actions">
            <button
              class="btn btn-sm btn-ghost"
              title="Health"
              @click="void wb.checkHealth(p.profile_id)"
            >
              {{ t('providers.health') }}
            </button>
            <button
              v-if="!p.is_active"
              class="btn btn-sm btn-ghost"
              @click="void wb.activateProvider(p.profile_id)"
            >
              Activate
            </button>
            <button
              v-if="p.scope === 'account'"
              class="btn btn-sm btn-ghost btn-danger"
              @click="void wb.deleteProvider(p.profile_id)"
            >
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 预设供应商对话框 -->
    <Teleport to="body">
      <div v-if="showPresets" class="overlay" @click.self="showPresets = false">
        <div class="preset-dialog">
          <div class="preset-dialog-header">
            <h3 class="dialog-title">{{ t('providers.addFromPreset') }}</h3>
            <p class="dialog-hint">{{ t('providers.addFromPresetHint') }}</p>
            <input
              v-model="searchQuery"
              class="field search-input"
              :placeholder="t('providers.displayName')"
            />
          </div>

          <div class="preset-list">
            <div v-if="wb.state.presets.length === 0" class="empty" style="padding:32px 0">
              <span class="spinner"></span>
            </div>
            <template v-else-if="filteredPresets.length === 0">
              <div class="empty" style="padding:32px 0">
                <span>{{ t('common.empty') }}</span>
              </div>
            </template>
            <template v-else>
              <div
                v-for="[cat, items] in presetsByCategory"
                :key="cat"
                class="preset-group"
              >
                <div class="preset-group-label">{{ categoryLabel(cat) }}</div>
                <div
                  v-for="preset in items"
                  :key="preset.provider_id"
                  class="preset-card"
                >
                  <div class="preset-card-info">
                    <span class="preset-card-name">{{ preset.display_name }}</span>
                    <span class="preset-card-model mono">{{ preset.model_id }}</span>
                    <span class="preset-card-url mono">{{ preset.base_url }}</span>
                  </div>
                  <div class="preset-card-actions">
                    <a
                      v-if="preset.website_url"
                      :href="preset.website_url"
                      target="_blank"
                      class="btn btn-sm btn-ghost"
                      rel="noopener"
                    >官网</a>
                    <button
                      v-if="addedProviderIds.has(preset.provider_id)"
                      class="btn btn-sm"
                      disabled
                    >
                      {{ t('providers.presetAdded') }}
                    </button>
                    <button
                      v-else
                      class="btn btn-sm btn-primary"
                      :disabled="addingIds.has(preset.provider_id)"
                      @click="void handleAddPreset(preset)"
                    >
                      <span v-if="addingIds.has(preset.provider_id)" class="spinner" style="width:14px;height:14px;"></span>
                      <span v-else>{{ t('providers.presetAdd') }}</span>
                    </button>
                  </div>
                </div>
              </div>
            </template>
          </div>

          <div class="dialog-actions">
            <button class="btn" @click="showPresets = false">
              {{ t('common.close') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 自定义创建对话框 -->
    <Teleport to="body">
      <div v-if="showCreate" class="overlay" @click.self="showCreate = false">
        <div class="dialog">
          <h3 class="dialog-title">{{ t('providers.customCreate') }}</h3>
          <input v-model="form.provider_id" class="field" :placeholder="t('providers.providerId')" />
          <input v-model="form.display_name" class="field" :placeholder="t('providers.displayName')" />
          <input v-model="form.base_url" class="field" :placeholder="t('providers.baseUrl')" />
          <input v-model="form.model_id" class="field" :placeholder="t('providers.modelId')" />
          <input v-model="form.api_key_env" class="field" :placeholder="t('providers.apiKeyEnv')" />
          <div class="dialog-actions">
            <button class="btn" @click="showCreate = false">{{ t('common.cancel') }}</button>
            <button
              class="btn btn-primary"
              :disabled="!form.provider_id.trim() || !form.display_name.trim()"
              @click="handleCreate"
            >
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
  gap: 10px;
  animation: rise 0.2s ease;
}
.preset-dialog {
  width: 100%;
  max-width: 600px;
  max-height: 80vh;
  background: var(--surface);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  padding: 26px 28px 22px;
  display: flex;
  flex-direction: column;
  animation: rise 0.2s ease;
}
@keyframes rise {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}
.dialog-title {
  font-size: 16px;
  color: var(--text);
  margin-bottom: 2px;
}
.dialog-hint {
  font-size: 13px;
  color: var(--text-dim);
  margin: 0 0 8px 0;
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
.preset-dialog-header {
  flex-shrink: 0;
}
.preset-dialog-header .search-input {
  margin-top: 6px;
}
.preset-list {
  flex: 1;
  overflow-y: auto;
  margin: 12px 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.preset-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.preset-group-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 4px 2px;
  border-bottom: 1px solid var(--border);
}
.preset-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  background: var(--bg);
  transition: background 0.12s;
}
.preset-card:hover {
  background: var(--surface-hover);
}
.preset-card-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.preset-card-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
}
.preset-card-model {
  font-size: 12px;
  color: var(--text-dim);
}
.preset-card-url {
  font-size: 11px;
  color: var(--text-dim);
  opacity: 0.7;
}
.preset-card-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}
.header-actions {
  display: flex;
  gap: 6px;
}
</style>