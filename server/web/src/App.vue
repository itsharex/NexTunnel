<template>
  <div class="dashboard-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">
          NT
        </div>
        <div>
          <strong>NexTunnel</strong>
          <span>Server Dashboard</span>
        </div>
      </div>

      <nav
        class="nav-list"
        aria-label="Dashboard navigation"
      >
        <a href="#overview">总览</a>
        <a href="#nodes">节点</a>
        <a href="#traffic">流量</a>
        <a href="#acl">ACL</a>
        <a href="#alerts">告警</a>
      </nav>

      <div class="sidebar-health">
        <span
          class="health-dot"
          :class="healthStatusClass"
        />
        <div>
          <strong>API {{ healthStatus }}</strong>
          <span>{{ lastRefreshLabel }}</span>
        </div>
      </div>
    </aside>

    <main class="main">
      <section
        v-if="!isAuthenticated"
        class="login-surface"
      >
        <div class="login-panel">
          <div>
            <p class="eyebrow">
              Dashboard Login
            </p>
            <h1>登录管理控制台</h1>
            <p class="login-copy">
              使用后端 Dashboard 管理员账户访问节点、流量、ACL 和告警数据。
            </p>
          </div>

          <form
            class="login-form"
            @submit.prevent="handleLogin"
          >
            <label>
              用户名
              <input
                v-model.trim="loginForm.username"
                autocomplete="username"
                type="text"
              >
            </label>
            <label>
              密码
              <input
                v-model="loginForm.password"
                autocomplete="current-password"
                type="password"
              >
            </label>
            <button
              class="btn primary"
              type="submit"
              :disabled="isSubmitting"
            >
              {{ isSubmitting ? '登录中' : '登录' }}
            </button>
          </form>

          <p
            v-if="errorMessage"
            class="message error"
          >
            {{ errorMessage }}
          </p>
        </div>
      </section>

      <template v-else>
        <header class="topbar">
          <div>
            <p class="eyebrow">
              生产控制台
            </p>
            <h1>全球加速运行面板</h1>
          </div>
          <div class="topbar-actions">
            <span class="user-chip">{{ activeUserLabel }}</span>
            <button
              class="btn secondary"
              type="button"
              :disabled="isLoading"
              @click="refreshDashboard"
            >
              {{ isLoading ? '刷新中' : '刷新' }}
            </button>
            <button
              class="btn ghost"
              type="button"
              @click="handleLogout"
            >
              退出
            </button>
          </div>
        </header>

        <p
          v-if="errorMessage"
          class="message error"
        >
          {{ errorMessage }}
        </p>
        <p
          v-if="successMessage"
          class="message success"
        >
          {{ successMessage }}
        </p>

        <section
          id="overview"
          class="metric-grid"
          aria-label="Dashboard metrics"
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

        <section class="layout-grid">
          <article
            id="nodes"
            class="panel map-panel"
          >
            <div class="panel-header">
              <div>
                <p class="eyebrow">
                  节点地图
                </p>
                <h2>区域分布与在线状态</h2>
              </div>
              <select
                v-model="selectedRegion"
                class="select-control"
                aria-label="筛选区域"
              >
                <option
                  v-for="region in regionOptions"
                  :key="region"
                  :value="region"
                >
                  {{ region }}
                </option>
              </select>
            </div>

            <div
              class="node-map"
              aria-label="节点区域地图"
            >
              <span
                v-for="(node, index) in filteredNodes"
                :key="node.node_id"
                class="map-dot"
                :class="node.online ? 'online' : 'offline'"
                :style="nodePosition(node, index)"
                :title="`${node.node_id} · ${node.region}`"
              />
            </div>

            <div class="node-table compact">
              <button
                v-for="node in filteredNodes"
                :key="node.node_id"
                class="node-row"
                type="button"
                :class="{ selected: selectedNodeID === node.node_id }"
                @click="selectNode(node.node_id)"
              >
                <span
                  class="status-pill"
                  :class="node.online ? 'online' : 'offline'"
                >
                  {{ statusLabel(node.online) }}
                </span>
                <strong>{{ node.node_id }}</strong>
                <span>{{ node.region || '未分区' }}</span>
                <span>{{ formatRelativeTime(node.last_seen) }}</span>
              </button>
            </div>
          </article>

          <article
            id="traffic"
            class="panel traffic-panel"
          >
            <div class="panel-header">
              <div>
                <p class="eyebrow">
                  流量
                </p>
                <h2>节点带宽分布</h2>
              </div>
              <span class="panel-count">{{ trafficBars.length }} 条样本</span>
            </div>

            <div class="traffic-bars">
              <div
                v-for="bar in trafficBars"
                :key="bar.label"
                class="traffic-row"
              >
                <div class="traffic-label">
                  <strong>{{ bar.label }}</strong>
                  <span>{{ bar.detail }}</span>
                </div>
                <div
                  class="bar-track"
                  aria-hidden="true"
                >
                  <span
                    class="rx-bar"
                    :style="{ width: `${bar.rxPercent}%` }"
                  />
                  <span
                    class="tx-bar"
                    :style="{ width: `${bar.txPercent}%` }"
                  />
                </div>
              </div>
            </div>
          </article>
        </section>

        <section class="panel node-detail-panel">
          <div class="panel-header">
            <div>
              <p class="eyebrow">
                节点清单
              </p>
              <h2>Relay 节点管理</h2>
            </div>
          </div>

          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>节点</th>
                  <th>区域</th>
                  <th>NAT</th>
                  <th>接收</th>
                  <th>发送</th>
                  <th>最后心跳</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="node in filteredNodes"
                  :key="node.node_id"
                >
                  <td>
                    <span
                      class="status-pill"
                      :class="node.online ? 'online' : 'offline'"
                    >
                      {{ statusLabel(node.online) }}
                    </span>
                    <strong>{{ node.node_id }}</strong>
                  </td>
                  <td>{{ node.region || '未分区' }}</td>
                  <td>{{ node.nat_type || '未知' }}</td>
                  <td>{{ formatBytes(node.rx_bytes) }}</td>
                  <td>{{ formatBytes(node.tx_bytes) }}</td>
                  <td>{{ formatRelativeTime(node.last_seen) }}</td>
                  <td>
                    <button
                      class="btn danger compact-btn"
                      type="button"
                      @click="removeNode(node.node_id)"
                    >
                      删除
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="layout-grid">
          <article
            id="acl"
            class="panel acl-panel"
          >
            <div class="panel-header">
              <div>
                <p class="eyebrow">
                  访问控制
                </p>
                <h2>ACL 规则</h2>
              </div>
              <span class="panel-count">{{ snapshot.aclRules.length }} 条规则</span>
            </div>

            <form
              class="acl-form"
              @submit.prevent="submitACLRule"
            >
              <label>
                来源
                <input
                  v-model.trim="aclForm.source"
                  placeholder="node-a 或 *"
                  type="text"
                >
              </label>
              <label>
                目标
                <input
                  v-model.trim="aclForm.target"
                  placeholder="node-b 或 10.0.0.0/24"
                  type="text"
                >
              </label>
              <label>
                协议
                <select v-model="aclForm.protocol">
                  <option value="tcp">TCP</option>
                  <option value="udp">UDP</option>
                  <option value="icmp">ICMP</option>
                  <option value="any">ANY</option>
                </select>
              </label>
              <label>
                动作
                <select v-model="aclForm.action">
                  <option value="allow">允许</option>
                  <option value="deny">拒绝</option>
                </select>
              </label>
              <label>
                优先级
                <input
                  v-model.number="aclForm.priority"
                  min="0"
                  step="1"
                  type="number"
                >
              </label>
              <label class="checkbox-label">
                <input
                  v-model="aclForm.enabled"
                  type="checkbox"
                >
                启用
              </label>
              <button
                class="btn primary"
                type="submit"
                :disabled="isSubmitting"
              >
                添加规则
              </button>
            </form>

            <div class="acl-list">
              <div
                v-for="rule in sortedACLRules"
                :key="rule.id"
                class="acl-row"
              >
                <span
                  class="status-pill"
                  :class="rule.enabled ? 'online' : 'offline'"
                >
                  {{ rule.enabled ? '启用' : '停用' }}
                </span>
                <strong>{{ rule.source }} → {{ rule.target }}</strong>
                <span>{{ rule.protocol.toUpperCase() }} · {{ aclActionLabel(rule.action) }} · P{{ rule.priority }}</span>
                <button
                  class="btn ghost compact-btn"
                  type="button"
                  @click="removeACLRule(rule.id)"
                >
                  删除
                </button>
              </div>
            </div>
          </article>

          <article
            id="alerts"
            class="panel alert-panel"
          >
            <div class="panel-header">
              <div>
                <p class="eyebrow">
                  告警
                </p>
                <h2>待处理事件</h2>
              </div>
              <span class="panel-count">{{ unackedAlerts.length }} 未确认</span>
            </div>

            <div class="alert-list">
              <div
                v-for="alert in recentAlerts"
                :key="alert.id"
                class="alert-row"
              >
                <span
                  class="severity"
                  :class="severityClass(alert.level)"
                >
                  {{ alert.level }}
                </span>
                <div>
                  <strong>{{ alert.rule_name || alert.message }}</strong>
                  <p>{{ alert.message }}</p>
                  <span>{{ alert.node_id || 'global' }} · {{ formatDateTime(alert.created_at) }}</span>
                </div>
                <button
                  class="btn secondary compact-btn"
                  type="button"
                  :disabled="alert.acked"
                  @click="ackAlert(alert.id)"
                >
                  {{ alert.acked ? '已确认' : '确认' }}
                </button>
              </div>
            </div>
          </article>
        </section>
      </template>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import {
  acknowledgeAlert,
  clearStoredToken,
  createACLRule,
  deleteACLRule,
  deleteNode,
  fetchDashboardSnapshot,
  fetchHealth,
  login,
  readStoredToken,
} from './api'
import { formatBandwidth, formatBytes, formatDateTime, formatNumber, formatRelativeTime, statusLabel } from './formatters'
import type { ACLFormState, ACLRuleView, DashboardSnapshot, NodeStatus, TrafficStats, User } from './types'

