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
            <n-tag
              size="small"
              round
              :type="isSavingConnection ? 'warning' : 'success'"
              :bordered="false"
            >
              {{ isSavingConnection ? t('settings.autoSaving') : t('settings.autoApplied') }}
            </n-tag>
          </PanelTitle>
        </template>

        <div class="settings-node-panel">
          <div class="node-toolbar">
            <n-input
              v-model:value="nodeSearchKeyword"
              class="node-search"
              clearable
              :placeholder="t('connection.nodes.searchPlaceholder')"
            >
              <template #prefix>
                <Search :size="15" />
              </template>
            </n-input>
            <div class="node-toolbar-actions">
              <n-button
                size="small"
                type="primary"
                @click="handleOpenCreateNodeModal"
              >
                <template #icon>
                  <Plus :size="15" />
                </template>
                {{ t('settings.addServerNode') }}
              </n-button>
              <n-button
                size="small"
                type="error"
                secondary
                :disabled="connectionForm.nodes.length <= 1"
                @click="handleDeleteServerNode(connectionForm.active_node_id)"
              >
                {{ t('settings.deleteServerNode') }}
              </n-button>
              <n-button-group size="small">
                <n-button
                  :type="nodeViewMode === 'card' ? 'primary' : 'default'"
                  secondary
                  :title="t('connection.nodes.cardView')"
                  @click="nodeViewMode = 'card'"
                >
                  <template #icon>
                    <LayoutGrid :size="15" />
                  </template>
                </n-button>
                <n-button
                  :type="nodeViewMode === 'table' ? 'primary' : 'default'"
                  secondary
                  :title="t('connection.nodes.tableView')"
                  @click="nodeViewMode = 'table'"
                >
                  <template #icon>
                    <Table2 :size="15" />
                  </template>
                </n-button>
              </n-button-group>
            </div>
          </div>

          <n-empty
            v-if="filteredServerNodes.length === 0"
            class="compact-empty"
            :description="connectionForm.nodes.length === 0 ? t('connection.nodes.empty') : t('connection.nodes.noSearchResult')"
          />

          <div
            v-else-if="nodeViewMode === 'card'"
            class="server-node-grid"
          >
            <article
              v-for="node in filteredServerNodes"
              :key="node.id"
              class="server-node-card"
              :class="{ selected: isActiveServerNode(node), default: isActiveServerNode(node) }"
              role="button"
              tabindex="0"
              @click="handleActiveServerNodeChange(node.id)"
              @keydown.enter.prevent="handleActiveServerNodeChange(node.id)"
            >
              <div class="node-card-head">
                <div>
                  <strong>{{ getServerNodeName(node) }}</strong>
                  <span>{{ node.relay_addr || t('connection.nodes.missingRelay') }}</span>
                </div>
                <n-tag
                  round
                  size="small"
                  :type="isActiveServerNode(node) ? 'success' : 'default'"
                  :bordered="false"
                >
                  <CheckCircle2
                    v-if="isActiveServerNode(node)"
                    :size="13"
                  />
                  {{ isActiveServerNode(node) ? t('connection.nodes.defaultNode') : t('connection.nodes.ready') }}
                </n-tag>
              </div>
              <div class="node-endpoints">
                <span>{{ t('connection.nodes.controlPlane') }}</span>
                <strong>{{ node.control_plane_url || t('connection.nodes.notConfigured') }}</strong>
                <span>{{ t('connection.nodes.stun') }}</span>
                <strong>{{ node.stun_server || t('connection.nodes.notConfigured') }}</strong>
              </div>
              <span class="node-select-hint">
                {{ isActiveServerNode(node) ? t('connection.nodes.selected') : t('connection.nodes.clickToSelect') }}
              </span>
              <div class="node-card-actions">
                <n-button
                  size="tiny"
                  secondary
                  @click.stop="handleOpenEditNodeModal(node.id)"
                >
                  {{ t('connection.nodes.editNode') }}
                </n-button>
                <n-button
                  size="tiny"
                  type="error"
                  secondary
                  :disabled="connectionForm.nodes.length <= 1"
                  @click.stop="handleDeleteServerNode(node.id)"
                >
                  {{ t('connection.nodes.deleteNode') }}
                </n-button>
              </div>
            </article>
          </div>

          <n-data-table
            v-else
            class="server-node-table"
            :columns="serverNodeColumns"
            :data="filteredServerNodes"
            :pagination="serverNodePagination"
            :row-key="getServerNodeRowKey"
            :row-class-name="getServerNodeRowClassName"
            :row-props="getServerNodeRowProps"
          />
        </div>

        <n-modal
          v-model:show="showNodeModal"
          preset="card"
          class="server-node-modal"
          :title="nodeModalTitle"
          :bordered="false"
          :segmented="{ content: true, footer: 'soft' }"
        >
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
              <n-form-item-gi :label="t('settings.serverNodeName')">
                <n-input v-model:value="nodeDraft.name" />
              </n-form-item-gi>
              <n-form-item-gi :label="t('connection.relayAddress')">
                <n-input v-model:value="nodeDraft.relay_addr" />
              </n-form-item-gi>
              <n-form-item-gi :label="t('connection.relayToken')">
                <n-input
                  v-model:value="nodeDraft.relay_token"
                  type="password"
                  show-password-on="click"
                />
              </n-form-item-gi>
              <n-form-item-gi :label="t('settings.controlPlaneURL')">
                <n-input
                  v-model:value="nodeDraft.control_plane_url"
                  placeholder="https://control.example.com:9090"
                />
              </n-form-item-gi>
              <n-form-item-gi :label="t('settings.controlPlaneToken')">
                <n-input
                  v-model:value="nodeDraft.control_plane_token"
                  type="password"
                  show-password-on="click"
                />
              </n-form-item-gi>
              <n-form-item-gi :label="t('settings.stunServer')">
                <n-input v-model:value="nodeDraft.stun_server" />
              </n-form-item-gi>
              <n-form-item-gi :label="t('settings.stunAltServer')">
                <n-input v-model:value="nodeDraft.stun_alt_server" />
              </n-form-item-gi>
            </n-grid>
          </n-form>

          <template #footer>
            <div class="modal-actions">
              <n-button
                secondary
                @click="showNodeModal = false"
              >
                {{ t('common.cancel') }}
              </n-button>
              <n-button
                type="primary"
                :disabled="!canSaveNodeDraft"
                @click="handleSaveNodeDraft"
              >
                {{ t('common.save') }}
              </n-button>
            </div>
          </template>
        </n-modal>
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
            <n-tag
              size="small"
              round
              :type="isSavingConnection ? 'warning' : 'success'"
              :bordered="false"
            >
              {{ isSavingConnection ? t('settings.autoSaving') : t('settings.autoApplied') }}
            </n-tag>
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
            <n-tag
              size="small"
              round
              :type="isSavingAppearance ? 'warning' : 'success'"
              :bordered="false"
            >
              {{ isSavingAppearance ? t('settings.autoSaving') : t('settings.autoApplied') }}
            </n-tag>
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
            <n-tag
              size="small"
              round
              :type="isSavingGeneral ? 'warning' : 'success'"
              :bordered="false"
            >
              {{ isSavingGeneral ? t('settings.autoSaving') : t('settings.autoApplied') }}
            </n-tag>
          </PanelTitle>
        </template>

        <div class="setting-list">
          <label class="setting-row interactive-row">
            <div>
              <strong>{{ t('settings.language') }}</strong>
              <span>{{ t('settings.languageHint') }}</span>
            </div>
            <n-select
              v-model:value="generalForm.language"
              class="setting-inline-select"
              :options="languageOptions"
            />
          </label>
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
              <strong>{{ t('settings.startMinimized') }}</strong>
              <span>{{ t('settings.startMinimizedHint') }}</span>
            </div>
            <n-switch v-model:value="generalForm.start_minimized" />
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
import { CheckCircle2, LayoutGrid, Plus, Search, Table2 } from 'lucide-vue-next'
import {
  NButton,
  NButtonGroup,
  NCard,
  NDataTable,
  NEmpty,
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NModal,
  NRadio,
  NSelect,
  NSwitch,
  NTag,
  useMessage,
  type DataTableColumns,
  type DataTableRowKey,
  type SelectOption,
} from 'naive-ui'
import LocalPortManager from '../components/LocalPortManager.vue'
import { useTunnelStore } from '../stores/tunnel'
import type { AppearanceSettings, GeneralSettings, ServerNodeSettings, ServerSettings } from '../api/app'

