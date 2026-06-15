<template>
  <section class="client-dashboard">
    <template v-if="viewMode === 'overview'">
      <section class="overview-hero">
        <div class="connection-strip">
          <div class="connection-state">
            <n-tag
              round
              :type="statusTagType"
              :bordered="false"
            >
              <span class="status-dot" />
              {{ statusLabel }}
            </n-tag>
            <div>
              <strong>{{ heroTitle }}</strong>
              <span>{{ heroSubtitle }}</span>
            </div>
          </div>

          <n-form
            class="quick-form"
            label-placement="left"
            :show-feedback="false"
          >
            <n-input
              v-model:value="relayForm.server_addr"
              class="relay-input"
              placeholder="127.0.0.1:7000"
            />
            <n-input
              v-model:value="relayForm.auth_token"
              class="token-input"
              type="password"
              show-password-on="click"
              :placeholder="t('connection.relayTokenPlaceholder')"
            />
            <n-button
              type="primary"
              :loading="store.isConnecting"
              :disabled="!canConnect || store.isConnecting"
              @click="handlePrimaryConnection"
            >
              {{ primaryButtonLabel }}
            </n-button>
          </n-form>
        </div>

        <n-alert
          v-if="store.lastError"
          class="error-alert"
          type="error"
          :bordered="false"
        >
          {{ store.lastError }}
        </n-alert>
      </section>

      <section class="overview-grid">
        <div class="metrics-strip">
          <article
            v-for="metric in summaryMetrics"
            :key="metric.label"
            class="metric-tile"
          >
            <span>{{ metric.label }}</span>
            <strong>{{ metric.value }}</strong>
            <small>{{ metric.hint }}</small>
          </article>
        </div>

        <n-card
          class="traffic-panel"
          :bordered="false"
        >
          <TrafficFlowChart
            :samples="store.trafficHistory"
            :title="t('traffic.title')"
            :subtitle="t('traffic.subtitle')"
            :upload-label="t('metrics.upload')"
            :download-label="t('metrics.download')"
            :empty-text="t('traffic.empty')"
          />
        </n-card>

        <n-card
          class="recent-log-panel"
          :bordered="false"
        >
          <template #header>
            <div class="panel-title compact">
              <div>
                <strong>{{ t('activity.recentTitle') }}</strong>
                <span>{{ t('activity.recentSubtitle') }}</span>
              </div>
            </div>
          </template>

          <div
            v-if="recentLogs.length > 0"
            class="recent-log-list"
          >
            <article
              v-for="log in recentLogs"
              :key="log.id"
              class="recent-log-item"
            >
              <span
                class="log-level"
                :class="log.level"
              />
              <div>
                <strong>{{ log.title }}</strong>
                <small>{{ formatLogTime(log.created_at) }} · {{ translateLogCategory(log.category) }}</small>
              </div>
            </article>
          </div>
          <n-empty
            v-else
            class="compact-empty"
            :description="t('activity.empty')"
          />
        </n-card>
      </section>
    </template>

    <template v-else>
      <section class="tunnels-layout">
        <n-card
          class="tunnel-panel"
          :bordered="false"
        >
          <template #header>
            <div class="panel-title">
              <div>
                <strong>{{ t('tunnel.title') }}</strong>
                <span>{{ t('tunnel.subtitle') }}</span>
              </div>
              <n-button
                type="primary"
                size="small"
                @click="showForm = !showForm"
              >
                {{ showForm ? t('tunnel.cancel') : t('tunnel.newTunnel') }}
              </n-button>
            </div>
          </template>

          <n-collapse-transition :show="showForm">
            <n-form
              class="tunnel-form"
              label-placement="top"
              :show-feedback="false"
            >
              <n-grid
                :cols="6"
                :x-gap="10"
                :y-gap="10"
                responsive="screen"
              >
                <n-form-item-gi :label="t('tunnel.name')">
                  <n-input v-model:value="form.name" />
                </n-form-item-gi>
                <n-form-item-gi :label="t('tunnel.protocol')">
                  <n-select
                    v-model:value="form.proxy_type"
                    :options="protocolOptions"
                  />
                </n-form-item-gi>
                <n-form-item-gi :label="t('tunnel.localAddress')">
                  <n-input v-model:value="form.local_addr" />
                </n-form-item-gi>
                <n-form-item-gi :label="t('tunnel.localPort')">
                  <n-input-number
                    v-model:value="form.local_port"
                    :min="1"
                    :max="65535"
                  />
                </n-form-item-gi>
                <n-form-item-gi :label="t('tunnel.remotePort')">
                  <n-input-number
                    v-model:value="form.remote_port"
                    :min="0"
                    :max="65535"
                  />
                </n-form-item-gi>
                <n-form-item-gi label=" ">
                  <n-button
                    type="primary"
                    block
                    :disabled="!canCreateTunnel"
                    @click="handleCreate"
                  >
                    {{ t('tunnel.create') }}
                  </n-button>
                </n-form-item-gi>
              </n-grid>
            </n-form>
          </n-collapse-transition>

          <n-empty
            v-if="store.tunnels.length === 0 && !showForm"
            class="empty-state"
            :description="t('tunnel.emptyTitle')"
          >
            <template #extra>
              <span>{{ t('tunnel.emptyText') }}</span>
            </template>
          </n-empty>

          <div
            v-else
            class="tunnel-list"
          >
            <article
              v-for="tunnel in store.tunnels"
              :key="tunnel.id"
              class="tunnel-item"
            >
              <div class="tunnel-main">
                <n-tag
                  round
                  size="small"
                  type="info"
                  :bordered="false"
                >
                  {{ tunnel.proxy_type.toUpperCase() }}
                </n-tag>
                <div>
                  <strong>{{ tunnel.name }}</strong>
                  <span>{{ t('tunnel.localEndpoint') }} {{ tunnel.local_addr }}:{{ tunnel.local_port }}</span>
                </div>
              </div>

              <div class="tunnel-meta">
                <span>{{ t('tunnel.remoteEndpoint') }} :{{ tunnel.remote_port }}</span>
                <n-tag
                  round
                  size="small"
                  :type="getConnectionTypeTagType(tunnel.connection_type)"
                  :bordered="false"
                >
                  {{ translateConnectionType(tunnel.connection_type) }}
                </n-tag>
                <n-tag
                  round
                  size="small"
                  :type="getTunnelTagType(tunnel.status)"
                  :bordered="false"
                >
                  {{ translateStatus(tunnel.status) }}
                </n-tag>
              </div>

              <n-space>
                <n-button
                  v-if="!isTunnelRunning(tunnel.status)"
                  size="small"
                  type="primary"
                  :disabled="!store.isConnected || store.busyTunnelIds.has(tunnel.id)"
                  @click="handleStart(tunnel.id)"
                >
                  {{ t('tunnel.start') }}
                </n-button>
                <n-button
                  v-else
                  size="small"
                  secondary
                  :disabled="store.busyTunnelIds.has(tunnel.id)"
                  @click="handleStop(tunnel.id)"
                >
                  {{ t('tunnel.stop') }}
                </n-button>
                <n-button
                  size="small"
                  type="error"
                  secondary
                  :disabled="store.busyTunnelIds.has(tunnel.id)"
                  @click="handleDelete(tunnel.id)"
                >
                  {{ t('tunnel.delete') }}
                </n-button>
              </n-space>
            </article>
          </div>
        </n-card>

        <n-card
          class="port-panel"
          :bordered="false"
        >
          <LocalPortManager @use-port="handleUsePort" />
        </n-card>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NAlert,
  NButton,
  NCard,
  NCollapseTransition,
  NEmpty,
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NTag,
  type SelectOption,
} from 'naive-ui'
import LocalPortManager from '../components/LocalPortManager.vue'
import TrafficFlowChart from '../components/TrafficFlowChart.vue'
import { useTunnelStore } from '../stores/tunnel'
import type { FavoritePortInfo, LocalPortScanResult } from '../api/app'

