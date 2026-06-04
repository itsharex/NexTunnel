<template>
  <section
    id="client-console"
    class="status-view"
  >
    <div class="hero-panel">
      <div class="hero-copy">
        <div
          class="status-indicator"
          :class="store.connectionStatus"
        >
          <span class="dot" />
          <span>{{ statusLabel }}</span>
        </div>
        <h2>Build secure tunnels from one focused desktop console.</h2>
        <p>
          Relay connection, local tunnel lifecycle, traffic telemetry and future P2P capability are organized in a
          single operational workspace.
        </p>
      </div>

      <div
        class="topology-card"
        aria-label="Client topology preview"
      >
        <div class="topology-line line-a" />
        <div class="topology-line line-b" />
        <div class="topology-line line-c" />
        <span class="topology-node node-local">Local</span>
        <span class="topology-node node-relay">Relay</span>
        <span class="topology-node node-edge">Edge</span>
      </div>
    </div>

    <div class="metric-grid">
      <article
        v-for="metric in summaryMetrics"
        :key="metric.label"
        class="metric-card"
      >
        <span class="metric-label">{{ metric.label }}</span>
        <strong>{{ metric.value }}</strong>
        <span class="metric-hint">{{ metric.hint }}</span>
      </article>
    </div>

    <div class="workspace-grid">
      <section class="panel connection-panel">
        <div class="panel-header">
          <div>
            <span class="panel-kicker">Relay Server</span>
            <h3>Connection Control</h3>
          </div>
          <span class="connection-address">{{ relayForm.server_addr || 'not configured' }}</span>
        </div>

        <div class="server-form">
          <label class="field">
            <span>Address</span>
            <input
              v-model.trim="relayForm.server_addr"
              class="input"
              placeholder="127.0.0.1:7000"
            >
          </label>
          <label class="field">
            <span>Token</span>
            <input
              v-model="relayForm.auth_token"
              class="input"
              type="password"
              autocomplete="off"
            >
          </label>
          <div class="button-row">
            <button
              class="btn"
              :disabled="!canConnect || store.isConnecting || store.isConnected"
              @click="handleConnect"
            >
              {{ store.isConnecting ? 'Connecting' : 'Connect' }}
            </button>
            <button
              class="btn btn-secondary"
              :disabled="!store.isConnected"
              @click="handleDisconnect"
            >
              Disconnect
            </button>
          </div>
        </div>

        <p
          v-if="store.lastError"
          class="error-text"
        >
          {{ store.lastError }}
        </p>
      </section>

      <section class="panel capability-panel">
        <div class="panel-header">
          <div>
            <span class="panel-kicker">Path Intelligence</span>
            <h3>P2P and NAT Preview</h3>
          </div>
          <span class="planned-pill">Planned</span>
        </div>

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
            <span
              class="capability-state"
              :class="item.stateClass"
            >{{ item.state }}</span>
          </div>
        </div>
      </section>
    </div>

    <section class="panel tunnel-panel">
      <div class="panel-header">
        <div>
          <span class="panel-kicker">Tunnels</span>
          <h3>Local Proxy Routes</h3>
        </div>
        <button
          class="btn btn-sm"
          @click="showForm = !showForm"
        >
          {{ showForm ? 'Cancel' : 'New Tunnel' }}
        </button>
      </div>

      <div
        v-if="showForm"
        class="tunnel-form"
      >
        <input
          v-model.trim="form.name"
          placeholder="Name"
          class="input"
        >
        <select
          v-model="form.proxy_type"
          class="input"
        >
          <option value="tcp">
            TCP
          </option>
          <option value="http">
            HTTP
          </option>
        </select>
        <input
          v-model.trim="form.local_addr"
          placeholder="Local Address"
          class="input"
        >
        <input
          v-model.number="form.local_port"
          type="number"
          min="1"
          max="65535"
          placeholder="Local Port"
          class="input"
        >
        <input
          v-model.number="form.remote_port"
          type="number"
          min="0"
          max="65535"
          placeholder="Remote Port"
          class="input"
        >
        <button
          class="btn"
          :disabled="!canCreateTunnel"
          @click="handleCreate"
        >
          Create
        </button>
      </div>

      <div
        v-if="store.tunnels.length === 0 && !showForm"
        class="empty-state"
      >
        <div class="empty-orbit" />
        <p>No tunnels configured.</p>
        <span>Create a tunnel after connecting to a relay server.</span>
      </div>

      <div
        v-else
        class="tunnel-list"
      >
        <article
          v-for="tunnel in store.tunnels"
          :key="tunnel.id"
          class="tunnel-item"
        >
          <div class="tunnel-info">
            <span class="tunnel-name">{{ tunnel.name }}</span>
            <span class="tunnel-type">{{ tunnel.proxy_type.toUpperCase() }}</span>
            <span class="tunnel-addr">{{ tunnel.local_addr }}:{{ tunnel.local_port }} to :{{ tunnel.remote_port }}</span>
          </div>
          <div class="tunnel-actions">
            <span
              class="tunnel-status"
              :class="tunnel.status"
            >{{ tunnel.status }}</span>
            <button
              v-if="!isTunnelRunning(tunnel.status)"
              class="btn btn-sm"
              :disabled="!store.isConnected || store.busyTunnelIds.has(tunnel.id)"
              @click="handleStart(tunnel.id)"
            >
              Start
            </button>
            <button
              v-else
              class="btn btn-sm btn-secondary"
              :disabled="store.busyTunnelIds.has(tunnel.id)"
              @click="handleStop(tunnel.id)"
            >
              Stop
            </button>
            <button
              class="btn btn-sm btn-danger"
              :disabled="store.busyTunnelIds.has(tunnel.id)"
              @click="handleDelete(tunnel.id)"
            >
              Delete
            </button>
          </div>
        </article>
      </div>
    </section>

    <section class="panel activity-panel">
      <div class="panel-header">
        <div>
          <span class="panel-kicker">Activity</span>
          <h3>Operational Timeline</h3>
        </div>
        <span class="planned-pill">Live stream planned</span>
      </div>

      <div class="activity-list">
        <div
          v-for="event in activityEvents"
          :key="event.title"
          class="activity-item"
        >
          <span class="activity-time">{{ event.time }}</span>
          <div>
            <strong>{{ event.title }}</strong>
            <p>{{ event.detail }}</p>
          </div>
        </div>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useTunnelStore } from '../stores/tunnel'

