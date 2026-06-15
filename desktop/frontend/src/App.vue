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
              <div class="brand-copy">
                <strong class="brand-title"><span>Nex</span>Tunnel</strong>
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
              <span class="window-icon minimise-icon" />
            </n-button>
            <n-button
              quaternary
              circle
              size="small"
              :title="t('window.maximise')"
              @click="toggleMaximiseWindow"
            >
              <span class="window-icon maximise-icon" />
            </n-button>
            <n-button
              quaternary
              circle
              size="small"
              class="close-button"
              :title="t('window.close')"
              @click="closeWindow"
            >
              <span class="window-icon close-icon" />
            </n-button>
          </div>
        </header>

        <div class="workspace">
          <aside class="sidebar">
            <div class="sidebar-logo">
              <img
                :src="sidebarLogoImage"
                alt="NexTunnel"
              >
            </div>

            <nav
              class="sidebar-nav"
              aria-label="NexTunnel client navigation"
            >
              <span
                class="nav-active-indicator"
                :style="{ transform: `translateY(${activeNavIndex * NAV_ITEM_STEP}px)` }"
                aria-hidden="true"
              />
              <button
                v-for="item in navItems"
                :key="item.key"
                class="nav-button"
                :class="{ active: item.active, disabled: item.disabled }"
                type="button"
                :disabled="item.disabled"
                @click="activeView = item.key"
              >
                <n-icon
                  class="nav-icon"
                  :component="item.icon"
                  :size="22"
                />
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
                <h1>{{ activeTitle }}</h1>
              </div>
              <n-space align="center">
                <n-tag
                  round
                  :type="connectionTagType"
                  :bordered="false"
                >
                  {{ connectionLabel }}
                </n-tag>
              </n-space>
            </section>

            <Transition
              name="view-switch"
              mode="out-in"
            >
              <StatusView
                v-if="activeView === 'overview' || activeView === 'tunnels'"
                :key="activeView"
                :view-mode="activeView === 'tunnels' ? 'tunnels' : 'overview'"
              />
              <NetworkView
                v-else-if="activeView === 'network'"
                key="network"
              />
              <LogsView
                v-else-if="activeView === 'logs'"
                key="logs"
              />
              <SettingsView
                v-else
                key="settings"
              />
            </Transition>
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
  NIcon,
  NMessageProvider,
  NSelect,
  NSpace,
  NTag,
  zhCN,
  type GlobalThemeOverrides,
} from 'naive-ui'
import { LayoutDashboard, Network, Route, ScrollText, Settings, type LucideIcon } from 'lucide-vue-next'
import StatusView from './views/StatusView.vue'
import NetworkView from './views/NetworkView.vue'
import LogsView from './views/LogsView.vue'
import SettingsView from './views/SettingsView.vue'
import { GetVersion } from './api/app'
import { closeWindow, minimiseWindow, toggleMaximiseWindow } from './api/window'
import { SUPPORTED_LOCALES, type SupportedLocale } from './i18n'
import { useTunnelStore } from './stores/tunnel'
import sidebarLogoImage from './assets/logo.png'

interface NavItem {
  key: AppView
  label: string
  icon: LucideIcon
  active: boolean
  disabled: boolean
}

type AppView = 'overview' | 'tunnels' | 'network' | 'logs' | 'settings'

const { t, locale } = useI18n()
const store = useTunnelStore()
const version = ref('0.0.0')
const currentLocale = ref<SupportedLocale>('zh-CN')
const activeView = ref<AppView>('overview')
const NAV_ITEM_STEP = 70

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
    borderRadius: '8px',
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
    icon: LayoutDashboard,
    active: activeView.value === 'overview',
    disabled: false,
  },
  {
    key: 'tunnels',
    label: t('nav.tunnels'),
    icon: Route,
    active: activeView.value === 'tunnels',
    disabled: false,
  },
  {
    key: 'network',
    label: t('nav.network'),
    icon: Network,
    active: activeView.value === 'network',
    disabled: false,
  },
  {
    key: 'logs',
    label: t('nav.logs'),
    icon: ScrollText,
    active: activeView.value === 'logs',
    disabled: false,
  },
  {
    key: 'settings',
    label: t('nav.settings'),
    icon: Settings,
    active: activeView.value === 'settings',
    disabled: false,
  },
])

