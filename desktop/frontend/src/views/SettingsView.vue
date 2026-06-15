<template>
  <section class="settings-view">
    <aside class="settings-sidebar">
      <button
        v-for="section in sections"
        :key="section.key"
        class="settings-nav-item"
        :class="{ active: activeSection === section.key }"
        type="button"
        @click="activeSection = section.key"
      >
        <span>{{ section.symbol }}</span>
        <strong>{{ section.label }}</strong>
        <small>{{ section.description }}</small>
      </button>
    </aside>

    <div class="settings-content">
      <n-card
        v-if="activeSection === 'connection'"
        class="settings-panel"
        :bordered="false"
      >
        <template #header>
          <div class="panel-title">
            <div>
              <strong>{{ t('settings.connectionTitle') }}</strong>
              <span>{{ t('settings.connectionSubtitle') }}</span>
            </div>
            <n-button
              type="primary"
              :loading="isSaving"
              @click="handleSave"
            >
              {{ t('settings.save') }}
            </n-button>
          </div>
        </template>

        <n-form
          label-placement="top"
          :show-feedback="false"
        >
          <n-grid
            :cols="2"
            :x-gap="14"
            :y-gap="14"
            responsive="screen"
          >
            <n-form-item-gi :label="t('connection.relayAddress')">
              <n-input v-model:value="form.relay_addr" />
            </n-form-item-gi>
            <n-form-item-gi :label="t('connection.relayToken')">
              <n-input
                v-model:value="form.relay_token"
                type="password"
                show-password-on="click"
              />
            </n-form-item-gi>
            <n-form-item-gi :label="t('settings.controlPlaneURL')">
              <n-input
                v-model:value="form.control_plane_url"
                placeholder="https://control.example.com:9090"
              />
            </n-form-item-gi>
            <n-form-item-gi :label="t('settings.controlPlaneToken')">
              <n-input
                v-model:value="form.control_plane_token"
                type="password"
                show-password-on="click"
              />
            </n-form-item-gi>
          </n-grid>
        </n-form>
      </n-card>

      <n-card
        v-else-if="activeSection === 'network'"
        class="settings-panel"
        :bordered="false"
      >
        <template #header>
          <div class="panel-title">
            <div>
              <strong>{{ t('settings.networkTitle') }}</strong>
              <span>{{ t('settings.networkSubtitle') }}</span>
            </div>
            <n-button
              type="primary"
              :loading="isSaving"
              @click="handleSave"
            >
              {{ t('settings.save') }}
            </n-button>
          </div>
        </template>

        <n-form
          label-placement="top"
          :show-feedback="false"
        >
          <n-grid
            :cols="2"
            :x-gap="14"
            :y-gap="14"
            responsive="screen"
          >
            <n-form-item-gi :label="t('settings.stunServer')">
              <n-input v-model:value="form.stun_server" />
            </n-form-item-gi>
            <n-form-item-gi :label="t('settings.stunAltServer')">
              <n-input v-model:value="form.stun_alt_server" />
            </n-form-item-gi>
          </n-grid>
        </n-form>

        <div class="diagnostic-grid">
          <div
            v-for="item in networkFacts"
            :key="item.label"
            class="setting-fact"
          >
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </n-card>

      <n-card
        v-else-if="activeSection === 'ports'"
        class="settings-panel"
        :bordered="false"
      >
        <LocalPortManager @use-port="handleUsePortFromSettings" />
      </n-card>

      <n-card
        v-else-if="activeSection === 'security'"
        class="settings-panel"
        :bordered="false"
      >
        <template #header>
          <div class="panel-title compact">
            <div>
              <strong>{{ t('settings.securityTitle') }}</strong>
              <span>{{ t('settings.securitySubtitle') }}</span>
            </div>
          </div>
        </template>

        <div class="setting-list">
          <div
            v-for="item in securityItems"
            :key="item.label"
            class="setting-row"
          >
            <div>
              <strong>{{ item.label }}</strong>
              <span>{{ item.description }}</span>
            </div>
            <n-tag
              round
              :type="item.ok ? 'success' : 'warning'"
              :bordered="false"
            >
              {{ item.state }}
            </n-tag>
          </div>
        </div>
      </n-card>

      <n-card
        v-else-if="activeSection === 'appearance'"
        class="settings-panel"
        :bordered="false"
      >
        <template #header>
          <div class="panel-title compact">
            <div>
              <strong>{{ t('settings.appearanceTitle') }}</strong>
              <span>{{ t('settings.appearanceSubtitle') }}</span>
            </div>
          </div>
        </template>

        <div class="appearance-grid">
          <div
            v-for="item in appearanceItems"
            :key="item.label"
            class="appearance-item"
          >
            <span :style="{ background: item.color }" />
            <div>
              <strong>{{ item.label }}</strong>
              <small>{{ item.value }}</small>
            </div>
          </div>
        </div>
      </n-card>

      <n-card
        v-else
        class="settings-panel"
        :bordered="false"
      >
        <template #header>
          <div class="panel-title compact">
            <div>
              <strong>{{ t('settings.aboutTitle') }}</strong>
              <span>{{ t('settings.aboutSubtitle') }}</span>
            </div>
          </div>
        </template>

        <div class="about-panel">
          <img
            src="../assets/logo.png"
            alt="NexTunnel"
          >
          <div>
            <strong>NexTunnel</strong>
            <span>{{ t('app.subtitle') }}</span>
          </div>
        </div>
      </n-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NCard, NForm, NFormItemGi, NGrid, NInput, NTag, useMessage } from 'naive-ui'
