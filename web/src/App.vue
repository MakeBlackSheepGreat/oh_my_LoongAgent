<script setup lang="ts">
// 根组件：根据路由切换登录/工作台视图。
import { useRouter } from './router'
import LoginView from './views/LoginView.vue'
import WorkbenchView from './views/WorkbenchView.vue'

const router = useRouter()
</script>

<template>
  <LoginView v-if="router.current.value === 'login'" />
  <WorkbenchView v-else />
</template>