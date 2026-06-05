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
              <img
                class="brand-icon"
                :src="logoImage"
                alt="NexTunnel"
              >
              <div class="brand-copy">
                <img
                  class="brand-wordmark"
                  :src="textLogoImage"
                  alt="NexTunnel"
                >
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
            <div class="sidebar-logo">
              <img
                :src="logoImage"
                alt="NexTunnel"
              >
            </div>

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
import logoImage from './assets/logo.png'
import textLogoImage from './assets/text-logo.png'

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
    primaryColorHover: '#33f6f6',
    primaryColorPressed: '#00d5d5',
    primaryColorSuppl: '#8a2be2',
    borderRadius: '8px',
    bodyColor: '#091120',
    cardColor: 'rgba(18, 31, 52, 0.82)',
    modalColor: '#111c2f',
    popoverColor: '#111c2f',
    textColorBase: '#feffff',
    textColor1: '#feffff',
    textColor2: '#d2e0ec',
    textColor3: '#a8a9a9',
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
  /* 品牌色来自 assets/palette.png，统一桌面端和后续服务端管理台的视觉基线。 */
  --nex-cyan: #00ffff;
  --tunnel-violet: #8a2be2;
  --data-blue: #0000ff;
  --neutral-grey: #a8a9a9;
  --future-white: #feffff;
  --bg-dark: #091120;
  --sidebar-bg: #0c1628;
  --surface-bg: rgba(18, 31, 52, 0.82);
  --surface-strong: rgba(9, 17, 32, 0.94);
  --line-soft: rgba(168, 169, 169, 0.16);
  --line-cyan: rgba(0, 255, 255, 0.18);
  --text-main: var(--future-white);
  --text-dim: #b8c5d3;
  --text-muted: var(--neutral-grey);
  --success: #10b981;
  --warning: #f59e0b;
  --danger: #ef4444;
  --accent-gradient: linear-gradient(135deg, var(--nex-cyan), var(--tunnel-violet));
  /* 标题栏高度按商业桌面软件比例设置，并同步参与主布局高度计算。 */
  --titlebar-height: 56px;
  --sidebar-width: 88px;
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
    radial-gradient(circle at 22% 12%, rgba(0, 255, 255, 0.11), transparent 28%),
    radial-gradient(circle at 86% 20%, rgba(138, 43, 226, 0.15), transparent 30%),
    linear-gradient(145deg, rgba(0, 0, 255, 0.08), transparent 38%),
    var(--bg-dark);
  overflow: hidden;
}

.network-background {
  position: fixed;
  inset: var(--titlebar-height) 0 0 var(--sidebar-width);
  pointer-events: none;
  opacity: 0.22;
  background-image:
    linear-gradient(90deg, rgba(148, 163, 184, 0.12) 1px, transparent 1px),
    linear-gradient(0deg, rgba(148, 163, 184, 0.1) 1px, transparent 1px);
  background-size: 54px 54px;
}

.titlebar {
  position: relative;
  z-index: 3;
  height: var(--titlebar-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--line-soft);
  background:
    linear-gradient(90deg, rgba(11, 23, 42, 0.98), rgba(12, 19, 35, 0.92)),
    rgba(9, 17, 32, 0.96);
  backdrop-filter: blur(20px);
}

.titlebar-drag-region {
  height: 100%;
  flex: 1;
  display: flex;
  align-items: center;
  padding-left: 18px;
  --wails-draggable: drag;
}

.brand-lockup {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-icon {
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  border-radius: 11px;
  object-fit: cover;
  box-shadow: 0 10px 26px rgba(0, 255, 255, 0.16);
}

.brand-copy {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-wordmark {
  width: 164px;
  max-height: 42px;
  object-fit: contain;
}

.brand-copy span {
  color: var(--text-dim);
  font-size: 13px;
  white-space: nowrap;
}

.titlebar-actions {
  height: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  --wails-draggable: no-drag;
}

.language-select {
  width: 128px;
}

.window-icon {
  color: var(--text-main);
  font-size: 15px;
  line-height: 1;
}

.titlebar-actions .n-button {
  width: 34px;
  height: 34px;
}

.close-button:hover {
  background: rgba(239, 68, 68, 0.18);
}

.workspace {
  height: calc(100dvh - var(--titlebar-height));
  display: grid;
  grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
}

.sidebar {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
  padding: 18px 10px;
  border-right: 1px solid var(--line-soft);
  background:
    linear-gradient(180deg, rgba(13, 27, 47, 0.98), rgba(6, 12, 25, 0.98)),
    var(--sidebar-bg);
}

.sidebar-logo {
  width: 52px;
  height: 52px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(0, 255, 255, 0.18);
  border-radius: 15px;
  background: rgba(255, 255, 255, 0.04);
  box-shadow: 0 12px 30px rgba(0, 255, 255, 0.1);
}

.sidebar-logo img {
  width: 42px;
  height: 42px;
  border-radius: 11px;
  object-fit: cover;
}

.sidebar-nav {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.nav-button {
  width: 100%;
  min-height: 60px;
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
  background: linear-gradient(90deg, rgba(0, 255, 255, 0.12), rgba(138, 43, 226, 0.08));
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
  padding: 24px 28px 28px;
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
  font-size: 13px;
  font-weight: 700;
}

.content-header h1 {
  margin: 4px 0 0;
  color: var(--text-main);
  font-size: 30px;
  line-height: 1.16;
}

.n-card {
  backdrop-filter: blur(12px);
}
</style>