withDefaults(
  defineProps<{
    viewMode?: 'overview' | 'tunnels'
  }>(),
  {
    viewMode: 'overview',
  },
)

interface SummaryMetric {
  label: string
  value: string
  hint: string
}

type TagType = 'default' | 'error' | 'success' | 'warning' | 'info'
type PortLike = FavoritePortInfo | LocalPortScanResult

const RECENT_LOG_LIMIT = 6
const store = useTunnelStore()
const { t } = useI18n()
const showForm = ref(false)
const relayForm = ref({
  server_addr: store.serverAddr,
  auth_token: store.authToken,
})
const form = ref({
  name: '',
  proxy_type: 'tcp',
  local_addr: '127.0.0.1',
  local_port: 8080,
  remote_port: 80,
})

const protocolOptions: SelectOption[] = [
  { label: 'TCP', value: 'tcp' },
  { label: 'HTTP', value: 'http' },
]

const statusLabel = computed(() => translateStatus(store.connectionStatus))
const statusTagType = computed<TagType>(() => {
  if (store.connectionStatus === 'connected') return 'success'
  if (store.connectionStatus === 'reconnecting') return 'warning'
  return 'error'
})
const heroTitle = computed(() => (store.isConnected ? t('connection.connectedTitle') : t('connection.readyTitle')))
const heroSubtitle = computed(() =>
  store.isConnected ? t('connection.connectedSubtitle') : t('connection.readySubtitle'),
)
const primaryButtonLabel = computed(() => {
  if (store.isConnecting) return t('connection.connecting')
  return store.isConnected ? t('connection.disconnect') : t('connection.connectNow')
})