type SettingsSection = 'connection' | 'network' | 'ports' | 'security' | 'appearance' | 'general' | 'about'
type NodeViewMode = 'card' | 'table'
type NodeModalMode = 'create' | 'edit'

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
  active_node_id: 'default',
  nodes: [
    {
      id: 'default',
      name: '默认节点',
      relay_addr: '127.0.0.1:7000',
      relay_token: '',
      control_plane_url: '',
      control_plane_token: '',
      stun_server: 'stun.l.google.com:19302',
      stun_alt_server: 'stun.l.google.com:19302',
    },
  ],
})

const createEmptyServerNode = (): ServerNodeSettings => ({
  id: '',
  name: '',
  relay_addr: '',
  relay_token: '',
  control_plane_url: '',
  control_plane_token: '',
  stun_server: '',
  stun_alt_server: '',
})

const createAppearanceForm = (): AppearanceSettings => ({
  theme_mode: 'dark',
  motion_level: 'normal',
  accent_color: '#00ffff',
})

const createGeneralForm = (): GeneralSettings => ({
  auto_connect: false,
  minimize_to_tray: false,
  start_minimized: false,
  export_include_tokens: false,
  tray_supported: false,
  language: 'zh-CN',
})

const AUTO_SAVE_DELAY_MS = 520

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
const nodeSearchKeyword = ref('')
const nodeViewMode = ref<NodeViewMode>('card')
const showNodeModal = ref(false)
const nodeModalMode = ref<NodeModalMode>('create')
const editingNodeID = ref('')
const connectionForm = reactive<ServerSettings>(createServerSettingsForm())
const nodeDraft = reactive<ServerNodeSettings>(createEmptyServerNode())
const appearanceForm = reactive<AppearanceSettings>(createAppearanceForm())
const generalForm = reactive<GeneralSettings>(createGeneralForm())
const isHydratingForms = ref(true)
let connectionSaveTimer: ReturnType<typeof setTimeout> | undefined
let appearanceSaveTimer: ReturnType<typeof setTimeout> | undefined
let generalSaveTimer: ReturnType<typeof setTimeout> | undefined

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

