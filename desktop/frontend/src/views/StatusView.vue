<template>
  <section class="client-dashboard">
    <div class="dashboard-grid">
      <n-card
        class="connect-panel"
        :bordered="false"
      >
        <div class="connect-layout">
          <div class="connect-copy">
            <n-tag
              round
              :type="statusTagType"
              :bordered="false"
            >
              <span class="status-dot" />
              {{ statusLabel }}
            </n-tag>
            <h2>{{ heroTitle }}</h2>
            <p>{{ heroSubtitle }}</p>

            <div class="relay-summary">
              <span>{{ t('connection.currentRelay') }}</span>
              <strong>{{ relayForm.server_addr || t('connection.notConfigured') }}</strong>
            </div>
          </div>

          <div class="connect-action">
            <button
              class="connect-button"
              :class="{ active: store.isConnected }"
              type="button"
              :disabled="!canConnect || store.isConnecting"
              @click="handlePrimaryConnection"
            >
              <span>{{ primaryButtonLabel }}</span>
            </button>
            <span class="connect-caption">{{ store.isConnected ? statusLabel : t('status.idle') }}</span>
          </div>
        </div>
      </n-card>

      <n-card
        class="settings-panel"
        :bordered="false"
      >
        <template #header>
          {{ t('connection.quickConnect') }}
        </template>

        <n-form
          label-placement="top"
          :show-feedback="false"
        >
          <n-grid
            :cols="2"
            :x-gap="12"
            :y-gap="12"
            responsive="screen"
          >
            <n-form-item-gi :label="t('connection.relayAddress')">
              <n-input
                v-model:value="relayForm.server_addr"
                placeholder="127.0.0.1:7000"
              />
            </n-form-item-gi>
            <n-form-item-gi :label="t('connection.relayToken')">
              <n-input
                v-model:value="relayForm.auth_token"
                type="password"
                show-password-on="click"
                :placeholder="t('connection.relayTokenPlaceholder')"
              />
            </n-form-item-gi>
          </n-grid>
        </n-form>

        <n-alert
          v-if="store.lastError"
          class="error-alert"
          type="error"
          :bordered="false"
        >
          {{ store.lastError }}
        </n-alert>
      </n-card>
    </div>

    <div class="stats-grid">
      <n-card
        v-for="metric in summaryMetrics"
        :key="metric.label"
        class="stat-card"
        :bordered="false"
      >
        <n-statistic
          :label="metric.label"
          :value="metric.value"
        />
        <span>{{ metric.hint }}</span>
      </n-card>
    </div>

    <div class="detail-grid">
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

      <div class="side-stack">
        <n-card
          class="capability-card"
          :bordered="false"
        >
          <template #header>
            <div class="panel-title compact">
              <div>
                <strong>{{ t('capability.title') }}</strong>
                <span>{{ t('capability.subtitle') }}</span>
              </div>
            </div>
          </template>

          <div class="capability-list">
            <div
              v-for="item in capabilityItems"
              :key="item.name"
              class="capability-item"
            >
              <div>
                <strong>{{ item.name }}</strong>
                <span>{{ item.detail }}</span>
              </div>
              <n-tag
                round
                size="small"
                :type="item.active ? 'success' : 'info'"
                :bordered="false"
              >
                {{ item.state }}
              </n-tag>
            </div>
          </div>
        </n-card>

        <n-card
          class="log-panel"
          :bordered="false"
        >
          <template #header>
            <div class="panel-title compact">
              <div>
                <strong>{{ t('activity.title') }}</strong>
                <span>{{ t('activity.subtitle') }}</span>
              </div>
            </div>
          </template>

          <div class="log-terminal">
            <div
              v-for="event in activityEvents"
              :key="event.title"
              class="log-line"
            >
              <span class="log-time">[{{ event.time }}]</span>
              <span>{{ event.title }} - {{ event.detail }}</span>
            </div>
          </div>
        </n-card>
      </div>
    </div>
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
  NStatistic,
  NTag,
  type SelectOption,
} from 'naive-ui'
import { useTunnelStore } from '../stores/tunnel'

interface CapabilityItem {
  name: string
  detail: string
  state: string
  active: boolean
}

interface ActivityEvent {
  time: string
  title: string
  detail: string
}

type TagType = 'default' | 'error' | 'success' | 'warning' | 'info'

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

const summaryMetrics = computed(() => [
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
    label: t('metrics.latency'),
    value: '-- ms',
    hint: t('metrics.realtimePending'),
  },
  {
    label: t('metrics.tunnelCount'),
    value: `${store.tunnelCount}`,
    hint: t('metrics.activeRoutes', { count: store.trafficStats.tunnels || 0 }),
  },
])

const capabilityItems = computed<CapabilityItem[]>(() => [
  {
    name: t('capability.p2p'),
    detail: t('capability.p2pDetail'),
    state: store.p2pStatus || t('status.planned'),
    active: Boolean(store.p2pStatus),
  },
  {
    name: t('capability.nat'),
    detail: t('capability.natDetail'),
    state: store.natType || t('status.waiting'),
    active: Boolean(store.natType),
  },
  {
    name: t('capability.quic'),
    detail: t('capability.quicDetail'),
    state: t('capability.comingSoon'),
    active: false,
  },
])

