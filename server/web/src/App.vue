<template>
  <div class="server-shell">
    <div
      class="network-background"
      aria-hidden="true"
    />

    <aside class="console-sidebar">
      <div class="brand-block">
        <div class="brand-mark">
          NT
        </div>
        <div>
          <p class="brand-name">
            NexTunnel
          </p>
          <p class="brand-caption">
            Server Console
          </p>
        </div>
      </div>

      <nav
        class="sidebar-nav"
        aria-label="Server navigation"
      >
        <a
          class="nav-item active"
          href="#overview"
        >
          <span class="nav-code">01</span>
          <span>Overview</span>
        </a>
        <a
          class="nav-item"
          href="#relays"
        >
          <span class="nav-code">02</span>
          <span>Relay Nodes</span>
        </a>
        <span class="nav-item disabled">
          <span class="nav-code">03</span>
          <span>Access Control</span>
          <span class="nav-badge">Coming soon</span>
        </span>
        <span class="nav-item disabled">
          <span class="nav-code">04</span>
          <span>Audit Stream</span>
          <span class="nav-badge">Coming soon</span>
        </span>
      </nav>

      <div class="sidebar-status">
        <span class="status-dot" />
        <div>
          <strong>Control Plane</strong>
          <p>Visual prototype only</p>
        </div>
      </div>
    </aside>

    <main class="console-main">
      <header class="console-header">
        <div>
          <p class="header-kicker">
            Server Management Web
          </p>
          <h1>Global relay operations and control-plane readiness.</h1>
        </div>
        <div class="header-actions">
          <button
            class="btn btn-secondary"
            disabled
          >
            Sync API
          </button>
          <button
            class="btn"
            disabled
          >
            Deploy Relay
          </button>
        </div>
      </header>

      <section
        id="overview"
        class="overview-grid"
      >
        <article
          v-for="metric in metrics"
          :key="metric.label"
          class="metric-card"
        >
          <span>{{ metric.label }}</span>
          <strong>{{ metric.value }}</strong>
          <p>{{ metric.detail }}</p>
        </article>
      </section>

      <section class="map-panel">
        <div class="panel-header">
          <div>
            <span class="panel-kicker">Topology</span>
            <h2>Relay mesh preview</h2>
          </div>
          <span class="planned-pill">API wiring planned</span>
        </div>

        <div
          class="world-map"
          aria-label="Global relay topology preview"
        >
          <div class="route route-primary" />
          <div class="route route-secondary" />
          <div class="route route-tertiary" />
          <span
            v-for="node in topologyNodes"
            :key="node.name"
            class="map-node"
            :class="node.className"
          >
            {{ node.name }}
          </span>
        </div>
      </section>

      <section
        id="relays"
        class="content-grid"
      >
        <div class="panel relay-panel">
          <div class="panel-header">
            <div>
              <span class="panel-kicker">Relays</span>
              <h2>Node fleet</h2>
            </div>
            <span class="planned-pill">Static data</span>
          </div>

          <div class="relay-list">
            <article
              v-for="node in relayNodes"
              :key="node.id"
              class="relay-item"
            >
              <div>
                <strong>{{ node.name }}</strong>
                <p>{{ node.region }} · {{ node.endpoint }}</p>
              </div>
              <div class="relay-stats">
                <span>{{ node.clients }} clients</span>
                <span>{{ node.throughput }}</span>
              </div>
              <span
                class="node-state"
                :class="node.stateClass"
              >{{ node.state }}</span>
            </article>
          </div>
        </div>

        <div class="panel health-panel">
          <div class="panel-header">
            <div>
              <span class="panel-kicker">Control Plane</span>
              <h2>Service readiness</h2>
            </div>
          </div>

          <div class="service-list">
            <div
              v-for="service in services"
              :key="service.name"
              class="service-item"
            >
              <span
                class="service-dot"
                :class="service.stateClass"
              />
              <div>
                <strong>{{ service.name }}</strong>
                <p>{{ service.detail }}</p>
              </div>
              <span class="service-state">{{ service.state }}</span>
            </div>
          </div>
        </div>
      </section>

      <section class="content-grid lower-grid">
        <div class="panel acl-panel">
          <div class="panel-header">
            <div>
              <span class="panel-kicker">Security</span>
              <h2>ACL and token posture</h2>
            </div>
            <button
              class="btn btn-secondary btn-sm"
              disabled
            >
              Configure
            </button>
          </div>

          <div class="acl-summary">
            <div
              v-for="item in aclItems"
              :key="item.label"
              class="acl-item"
            >
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
              <p>{{ item.detail }}</p>
            </div>
          </div>
        </div>

        <div class="panel alert-panel">
          <div class="panel-header">
            <div>
              <span class="panel-kicker">Alerts</span>
              <h2>Operational timeline</h2>
            </div>
            <span class="planned-pill">Read-only</span>
          </div>

          <div class="alert-list">
            <div
              v-for="alert in alerts"
              :key="alert.title"
              class="alert-item"
            >
              <span class="alert-time">{{ alert.time }}</span>
              <div>
                <strong>{{ alert.title }}</strong>
                <p>{{ alert.detail }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
interface MetricItem {
  label: string
  value: string
  detail: string
}

interface TopologyNode {
  name: string
  className: string
}

interface RelayNode {
  id: string
  name: string
  region: string
  endpoint: string
  clients: number
  throughput: string
  state: string
  stateClass: string
}

interface ServiceItem {
  name: string
  detail: string
  state: string
  stateClass: string
}

interface SummaryItem {
  label: string
  value: string
  detail: string
}

interface AlertItem {
  time: string
  title: string
  detail: string
}

const metrics: MetricItem[] = [
  { label: 'Relay nodes', value: '8', detail: 'Static preview across four regions' },
  { label: 'Connected clients', value: '1,284', detail: 'Mocked active sessions' },
  { label: 'Data throughput', value: '12.8 Gbps', detail: 'Aggregated relay traffic view' },
  { label: 'Open alerts', value: '3', detail: 'No live alert stream connected' },
]

const topologyNodes: TopologyNode[] = [
  { name: 'SFO', className: 'node-sfo' },
  { name: 'FRA', className: 'node-fra' },
  { name: 'SIN', className: 'node-sin' },
  { name: 'NRT', className: 'node-nrt' },
  { name: 'GRU', className: 'node-gru' },
]

const relayNodes: RelayNode[] = [
  {
    id: 'relay-sfo-01',
    name: 'relay-sfo-01',
    region: 'US West',
    endpoint: 'sfo.relay.nextunnel.local:7000',
    clients: 312,
    throughput: '3.1 Gbps',
    state: 'Healthy',
    stateClass: 'healthy',
  },
  {
    id: 'relay-fra-02',
    name: 'relay-fra-02',
    region: 'EU Central',
    endpoint: 'fra.relay.nextunnel.local:7000',
    clients: 418,
    throughput: '4.4 Gbps',
    state: 'Healthy',
    stateClass: 'healthy',
  },
  {
    id: 'relay-sin-01',
    name: 'relay-sin-01',
    region: 'AP Southeast',
    endpoint: 'sin.relay.nextunnel.local:7000',
    clients: 276,
    throughput: '2.8 Gbps',
    state: 'Degraded',
    stateClass: 'warning',
  },
]

const services: ServiceItem[] = [
  {
    name: 'Node registry',
    detail: 'Registration and heartbeat API will bind here.',
    state: 'Planned',
    stateClass: 'planned',
  },
  {
    name: 'Peer directory',
    detail: 'Peer lookup, key exchange and route policy are not wired yet.',
    state: 'Planned',
    stateClass: 'planned',
  },
  {
    name: 'Dashboard API',
    detail: 'Static console shell is ready for authenticated API integration.',
    state: 'Prototype',
    stateClass: 'active',
  },
  {
    name: 'QUIC relay',
    detail: 'Stream telemetry and limits will use the relay transport layer.',
    state: 'Coming soon',
    stateClass: 'planned',
  },
]

const aclItems: SummaryItem[] = [
  { label: 'Token mode', value: 'Bearer', detail: 'Configured in server runtime, not stored in the Web shell' },
  { label: 'ACL policies', value: '0', detail: 'Policy editor remains disabled until API schema is stable' },
  { label: 'CORS mode', value: 'Allowlist', detail: 'Expected production posture for the Dashboard API' },
]

const alerts: AlertItem[] = [
  {
    time: '09:42',
    title: 'AP Southeast relay latency elevated',
    detail: 'Mock alert showing how degraded regions will be presented.',
  },
  {
    time: '08:15',
    title: 'Control Plane API integration pending',
    detail: 'The page intentionally avoids live calls until authentication is wired.',
  },
  {
    time: 'Yesterday',
    title: 'Dashboard shell refreshed',
    detail: 'Visual system aligned with the NexTunnel desktop application.',
  },
]
</script>

<style scoped>
:global(:root) {
  --nex-cyan: #00ffff;
  --tunnel-violet: #8a2be2;
  --data-blue: #0000ff;
  --neutral-grey: #a8a9a9;
  --future-white: #feffff;
  --shell-bg: #eef4fb;
  --console-bg: #07111f;
  --console-panel: #0c1b2d;
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
  --sidebar-width: 252px;
}

:global(*) {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

:global(body) {
  min-width: 320px;
  min-height: 100dvh;
  background: var(--shell-bg);
  color: var(--text-primary);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

:global(button) {
  font: inherit;
}

.server-shell {
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
    radial-gradient(circle at 9% 28%, rgba(0, 255, 255, 0.16), transparent 28%),
    radial-gradient(circle at 78% 14%, rgba(138, 43, 226, 0.16), transparent 30%),
    linear-gradient(120deg, rgba(255, 255, 255, 0.94), rgba(237, 245, 253, 0.82)),
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

.console-sidebar,
.console-main {
  border: 1px solid rgba(255, 255, 255, 0.68);
  background:
    linear-gradient(180deg, rgba(10, 24, 42, 0.98), rgba(5, 12, 24, 0.98)),
    var(--console-bg);
  box-shadow: var(--shadow-shell);
}

.console-sidebar {
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
  border-radius: var(--radius-panel);
  background: linear-gradient(135deg, var(--nex-cyan), var(--tunnel-violet));
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
  opacity: 0.6;
}

.nav-code {
  color: var(--nex-cyan);
  font-family: ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace;
  font-size: 11px;
}

.nav-badge,
.planned-pill {
  min-width: 0;
  max-width: 100%;
  border: 1px solid rgba(138, 43, 226, 0.28);
  border-radius: 999px;
  background: rgba(138, 43, 226, 0.1);
  color: #d5b7ff;
  font-size: 11px;
  line-height: 1;
  overflow: hidden;
  padding: 7px 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-status {
  margin-top: auto;
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  border: 1px solid rgba(0, 255, 255, 0.12);
  border-radius: var(--radius-panel);
  background: rgba(0, 255, 255, 0.06);
  padding: 12px;
}

.status-dot {
  width: 8px;
  height: 8px;
  margin-top: 5px;
  border-radius: 50%;
  background: var(--status-warning);
  box-shadow: 0 0 16px var(--status-warning);
}

.sidebar-status strong {
  font-size: 13px;
}

.sidebar-status p {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 12px;
}

.console-main {
  min-width: 0;
  min-height: calc(100dvh - 36px);
  padding: 22px;
  border-radius: var(--radius-shell);
  overflow: auto;
}

.console-header {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.header-kicker,
.panel-kicker,
.metric-card span {
  color: var(--nex-cyan);
  font-size: 11px;
  font-weight: 760;
}

.console-header h1 {
  max-width: 760px;
  margin-top: 6px;
  font-size: 32px;
  line-height: 1.12;
}

.header-actions {
  display: flex;
  align-items: flex-start;
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

.btn:disabled {
  cursor: not-allowed;
  filter: grayscale(0.4);
  opacity: 0.48;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-primary);
}

.btn-sm {
  min-height: 32px;
  padding: 6px 12px;
  font-size: 12px;
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.metric-card,
.panel {
  min-width: 0;
  border: 1px solid var(--console-border-muted);
  border-radius: var(--radius-panel);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.055), rgba(255, 255, 255, 0.025)),
    rgba(10, 25, 43, 0.92);
  box-shadow: var(--shadow-panel);
}

.metric-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 118px;
  padding: 16px;
}

.metric-card strong {
  color: var(--text-primary);
  font-size: 26px;
  line-height: 1.1;
}

.metric-card p {
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.map-panel,
.panel {
  padding: 18px;
}

.map-panel {
  min-width: 0;
  border: 1px solid var(--console-border-muted);
  border-radius: var(--radius-panel);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.055), rgba(255, 255, 255, 0.025)),
    rgba(10, 25, 43, 0.92);
  box-shadow: var(--shadow-panel);
  margin-bottom: 16px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
  margin-bottom: 16px;
}

.panel-header h2 {
  margin-top: 5px;
  color: var(--text-primary);
  font-size: 18px;
  line-height: 1.2;
}

.world-map {
  position: relative;
  width: 100%;
  min-width: 0;
  min-height: 360px;
  border: 1px solid rgba(0, 255, 255, 0.12);
  border-radius: var(--radius-panel);
  background:
    radial-gradient(ellipse at 28% 46%, rgba(122, 151, 177, 0.34) 0 12%, transparent 13%),
    radial-gradient(ellipse at 50% 39%, rgba(122, 151, 177, 0.28) 0 15%, transparent 16%),
    radial-gradient(ellipse at 72% 50%, rgba(122, 151, 177, 0.28) 0 13%, transparent 14%),
    radial-gradient(ellipse at 55% 68%, rgba(122, 151, 177, 0.18) 0 6%, transparent 7%),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.045) 0 1px, transparent 1px 52px),
    repeating-linear-gradient(0deg, rgba(255, 255, 255, 0.035) 0 1px, transparent 1px 52px),
    linear-gradient(120deg, rgba(0, 0, 255, 0.14), rgba(0, 255, 255, 0.06), rgba(138, 43, 226, 0.12));
  overflow: hidden;
}

.route {
  position: absolute;
  height: 2px;
  border-radius: 999px;
  background: linear-gradient(90deg, transparent, var(--nex-cyan), var(--tunnel-violet), transparent);
  transform-origin: left center;
}

.route-primary {
  width: 48%;
  left: 18%;
  top: 48%;
  transform: rotate(-8deg);
}

.route-secondary {
  width: 35%;
  left: 46%;
  top: 47%;
  transform: rotate(18deg);
}

.route-tertiary {
  width: 42%;
  left: 25%;
  top: 64%;
  transform: rotate(12deg);
}

.map-node {
  position: absolute;
  display: grid;
  place-items: center;
  min-width: 58px;
  border: 1px solid rgba(0, 255, 255, 0.28);
  border-radius: 999px;
  background: rgba(8, 22, 39, 0.94);
  color: var(--text-primary);
  font-family: ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  font-weight: 800;
  padding: 8px 11px;
  box-shadow: 0 0 30px rgba(0, 255, 255, 0.16);
}

.node-sfo {
  left: 18%;
  top: 43%;
}

.node-fra {
  left: 48%;
  top: 36%;
}

.node-sin {
  right: 20%;
  bottom: 25%;
}

.node-nrt {
  right: 12%;
  top: 39%;
}

.node-gru {
  left: 35%;
  bottom: 20%;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(320px, 0.75fr);
  gap: 16px;
  margin-bottom: 16px;
}

.lower-grid {
  grid-template-columns: minmax(0, 0.95fr) minmax(0, 1.05fr);
}

.relay-list,
.service-list,
.alert-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.relay-item,
.service-item,
.alert-item,
.acl-item {
  border: 1px solid rgba(168, 169, 169, 0.14);
  border-radius: var(--radius-panel);
  background: rgba(255, 255, 255, 0.035);
}

.relay-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 14px;
  align-items: center;
  padding: 13px;
}