const summaryMetrics = computed<SummaryMetric[]>(() => [
  {
    label: t('metrics.upload'),
    value: formatBytes(store.trafficStats.bytes_out),
    hint: t('metrics.outboundTraffic'),
  },
  {
    label: t('metrics.download'),
    value: formatBytes(store.trafficStats.bytes_in),
    hint: t('metrics.inboundTraffic'),
  },
  {
    label: t('metrics.tunnelCount'),
    value: `${store.tunnelCount}`,
    hint: t('metrics.activeRoutes', { count: store.trafficStats.tunnels || 0 }),
  },
])

const recentLogs = computed(() => store.activityLogs.slice(0, RECENT_LOG_LIMIT))
const canConnect = computed(() => relayForm.value.server_addr.trim().length > 0)
const canCreateTunnel = computed(() => {
  return (
    form.value.name.trim().length > 0 &&
    form.value.local_addr.trim().length > 0 &&
    form.value.local_port > 0 &&
    form.value.local_port <= 65535 &&
    form.value.remote_port >= 0 &&
    form.value.remote_port <= 65535
  )
})

// translateStatus 将后端状态字符串映射为当前语言文案。
const translateStatus = (status: string): string => {
  const normalizedStatus = status || 'unknown'
  const key = `status.${normalizedStatus}`
  const translated = t(key)
  return translated === key ? normalizedStatus : translated
}

const translateConnectionType = (connectionType: string): string => {
  const normalizedType = connectionType || 'standby'
  const key = `connectionTypes.${normalizedType}`
  const translated = t(key)
  return translated === key ? normalizedType : translated
}

const translateLogCategory = (category: string): string => {
  const key = `logs.categories.${category}`
  const translated = t(key)
  return translated === key ? category : translated
}

