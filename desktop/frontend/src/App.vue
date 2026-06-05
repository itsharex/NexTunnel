<template>
  <n-config-provider
    :theme="darkTheme"
    :theme-overrides="themeOverrides"
    :locale="naiveLocale"
    :date-locale="naiveDateLocale"
  >
    <n-message-provider>
      <div class="app-shell">
        <div
          class="network-background"
          aria-hidden="true"
        />

        <header class="titlebar">
          <div class="titlebar-drag-region">
            <div class="brand-lockup">
              <div class="brand-mark">
                NT
              </div>
              <div class="brand-copy">
                <strong>{{ t('app.productName') }}</strong>
                <span>{{ t('app.subtitle') }}</span>
              </div>
            </div>
          </div>

          <div class="titlebar-actions">
            <n-select
              v-model:value="currentLocale"
              class="language-select"
              size="small"
              :options="localeOptions"
              @update:value="handleLocaleChange"
            />
            <n-button
              quaternary
              circle
              size="small"
              :title="t('window.minimise')"
              @click="minimiseWindow"
            >
              <span class="window-icon">-</span>
            </n-button>
            <n-button
              quaternary
              circle
              size="small"
              :title="t('window.maximise')"
              @click="toggleMaximiseWindow"
            >
              <span class="window-icon">□</span>
            </n-button>
            <n-button
              quaternary
              circle
              size="small"
              class="close-button"
              :title="t('window.close')"
              @click="closeWindow"
            >
              <span class="window-icon">×</span>
            </n-button>
          </div>
        </header>

        <div class="workspace">
          <aside class="sidebar">
            <nav
              class="sidebar-nav"
              aria-label="NexTunnel client navigation"
            >
              <button
                v-for="item in navItems"
                :key="item.key"
                class="nav-button"
                :class="{ active: item.active, disabled: item.disabled }"
                type="button"
                :disabled="item.disabled"
              >
                <span class="nav-symbol">{{ item.symbol }}</span>
                <span>{{ item.label }}</span>
                <n-tag
                  v-if="item.disabled"
                  size="small"
                  round
                  type="info"
                  :bordered="false"
                >
                  {{ t('nav.planned') }}
                </n-tag>
              </button>
            </nav>

            <div class="sidebar-footer">
              <n-tag
                round
                type="success"
                :bordered="false"
              >
                {{ t('app.localAgent') }}
              </n-tag>
              <span>{{ t('app.version', { version }) }}</span>
            </div>
          </aside>

          <main class="content-shell">
            <section class="content-header">
              <div>
                <span class="section-kicker">{{ t('app.productName') }}</span>
                <h1>{{ t('app.title') }}</h1>
              </div>
              <n-space align="center">
                <n-tag
                  round
                  type="info"
                  :bordered="false"
                >
                  {{ t('app.preview') }}
                </n-tag>
              </n-space>
            </section>

            <StatusView />
          </main>
        </div>
      </div>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  darkTheme,
  dateZhCN,
  enUS,
  NButton,
  NConfigProvider,
  NMessageProvider,
  NSelect,
  NSpace,
  NTag,
  zhCN,
  type GlobalThemeOverrides,
} from 'naive-ui'
import StatusView from './views/StatusView.vue'
import { GetVersion } from './api/app'
import { closeWindow, minimiseWindow, toggleMaximiseWindow } from './api/window'
import { SUPPORTED_LOCALES, type SupportedLocale } from './i18n'

interface NavItem {
  key: string
  label: string
  symbol: string
  active: boolean
  disabled: boolean
}

const { t, locale } = useI18n()
const version = ref('0.0.0')
const currentLocale = ref<SupportedLocale>('zh-CN')

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#00ffff',
    primaryColorHover: '#4dffff',
    primaryColorPressed: '#00d5d5',
    primaryColorSuppl: '#8a2be2',
    borderRadius: '8px',
    bodyColor: '#0f172a',
    cardColor: 'rgba(30, 41, 59, 0.72)',
    modalColor: '#111c2f',
    popoverColor: '#111c2f',
    textColorBase: '#f8fafc',
    textColor1: '#f8fafc',
    textColor2: '#cbd5e1',
    textColor3: '#94a3b8',
  },
  Button: {
    borderRadiusMedium: '8px',
    borderRadiusSmall: '8px',
  },
  Card: {
    borderRadius: '12px',
  },
}

const localeOptions = computed(() => [
  {
    label: t('settings.simplifiedChinese'),
    value: 'zh-CN',
  },
  {
    label: t('settings.english'),
    value: 'en-US',
  },
])

const naiveLocale = computed(() => (currentLocale.value === 'zh-CN' ? zhCN : enUS))
const naiveDateLocale = computed(() => (currentLocale.value === 'zh-CN' ? dateZhCN : null))

const navItems = computed<NavItem[]>(() => [
  {
    key: 'overview',
    label: t('nav.overview'),
    symbol: '⌂',
    active: true,
    disabled: false,
  },
  {
    key: 'tunnels',
    label: t('nav.tunnels'),
    symbol: '⇄',
    active: false,
    disabled: false,
  },
  {
    key: 'network',
    label: t('nav.network'),
    symbol: '◎',
    active: false,
    disabled: true,
  },
  {
    key: 'settings',
    label: t('nav.settings'),
    symbol: '⚙',
    active: false,
    disabled: true,
  },
])