const REFRESH_INTERVAL_MS = 30_000
const ALL_REGIONS = '全部区域'
const DEFAULT_ACL_ACTION = 'allow'
const DEFAULT_ACL_PROTOCOL = 'tcp'
const MAP_FALLBACK_TOP_OFFSET = 16
const MAP_FALLBACK_LEFT_OFFSET = 12

const REGION_COORDINATES: Record<string, { left: number; top: number }> = {
  'us-east': { left: 27, top: 38 },
  'us-west': { left: 17, top: 42 },
  'eu-central': { left: 50, top: 35 },
  'ap-southeast': { left: 73, top: 58 },
  'ap-northeast': { left: 80, top: 41 },
  'sa-east': { left: 38, top: 72 },
}

const createEmptySnapshot = (): DashboardSnapshot => ({
  nodes: [],
  stats: [],
  aclRules: [],
  alerts: [],
  users: [],
})

const createEmptyACLForm = (): ACLFormState => ({
  source: '*',
  target: '',
  action: DEFAULT_ACL_ACTION,
  protocol: DEFAULT_ACL_PROTOCOL,
  priority: 100,
  enabled: true,
})

const token = ref(readStoredToken())
const authUser = ref<User | null>(null)
const snapshot = ref<DashboardSnapshot>(createEmptySnapshot())
const selectedRegion = ref(ALL_REGIONS)
const selectedNodeID = ref('')
const healthStatus = ref('检测中')
const lastRefreshAt = ref('')
const isLoading = ref(false)
const isSubmitting = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const loginForm = reactive({ username: 'admin', password: '' })
const aclForm = reactive<ACLFormState>(createEmptyACLForm())
let refreshTimer: number | undefined

