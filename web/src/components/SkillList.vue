<script setup lang="ts">
// 技能列表面板。
import { onMounted } from 'vue'
import { useWorkbench } from '../stores/workbench'
import { t } from '../i18n'

const wb = useWorkbench()

onMounted(() => void wb.loadSkills())
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <span class="panel-title">{{ t('skills.title') }}</span>
    </div>

    <div class="panel-body">
      <div v-if="wb.state.loading.skills" class="empty">
        <span class="spinner"></span>
      </div>
      <div v-else-if="wb.state.skills.length === 0" class="empty">
        <div class="empty-mark">🧩</div>
        <span>{{ t('skills.empty') }}</span>
      </div>
      <div v-else class="list">
        <div
          v-for="s in wb.state.skills"
          :key="s.skill_id"
          class="list-item"
        >
          <div class="list-item-main">
            <span class="list-item-title">{{ s.title }}</span>
            <span class="list-item-sub">{{ s.description }}</span>
          </div>
          <span class="meta mono">{{ t('skills.version', { version: s.version }) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>