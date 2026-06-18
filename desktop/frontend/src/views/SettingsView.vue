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
          <PanelTitle
            :title="t('settings.connectionTitle')"
            :subtitle="t('settings.connectionSubtitle')"
          >
            <n-button
              type="primary"
              :loading="isSavingConnection"
              @click="handleSaveConnection"
            >
              {{ t('settings.save') }}
            </n-button>
          </PanelTitle>
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
              <n-input v-model:value="connectionForm.relay_addr" />
            </n-form-item-gi>
            <n-form-item-gi :label="t('connection.relayToken')">
              <n-input
                v-model:value="connectionForm.relay_token"
                type="password"
                show-password-on="click"
              />
            </n-form-item-gi>
            <n-form-item-gi :label="t('settings.controlPlaneURL')">
              <n-input
                v-model:value="connectionForm.control_plane_url"
                placeholder="https://control.example.com:9090"
              />
            </n-form-item-gi>
            <n-form-item-gi :label="t('settings.controlPlaneToken')">
              <n-input
                v-model:value="connectionForm.control_plane_token"
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
          <PanelTitle
            :title="t('settings.networkTitle')"
            :subtitle="t('settings.networkSubtitle')"
          >
            <n-button
              type="primary"
              :loading="isSavingConnection"
              @click="handleSaveConnection"
            >
              {{ t('settings.save') }}
            </n-button>
          </PanelTitle>
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
              <n-input v-model:value="connectionForm.stun_server" />
            </n-form-item-gi>
            <n-form-item-gi :label="t('settings.stunAltServer')">
              <n-input v-model:value="connectionForm.stun_alt_server" />
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
          <PanelTitle
            :title="t('settings.securityTitle')"
            :subtitle="t('settings.securitySubtitle')"
          />
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
          <PanelTitle
            :title="t('settings.appearanceTitle')"
            :subtitle="t('settings.appearanceSubtitle')"
          >
            <n-button
              type="primary"
              :loading="isSavingAppearance"
              @click="handleSaveAppearance"
            >
              {{ t('settings.save') }}
            </n-button>
          </PanelTitle>
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
            <n-form-item-gi :label="t('settings.themeMode')">
              <n-select
                v-model:value="appearanceForm.theme_mode"
                :options="themeOptions"
              />
            </n-form-item-gi>
            <n-form-item-gi :label="t('settings.language')">
              <n-select
                v-model:value="appearanceForm.language"
                :options="languageOptions"
              />
            </n-form-item-gi>
            <n-form-item-gi :label="t('settings.motionLevel')">
              <n-select
                v-model:value="appearanceForm.motion_level"
                :options="motionOptions"
              />
            </n-form-item-gi>
            <n-form-item-gi :label="t('settings.accentColor')">
              <n-input v-model:value="appearanceForm.accent_color" />
            </n-form-item-gi>
          </n-grid>
        </n-form>

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
        v-else-if="activeSection === 'general'"
        class="settings-panel"
        :bordered="false"
      >
        <template #header>
          <PanelTitle
            :title="t('settings.generalTitle')"
            :subtitle="t('settings.generalSubtitle')"
          >
            <n-button
              type="primary"
              :loading="isSavingGeneral"
              @click="handleSaveGeneral"
            >
              {{ t('settings.save') }}
            </n-button>
          </PanelTitle>
        </template>

        <div class="setting-list">
          <label class="setting-row interactive-row">
            <div>
              <strong>{{ t('settings.autoConnect') }}</strong>
              <span>{{ t('settings.autoConnectHint') }}</span>
            </div>
            <n-switch v-model:value="generalForm.auto_connect" />
          </label>
          <label class="setting-row interactive-row">
            <div>
              <strong>{{ t('settings.minimizeToTray') }}</strong>
              <span>{{ t('settings.minimizeToTrayHint') }}</span>
            </div>
            <n-switch
              v-model:value="generalForm.minimize_to_tray"
              :disabled="!generalForm.tray_supported"
            />
          </label>
          <label class="setting-row interactive-row">
            <div>
              <strong>{{ t('settings.autoStart') }}</strong>
              <span>{{ t('settings.autoStartHint') }}</span>
            </div>
            <n-switch
              v-model:value="autoStartModel"
              :loading="isSavingAutoStart"
            />
          </label>
          <label class="setting-row interactive-row">
            <div>
              <strong>{{ t('settings.includeSensitiveExport') }}</strong>
              <span>{{ t('settings.includeSensitiveExportHint') }}</span>
            </div>
            <n-switch v-model:value="generalForm.export_include_tokens" />
          </label>
        </div>

        <div class="action-grid">
          <n-button
            secondary
            :loading="isExporting"
            @click="handleExportConfig"
          >
            {{ t('settings.exportConfig') }}
          </n-button>
          <n-input
            v-model:value="importText"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 8 }"
            :placeholder="t('settings.importPlaceholder')"
          />
          <n-button
            secondary
            :disabled="importText.trim().length === 0"
            :loading="isImporting"
            @click="handleImportConfig"
          >
            {{ t('settings.importConfig') }}
          </n-button>
        </div>

        <n-input
          v-if="exportText"
          v-model:value="exportText"
          class="export-output"
          type="textarea"
          readonly
          :autosize="{ minRows: 4, maxRows: 10 }"
        />
      </n-card>

      <n-card
        v-else
        class="settings-panel"
        :bordered="false"
      >
        <template #header>
          <PanelTitle
            :title="t('settings.aboutTitle')"
            :subtitle="t('settings.aboutSubtitle')"
          />
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

        <div class="action-grid compact-actions">
          <n-button
            secondary
            :loading="isCheckingUpdate"
            @click="handleCheckUpdate"
          >
            {{ t('settings.checkUpdate') }}
          </n-button>
          <n-button
            secondary
            :loading="isCollectingDiagnostics"
            @click="handleCollectDiagnostics"
          >
            {{ t('settings.collectDiagnostics') }}
          </n-button>
        </div>

        <div
          v-if="store.updateInfo"
          class="setting-row"
        >
          <div>
            <strong>{{ updateStatusText }}</strong>
            <span>{{ store.updateInfo.error || store.updateInfo.url || store.updateInfo.current_version }}</span>
          </div>
          <n-tag
            round
            :type="store.updateInfo.available ? 'warning' : 'success'"
            :bordered="false"
          >
            {{ store.updateInfo.latest_version || store.updateInfo.current_version }}
          </n-tag>
        </div>

        <n-input
          v-if="store.diagnosticsInfo?.text"
          v-model:value="store.diagnosticsInfo.text"
          class="export-output"
          type="textarea"
          readonly
          :autosize="{ minRows: 8, maxRows: 14 }"
        />
      </n-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NCard,
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NSelect,
  NSwitch,
  NTag,
  useMessage,
  type SelectOption,
} from 'naive-ui'
import LocalPortManager from '../components/LocalPortManager.vue'
import { useTunnelStore } from '../stores/tunnel'
import type { AppearanceSettings, GeneralSettings, ServerSettings } from '../api/app'