const isAuthenticated = computed(() => token.value.length > 0)

const healthStatusClass = computed(() => (healthStatus.value === 'ok' ? 'online' : 'offline'))

const activeUserLabel = computed(() => {
  if (authUser.value) {
    return `${authUser.value.username} · ${authUser.value.role}`
  }
  return '已认证'
})

const lastRefreshLabel = computed(() => {
  if (!lastRefreshAt.value) {
    return '等待首次刷新'
  }
  return `上次刷新 ${formatRelativeTime(lastRefreshAt.value)}`
})

const onlineNodes = computed(() => snapshot.value.nodes.filter((node) => node.online))
const offlineNodes = computed(() => snapshot.value.nodes.filter((node) => !node.online))
const unackedAlerts = computed(() => snapshot.value.alerts.filter((alert) => !alert.acked))

const aggregateStats = computed(() => {
  const initialValue: TrafficStats = {
    rx_bytes: 0,
    tx_bytes: 0,
    rx_bandwidth_bps: 0,
    tx_bandwidth_bps: 0,
    connections: 0,
    timestamp: new Date().toISOString(),
  }

  return snapshot.value.stats.reduce<TrafficStats>((total, item) => ({
    ...total,
    rx_bytes: total.rx_bytes + item.rx_bytes,
    tx_bytes: total.tx_bytes + item.tx_bytes,
    rx_bandwidth_bps: total.rx_bandwidth_bps + item.rx_bandwidth_bps,
    tx_bandwidth_bps: total.tx_bandwidth_bps + item.tx_bandwidth_bps,
    connections: total.connections + item.connections,
  }), initialValue)
})