// handleLocaleChange 切换当前界面语言，默认语言为简体中文。
const handleLocaleChange = (value: SupportedLocale): void => {
  if (!SUPPORTED_LOCALES.includes(value)) return
  currentLocale.value = value
  locale.value = value
}

// loadVersion 兼容 Wails 注入失败场景，保证普通浏览器预览仍可打开。
const loadVersion = async (): Promise<void> => {
  try {
    version.value = await GetVersion()
  } catch {
    version.value = 'preview'
  }
}

onMounted(async () => {
  locale.value = currentLocale.value
  await loadVersion()
})
</script>

<style>
:root {
  --nex-cyan: #00ffff;
  --tunnel-violet: #8a2be2;
  --data-blue: #0000ff;
  --bg-dark: #0f172a;
  --sidebar-bg: #111c2f;
  --surface-bg: rgba(30, 41, 59, 0.72);
  --surface-strong: rgba(15, 23, 42, 0.9);
  --line-soft: rgba(148, 163, 184, 0.16);
  --line-cyan: rgba(0, 255, 255, 0.2);
  --text-main: #f8fafc;
  --text-dim: #94a3b8;
  --text-muted: #64748b;
  --success: #10b981;
  --warning: #f59e0b;
  --danger: #ef4444;
  --accent-gradient: linear-gradient(135deg, var(--nex-cyan), var(--tunnel-violet));
}

* {
  box-sizing: border-box;
}

html,
body,
#app {
  width: 100%;
  height: 100%;
  margin: 0;
  overflow: hidden;
}

body {
  background: var(--bg-dark);
  color: var(--text-main);
  font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
}

button,
input,
select {
  font: inherit;
}

.app-shell {
  position: relative;
  height: 100dvh;
  min-width: 1080px;
  background:
    radial-gradient(circle at 24% 14%, rgba(0, 255, 255, 0.14), transparent 28%),
    radial-gradient(circle at 86% 22%, rgba(138, 43, 226, 0.18), transparent 30%),
    var(--bg-dark);
  overflow: hidden;
}

.network-background {
  position: fixed;
  inset: 42px 0 0 80px;
  pointer-events: none;
  opacity: 0.28;
  background-image:
    linear-gradient(90deg, rgba(148, 163, 184, 0.12) 1px, transparent 1px),
    linear-gradient(0deg, rgba(148, 163, 184, 0.1) 1px, transparent 1px);
  background-size: 54px 54px;
}

.titlebar {
  position: relative;
  z-index: 3;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--line-soft);
  background: rgba(15, 23, 42, 0.92);
  backdrop-filter: blur(18px);
}

.titlebar-drag-region {
  height: 100%;
  flex: 1;
  display: flex;
  align-items: center;
  padding-left: 16px;
  --wails-draggable: drag;
}

.brand-lockup {
  display: flex;
  align-items: center;
  gap: 10px;
}

.brand-mark {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  background: var(--accent-gradient);
  color: #06111f;
  font-size: 12px;
  font-weight: 900;
  box-shadow: 0 0 18px rgba(0, 255, 255, 0.25);
}

.brand-copy {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.brand-copy strong {
  color: var(--text-main);
  font-size: 13px;
  letter-spacing: 0;
}

.brand-copy span {
  color: var(--text-dim);
  font-size: 12px;
}

.titlebar-actions {
  height: 100%;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
}

.language-select {
  width: 116px;
}

.window-icon {
  color: var(--text-main);
  font-size: 14px;
  line-height: 1;
}

.close-button:hover {
  background: rgba(239, 68, 68, 0.18);
}

.workspace {
  height: calc(100dvh - 42px);
  display: grid;
  grid-template-columns: 80px minmax(0, 1fr);
}

.sidebar {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  padding: 18px 10px;
  border-right: 1px solid var(--line-soft);
  background: linear-gradient(180deg, rgba(17, 28, 47, 0.98), rgba(9, 15, 28, 0.98));
}

.sidebar-nav {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.nav-button {
  width: 100%;
  min-height: 58px;
  display: grid;
  place-items: center;
  gap: 4px;
  border: 0;
  border-left: 3px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--text-dim);
  cursor: pointer;
  font-size: 11px;
  transition:
    background 160ms ease,
    color 160ms ease,
    border-color 160ms ease;
}

.nav-button.active,
.nav-button:hover:not(:disabled) {
  border-left-color: var(--nex-cyan);
  background: rgba(0, 255, 255, 0.1);
  color: var(--nex-cyan);
}

.nav-button.disabled {
  cursor: not-allowed;
  opacity: 0.52;
}

.nav-symbol {
  font-size: 18px;
  line-height: 1;
}

.sidebar-footer {
  width: 100%;
  margin-top: auto;
  display: grid;
  place-items: center;
  gap: 8px;
  color: var(--text-muted);
  font-size: 11px;
  text-align: center;
}

.content-shell {
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 22px 26px 26px;
  overflow: auto;
}

.content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.section-kicker {
  color: var(--nex-cyan);
  font-size: 12px;
  font-weight: 700;
}

.content-header h1 {
  margin: 4px 0 0;
  color: var(--text-main);
  font-size: 28px;
  line-height: 1.16;
}

.n-card {
  backdrop-filter: blur(12px);
}
</style>