type SettingsSection = 'connection' | 'network' | 'ports' | 'security' | 'appearance' | 'general' | 'about'

interface SettingsNavItem {
  key: SettingsSection
  label: string
  description: string
  symbol: string
}

const PanelTitle = defineComponent({
  props: {
    title: { type: String, required: true },
    subtitle: { type: String, required: true },
  },
  setup(props, { slots }) {
    return () =>
      h('div', { class: 'panel-title' }, [
        h('div', [h('strong', props.title), h('span', props.subtitle)]),
        slots.default?.(),
      ])
  },
})

const createServerSettingsForm = (): ServerSettings => ({
  relay_addr: '127.0.0.1:7000',
  relay_token: '',
  control_plane_url: '',
  control_plane_token: '',
  stun_server: 'stun.l.google.com:19302',
  stun_alt_server: 'stun.l.google.com:19302',
})

const createAppearanceForm = (): AppearanceSettings => ({
  theme_mode: 'dark',
  motion_level: 'normal',
  language: 'zh-CN',
  accent_color: '#00ffff',
})

const createGeneralForm = (): GeneralSettings => ({
  auto_connect: false,
  minimize_to_tray: false,
  start_minimized: false,
  export_include_tokens: false,
  tray_supported: false,
})

const store = useTunnelStore()
const message = useMessage()
const { t } = useI18n()
const isSavingConnection = ref(false)
const isSavingAppearance = ref(false)
const isSavingGeneral = ref(false)
const isSavingAutoStart = ref(false)
const isExporting = ref(false)
const isImporting = ref(false)
const isCheckingUpdate = ref(false)
const isCollectingDiagnostics = ref(false)
const activeSection = ref<SettingsSection>('connection')
const exportText = ref('')
const importText = ref('')
const connectionForm = reactive<ServerSettings>(createServerSettingsForm())
const appearanceForm = reactive<AppearanceSettings>(createAppearanceForm())
const generalForm = reactive<GeneralSettings>(createGeneralForm())