const metrics = computed(() => [
  {
    label: '在线节点',
    value: `${onlineNodes.value.length}/${snapshot.value.nodes.length}`,
    detail: `${offlineNodes.value.length} 个离线节点`,
  },
  {
    label: '活跃连接',
    value: formatNumber(aggregateStats.value.connections),
    detail: '来自统计 API 的连接总数',
  },
  {
    label: '实时带宽',
    value: formatBandwidth(aggregateStats.value.rx_bandwidth_bps + aggregateStats.value.tx_bandwidth_bps),
    detail: `${formatBytes(aggregateStats.value.rx_bytes + aggregateStats.value.tx_bytes)} 累计流量`,
  },
  {
    label: '告警',
    value: formatNumber(unackedAlerts.value.length),
    detail: `${snapshot.value.aclRules.length} 条 ACL 规则`,
  },
])

const regionOptions = computed(() => {
  const regions = new Set(snapshot.value.nodes.map((node) => node.region || '未分区'))
  return [ALL_REGIONS, ...Array.from(regions).sort()]
})

const filteredNodes = computed(() => {
  if (selectedRegion.value === ALL_REGIONS) {
    return snapshot.value.nodes
  }
  return snapshot.value.nodes.filter((node) => (node.region || '未分区') === selectedRegion.value)
})

const sortedACLRules = computed(() => [...snapshot.value.aclRules].sort((a, b) => a.priority - b.priority))

const recentAlerts = computed(() =>
  [...snapshot.value.alerts]
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    .slice(0, 8),
)

const maxBandwidth = computed(() => {
  const values = snapshot.value.stats.map((item) => item.rx_bandwidth_bps + item.tx_bandwidth_bps)
  return Math.max(1, ...values)
})

const trafficBars = computed(() =>
  snapshot.value.stats.map((item) => {
    const totalBandwidth = item.rx_bandwidth_bps + item.tx_bandwidth_bps
    return {
      label: item.node_id || 'global',
      detail: `${formatBandwidth(totalBandwidth)} · ${item.connections} 连接`,
      rxPercent: Math.max(4, Math.round((item.rx_bandwidth_bps / maxBandwidth.value) * 100)),
      txPercent: Math.max(4, Math.round((item.tx_bandwidth_bps / maxBandwidth.value) * 100)),
    }
  }),
)