import LocalPortManager from '../components/LocalPortManager.vue'
import { useTunnelStore } from '../stores/tunnel'
import type { ServerSettings } from '../api/app'

type SettingsSection = 'connection' | 'network' | 'ports' | 'security' | 'appearance' | 'about'

interface SettingsNavItem {
  key: SettingsSection
  label: string
  description: string
  symbol: string
}

const store = useTunnelStore()
const message = useMessage()
const { t } = useI18n()
const isSaving = ref(false)
const activeSection = ref<SettingsSection>('connection')
const form = reactive<ServerSettings>({
  relay_addr: '127.0.0.1:7000',
  relay_token: '',
  control_plane_url: '',
  control_plane_token: '',
  stun_server: 'stun.l.google.com:19302',
  stun_alt_server: 'stun.l.google.com:19302',
})

const sections = computed<SettingsNavItem[]>(() => [
  {
    key: 'connection',
    label: t('settings.sections.connection'),
    description: t('settings.sections.connectionHint'),
    symbol: '01',
  },
  {
    key: 'network',
    label: t('settings.sections.network'),
    description: t('settings.sections.networkHint'),
    symbol: '02',
  },
  {
    key: 'ports',
    label: t('settings.sections.ports'),
    description: t('settings.sections.portsHint'),
    symbol: '03',
  },
  {
    key: 'security',
    label: t('settings.sections.security'),
    description: t('settings.sections.securityHint'),
    symbol: '04',
  },
  {
    key: 'appearance',
    label: t('settings.sections.appearance'),
    description: t('settings.sections.appearanceHint'),
    symbol: '05',
  },
  {
    key: 'about',
    label: t('settings.sections.about'),
    description: t('settings.sections.aboutHint'),
    symbol: '06',
  },
])

const networkFacts = computed(() => [
  { label: t('network.natType'), value: store.natType || t('status.waiting') },
  { label: t('network.platformName'), value: store.runtimeStatus?.tun.PlatformName || '--' },
  { label: t('network.adminRequired'), value: store.runtimeStatus?.tun.NeedsAdminPrivilege ? t('network.required') : t('network.notRequired') },
])

const securityItems = computed(() => [
  {
    label: t('settings.securityToken'),
    description: t('settings.securityTokenHint'),
    state: form.relay_token ? t('settings.configured') : t('settings.notConfigured'),
    ok: Boolean(form.relay_token),
  },
  {
    label: t('settings.securityControlPlane'),
    description: t('settings.securityControlPlaneHint'),
    state: form.control_plane_token ? t('settings.configured') : t('settings.optional'),
    ok: true,
  },
  {
    label: t('settings.securityLoopbackScan'),
    description: t('settings.securityLoopbackScanHint'),
    state: t('settings.enabled'),
    ok: true,
  },
])