interface CapabilityItem {
  name: string
  detail: string
  state: string
  stateClass: string
}

interface ActivityEvent {
  time: string
  title: string
  detail: string
}

const store = useTunnelStore()
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

const statusLabel = computed(() => {
  switch (store.connectionStatus) {
    case 'connected':
      return 'Connected'
    case 'reconnecting':
      return 'Reconnecting'
    default:
      return 'Disconnected'
  }
})

const summaryMetrics = computed(() => [
  {
    label: 'Tunnels',
    value: `${store.tunnelCount}`,
    hint: `${store.trafficStats.tunnels || 0} active routes reported`,
  },
  {
    label: 'Traffic In',
    value: formatBytes(store.trafficStats.bytes_in),
    hint: 'Inbound relay traffic',
  },
  {
    label: 'Traffic Out',
    value: formatBytes(store.trafficStats.bytes_out),
    hint: 'Outbound relay traffic',
  },
  {
    label: 'NAT Type',
    value: store.natType || 'Unknown',
    hint: store.p2pStatus || 'Detection pending',
  },
])

const capabilityItems = computed<CapabilityItem[]>(() => [
  {
    name: 'P2P Path',
    detail: 'Scheduler integration will switch between direct and relay paths.',
    state: store.p2pStatus || 'Planned',
    stateClass: store.p2pStatus ? 'active' : 'planned',
  },
  {
    name: 'NAT Detection',
    detail: 'Runtime NAT classification is surfaced when the local agent reports it.',
    state: store.natType || 'Waiting',
    stateClass: store.natType ? 'active' : 'planned',
  },
  {
    name: 'QUIC Relay',
    detail: 'Stream migration controls remain a server-side roadmap item.',
    state: 'Coming soon',
    stateClass: 'planned',
  },
])

const activityEvents: ActivityEvent[] = [
  {
    time: 'Now',
    title: 'Client console ready',
    detail: 'Local configuration and tunnel controls are available in the desktop shell.',
  },
  {
    time: 'Next',
    title: 'Path migration hooks',
    detail: 'Future releases can connect the scheduler, QUIC relay and P2P engine here.',
  },
  {
    time: 'Later',
    title: 'Security posture feed',
    detail: 'Token expiry, ACL updates and certificate health can be surfaced in this stream.',
  },
]

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