const setFeedback = (type: 'error' | 'success', message: string): void => {
  errorMessage.value = type === 'error' ? message : ''
  successMessage.value = type === 'success' ? message : ''
}

const describeError = (error: unknown): string => (error instanceof Error ? error.message : '未知错误')

const loadHealth = async (): Promise<void> => {
  try {
    healthStatus.value = await fetchHealth()
  } catch {
    healthStatus.value = '不可用'
  }
}

const loadSnapshot = async (): Promise<void> => {
  if (!token.value) {
    return
  }

  isLoading.value = true
  try {
    // 所有核心面板共享同一次快照刷新，避免多个组件重复打 API。
    snapshot.value = await fetchDashboardSnapshot(token.value)
    lastRefreshAt.value = new Date().toISOString()
    if (!authUser.value && snapshot.value.users.length > 0) {
      authUser.value = snapshot.value.users[0]
    }
    setFeedback('success', '数据已刷新')
  } catch (error) {
    setFeedback('error', `刷新失败：${describeError(error)}`)
    if (describeError(error).includes('token')) {
      clearStoredToken()
      token.value = ''
    }
  } finally {
    isLoading.value = false
  }
}

const refreshDashboard = async (): Promise<void> => {
  await Promise.all([loadHealth(), loadSnapshot()])
}

const handleLogin = async (): Promise<void> => {
  if (!loginForm.username || !loginForm.password) {
    setFeedback('error', '请输入用户名和密码')
    return
  }

  isSubmitting.value = true
  try {
    const response = await login(loginForm.username, loginForm.password)
    token.value = response.token
    authUser.value = response.user
    loginForm.password = ''
    setFeedback('success', '登录成功')
    await refreshDashboard()
  } catch (error) {
    setFeedback('error', `登录失败：${describeError(error)}`)
  } finally {
    isSubmitting.value = false
  }
}

const handleLogout = (): void => {
  clearStoredToken()
  token.value = ''
  authUser.value = null
  snapshot.value = createEmptySnapshot()
  setFeedback('success', '已退出登录')
}

const nodePosition = (node: NodeStatus, index: number): Record<string, string> => {
  const key = node.region.toLowerCase()
  const knownPosition = REGION_COORDINATES[key]
  if (knownPosition) {
    return { left: `${knownPosition.left}%`, top: `${knownPosition.top}%` }
  }

  // 未知区域使用确定性散列位置，保证刷新后节点不会随机跳动。
  const left = MAP_FALLBACK_LEFT_OFFSET + ((index * 17) % 72)
  const top = MAP_FALLBACK_TOP_OFFSET + ((index * 23) % 68)
  return { left: `${left}%`, top: `${top}%` }
}

const selectNode = (nodeID: string): void => {
  selectedNodeID.value = selectedNodeID.value === nodeID ? '' : nodeID
}

const buildACLRule = (): ACLRuleView => ({
  id: `acl-${Date.now()}`,
  source: aclForm.source,
  target: aclForm.target,
  action: aclForm.action,
  protocol: aclForm.protocol,
  priority: Number.isFinite(aclForm.priority) ? aclForm.priority : 100,
  enabled: aclForm.enabled,
  created_at: new Date().toISOString(),
})

const submitACLRule = async (): Promise<void> => {
  if (!aclForm.source || !aclForm.target) {
    setFeedback('error', 'ACL 来源和目标不能为空')
    return
  }

  isSubmitting.value = true
  try {
    await createACLRule(token.value, buildACLRule())
    Object.assign(aclForm, createEmptyACLForm())
    setFeedback('success', 'ACL 规则已添加')
    await loadSnapshot()
  } catch (error) {
    setFeedback('error', `添加 ACL 失败：${describeError(error)}`)
  } finally {
    isSubmitting.value = false
  }
}

