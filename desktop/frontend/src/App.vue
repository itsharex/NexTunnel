<template>
  <div
    id="app"
    class="app-shell"
  >
    <div
      class="network-background"
      aria-hidden="true"
    />

    <aside class="app-sidebar">
      <div class="brand-block">
        <div class="brand-mark">
          NT
        </div>
        <div>
          <p class="brand-name">
            NexTunnel
          </p>
          <p class="brand-caption">
            NAT traversal workspace
          </p>
        </div>
      </div>

      <nav
        class="sidebar-nav"
        aria-label="Desktop navigation"
      >
        <a
          class="nav-item active"
          href="#client-console"
        >
          <span class="nav-icon">01</span>
          <span>Client Console</span>
        </a>
        <span class="nav-item disabled">
          <span class="nav-icon">02</span>
          <span>Policy Center</span>
          <span class="nav-badge">Planned</span>
        </span>
        <span class="nav-item disabled">
          <span class="nav-icon">03</span>
          <span>Diagnostics</span>
          <span class="nav-badge">Planned</span>
        </span>
      </nav>

      <div class="sidebar-footer">
        <span class="version">v{{ version }}</span>
        <span class="runtime-pill">Local agent</span>
      </div>
    </aside>

    <main class="app-main">
      <header class="app-header">
        <div>
          <p class="header-kicker">
            Desktop Application
          </p>
          <h1>NexTunnel Client Console</h1>
        </div>
        <div class="header-actions">
          <span class="status-chip">Design preview</span>
          <span class="status-chip muted">Feature-ready shell</span>
        </div>
      </header>

      <StatusView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import StatusView from './views/StatusView.vue'
import { GetVersion } from './api/app'

const version = ref('0.0.0')

// loadVersion 兼容 Wails 注入失败场景，保证视觉预览仍可打开。
const loadVersion = async (): Promise<void> => {
  try {
    version.value = await GetVersion()
  } catch {
    version.value = 'unknown'
  }
}

onMounted(async () => {
  await loadVersion()
})
</script>

<style>
:root {
  --nex-cyan: #00ffff;
  --tunnel-violet: #8a2be2;
  --data-blue: #0000ff;
  --neutral-grey: #a8a9a9;
  --future-white: #feffff;
  --shell-bg: #eef4fb;
  --console-bg: #07111f;
  --console-panel: #0c1b2d;
  --console-panel-strong: #10243a;
  --console-border: rgba(0, 255, 255, 0.16);
  --console-border-muted: rgba(168, 169, 169, 0.18);
  --text-primary: #f7fbff;
  --text-secondary: #9fb2c7;
  --text-muted: #6f8298;
  --status-success: #24e6a1;
  --status-warning: #ffc857;
  --status-danger: #ff5c7a;
  --shadow-shell: 0 24px 80px rgba(7, 17, 31, 0.18);
  --shadow-panel: 0 16px 42px rgba(0, 0, 0, 0.22);
  --radius-panel: 8px;
  --radius-shell: 18px;
  --sidebar-width: 248px;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  min-width: 320px;
  min-height: 100dvh;
  background: var(--shell-bg);
  color: var(--text-primary);
}

#app {
  min-height: 100dvh;
}

button,
input,
select {
  font: inherit;
}

button {
  transition:
    transform 140ms ease,
    box-shadow 140ms ease,
    border-color 140ms ease,
    opacity 140ms ease;
}

button:active:not(:disabled) {
  transform: translateY(1px);
}

.app-shell {
  position: relative;
  display: grid;
  grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
  gap: 18px;
  min-height: 100dvh;
  padding: 18px;
  overflow: hidden;
}

.network-background {
  position: fixed;
  inset: 0;
  z-index: -1;
  background:
    radial-gradient(circle at 11% 22%, rgba(0, 255, 255, 0.16), transparent 28%),
    radial-gradient(circle at 82% 12%, rgba(138, 43, 226, 0.16), transparent 30%),
    linear-gradient(120deg, rgba(255, 255, 255, 0.92), rgba(236, 244, 253, 0.82)),
    repeating-linear-gradient(35deg, rgba(7, 17, 31, 0.07) 0 1px, transparent 1px 68px);
}

