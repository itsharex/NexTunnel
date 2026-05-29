<template>
  <div id="app">
    <header class="app-header">
      <h1>NexTunnel</h1>
      <span class="version">v{{ version }}</span>
    </header>
    <main class="app-main">
      <StatusView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import StatusView from './views/StatusView.vue'
import { GetVersion } from './api/app'

const version = ref('0.0.0')

onMounted(async () => {
  try {
    version.value = await GetVersion()
  } catch {
    version.value = 'unknown'
  }
})
</script>

<style>
:root {
  --color-bg: #1b2636;
  --color-surface: #243447;
  --color-primary: #4fc3f7;
  --color-text: #e0e0e0;
  --color-text-secondary: #90a4ae;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background-color: var(--color-bg);
  color: var(--color-text);
}

.app-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 24px;
  background-color: var(--color-surface);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.app-header h1 {
  font-size: 20px;
  font-weight: 600;
  color: var(--color-primary);
}

.version {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.app-main {
  padding: 24px;
}
</style>
