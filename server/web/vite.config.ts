import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
  },
  server: {
    fs: {
      // 允许服务端 Web 直接复用桌面端标准 logo 资源，避免维护两份品牌素材。
      allow: ['..', '../../desktop/frontend/src/assets/logo'],
    },
  },
  resolve: {
    alias: {
      '@': '/src',
      '@shared-logo': '../../desktop/frontend/src/assets/logo',
    },
  },
})