const sections = computed<SettingsNavItem[]>(() => [
  { key: 'connection', label: t('settings.sections.connection'), description: t('settings.sections.connectionHint'), symbol: '01' },
  { key: 'network', label: t('settings.sections.network'), description: t('settings.sections.networkHint'), symbol: '02' },
  { key: 'ports', label: t('settings.sections.ports'), description: t('settings.sections.portsHint'), symbol: '03' },
  { key: 'security', label: t('settings.sections.security'), description: t('settings.sections.securityHint'), symbol: '04' },
  { key: 'appearance', label: t('settings.sections.appearance'), description: t('settings.sections.appearanceHint'), symbol: '05' },
  { key: 'general', label: t('settings.sections.general'), description: t('settings.sections.generalHint'), symbol: '06' },
  { key: 'about', label: t('settings.sections.about'), description: t('settings.sections.aboutHint'), symbol: '07' },
])

const themeOptions = computed<SelectOption[]>(() => [
  { label: t('settings.themeDark'), value: 'dark' },
  { label: t('settings.themeLight'), value: 'light' },
  { label: t('settings.themeSystem'), value: 'system' },
])

const motionOptions = computed<SelectOption[]>(() => [
  { label: t('settings.motionNormal'), value: 'normal' },
  { label: t('settings.motionReduced'), value: 'reduced' },
])

const languageOptions = computed<SelectOption[]>(() => [
  { label: t('settings.simplifiedChinese'), value: 'zh-CN' },
  { label: t('settings.english'), value: 'en-US' },
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
    state: connectionForm.relay_token ? t('settings.configured') : t('settings.notConfigured'),
    ok: Boolean(connectionForm.relay_token),
  },
  {
    label: t('settings.securityControlPlane'),
    description: t('settings.securityControlPlaneHint'),
    state: connectionForm.control_plane_token ? t('settings.configured') : t('settings.optional'),
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
  { label: t('settings.accentColor'), value: appearanceForm.accent_color, color: appearanceForm.accent_color },
])

const autoStartModel = computed({
  get: () => store.autoStartEnabled,
  set: (enabled: boolean) => {
    void handleSetAutoStart(enabled)
  },
})

const updateStatusText = computed(() => {
  if (!store.updateInfo) return ''
  if (store.updateInfo.error) return t('settings.updateCheckFailed')
  return store.updateInfo.available ? t('settings.updateAvailable') : t('settings.noUpdate')
})

const fillConnectionForm = (settings: ServerSettings): void => {
  Object.assign(connectionForm, settings)
}

const fillAppearanceForm = (settings: AppearanceSettings): void => {
  Object.assign(appearanceForm, settings)
}

const fillGeneralForm = (settings: GeneralSettings): void => {
  Object.assign(generalForm, settings)
}