.network-background::before {
  content: '';
  position: absolute;
  inset: 0;
  opacity: 0.42;
  background-image:
    linear-gradient(90deg, rgba(9, 39, 70, 0.08) 1px, transparent 1px),
    linear-gradient(0deg, rgba(9, 39, 70, 0.08) 1px, transparent 1px);
  background-size: 88px 88px;
}

.app-sidebar,
.app-main {
  position: relative;
  border: 1px solid rgba(255, 255, 255, 0.68);
  background:
    linear-gradient(180deg, rgba(10, 24, 42, 0.98), rgba(5, 12, 24, 0.98)),
    var(--console-bg);
  box-shadow: var(--shadow-shell);
}

.app-sidebar {
  display: flex;
  flex-direction: column;
  gap: 28px;
  min-width: 0;
  min-height: calc(100dvh - 36px);
  padding: 18px;
  border-radius: var(--radius-shell);
}

.brand-block {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-mark {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  background:
    linear-gradient(135deg, rgba(0, 255, 255, 0.92), rgba(138, 43, 226, 0.9)),
    #0c1b2d;
  color: #05101f;
  font-size: 18px;
  font-weight: 900;
  box-shadow: 0 12px 28px rgba(0, 255, 255, 0.18);
}

.brand-name {
  font-size: 20px;
  font-weight: 800;
  background: linear-gradient(90deg, var(--nex-cyan), var(--tunnel-violet));
  background-clip: text;
  color: transparent;
  overflow-wrap: anywhere;
}

.brand-caption {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 12px;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.nav-item {
  min-height: 42px;
  min-width: 0;
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) minmax(0, auto);
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid transparent;
  border-radius: var(--radius-panel);
  color: var(--text-secondary);
  font-size: 13px;
  text-decoration: none;
}

.nav-item.active {
  border-color: var(--console-border);
  background: linear-gradient(90deg, rgba(0, 255, 255, 0.14), rgba(138, 43, 226, 0.06));
  color: var(--text-primary);
}

.nav-item.disabled {
  opacity: 0.58;
}

.nav-icon {
  color: var(--nex-cyan);
  font-family: ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace;
  font-size: 11px;
}

.nav-badge,
.runtime-pill,
.status-chip {
  min-width: 0;
  max-width: 100%;
  border: 1px solid rgba(0, 255, 255, 0.2);
  border-radius: 999px;
  background: rgba(0, 255, 255, 0.08);
  color: var(--nex-cyan);
  font-size: 11px;
  line-height: 1;
  overflow: hidden;
  padding: 6px 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-footer {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
  color: var(--text-muted);
}

.version {
  font-size: 12px;
}

.runtime-pill {
  width: fit-content;
}

.app-main {
  min-width: 0;
  min-height: calc(100dvh - 36px);
  padding: 22px;
  border-radius: var(--radius-shell);
  overflow: auto;
}

.app-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 18px;
}

.header-kicker {
  margin-bottom: 5px;
  color: var(--nex-cyan);
  font-size: 12px;
  font-weight: 600;
}

.app-header h1 {
  color: var(--text-primary);
  font-size: 28px;
  font-weight: 780;
  line-height: 1.16;
}

.header-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.status-chip.muted {
  border-color: rgba(168, 169, 169, 0.2);
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-secondary);
}

@media (max-width: 980px) {
  .app-shell {
    grid-template-columns: 1fr;
  }

  .app-sidebar {
    min-height: auto;
  }

  .sidebar-nav {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 700px) {
  .app-shell {
    padding: 10px;
  }

  .app-main,
  .app-sidebar {
    min-height: auto;
    border-radius: 12px;
  }

  .app-main {
    padding: 14px;
  }

  .app-header {
    flex-direction: column;
  }

  .app-header h1 {
    font-size: 23px;
  }

  .sidebar-nav {
    grid-template-columns: 1fr;
  }
}
</style>