const activeServerNode = computed(() => {
  return connectionForm.nodes.find((node) => node.id === connectionForm.active_node_id) ?? connectionForm.nodes[0]
})

const getServerNodeName = (node: ServerNodeSettings): string => {
  return node.name || node.relay_addr || t('connection.nodes.unnamedNode')
}

const isActiveServerNode = (node: ServerNodeSettings): boolean => {
  return node.id === connectionForm.active_node_id
}

const getServerNodeRowKey = (node: ServerNodeSettings): DataTableRowKey => node.id

const getServerNodeRowClassName = (node: ServerNodeSettings): string => {
  return isActiveServerNode(node) ? 'server-node-row selected' : 'server-node-row'
}

const getServerNodeRowProps = (node: ServerNodeSettings): Record<string, unknown> => ({
  onClick: () => handleActiveServerNodeChange(node.id),
})

const canSaveNodeDraft = computed(() => nodeDraft.name.trim().length > 0 && nodeDraft.relay_addr.trim().length > 0)

const nodeModalTitle = computed(() => (
  nodeModalMode.value === 'create' ? t('connection.nodes.createNodeTitle') : t('connection.nodes.editNodeTitle')
))

const filteredServerNodes = computed(() => {
  const keyword = nodeSearchKeyword.value.trim().toLowerCase()
  if (!keyword) return connectionForm.nodes
  return connectionForm.nodes.filter((node) =>
    [
      node.name,
      node.relay_addr,
      node.control_plane_url,
      node.stun_server,
      node.stun_alt_server,
    ].some((value) => (value ?? '').toLowerCase().includes(keyword)),
  )
})

const serverNodePagination = computed(() => ({
  pageSize: 8,
  showSizePicker: false,
}))