const removeACLRule = async (ruleID: string): Promise<void> => {
  try {
    await deleteACLRule(token.value, ruleID)
    setFeedback('success', 'ACL 规则已删除')
    await loadSnapshot()
  } catch (error) {
    setFeedback('error', `删除 ACL 失败：${describeError(error)}`)
  }
}

const removeNode = async (nodeID: string): Promise<void> => {
  try {
    await deleteNode(token.value, nodeID)
    setFeedback('success', '节点已删除')
    await loadSnapshot()
  } catch (error) {
    setFeedback('error', `删除节点失败：${describeError(error)}`)
  }
}

const ackAlert = async (alertID: string): Promise<void> => {
  try {
    await acknowledgeAlert(token.value, alertID, authUser.value?.username ?? 'dashboard')
    setFeedback('success', '告警已确认')
    await loadSnapshot()
  } catch (error) {
    setFeedback('error', `确认告警失败：${describeError(error)}`)
  }
}

const aclActionLabel = (action: string): string => (action === 'deny' ? '拒绝' : '允许')

const severityClass = (level: string): string => {
  const normalizedLevel = level.toLowerCase()
  if (normalizedLevel === 'critical') {
    return 'critical'
  }
  if (normalizedLevel === 'warning') {
    return 'warning'
  }
  return 'info'
}

onMounted(() => {
  void refreshDashboard()
  refreshTimer = window.setInterval(() => {
    void refreshDashboard()
  }, REFRESH_INTERVAL_MS)
})

