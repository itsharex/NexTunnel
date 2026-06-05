import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    rollupOptions: {
      output: {
        // manualChunks 将 Naive UI 与 Vue 生态拆分，降低主入口 chunk 体积。
        manualChunks: {
          vue: ['vue', 'pinia', 'vue-i18n'],
          naive: ['naive-ui'],
        },
      },
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
})
