import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 代理 /api 到 Go 后端；开发期前端 48623、后端 8000。
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 48623,
    proxy: {
      '/api': 'http://127.0.0.1:8000',
      '/health': 'http://127.0.0.1:8000',
    },
  },
})