.relay-item strong,
.service-item strong,
.alert-item strong {
  color: var(--text-primary);
  font-size: 13px;
}

.relay-item p,
.service-item p,
.alert-item p,
.acl-item p {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.relay-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  text-align: right;
}

.node-state {
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-secondary);
  font-size: 12px;
  padding: 6px 9px;
}

.node-state.healthy {
  color: var(--status-success);
}

.node-state.warning {
  color: var(--status-warning);
}

.service-item {
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: start;
  padding: 13px;
}

.service-dot {
  width: 8px;
  height: 8px;
  margin-top: 5px;
  border-radius: 50%;
  background: var(--status-warning);
  box-shadow: 0 0 16px currentColor;
}

.service-dot.active {
  background: var(--status-success);
  color: var(--status-success);
}

.service-dot.planned {
  background: var(--tunnel-violet);
  color: var(--tunnel-violet);
}

.service-state {
  color: var(--text-secondary);
  font-size: 12px;
}

.acl-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.acl-item {
  min-height: 126px;
  padding: 14px;
}

.acl-item span {
  color: var(--nex-cyan);
  font-size: 11px;
  font-weight: 760;
}

.acl-item strong {
  display: block;
  margin-top: 10px;
  color: var(--text-primary);
  font-size: 22px;
}