const activeNavIndex = computed(() => Math.max(0, navItems.value.findIndex((item) => item.key === activeView.value)))

const activeTitle = computed(() => {
  const match = navItems.value.find((item) => item.key === activeView.value)
  return match?.label ?? t('app.title')
})

const connectionLabel = computed(() => {
  const key = `status.${store.connectionStatus || 'disconnected'}`
  const translated = t(key)
  return translated === key ? store.connectionStatus : translated
})

const connectionTagType = computed(() => {
  if (store.connectionStatus === 'connected') return 'success'
  if (store.connectionStatus === 'reconnecting') return 'warning'
  return 'error'
})

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
  await store.loadServerSettings()
  await store.refreshRuntimeStatus()
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
  --ease-standard: cubic-bezier(0.4, 0, 0.2, 1);
  --ease-decelerate: cubic-bezier(0, 0, 0.2, 1);
  --duration-micro: 120ms;
  --duration-small: 180ms;
  --duration-medium: 260ms;
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
    linear-gradient(135deg, rgba(0, 255, 255, 0.06), transparent 32%),
    linear-gradient(225deg, rgba(138, 43, 226, 0.08), transparent 36%),
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
  min-width: 0;
}

.brand-copy {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-title {
  color: var(--tunnel-violet);
  font-size: 22px;
  font-weight: 900;
  letter-spacing: 0;
  line-height: 1;
  text-shadow: 0 8px 24px rgba(138, 43, 226, 0.28);
}

.brand-title span {
  color: #1da1f2;
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
  position: relative;
  width: 14px;
  height: 14px;
  display: inline-block;
  color: var(--text-main);
}

.minimise-icon::before,
.maximise-icon::before,
.close-icon::before,
.close-icon::after {
  content: '';
  position: absolute;
  background: currentColor;
}

.minimise-icon::before {
  left: 2px;
  right: 2px;
  top: 7px;
  height: 1.5px;
}

.maximise-icon::before {
  inset: 2px;
  border: 1.5px solid currentColor;
  background: transparent;
}

.close-icon::before,
.close-icon::after {
  left: 2px;
  right: 2px;
  top: 6px;
  height: 1.5px;
}

.close-icon::before {
  transform: rotate(45deg);
}

.close-icon::after {
  transform: rotate(-45deg);
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
  width: 72px;
  height: 72px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(0, 255, 255, 0.18);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
  box-shadow: 0 12px 30px rgba(0, 255, 255, 0.1);
}

.sidebar-logo img {
  width: 68px;
  height: 68px;
  border-radius: 8px;
  object-fit: cover;
}

.sidebar-nav {
  position: relative;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.nav-active-indicator {
  position: absolute;
  left: -10px;
  top: 0;
  width: 4px;
  height: 60px;
  border-radius: 999px;
  background: var(--accent-gradient);
  box-shadow: 0 0 18px rgba(0, 255, 255, 0.36);
  pointer-events: none;
  transition: transform var(--duration-medium) var(--ease-standard);
}

.nav-button {
  width: 100%;
  min-height: 60px;
  display: grid;
  place-items: center;
  gap: 4px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text-dim);
  cursor: pointer;
  font-size: 11px;
  transition: transform var(--duration-small) var(--ease-standard);
}

.nav-button.active,
.nav-button:hover:not(:disabled) {
  transform: translateY(-1px);
  background: linear-gradient(90deg, rgba(0, 255, 255, 0.12), rgba(138, 43, 226, 0.08));
  color: var(--nex-cyan);
}

.nav-button.disabled {
  cursor: not-allowed;
  opacity: 0.52;
}

.nav-icon {
  color: currentColor;
  transition: transform var(--duration-small) var(--ease-standard), opacity var(--duration-small) var(--ease-standard);
}

.nav-button:hover:not(:disabled) .nav-icon,
.nav-button.active .nav-icon {
  transform: translateY(-1px) scale(1.06);
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

.view-switch-enter-active,
.view-switch-leave-active {
  transition: opacity var(--duration-medium) var(--ease-standard), transform var(--duration-medium) var(--ease-standard);
}

.view-switch-enter-from,
.view-switch-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }

  .view-switch-enter-from,
  .view-switch-leave-to {
    transform: none;
  }
}
</style>