const activityEvents = computed<ActivityEvent[]>(() => [
  {
    time: 'Now',
    title: t('activity.ready'),
    detail: t('activity.readyDetail'),
  },
  {
    time: 'Next',
    title: t('activity.migration'),
    detail: t('activity.migrationDetail'),
  },
  {
    time: 'Later',
    title: t('activity.security'),
    detail: t('activity.securityDetail'),
  },
])

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

// formatBytes 将累计字节数格式化为桌面端指标展示文本。
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / Math.pow(1024, unitIndex)).toFixed(1)} ${units[unitIndex]}`
}

const isTunnelRunning = (status: string): boolean => status === 'active' || status === 'running'

const getTunnelTagType = (status: string): TagType => {
  if (status === 'active' || status === 'running') return 'success'
  if (status === 'error') return 'error'
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
  await store.loadTunnels()
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
  gap: 16px;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(340px, 0.85fr);
  gap: 16px;
}

.connect-panel,
.settings-panel,
.stat-card,
.tunnel-panel,
.capability-card,
.log-panel {
  border: 1px solid var(--line-soft);
  background: var(--surface-bg);
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.22);
}

.connect-layout {
  min-height: 260px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 260px;
  align-items: center;
  gap: 26px;
}

.connect-copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 14px;
}

.connect-copy h2 {
  max-width: 620px;
  margin: 0;
  color: var(--text-main);
  font-size: 36px;
  line-height: 1.12;
}

.connect-copy p {
  max-width: 640px;
  margin: 0;
  color: var(--text-dim);
  font-size: 14px;
  line-height: 1.75;
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

.relay-summary {
  display: grid;
  gap: 4px;
  min-width: 260px;
  padding: 12px 14px;
  border: 1px solid var(--line-cyan);
  border-radius: 12px;
  background: rgba(0, 255, 255, 0.06);
}

.relay-summary span {
  color: var(--text-dim);
  font-size: 12px;
}

.relay-summary strong {
  color: var(--nex-cyan);
  font-family: Consolas, 'SFMono-Regular', monospace;
  font-size: 13px;
}

.connect-action {
  display: grid;
  place-items: center;
  gap: 12px;
}

.connect-button {
  width: 202px;
  height: 202px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 50%;
  background:
    radial-gradient(circle at 35% 28%, rgba(255, 255, 255, 0.45), transparent 22%),
    var(--accent-gradient);
  color: white;
  cursor: pointer;
  font-size: 20px;
  font-weight: 800;
  letter-spacing: 0;
  box-shadow:
    0 0 34px rgba(138, 43, 226, 0.42),
    inset 0 0 24px rgba(255, 255, 255, 0.18);
  transition:
    transform 180ms ease,
    box-shadow 180ms ease,
    filter 180ms ease;
}

.connect-button:hover:not(:disabled) {
  transform: scale(1.035);
  box-shadow:
    0 0 52px rgba(0, 255, 255, 0.42),
    inset 0 0 24px rgba(255, 255, 255, 0.2);
}

.connect-button:disabled {
  cursor: not-allowed;
  filter: grayscale(0.45);
  opacity: 0.64;
}

.connect-button.active {
  background:
    radial-gradient(circle at 35% 28%, rgba(255, 255, 255, 0.35), transparent 22%),
    linear-gradient(135deg, #ef4444, #8a2be2);
}

.connect-caption {
  color: var(--text-dim);
  font-size: 13px;
}

.error-alert {
  margin-top: 12px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.stat-card span {
  display: block;
  margin-top: 8px;
  color: var(--text-dim);
  font-size: 12px;
}

.detail-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(340px, 0.65fr);
  gap: 16px;
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

.tunnel-form {
  margin-bottom: 14px;
  padding: 14px;
  border: 1px solid var(--line-soft);
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.62);
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
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.58);
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

.side-stack {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.capability-list {
  display: grid;
  gap: 10px;
}

.capability-item {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--line-soft);
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.55);
}

.capability-item div {
  display: grid;
  gap: 4px;
}

.capability-item strong {
  color: var(--text-main);
  font-size: 13px;
}

.capability-item span {
  color: var(--text-dim);
  font-size: 12px;
  line-height: 1.5;
}

.log-terminal {
  height: 184px;
  overflow: auto;
  padding: 14px;
  border: 1px solid rgba(16, 185, 129, 0.2);
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.72);
  color: #10b981;
  font-family: Consolas, 'SFMono-Regular', monospace;
  font-size: 12px;
}

.log-line {
  display: flex;
  gap: 10px;
  margin-bottom: 7px;
  line-height: 1.5;
}

.log-time {
  flex: 0 0 auto;
  color: #64748b;
}

@media (max-width: 1180px) {
  .dashboard-grid,
  .detail-grid {
    grid-template-columns: 1fr;
  }

  .stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