.alert-item {
  display: grid;
  grid-template-columns: 74px minmax(0, 1fr);
  gap: 12px;
  padding: 12px;
}

.alert-time {
  color: var(--nex-cyan);
  font-family: ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
}

@media (max-width: 1180px) {
  .overview-grid,
  .content-grid,
  .lower-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .acl-summary {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 980px) {
  .server-shell {
    grid-template-columns: 1fr;
  }

  .console-sidebar {
    min-height: auto;
  }

  .sidebar-nav {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .server-shell {
    padding: 10px;
  }

  .console-main,
  .console-sidebar {
    min-height: auto;
    border-radius: 12px;
  }

  .console-main {
    padding: 14px;
  }

  .console-header,
  .panel-header {
    flex-direction: column;
  }

  .console-header h1 {
    font-size: 25px;
    overflow-wrap: anywhere;
  }

  .header-actions,
  .sidebar-nav,
  .overview-grid,
  .content-grid,
  .lower-grid {
    grid-template-columns: 1fr;
    width: 100%;
  }

  .header-actions {
    display: grid;
  }

  .world-map {
    min-height: 280px;
  }

  .relay-item,
  .service-item,
  .alert-item {
    grid-template-columns: 1fr;
  }

  .relay-stats {
    text-align: left;
  }
}
</style>