const serverNodeColumns = computed<DataTableColumns<ServerNodeSettings>>(() => [
  {
    key: 'selected',
    title: '',
    width: 58,
    render: (node) =>
      h(NRadio, {
        checked: isActiveServerNode(node),
        onUpdateChecked: () => handleActiveServerNodeChange(node.id),
        'aria-label': t('connection.nodes.selectNode'),
      }),
  },
  {
    key: 'name',
    title: t('connection.nodes.name'),
    render: (node) =>
      h('div', { class: 'node-table-name' }, [
        h('strong', getServerNodeName(node)),
        h('span', node.relay_addr || t('connection.nodes.missingRelay')),
      ]),
  },
  {
    key: 'relay_addr',
    title: t('connection.relayAddress'),
    ellipsis: { tooltip: true },
  },
  {
    key: 'control_plane_url',
    title: t('connection.nodes.controlPlane'),
    render: (node) => node.control_plane_url || t('connection.nodes.notConfigured'),
  },
  {
    key: 'stun_server',
    title: t('connection.nodes.stun'),
    render: (node) => node.stun_server || t('connection.nodes.notConfigured'),
  },
  {
    key: 'status',
    title: t('connection.nodes.status'),
    width: 120,
    render: (node) =>
      h(
        NTag,
        {
          round: true,
          size: 'small',
          type: isActiveServerNode(node) ? 'success' : 'default',
          bordered: false,
        },
        { default: () => (isActiveServerNode(node) ? t('connection.nodes.defaultNode') : t('connection.nodes.ready')) },
      ),
  },
  {
    key: 'actions',
    title: t('connection.nodes.actions'),
    width: 158,
    render: (node) =>
      h('div', { class: 'node-table-actions', onClick: (event: MouseEvent) => event.stopPropagation() }, [
        h(
          NButton,
          {
            size: 'tiny',
            secondary: true,
            onClick: () => handleOpenEditNodeModal(node.id),
          },
          { default: () => t('connection.nodes.editNode') },
        ),
        h(
          NButton,
          {
            size: 'tiny',
            type: 'error',
            secondary: true,
            disabled: connectionForm.nodes.length <= 1,
            onClick: () => handleDeleteServerNode(node.id),
          },
          { default: () => t('connection.nodes.deleteNode') },
        ),
      ]),
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
  Object.assign(connectionForm, cloneServerSettings(settings))
}

const fillAppearanceForm = (settings: AppearanceSettings): void => {
  Object.assign(appearanceForm, settings)
}

const fillGeneralForm = (settings: GeneralSettings): void => {
  Object.assign(generalForm, settings)
}

const cloneServerNode = (node: ServerNodeSettings): ServerNodeSettings => ({ ...node })

const cloneServerSettings = (settings: ServerSettings): ServerSettings => ({
  ...settings,
  nodes: (settings.nodes?.length ? settings.nodes : createServerSettingsForm().nodes).map(cloneServerNode),
})

// syncConnectionFieldsFromActiveServerNode 将默认节点写入兼容字段，供旧连接入口继续读取。
const syncConnectionFieldsFromActiveServerNode = (): void => {
  const node = activeServerNode.value
  if (!node) return
  connectionForm.relay_addr = node.relay_addr
  connectionForm.relay_token = node.relay_token
  connectionForm.control_plane_url = node.control_plane_url
  connectionForm.control_plane_token = node.control_plane_token
  connectionForm.stun_server = node.stun_server
  connectionForm.stun_alt_server = node.stun_alt_server
}

const syncActiveNodeNetworkFieldsFromConnectionFields = (): void => {
  const node = activeServerNode.value
  if (!node) return
  node.stun_server = connectionForm.stun_server
  node.stun_alt_server = connectionForm.stun_alt_server
}

const saveConnectionNow = async (): Promise<void> => {
  isSavingConnection.value = true
  try {
    if (activeSection.value === 'network') {
      syncActiveNodeNetworkFieldsFromConnectionFields()
    }
    syncConnectionFieldsFromActiveServerNode()
    await store.saveServerSettings({ ...connectionForm })
  } catch {
    message.error(store.lastError || t('settings.saveFailed'))
  } finally {
    isSavingConnection.value = false
  }
}

const saveAppearanceNow = async (): Promise<void> => {
  isSavingAppearance.value = true
  try {
    await store.saveAppearanceSettings({ ...appearanceForm })
  } catch {
    message.error(store.lastError || t('settings.saveFailed'))
  } finally {
    isSavingAppearance.value = false
  }
}

const saveGeneralNow = async (): Promise<void> => {
  isSavingGeneral.value = true
  try {
    await store.saveGeneralSettings({ ...generalForm })
  } catch {
    message.error(store.lastError || t('settings.saveFailed'))
  } finally {
    isSavingGeneral.value = false
  }
}

const queueConnectionSave = (): void => {
  if (isHydratingForms.value) return
  if (connectionSaveTimer) clearTimeout(connectionSaveTimer)
  connectionSaveTimer = setTimeout(() => {
    void saveConnectionNow()
  }, AUTO_SAVE_DELAY_MS)
}

const queueAppearanceSave = (): void => {
  if (isHydratingForms.value) return
  if (appearanceSaveTimer) clearTimeout(appearanceSaveTimer)
  appearanceSaveTimer = setTimeout(() => {
    void saveAppearanceNow()
  }, AUTO_SAVE_DELAY_MS)
}

const queueGeneralSave = (): void => {
  if (isHydratingForms.value) return
  if (generalSaveTimer) clearTimeout(generalSaveTimer)
  generalSaveTimer = setTimeout(() => {
    void saveGeneralNow()
  }, AUTO_SAVE_DELAY_MS)
}

const handleActiveServerNodeChange = (nodeID: string): void => {
  connectionForm.active_node_id = nodeID
  syncConnectionFieldsFromActiveServerNode()
  queueConnectionSave()
}

const fillNodeDraft = (node: ServerNodeSettings): void => {
  Object.assign(nodeDraft, cloneServerNode(node))
}

const createDefaultServerNode = (): ServerNodeSettings => {
  const nextIndex = connectionForm.nodes.length + 1
  return {
    id: `node-${Date.now()}`,
    name: t('settings.newServerNodeName', { count: nextIndex }),
    relay_addr: '127.0.0.1:7000',
    relay_token: '',
    control_plane_url: '',
    control_plane_token: '',
    stun_server: activeServerNode.value?.stun_server || connectionForm.stun_server || 'stun.l.google.com:19302',
    stun_alt_server: activeServerNode.value?.stun_alt_server || connectionForm.stun_alt_server || connectionForm.stun_server || 'stun.l.google.com:19302',
  }
}

const handleOpenCreateNodeModal = (): void => {
  nodeModalMode.value = 'create'
  editingNodeID.value = ''
  fillNodeDraft(createDefaultServerNode())
  showNodeModal.value = true
}

const handleOpenEditNodeModal = (nodeID: string): void => {
  const node = connectionForm.nodes.find((item) => item.id === nodeID)
  if (!node) return
  nodeModalMode.value = 'edit'
  editingNodeID.value = nodeID
  fillNodeDraft(node)
  showNodeModal.value = true
}

const handleSaveNodeDraft = (): void => {
  if (!canSaveNodeDraft.value) return
  const normalizedNode: ServerNodeSettings = {
    ...cloneServerNode(nodeDraft),
    id: nodeDraft.id || `node-${Date.now()}`,
    name: nodeDraft.name.trim(),
    relay_addr: nodeDraft.relay_addr.trim(),
    relay_token: nodeDraft.relay_token.trim(),
    control_plane_url: nodeDraft.control_plane_url.trim(),
    control_plane_token: nodeDraft.control_plane_token.trim(),
    stun_server: nodeDraft.stun_server.trim(),
    stun_alt_server: nodeDraft.stun_alt_server.trim(),
  }

  if (nodeModalMode.value === 'create') {
    connectionForm.nodes.push(normalizedNode)
    connectionForm.active_node_id = normalizedNode.id
  } else {
    const targetIndex = connectionForm.nodes.findIndex((node) => node.id === editingNodeID.value)
    if (targetIndex < 0) return
    connectionForm.nodes.splice(targetIndex, 1, normalizedNode)
  }

  syncConnectionFieldsFromActiveServerNode()
  showNodeModal.value = false
  queueConnectionSave()
}

const handleDeleteServerNode = (nodeID: string): void => {
  if (connectionForm.nodes.length <= 1) return
  const nextNodes = connectionForm.nodes.filter((node) => node.id !== nodeID)
  connectionForm.nodes.splice(0, connectionForm.nodes.length, ...nextNodes)
  if (connectionForm.active_node_id === nodeID) {
    connectionForm.active_node_id = connectionForm.nodes[0]?.id ?? 'default'
  }
  syncConnectionFieldsFromActiveServerNode()
  queueConnectionSave()
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
  () => connectionForm,
  () => {
    if (showNodeModal.value) return
    queueConnectionSave()
  },
  { deep: true },
)

watch(
  () => appearanceForm,
  () => queueAppearanceSave(),
  { deep: true },
)

watch(
  () => generalForm,
  () => queueGeneralSave(),
  { deep: true },
)

watch(
  () => store.generalSettings,
  (settings) => {
    if (isSavingGeneral.value) return
    isHydratingForms.value = true
    fillGeneralForm(settings)
    isHydratingForms.value = false
  },
  { deep: true },
)

onMounted(async () => {
  isHydratingForms.value = true
  await store.loadServerSettings()
  await store.loadAppearanceSettings()
  await store.loadGeneralSettings()
  await store.refreshRuntimeStatus()
  await store.loadFavoritePorts()
  fillConnectionForm(store.serverSettings)
  fillAppearanceForm(store.appearanceSettings)
  fillGeneralForm(store.generalSettings)
  isHydratingForms.value = false
})
</script>

<style scoped>
.settings-view {
  flex: 1 1 auto;
  min-height: 0;
  height: 100%;
  display: grid;
  grid-template-columns: 292px minmax(0, 1fr);
  gap: 18px;
  align-items: start;
  overflow: auto;
  padding-right: 2px;
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

.settings-node-panel {
  display: grid;
  gap: 14px;
  margin-top: 0;
}

.node-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.node-search {
  max-width: 420px;
}

.node-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.server-node-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 12px;
}

.server-node-card {
  position: relative;
  min-width: 0;
  min-height: 178px;
  display: grid;
  align-content: space-between;
  gap: 14px;
  padding: 14px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.012)),
    rgba(9, 17, 32, 0.58);
  cursor: pointer;
  transition:
    transform 180ms cubic-bezier(0.4, 0, 0.2, 1),
    border-color 180ms cubic-bezier(0.4, 0, 0.2, 1),
    background 180ms cubic-bezier(0.4, 0, 0.2, 1),
    box-shadow 180ms cubic-bezier(0.4, 0, 0.2, 1);
}