// formatBytes 将累计字节数格式化为桌面端指标展示文本。
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / Math.pow(1024, unitIndex)).toFixed(1)} ${units[unitIndex]}`
}

const formatLogTime = (value: string): string => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--:--'
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const isTunnelRunning = (status: string): boolean => status === 'active' || status === 'running'

const getTunnelTagType = (status: string): TagType => {
  if (status === 'active' || status === 'running') return 'success'
  if (status === 'error') return 'error'
  return 'default'
}

const getConnectionTypeTagType = (connectionType: string): TagType => {
  if (connectionType === 'p2p_direct') return 'success'
  if (connectionType === 'relay') return 'info'
  return 'default'
}

const handlePrimaryConnection = async (): Promise<void> => {
  if (store.isConnected) {
    await handleDisconnect()
    return
  }
  await handleConnect()
}

const handleConnect = async (): Promise<void> => {
  if (!canConnect.value) return
  await store.connectServer(relayForm.value)
}

const handleDisconnect = async (): Promise<void> => {
  await store.disconnectServer()
}

const handleCreate = async (): Promise<void> => {
  if (!canCreateTunnel.value) return
  await store.createTunnel(form.value)
  showForm.value = false
  form.value = { name: '', proxy_type: 'tcp', local_addr: '127.0.0.1', local_port: 8080, remote_port: 80 }
}

const handleUsePort = (port: PortLike): void => {
  form.value = {
    name: port.name || `local-${port.port}`,
    proxy_type: port.protocol || 'tcp',
    local_addr: '127.0.0.1',
    local_port: port.port,
    remote_port: port.port,
  }
  showForm.value = true
}

const handleDelete = async (id: string): Promise<void> => {
  await store.deleteTunnel(id)
}

const handleStart = async (id: string): Promise<void> => {
  await store.startTunnel(id)
}

const handleStop = async (id: string): Promise<void> => {
  await store.stopTunnel(id)
}

// refreshClientState 定期刷新桌面端状态，保持界面与本地代理同步。
const refreshClientState = async (): Promise<void> => {
  await store.refreshStatus()
  await store.loadTunnels()
}

let refreshTimer: ReturnType<typeof setInterval> | undefined

onMounted(async () => {
  await store.loadServerSettings()
  relayForm.value = {
    server_addr: store.serverAddr,
    auth_token: store.authToken,
  }
  await store.loadTunnels()
  await store.loadFavoritePorts()
  await store.loadActivityLogs({ limit: 100 })
  await store.refreshStatus()
  refreshTimer = setInterval(refreshClientState, 3000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style scoped>
.client-dashboard {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.overview-hero,
.traffic-panel,
.recent-log-panel,
.tunnel-panel,
.port-panel {
  border: 1px solid var(--line-soft);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.012)),
    var(--surface-bg);
  box-shadow: 0 14px 34px rgba(0, 0, 0, 0.16);
}

.overview-hero {
  padding: 16px;
  border-radius: 8px;
}

.connection-strip {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(520px, auto);
  align-items: center;
  gap: 18px;
}

.connection-state {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 14px;
}

.connection-state div {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.connection-state strong {
  color: var(--text-main);
  font-size: 18px;
}

.connection-state span:last-child {
  overflow: hidden;
  color: var(--text-dim);
  font-size: 13px;
  line-height: 1.5;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.quick-form {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(160px, 0.85fr) auto;
  align-items: center;
  gap: 10px;
}

.relay-input,
.token-input {
  min-width: 0;
}

.status-dot {
  width: 7px;
  height: 7px;
  display: inline-block;
  margin-right: 6px;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 12px currentColor;
}

.error-alert {
  margin-top: 12px;
}

.overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 0.36fr);
  grid-template-areas:
    'metrics logs'
    'traffic logs';
  gap: 16px;
}

.metrics-strip {
  grid-area: metrics;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.metric-tile {
  min-height: 104px;
  display: grid;
  align-content: center;
  gap: 7px;
  padding: 16px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.66);
}

.metric-tile span {
  color: var(--text-dim);
  font-size: 12px;
}

.metric-tile strong {
  color: var(--text-main);
  font-size: 25px;
  line-height: 1.1;
}

.metric-tile small {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.traffic-panel {
  grid-area: traffic;
}

.recent-log-panel {
  grid-area: logs;
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

.panel-title.compact {
  display: block;
}

.recent-log-list {
  display: grid;
  gap: 10px;
}

.recent-log-item {
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px solid rgba(168, 169, 169, 0.1);
}

.recent-log-item:last-child {
  border-bottom: 0;
}

.recent-log-item div {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.recent-log-item strong {
  overflow: hidden;
  color: var(--text-main);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.recent-log-item small {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-level {
  width: 8px;
  height: 28px;
  border-radius: 999px;
  background: var(--nex-cyan);
}

.log-level.warning {
  background: var(--warning);
}

.log-level.error {
  background: var(--danger);
}

.compact-empty {
  min-height: 180px;
  display: grid;
  place-items: center;
}

.tunnels-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(360px, 0.42fr);
  gap: 16px;
}

.tunnel-form {
  margin-bottom: 14px;
  padding: 14px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.68);
}

.empty-state {
  min-height: 220px;
  display: grid;
  place-items: center;
}

.tunnel-list {
  display: grid;
  gap: 10px;
}

.tunnel-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 14px;
  padding: 12px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.56);
  transition: transform var(--duration-small) var(--ease-standard), opacity var(--duration-small) var(--ease-standard);
}

.tunnel-item:hover {
  transform: translateY(-1px);
}

.tunnel-main {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
}

.tunnel-main div {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.tunnel-main strong {
  overflow: hidden;
  color: var(--text-main);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tunnel-main span,
.tunnel-meta span {
  color: var(--text-dim);
  font-size: 12px;
}

.tunnel-meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

@media (max-width: 1280px) {
  .connection-strip,
  .overview-grid,
  .tunnels-layout {
    grid-template-columns: 1fr;
    grid-template-areas:
      'metrics'
      'traffic'
      'logs';
  }

  .quick-form {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  }
}

@media (max-width: 1180px) {
  .metrics-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .tunnel-item {
    grid-template-columns: 1fr;
  }

  .tunnel-meta {
    flex-wrap: wrap;
  }
}

@media (prefers-reduced-motion: reduce) {
  .tunnel-item {
    transition: none;
  }

  .tunnel-item:hover {
    transform: none;
  }
}
</style>