const appearanceItems = computed(() => [
  { label: t('settings.brandCyan'), value: '#00ffff', color: '#00ffff' },
  { label: t('settings.brandViolet'), value: '#8a2be2', color: '#8a2be2' },
  { label: t('settings.brandDark'), value: '#091120', color: '#091120' },
])

const fillForm = (settings: ServerSettings): void => {
  Object.assign(form, settings)
}

const handleSave = async (): Promise<void> => {
  isSaving.value = true
  try {
    await store.saveServerSettings({ ...form })
    message.success(t('settings.saveSuccess'))
  } catch {
    message.error(store.lastError || t('settings.saveFailed'))
  } finally {
    isSaving.value = false
  }
}

const handleUsePortFromSettings = (): void => {
  message.info(t('ports.useFromTunnelPage'))
}

onMounted(async () => {
  await store.loadServerSettings()
  await store.refreshRuntimeStatus()
  await store.loadFavoritePorts()
  fillForm(store.serverSettings)
})
</script>

<style scoped>
.settings-view {
  display: grid;
  grid-template-columns: 292px minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

.settings-sidebar,
.settings-panel {
  border: 1px solid var(--line-soft);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.012)),
    var(--surface-bg);
  box-shadow: 0 18px 44px rgba(0, 0, 0, 0.2);
}

.settings-sidebar {
  position: sticky;
  top: 0;
  display: grid;
  gap: 8px;
  padding: 12px;
  border-radius: 8px;
}

.settings-nav-item {
  width: 100%;
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  grid-template-areas:
    'symbol label'
    'symbol hint';
  align-items: center;
  gap: 2px 10px;
  padding: 11px 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--text-dim);
  cursor: pointer;
  text-align: left;
  transition: transform 160ms cubic-bezier(0.4, 0, 0.2, 1);
}

.settings-nav-item:hover,
.settings-nav-item.active {
  transform: translateY(-1px);
  border-color: rgba(0, 255, 255, 0.22);
  background: rgba(0, 255, 255, 0.08);
  color: var(--text-main);
}

.settings-nav-item span {
  grid-area: symbol;
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: rgba(0, 255, 255, 0.1);
  color: var(--nex-cyan);
  font-family: Consolas, 'SFMono-Regular', monospace;
  font-size: 12px;
  font-weight: 800;
}

.settings-nav-item strong {
  grid-area: label;
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-nav-item small {
  grid-area: hint;
  overflow: hidden;
  color: var(--text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-content {
  min-width: 0;
}

.panel-title {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
}

.panel-title div,
.panel-title.compact {
  display: grid;
  gap: 4px;
}

.panel-title strong {
  color: var(--text-main);
  font-size: 16px;
}

.panel-title span {
  color: var(--text-dim);
  font-size: 12px;
}

.diagnostic-grid,
.setting-list,
.appearance-grid {
  display: grid;
  gap: 10px;
  margin-top: 16px;
}

.diagnostic-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.setting-fact,
.setting-row,
.appearance-item {
  min-height: 56px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.52);
}

.setting-fact span,
.setting-row span,
.appearance-item small {
  color: var(--text-dim);
  font-size: 12px;
}

.setting-fact strong,
.setting-row strong,
.appearance-item strong {
  color: var(--text-main);
  font-size: 13px;
}

.setting-row div,
.appearance-item div {
  display: grid;
  gap: 4px;
}

.appearance-item {
  justify-content: flex-start;
}

.appearance-item > span {
  width: 34px;
  height: 34px;
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: 8px;
}

.about-panel {
  display: flex;
  align-items: center;
  gap: 14px;
}

.about-panel img {
  width: 72px;
  height: 72px;
  border-radius: 8px;
  object-fit: cover;
}

.about-panel div {
  display: grid;
  gap: 6px;
}

.about-panel strong {
  color: var(--text-main);
  font-size: 24px;
}

.about-panel span {
  color: var(--text-dim);
  font-size: 13px;
}

@media (max-width: 1180px) {
  .settings-view {
    grid-template-columns: 1fr;
  }

  .settings-sidebar {
    position: static;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .diagnostic-grid {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .settings-nav-item {
    transition: none;
  }
}
</style>