const handleSaveConnection = async (): Promise<void> => {
  isSavingConnection.value = true
  try {
    await store.saveServerSettings({ ...connectionForm })
    message.success(t('settings.saveSuccess'))
  } catch {
    message.error(store.lastError || t('settings.saveFailed'))
  } finally {
    isSavingConnection.value = false
  }
}

const handleSaveAppearance = async (): Promise<void> => {
  isSavingAppearance.value = true
  try {
    await store.saveAppearanceSettings({ ...appearanceForm })
    message.success(t('settings.saveSuccess'))
  } catch {
    message.error(store.lastError || t('settings.saveFailed'))
  } finally {
    isSavingAppearance.value = false
  }
}

const handleSaveGeneral = async (): Promise<void> => {
  isSavingGeneral.value = true
  try {
    await store.saveGeneralSettings({ ...generalForm })
    message.success(t('settings.saveSuccess'))
  } catch {
    message.error(store.lastError || t('settings.saveFailed'))
  } finally {
    isSavingGeneral.value = false
  }
}

const handleSetAutoStart = async (enabled: boolean): Promise<void> => {
  isSavingAutoStart.value = true
  try {
    await store.setAutoStartEnabled(enabled)
    message.success(t('settings.saveSuccess'))
  } catch {
    message.error(store.lastError || t('settings.saveFailed'))
  } finally {
    isSavingAutoStart.value = false
  }
}

const handleExportConfig = async (): Promise<void> => {
  isExporting.value = true
  try {
    exportText.value = await store.exportConfig({ include_sensitive: generalForm.export_include_tokens })
    message.success(t('settings.exportSuccess'))
  } catch {
    message.error(store.lastError || t('settings.exportFailed'))
  } finally {
    isExporting.value = false
  }
}

const handleImportConfig = async (): Promise<void> => {
  isImporting.value = true
  try {
    await store.importConfig(importText.value)
    fillConnectionForm(store.serverSettings)
    fillAppearanceForm(store.appearanceSettings)
    fillGeneralForm(store.generalSettings)
    importText.value = ''
    message.success(t('settings.importSuccess'))
  } catch {
    message.error(store.lastError || t('settings.importFailed'))
  } finally {
    isImporting.value = false
  }
}

const handleCheckUpdate = async (): Promise<void> => {
  isCheckingUpdate.value = true
  try {
    const result = await store.checkForUpdate()
    message.info(result.error || updateStatusText.value)
  } catch {
    message.error(store.lastError || t('settings.updateCheckFailed'))
  } finally {
    isCheckingUpdate.value = false
  }
}

const handleCollectDiagnostics = async (): Promise<void> => {
  isCollectingDiagnostics.value = true
  try {
    await store.collectDiagnostics()
    message.success(t('settings.diagnosticsReady'))
  } catch {
    message.error(store.lastError || t('settings.diagnosticsFailed'))
  } finally {
    isCollectingDiagnostics.value = false
  }
}

const handleUsePortFromSettings = (): void => {
  message.info(t('ports.useFromTunnelPage'))
}

watch(
  () => store.generalSettings,
  (settings) => fillGeneralForm(settings),
  { deep: true },
)

onMounted(async () => {
  await store.loadServerSettings()
  await store.loadAppearanceSettings()
  await store.loadGeneralSettings()
  await store.refreshRuntimeStatus()
  await store.loadFavoritePorts()
  fillConnectionForm(store.serverSettings)
  fillAppearanceForm(store.appearanceSettings)
  fillGeneralForm(store.generalSettings)
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

.panel-title div {
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
.appearance-grid,
.action-grid {
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

.interactive-row {
  cursor: pointer;
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

.action-grid {
  grid-template-columns: minmax(160px, auto) minmax(0, 1fr) minmax(120px, auto);
  align-items: start;
}

.compact-actions {
  grid-template-columns: repeat(2, minmax(0, 180px));
}

.export-output {
  margin-top: 14px;
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
  .settings-view,
  .action-grid,
  .compact-actions {
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