.server-node-card:hover {
  transform: translateY(-1px);
  border-color: rgba(0, 255, 255, 0.24);
}

.server-node-card.selected {
  border-color: rgba(0, 255, 255, 0.54);
  background:
    linear-gradient(135deg, rgba(0, 255, 255, 0.12), rgba(138, 43, 226, 0.055)),
    rgba(9, 17, 32, 0.68);
  box-shadow:
    inset 0 0 0 1px rgba(0, 255, 255, 0.18),
    0 14px 34px rgba(0, 255, 255, 0.08);
}

.node-card-head {
  min-width: 0;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
}

.node-card-head > div {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.node-card-head strong,
.node-table-name strong {
  overflow: hidden;
  color: var(--text-main);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-card-head span,
.node-table-name span,
.node-select-hint {
  overflow: hidden;
  color: var(--text-dim);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-endpoints {
  display: grid;
  grid-template-columns: minmax(92px, auto) minmax(0, 1fr);
  gap: 8px 10px;
  align-items: center;
}

.node-endpoints span {
  color: var(--text-muted);
  font-size: 12px;
}

.node-endpoints strong {
  overflow: hidden;
  color: var(--text-main);
  font-family: Consolas, 'SFMono-Regular', monospace;
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-card-actions,
.node-table-actions,
.modal-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.node-card-actions {
  justify-content: flex-end;
}

.node-table-actions {
  justify-content: flex-start;
}

.modal-actions {
  justify-content: flex-end;
}

:global(.server-node-modal) {
  width: min(760px, calc(100vw - 48px));
}

.server-node-table {
  overflow: hidden;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
}

.node-table-name {
  min-width: 0;
  display: grid;
  gap: 3px;
}

:deep(.server-node-row) {
  cursor: pointer;
  transition:
    background 180ms cubic-bezier(0.4, 0, 0.2, 1),
    box-shadow 180ms cubic-bezier(0.4, 0, 0.2, 1);
}

:deep(.server-node-row:hover) {
  background: rgba(0, 255, 255, 0.045);
}

:deep(.server-node-row.selected td:first-child) {
  box-shadow: inset 3px 0 0 var(--nex-cyan);
}

:deep(.server-node-row.selected) {
  background: rgba(0, 255, 255, 0.075);
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

.setting-inline-select {
  width: 180px;
  flex: 0 0 180px;
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

  .setting-row {
    align-items: stretch;
    flex-direction: column;
  }

  .node-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .node-search {
    max-width: none;
  }

  .node-toolbar-actions {
    justify-content: flex-start;
  }

  .setting-inline-select {
    width: 100%;
    flex-basis: auto;
  }
}

@media (prefers-reduced-motion: reduce) {
  .settings-nav-item {
    transition: none;
  }

  .server-node-card,
  :deep(.server-node-row) {
    transition: none;
  }
}
</style>