// formatBytes 将后端累计字节数格式化为紧凑展示文本。
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / Math.pow(1024, unitIndex)).toFixed(1)} ${units[unitIndex]}`
}

const isTunnelRunning = (status: string): boolean => status === 'active' || status === 'running'

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

// refreshClientState 定期刷新桌面端状态，避免视觉重构改变原有数据流。
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
.status-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.hero-panel,
.panel,
.metric-card {
  min-width: 0;
  border: 1px solid var(--console-border-muted);
  border-radius: var(--radius-panel);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.055), rgba(255, 255, 255, 0.025)),
    rgba(10, 25, 43, 0.92);
  box-shadow: var(--shadow-panel);
}

.hero-panel {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 420px);
  gap: 22px;
  min-height: 260px;
  padding: 24px;
  overflow: hidden;
}

.hero-copy {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 16px;
  min-width: 0;
  max-width: 680px;
}

.hero-copy h2 {
  max-width: 640px;
  color: var(--text-primary);
  font-size: 38px;
  line-height: 1.05;
  font-weight: 820;
  overflow-wrap: anywhere;
}

.hero-copy p {
  max-width: 58ch;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.7;
}

.status-indicator {
  width: fit-content;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: 1px solid rgba(255, 92, 122, 0.28);
  border-radius: 999px;
  background: rgba(255, 92, 122, 0.08);
  color: var(--status-danger);
  padding: 8px 11px;
  font-size: 12px;
  font-weight: 700;
}

.status-indicator .dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 14px currentColor;
}

.status-indicator.connected {
  border-color: rgba(36, 230, 161, 0.3);
  background: rgba(36, 230, 161, 0.09);
  color: var(--status-success);
}

.status-indicator.reconnecting {
  border-color: rgba(255, 200, 87, 0.3);
  background: rgba(255, 200, 87, 0.1);
  color: var(--status-warning);
}

.topology-card {
  position: relative;
  width: 100%;
  min-width: 0;
  min-height: 212px;
  border: 1px solid rgba(0, 255, 255, 0.12);
  border-radius: var(--radius-panel);
  background:
    radial-gradient(circle at 50% 48%, rgba(0, 255, 255, 0.16), transparent 24%),
    linear-gradient(120deg, rgba(0, 0, 255, 0.12), rgba(138, 43, 226, 0.12)),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.045) 0 1px, transparent 1px 42px),
    repeating-linear-gradient(0deg, rgba(255, 255, 255, 0.035) 0 1px, transparent 1px 42px);
  overflow: hidden;
}

.topology-card::before {
  content: '';
  position: absolute;
  inset: 30px;
  border-radius: 50%;
  border: 1px solid rgba(0, 255, 255, 0.14);
}

.topology-line {
  position: absolute;
  height: 2px;
  border-radius: 999px;
  background: linear-gradient(90deg, transparent, var(--nex-cyan), var(--tunnel-violet), transparent);
  transform-origin: left center;
}

.line-a {
  width: 62%;
  left: 18%;
  top: 48%;
  transform: rotate(-18deg);
}

.line-b {
  width: 48%;
  left: 28%;
  top: 58%;
  transform: rotate(19deg);
}

.line-c {
  width: 44%;
  left: 18%;
  top: 35%;
  transform: rotate(24deg);
}

.topology-node {
  position: absolute;
  min-width: 56px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(0, 255, 255, 0.26);
  border-radius: 999px;
  background: rgba(8, 22, 39, 0.92);
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 700;
  padding: 9px 12px;
  box-shadow: 0 0 30px rgba(0, 255, 255, 0.14);
}

.node-local {
  left: 8%;
  bottom: 22%;
}

.node-relay {
  left: 45%;
  top: 35%;
}

.node-edge {
  right: 8%;
  top: 20%;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.metric-card {
  display: flex;
  flex-direction: column;
  gap: 7px;
  min-height: 116px;
  padding: 16px;
}

.metric-label,
.panel-kicker {
  color: var(--nex-cyan);
  font-size: 11px;
  font-weight: 750;
}

.metric-card strong {
  color: var(--text-primary);
  font-size: 25px;
  line-height: 1.1;
}

.metric-hint {
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.workspace-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(320px, 0.85fr);
  gap: 16px;
}

.panel {
  padding: 18px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
  margin-bottom: 16px;
}

.panel-header h3 {
  margin-top: 5px;
  color: var(--text-primary);
  font-size: 18px;
  line-height: 1.2;
}

.connection-address {
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  text-align: right;
  word-break: break-all;
}

.server-form {
  display: grid;
  grid-template-columns: minmax(180px, 1.3fr) minmax(140px, 0.9fr) auto;
  gap: 10px;
  align-items: end;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 7px;
  color: var(--text-secondary);
  font-size: 12px;
}

.input {
  width: 100%;
  min-height: 40px;
  border: 1px solid rgba(168, 169, 169, 0.24);
  border-radius: var(--radius-panel);
  background: rgba(255, 255, 255, 0.055);
  color: var(--text-primary);
  outline: none;
  padding: 9px 12px;
}

.input::placeholder {
  color: rgba(159, 178, 199, 0.64);
}

.input:focus {
  border-color: rgba(0, 255, 255, 0.68);
  box-shadow: 0 0 0 3px rgba(0, 255, 255, 0.08);
}

.button-row {
  display: flex;
  gap: 8px;
}

.btn {
  min-height: 40px;
  border: 1px solid rgba(0, 255, 255, 0.24);
  border-radius: var(--radius-panel);
  background: linear-gradient(90deg, var(--nex-cyan), var(--tunnel-violet));
  color: #04111f;
  cursor: pointer;
  font-size: 13px;
  font-weight: 800;
  padding: 9px 15px;
  white-space: nowrap;
}

.btn:hover:not(:disabled) {
  box-shadow: 0 8px 24px rgba(0, 255, 255, 0.16);
}

.btn:disabled {
  cursor: not-allowed;
  filter: grayscale(0.4);
  opacity: 0.45;
}

.btn-sm {
  min-height: 32px;
  padding: 6px 12px;
  font-size: 12px;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-primary);
}

.btn-danger {
  border-color: rgba(255, 92, 122, 0.24);
  background: rgba(255, 92, 122, 0.14);
  color: #ffdce4;
}

.error-text {
  margin-top: 12px;
  color: #ff9caf;
  font-size: 13px;
}

.planned-pill,
.capability-state {
  border: 1px solid rgba(138, 43, 226, 0.28);
  border-radius: 999px;
  background: rgba(138, 43, 226, 0.1);
  color: #d5b7ff;
  font-size: 11px;
  line-height: 1;
  padding: 7px 9px;
  white-space: nowrap;
}

.capability-list,
.activity-list,
.tunnel-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.capability-item,
.activity-item,
.tunnel-item {
  border: 1px solid rgba(168, 169, 169, 0.14);
  border-radius: var(--radius-panel);
  background: rgba(255, 255, 255, 0.035);
}

.capability-item {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  padding: 13px;
}

.capability-item div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.capability-item strong {
  color: var(--text-primary);
  font-size: 13px;
}

.capability-item span {
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.capability-state.active {
  border-color: rgba(36, 230, 161, 0.28);
  background: rgba(36, 230, 161, 0.1);
  color: var(--status-success);
}

.tunnel-form {
  display: grid;
  grid-template-columns: minmax(150px, 1fr) 110px minmax(150px, 1fr) 130px 130px auto;
  gap: 8px;
  margin-bottom: 12px;
}

.empty-state {
  display: grid;
  place-items: center;
  gap: 8px;
  min-height: 190px;
  border: 1px dashed rgba(0, 255, 255, 0.2);
  border-radius: var(--radius-panel);
  color: var(--text-secondary);
  text-align: center;
}

.empty-state p {
  color: var(--text-primary);
  font-weight: 700;
}

.empty-state span {
  max-width: 280px;
  color: var(--text-muted);
  font-size: 12px;
}

.empty-orbit {
  width: 54px;
  height: 54px;
  border: 1px solid rgba(0, 255, 255, 0.24);
  border-radius: 50%;
  background:
    radial-gradient(circle at center, rgba(0, 255, 255, 0.34) 0 5px, transparent 6px),
    radial-gradient(circle at 72% 30%, rgba(138, 43, 226, 0.8) 0 4px, transparent 5px);
}

.tunnel-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 13px;
}

.tunnel-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.tunnel-name {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 800;
}

.tunnel-type {
  border-radius: 6px;
  background: rgba(0, 255, 255, 0.1);
  color: var(--nex-cyan);
  font-size: 11px;
  font-weight: 800;
  padding: 4px 7px;
}

.tunnel-addr {
  min-width: 0;
  overflow: hidden;
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tunnel-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.tunnel-status {
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-secondary);
  font-size: 12px;
  padding: 5px 8px;
}

.tunnel-status.active,
.tunnel-status.running {
  color: var(--status-success);
}

.tunnel-status.error {
  color: var(--status-danger);
}

.activity-item {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr);
  gap: 12px;
  padding: 12px;
}

.activity-time {
  color: var(--nex-cyan);
  font-family: ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
}

.activity-item strong {
  color: var(--text-primary);
  font-size: 13px;
}

.activity-item p {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.5;
}

@media (max-width: 1180px) {
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .workspace-grid,
  .hero-panel {
    grid-template-columns: 1fr;
  }

  .tunnel-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .hero-panel,
  .panel {
    padding: 14px;
  }

  .hero-copy h2 {
    font-size: 26px;
    line-height: 1.12;
  }

  .topology-card {
    min-height: 190px;
  }

  .metric-grid,
  .server-form,
  .tunnel-form {
    grid-template-columns: 1fr;
  }

  .button-row,
  .tunnel-actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .tunnel-item {
    align-items: flex-start;
    flex-direction: column;
  }

  .tunnel-info {
    width: 100%;
    flex-wrap: wrap;
  }

  .activity-item {
    grid-template-columns: 1fr;
  }
}
</style>
