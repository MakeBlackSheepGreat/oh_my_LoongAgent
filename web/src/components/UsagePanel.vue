<script setup lang="ts">
// 用量面板：今日/本周/本月/累计切换 + 公共模型池。
import { onMounted } from 'vue'
import { useWorkbench } from '../stores/workbench'
import { t } from '../i18n'
import { useSession } from '../stores/session'

const wb = useWorkbench()
const session = useSession()

const windows = ['today', 'week', 'month', 'all'] as const

onMounted(() => void wb.loadUsage())

function formatTokens(n: number): string {
  return n.toLocaleString(session.state.me?.locale === 'en' ? 'en-US' : 'zh-CN')
}

function formatCurrency(n: number): string {
  return '$' + n.toFixed(6)
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <span class="panel-title">{{ t('usage.title') }}</span>
    </div>

    <div class="panel-body">
      <!-- 窗口切换 -->
      <div class="window-tabs">
        <button
          v-for="w in windows"
          :key="w"
          class="window-tab"
          :class="{ active: wb.state.window === w }"
          @click="void wb.loadUsage(w)"
        >
          {{ t('usage.window.' + w) }}
        </button>
      </div>

      <div v-if="wb.state.loading.usage" class="empty">
        <span class="spinner"></span>
      </div>

      <template v-else>
        <!-- 我的用量 -->
        <div class="usage-section">
          <h4 class="usage-section-title">{{ t('usage.my') }}</h4>
          <div v-if="!wb.state.usage" class="empty">
            <span>{{ t('usage.empty') }}</span>
          </div>
          <div v-else class="usage-grid">
            <div class="usage-cell">
              <span class="usage-value">{{ formatTokens(wb.state.usage.total_tokens) }}</span>
              <span class="usage-label">{{ t('usage.tokens') }}</span>
            </div>
            <div class="usage-cell">
              <span class="usage-value">{{ formatCurrency(wb.state.usage.total_cost_usd) }}</span>
              <span class="usage-label">{{ t('usage.cost') }}</span>
            </div>
            <div class="usage-cell">
              <span class="usage-value">{{ formatTokens(wb.state.usage.call_count) }}</span>
              <span class="usage-label">{{ t('usage.calls') }}</span>
            </div>
          </div>
        </div>

        <!-- 公共模型池 -->
        <div class="usage-section">
          <h4 class="usage-section-title">{{ t('usage.pool') }}</h4>
          <div v-if="!wb.state.pool || wb.state.pool.providers.length === 0" class="empty">
            <span>{{ t('usage.empty') }}</span>
          </div>
          <div v-else class="pool-list">
            <div
              v-for="pv in wb.state.pool.providers"
              :key="pv.provider_id"
              class="pool-item"
            >
              <div class="pool-item-main">
                <span class="mono" style="font-weight:500">{{ pv.model_id }}</span>
                <span class="meta">{{ pv.provider_id }}</span>
              </div>
              <div class="pool-stats">
                <span class="pool-stat">
                  <span class="pool-stat-val">{{ formatTokens(pv.total_tokens) }}</span>
                  <span class="pool-stat-label">T</span>
                </span>
                <span class="pool-stat">
                  <span class="pool-stat-val">{{ formatCurrency(pv.estimated_cost_usd) }}</span>
                  <span class="pool-stat-label">$</span>
                </span>
                <span class="pool-stat">
                  <span class="pool-stat-val">{{ formatTokens(pv.call_count) }}</span>
                  <span class="pool-stat-label">C</span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.window-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 14px;
  background: var(--surface-3);
  border-radius: var(--radius-sm);
  padding: 3px;
}
.window-tab {
  flex: 1;
  padding: 5px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-2);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.14s ease;
}
.window-tab.active {
  background: var(--surface);
  color: var(--text);
  box-shadow: var(--shadow-sm);
}
.window-tab:hover:not(.active) {
  color: var(--text);
}

.usage-section {
  margin-bottom: 16px;
}
.usage-section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 8px;
}
.usage-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}
.usage-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 10px 6px;
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  border: 1px solid var(--border);
}
.usage-value {
  font-size: 16px;
  font-weight: 700;
  color: var(--text);
  font-family: var(--mono);
}
.usage-label {
  font-size: 11px;
  color: var(--text-3);
}

.pool-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.pool-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  background: var(--surface-2);
}
.pool-item-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.pool-stats {
  display: flex;
  gap: 8px;
  flex: none;
}
.pool-stat {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0;
}
.pool-stat-val {
  font-family: var(--mono);
  font-size: 11px;
  font-weight: 600;
  color: var(--text);
}
.pool-stat-label {
  font-size: 9px;
  color: var(--text-3);
}
</style>