onUnmounted(() => {
  if (refreshTimer !== undefined) {
    window.clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
:global(:root) {
  --bg: #f3f6f8;
  --surface: #ffffff;
  --surface-muted: #eef3f5;
  --sidebar: #101820;
  --sidebar-soft: #172330;
  --border: #d9e1e5;
  --border-strong: #b8c5cc;
  --text: #10202a;
  --text-muted: #667883;
  --text-inverse: #edf7fa;
  --accent: #0f766e;
  --accent-strong: #0b5f59;
  --success: #138a5b;
  --warning: #b7791f;
  --danger: #b42318;
  --info: #2563eb;
  --radius: 8px;
  --shadow: 0 14px 40px rgb(16 32 42 / 0.08);
  --sidebar-width: 248px;
}

:global(*) {
  box-sizing: border-box;
}

:global(body) {
  margin: 0;
  min-width: 320px;
  min-height: 100dvh;
  background: var(--bg);
  color: var(--text);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

:global(button),
:global(input),
:global(select) {
  font: inherit;
}

.dashboard-shell {
  display: grid;
  grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
  min-height: 100dvh;
}

.sidebar {
  position: sticky;
  top: 0;
  display: flex;
  flex-direction: column;
  gap: 24px;
  height: 100dvh;
  padding: 20px;
  background: var(--sidebar);
  color: var(--text-inverse);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border-radius: var(--radius);
  background: var(--accent);
  color: #ffffff;
  font-weight: 850;
}

.brand strong,
.brand span,
.sidebar-health strong,
.sidebar-health span {
  display: block;
}

.brand span,
.sidebar-health span {
  color: #a9bcc6;
  font-size: 12px;
}

.nav-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.nav-list a {
  min-height: 40px;
  border-radius: var(--radius);
  color: #d9e8ed;
  display: flex;
  align-items: center;
  padding: 0 12px;
  text-decoration: none;
}

.nav-list a:hover {
  background: var(--sidebar-soft);
}

.sidebar-health {
  margin-top: auto;
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  border: 1px solid rgb(255 255 255 / 0.1);
  border-radius: var(--radius);
  background: var(--sidebar-soft);
  padding: 12px;
}

.health-dot,
.map-dot {
  border-radius: 999px;
}

.health-dot {
  width: 8px;
  height: 8px;
  margin-top: 6px;
  background: var(--danger);
}

.health-dot.online {
  background: var(--success);
}

.main {
  min-width: 0;
  padding: 22px;
}

.topbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.topbar h1,
.login-panel h1 {
  margin: 4px 0 0;
  font-size: 28px;
  line-height: 1.18;
}

.topbar-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.eyebrow {
  margin: 0;
  color: var(--accent);
  font-size: 12px;
  font-weight: 780;
}

.user-chip,
.panel-count,
.status-pill,
.severity {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  border-radius: 999px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 720;
}

.user-chip {
  border: 1px solid var(--border);
  background: var(--surface);
}

.btn {
  min-height: 38px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  color: var(--text);
  cursor: pointer;
  font-weight: 720;
  padding: 0 14px;
}

.btn:hover:not(:disabled) {
  border-color: var(--border-strong);
}

.btn:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.btn.primary {
  border-color: var(--accent);
  background: var(--accent);
  color: #ffffff;
}

.btn.secondary {
  border-color: #b7d8d3;
  background: #e7f4f2;
  color: var(--accent-strong);
}

.btn.ghost {
  background: transparent;
}

.btn.danger {
  border-color: #f0c7c1;
  background: #fff1f0;
  color: var(--danger);
}

.compact-btn {
  min-height: 30px;
  padding: 0 10px;
  font-size: 12px;
}

.message {
  margin: 0 0 12px;
  border-radius: var(--radius);
  padding: 10px 12px;
  font-size: 13px;
}

.message.error {
  border: 1px solid #f0c7c1;
  background: #fff1f0;
  color: var(--danger);
}

.message.success {
  border: 1px solid #b9decf;
  background: #eef9f4;
  color: var(--success);
}

.login-surface {
  display: grid;
  min-height: calc(100dvh - 44px);
  place-items: center;
}

.login-panel {
  width: min(460px, 100%);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: var(--shadow);
  padding: 24px;
}

.login-copy {
  color: var(--text-muted);
  line-height: 1.55;
}

.login-form,
.acl-form {
  display: grid;
  gap: 12px;
}

.login-form label,
.acl-form label {
  display: grid;
  gap: 6px;
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 700;
}

input,
select,
.select-control {
  width: 100%;
  min-height: 38px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: #ffffff;
  color: var(--text);
  padding: 0 10px;
}

input:focus,
select:focus,
.btn:focus-visible {
  outline: 2px solid #7dd3c7;
  outline-offset: 2px;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.metric-card,
.panel {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: var(--shadow);
}

.metric-card {
  min-height: 112px;
  padding: 16px;
}

.metric-card span {
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 740;
}

.metric-card strong {
  display: block;
  margin-top: 10px;
  font-size: 26px;
  line-height: 1.1;
}

.metric-card p {
  margin: 8px 0 0;
  color: var(--text-muted);
  font-size: 12px;
}

.layout-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(340px, 0.9fr);
  gap: 14px;
  margin-bottom: 14px;
}

.panel {
  min-width: 0;
  padding: 16px;
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.panel-header h2 {
  margin: 4px 0 0;
  font-size: 18px;
}

.panel-count {
  border: 1px solid var(--border);
  color: var(--text-muted);
}

.node-map {
  position: relative;
  min-height: 300px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background:
    linear-gradient(90deg, rgb(16 32 42 / 0.05) 1px, transparent 1px),
    linear-gradient(0deg, rgb(16 32 42 / 0.05) 1px, transparent 1px),
    linear-gradient(180deg, #f9fbfc, #edf4f6);
  background-size: 56px 56px, 56px 56px, auto;
  overflow: hidden;
}

.node-map::before {
  content: '';
  position: absolute;
  inset: 18% 12%;
  border-radius: 50%;
  border: 1px dashed rgb(15 118 110 / 0.25);
}

.map-dot {
  position: absolute;
  width: 15px;
  height: 15px;
  transform: translate(-50%, -50%);
  border: 2px solid #ffffff;
  box-shadow: 0 0 0 4px rgb(19 138 91 / 0.12);
}

.map-dot.online {
  background: var(--success);
}

.map-dot.offline {
  background: var(--danger);
  box-shadow: 0 0 0 4px rgb(180 35 24 / 0.12);
}

.node-table.compact {
  display: grid;
  gap: 8px;
  margin-top: 12px;
}

.node-row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr) 120px 110px;
  gap: 10px;
  align-items: center;
  width: 100%;
  min-height: 42px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: #ffffff;
  color: var(--text);
  cursor: pointer;
  padding: 8px 10px;
  text-align: left;
}

.node-row.selected {
  border-color: var(--accent);
  background: #effaf8;
}

.status-pill.online {
  background: #e9f7f1;
  color: var(--success);
}

.status-pill.offline {
  background: #fff1f0;
  color: var(--danger);
}

.traffic-bars {
  display: grid;
  gap: 14px;
}

.traffic-row {
  display: grid;
  gap: 8px;
}

.traffic-label {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: var(--text-muted);
  font-size: 12px;
}

.traffic-label strong {
  color: var(--text);
}

.bar-track {
  position: relative;
  height: 14px;
  border-radius: 999px;
  background: var(--surface-muted);
  overflow: hidden;
}

.rx-bar,
.tx-bar {
  position: absolute;
  inset-block: 0;
  border-radius: 999px;
}

.rx-bar {
  left: 0;
  background: var(--accent);
}

.tx-bar {
  right: 0;
  background: #7c8a93;
  opacity: 0.62;
}

.node-detail-panel {
  margin-bottom: 14px;
}

.table-wrap {
  overflow-x: auto;
}

table {
  width: 100%;
  min-width: 760px;
  border-collapse: collapse;
}

th,
td {
  border-bottom: 1px solid var(--border);
  padding: 12px 10px;
  text-align: left;
  vertical-align: middle;
  white-space: nowrap;
}

th {
  color: var(--text-muted);
  font-size: 12px;
}

td {
  font-size: 13px;
}

td:first-child {
  display: flex;
  align-items: center;
  gap: 8px;
}

.acl-form {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: 14px;
}

.checkbox-label {
  display: flex !important;
  align-items: center;
  gap: 8px !important;
}

.checkbox-label input {
  width: 16px;
  min-height: 16px;
}

.acl-list,
.alert-list {
  display: grid;
  gap: 8px;
}

.acl-row,
.alert-row {
  display: grid;
  align-items: center;
  gap: 10px;
  min-height: 54px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: #ffffff;
  padding: 10px;
}

.acl-row {
  grid-template-columns: 72px minmax(0, 1fr) minmax(160px, auto) auto;
}

.acl-row strong,
.acl-row span,
.alert-row strong,
.alert-row span,
.alert-row p {
  overflow-wrap: anywhere;
}

.alert-row {
  grid-template-columns: 82px minmax(0, 1fr) auto;
}

.alert-row p {
  margin: 4px 0;
  color: var(--text-muted);
  font-size: 12px;
}

.alert-row span {
  color: var(--text-muted);
  font-size: 12px;
}

.severity.info {
  background: #eef4ff;
  color: var(--info);
}

.severity.warning {
  background: #fff7e6;
  color: var(--warning);
}

.severity.critical {
  background: #fff1f0;
  color: var(--danger);
}

@media (max-width: 1180px) {
  .metric-grid,
  .layout-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .acl-form {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .dashboard-shell {
    grid-template-columns: 1fr;
  }

  .sidebar {
    position: static;
    height: auto;
  }

  .nav-list {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .main {
    padding: 14px;
  }

  .topbar,
  .panel-header {
    flex-direction: column;
  }

  .topbar-actions,
  .metric-grid,
  .layout-grid,
  .nav-list {
    grid-template-columns: 1fr;
    width: 100%;
  }

  .topbar-actions {
    display: grid;
  }

  .node-row,
  .acl-row,
  .alert-row {
    grid-template-columns: 1fr;
  }

  .node-map {
    min-height: 240px;
  }
}
</style>
