<template>
  <n-config-provider
    :theme="darkTheme"
    :theme-overrides="themeOverrides"
  >
    <n-message-provider>
      <div class="dashboard-shell">
        <aside class="sidebar">
          <div class="brand">
            <img
              class="brand-logo"
              :src="whiteLogo"
              alt="NexTunnel"
            >
            <div>
              <strong>NexTunnel</strong>
              <span>Server Dashboard</span>
            </div>
          </div>

          <nav
            class="nav-list"
            aria-label="Dashboard navigation"
          >
            <a
              v-for="item in navItems"
              :key="item.href"
              :href="item.href"
            >
              <span>{{ item.index }}</span>
              {{ item.label }}
            </a>
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
            <n-card
              class="login-panel"
              :bordered="false"
            >
              <div class="login-brand">
                <img
                  :src="blackLogo"
                  alt="NexTunnel"
                >
              </div>
              <div>
                <p class="eyebrow">
                  Dashboard Login
                </p>
                <h1>登录管理控制台</h1>
                <p class="login-copy">
                  使用后端 Dashboard 管理员账户访问节点、流量、ACL 和告警数据。
                </p>
              </div>

              <n-form
                class="login-form"
                label-placement="top"
                :show-feedback="false"
                @submit.prevent="handleLogin"
              >
                <n-form-item label="用户名">
                  <n-input
                    v-model:value="loginForm.username"
                    autocomplete="username"
                  />
                </n-form-item>
                <n-form-item label="密码">
                  <n-input
                    v-model:value="loginForm.password"
                    autocomplete="current-password"
                    type="password"
                    show-password-on="click"
                  />
                </n-form-item>
                <n-button
                  block
                  type="primary"
                  attr-type="submit"
                  :loading="isSubmitting"
                >
                  {{ isSubmitting ? '登录中' : '登录' }}
                </n-button>
              </n-form>

              <n-alert
                v-if="errorMessage"
                type="error"
                :bordered="false"
              >
                {{ errorMessage }}
              </n-alert>
            </n-card>
          </section>

          <template v-else>
            <header class="topbar">
              <div>
                <p class="eyebrow">
                  生产控制台
                </p>
                <h1>全球加速运行面板</h1>
              </div>
              <n-space align="center">
                <n-tag
                  round
                  type="info"
                  :bordered="false"
                >
                  {{ activeUserLabel }}
                </n-tag>
                <n-button
                  secondary
                  :loading="isLoading"
                  @click="refreshDashboard"
                >
                  {{ isLoading ? '刷新中' : '刷新' }}
                </n-button>
                <n-button
                  quaternary
                  @click="handleLogout"
                >
                  退出
                </n-button>
              </n-space>
            </header>

            <n-alert
              v-if="errorMessage"
              class="feedback-message"
              type="error"
              :bordered="false"
            >
              {{ errorMessage }}
            </n-alert>
            <n-alert
              v-if="successMessage"
              class="feedback-message"
              type="success"
              :bordered="false"
            >
              {{ successMessage }}
            </n-alert>

            <section
              id="overview"
              class="metric-grid"
              aria-label="Dashboard metrics"
            >
              <n-card
                v-for="metric in metrics"
                :key="metric.label"
                class="metric-card"
                :bordered="false"
              >
                <n-statistic
                  :label="metric.label"
                  :value="metric.value"
                />
                <span>{{ metric.detail }}</span>
              </n-card>
            </section>

            <section class="layout-grid">
              <n-card
                id="nodes"
                class="panel map-panel"
                :bordered="false"
              >
                <template #header>
                  <div class="panel-header">
                    <div>
                      <p class="eyebrow">
                        节点地图
                      </p>
                      <h2>区域分布与在线状态</h2>
                    </div>
                    <n-select
                      v-model:value="selectedRegion"
                      class="region-select"
                      :options="regionSelectOptions"
                    />
                  </div>
                </template>

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
                    <n-tag
                      round
                      size="small"
                      :type="node.online ? 'success' : 'error'"
                      :bordered="false"
                    >
                      {{ statusLabel(node.online) }}
                    </n-tag>
                    <strong>{{ node.node_id }}</strong>
                    <span>{{ node.region || '未分区' }}</span>
                    <span>{{ formatRelativeTime(node.last_seen) }}</span>
                  </button>
                </div>
              </n-card>

              <n-card
                id="traffic"
                class="panel traffic-panel"
                :bordered="false"
              >
                <template #header>
                  <div class="panel-header">
                    <div>
                      <p class="eyebrow">
                        流量
                      </p>
                      <h2>节点带宽分布</h2>
                    </div>
                    <n-tag
                      round
                      size="small"
                      :bordered="false"
                    >
                      {{ trafficBars.length }} 条样本
                    </n-tag>
                  </div>
                </template>

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
                  <n-empty
                    v-if="trafficBars.length === 0"
                    description="暂无流量样本"
                  />
                </div>
              </n-card>
            </section>

            <n-card
              class="panel node-detail-panel"
              :bordered="false"
            >
              <template #header>
                <div class="panel-header">
                  <div>
                    <p class="eyebrow">
                      节点清单
                    </p>
                    <h2>Relay 节点管理</h2>
                  </div>
                </div>
              </template>

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
                        <n-tag
                          round
                          size="small"
                          :type="node.online ? 'success' : 'error'"
                          :bordered="false"
                        >
                          {{ statusLabel(node.online) }}
                        </n-tag>
                        <strong>{{ node.node_id }}</strong>
                      </td>
                      <td>{{ node.region || '未分区' }}</td>
                      <td>{{ node.nat_type || '未知' }}</td>
                      <td>{{ formatBytes(node.rx_bytes) }}</td>
                      <td>{{ formatBytes(node.tx_bytes) }}</td>
                      <td>{{ formatRelativeTime(node.last_seen) }}</td>
                      <td>
                        <n-button
                          size="small"
                          type="error"
                          secondary
                          @click="removeNode(node.node_id)"
                        >
                          删除
                        </n-button>
                      </td>
                    </tr>
                  </tbody>
                </table>
                <n-empty
                  v-if="filteredNodes.length === 0"
                  description="暂无节点"
                />
              </div>
            </n-card>

            <section class="layout-grid">
              <n-card
                id="acl"
                class="panel acl-panel"
                :bordered="false"
              >
                <template #header>
                  <div class="panel-header">
                    <div>
                      <p class="eyebrow">
                        访问控制
                      </p>
                      <h2>ACL 规则</h2>
                    </div>
                    <n-tag
                      round
                      size="small"
                      :bordered="false"
                    >
                      {{ snapshot.aclRules.length }} 条规则
                    </n-tag>
                  </div>
                </template>

                <n-form
                  class="acl-form"
                  label-placement="top"
                  :show-feedback="false"
                  @submit.prevent="submitACLRule"
                >
                  <n-form-item label="来源">
                    <n-input
                      v-model:value="aclForm.source"
                      placeholder="node-a 或 *"
                    />
                  </n-form-item>
                  <n-form-item label="目标">
                    <n-input
                      v-model:value="aclForm.target"
                      placeholder="node-b 或 10.0.0.0/24"
                    />
                  </n-form-item>
                  <n-form-item label="协议">
                    <n-select
                      v-model:value="aclForm.protocol"
                      :options="protocolOptions"
                    />
                  </n-form-item>
                  <n-form-item label="动作">
                    <n-select
                      v-model:value="aclForm.action"
                      :options="actionOptions"
                    />
                  </n-form-item>
                  <n-form-item label="优先级">
                    <n-input-number
                      v-model:value="aclForm.priority"
                      :min="0"
                    />
                  </n-form-item>
                  <n-checkbox v-model:checked="aclForm.enabled">
                    启用
                  </n-checkbox>
                  <n-button
                    type="primary"
                    attr-type="submit"
                    :loading="isSubmitting"
                  >
                    添加规则
                  </n-button>
                </n-form>

                <div class="acl-list">
                  <div
                    v-for="rule in sortedACLRules"
                    :key="rule.id"
                    class="acl-row"
                  >
                    <n-tag
                      round
                      size="small"
                      :type="rule.enabled ? 'success' : 'warning'"
                      :bordered="false"
                    >
                      {{ rule.enabled ? '启用' : '停用' }}
                    </n-tag>
                    <strong>{{ rule.source }} -> {{ rule.target }}</strong>
                    <span>{{ rule.protocol.toUpperCase() }} · {{ aclActionLabel(rule.action) }} · P{{ rule.priority }}</span>
                    <n-button
                      size="small"
                      quaternary
                      @click="removeACLRule(rule.id)"
                    >
                      删除
                    </n-button>
                  </div>
                  <n-empty
                    v-if="sortedACLRules.length === 0"
                    description="暂无 ACL 规则"
                  />
                </div>
              </n-card>

              <n-card
                id="alerts"
                class="panel alert-panel"
                :bordered="false"
              >
                <template #header>
                  <div class="panel-header">
                    <div>
                      <p class="eyebrow">
                        告警
                      </p>
                      <h2>待处理事件</h2>
                    </div>
                    <n-tag
                      round
                      size="small"
                      type="warning"
                      :bordered="false"
                    >
                      {{ unackedAlerts.length }} 未确认
                    </n-tag>
                  </div>
                </template>

                <div class="alert-list">
                  <div
                    v-for="alert in recentAlerts"
                    :key="alert.id"
                    class="alert-row"
                  >
                    <n-tag
                      round
                      size="small"
                      :type="severityTagType(alert.level)"
                      :bordered="false"
                    >
                      {{ alert.level }}
                    </n-tag>
                    <div>
                      <strong>{{ alert.rule_name || alert.message }}</strong>
                      <p>{{ alert.message }}</p>
                      <span>{{ alert.node_id || 'global' }} · {{ formatDateTime(alert.created_at) }}</span>
                    </div>
                    <n-button
                      size="small"
                      secondary
                      :disabled="alert.acked"
                      @click="ackAlert(alert.id)"
                    >
                      {{ alert.acked ? '已确认' : '确认' }}
                    </n-button>
                  </div>
                  <n-empty
                    v-if="recentAlerts.length === 0"
                    description="暂无告警"
                  />
                </div>
              </n-card>
            </section>
          </template>
        </main>
      </div>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import {
  darkTheme,
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NConfigProvider,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NMessageProvider,
  NSelect,
  NSpace,
  NStatistic,
  NTag,
  type GlobalThemeOverrides,
  type SelectOption,
} from 'naive-ui'
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
import blackLogo from '@shared-logo/black-logo.png'
import whiteLogo from '@shared-logo/white-logo.png'

const REFRESH_INTERVAL_MS = 30_000
const ALL_REGIONS = '全部区域'
const DEFAULT_ACL_ACTION = 'allow'
const DEFAULT_ACL_PROTOCOL = 'tcp'
const MAP_FALLBACK_TOP_OFFSET = 16
const MAP_FALLBACK_LEFT_OFFSET = 12

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

const navItems = [
  { href: '#overview', label: '总览', index: '01' },
  { href: '#nodes', label: '节点', index: '02' },
  { href: '#traffic', label: '流量', index: '03' },
  { href: '#acl', label: 'ACL', index: '04' },
  { href: '#alerts', label: '告警', index: '05' },
] as const

const protocolOptions: SelectOption[] = [
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
  { label: 'ICMP', value: 'icmp' },
  { label: 'ANY', value: 'any' },
]

const actionOptions: SelectOption[] = [
  { label: '允许', value: 'allow' },
  { label: '拒绝', value: 'deny' },
]

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

const regionSelectOptions = computed<SelectOption[]>(() =>
  regionOptions.value.map((region) => ({ label: region, value: region })),
)

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

const severityTagType = (level: string): 'default' | 'error' | 'success' | 'warning' | 'info' => {
  const normalizedLevel = level.toLowerCase()
  if (normalizedLevel === 'critical') {
    return 'error'
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
  --nex-cyan: #00ffff;
  --tunnel-violet: #8a2be2;
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
  --sidebar-width: 248px;
}

:global(*) {
  box-sizing: border-box;
}

:global(body) {
  margin: 0;
  min-width: 320px;
  min-height: 100dvh;
  overflow-x: hidden;
  background:
    radial-gradient(circle at 22% 12%, rgba(0, 255, 255, 0.1), transparent 28%),
    radial-gradient(circle at 86% 20%, rgba(138, 43, 226, 0.13), transparent 30%),
    var(--bg-dark);
  color: var(--text-main);
  font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
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
  border-right: 1px solid var(--line-soft);
  background:
    linear-gradient(180deg, rgba(13, 27, 47, 0.98), rgba(6, 12, 25, 0.98)),
    var(--sidebar-bg);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-logo {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  object-fit: cover;
  box-shadow: 0 12px 30px rgba(0, 255, 255, 0.1);
}

.brand strong,
.brand span,
.sidebar-health strong,
.sidebar-health span {
  display: block;
}

.brand strong {
  color: var(--text-main);
  font-size: 17px;
}

.brand span,
.sidebar-health span {
  color: var(--text-dim);
  font-size: 12px;
}

.nav-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.nav-list a {
  min-height: 42px;
  display: flex;
  align-items: center;
  gap: 10px;
  border-radius: 8px;
  color: var(--text-dim);
  padding: 0 12px;
  text-decoration: none;
}

.nav-list a:hover {
  background: linear-gradient(90deg, rgba(0, 255, 255, 0.12), rgba(138, 43, 226, 0.08));
  color: var(--nex-cyan);
}

.nav-list span {
  font-family: Consolas, 'SFMono-Regular', monospace;
  font-size: 11px;
  font-weight: 800;
}

.sidebar-health {
  margin-top: auto;
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  border: 1px solid rgba(0, 255, 255, 0.16);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
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
  box-shadow: 0 0 12px rgba(16, 185, 129, 0.54);
}

.main {
  min-width: 0;
  padding: 24px 28px 28px;
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
  color: var(--text-main);
  font-size: 30px;
  line-height: 1.16;
}

.eyebrow {
  margin: 0;
  color: var(--nex-cyan);
  font-size: 12px;
  font-weight: 780;
}

.feedback-message {
  margin-bottom: 12px;
}

.login-surface {
  display: grid;
  min-height: calc(100dvh - 56px);
  place-items: center;
}

.login-panel {
  width: min(480px, 100%);
  border: 1px solid var(--line-soft);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.012)),
    var(--surface-bg);
  box-shadow: 0 18px 44px rgba(0, 0, 0, 0.2);
}

.login-brand img {
  width: 180px;
  height: auto;
  margin-bottom: 12px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.94);
}

.login-copy {
  color: var(--text-dim);
  line-height: 1.65;
}

.login-form {
  margin: 20px 0 14px;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}

.metric-card,
.panel {
  border: 1px solid var(--line-soft);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.012)),
    var(--surface-bg);
  box-shadow: 0 18px 44px rgba(0, 0, 0, 0.2);
}

.metric-card {
  min-height: 118px;
}

.metric-card span {
  display: block;
  margin-top: 8px;
  color: var(--text-dim);
  font-size: 12px;
}

.layout-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(360px, 0.9fr);
  gap: 18px;
  margin-bottom: 18px;
}

.panel-header {
  width: 100%;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.panel-header h2 {
  margin: 4px 0 0;
  color: var(--text-main);
  font-size: 18px;
}

.region-select {
  width: 160px;
}

.node-map {
  position: relative;
  min-height: 300px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background:
    linear-gradient(90deg, rgba(148, 163, 184, 0.1) 1px, transparent 1px),
    linear-gradient(0deg, rgba(148, 163, 184, 0.1) 1px, transparent 1px),
    linear-gradient(180deg, rgba(9, 17, 32, 0.78), rgba(18, 31, 52, 0.54));
  background-size: 56px 56px, 56px 56px, auto;
  overflow: hidden;
}

.node-map::before {
  content: '';
  position: absolute;
  inset: 18% 12%;
  border-radius: 50%;
  border: 1px dashed rgba(0, 255, 255, 0.22);
}

.map-dot {
  position: absolute;
  width: 15px;
  height: 15px;
  transform: translate(-50%, -50%);
  border: 2px solid rgba(255, 255, 255, 0.9);
}

.map-dot.online {
  background: var(--success);
  box-shadow: 0 0 0 4px rgba(16, 185, 129, 0.16);
}

.map-dot.offline {
  background: var(--danger);
  box-shadow: 0 0 0 4px rgba(239, 68, 68, 0.14);
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
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.56);
  color: var(--text-main);
  cursor: pointer;
  padding: 8px 10px;
  text-align: left;
}

.node-row.selected {
  border-color: var(--nex-cyan);
  background: linear-gradient(90deg, rgba(0, 255, 255, 0.12), rgba(138, 43, 226, 0.08));
}

.node-row span,
td {
  color: var(--text-dim);
}

.traffic-bars,
.acl-list,
.alert-list {
  display: grid;
  gap: 10px;
}

.traffic-row {
  display: grid;
  gap: 8px;
}

.traffic-label {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: var(--text-dim);
  font-size: 12px;
}

.traffic-label strong {
  color: var(--text-main);
}

.bar-track {
  position: relative;
  height: 14px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
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
  background: var(--nex-cyan);
}

.tx-bar {
  right: 0;
  background: var(--tunnel-violet);
  opacity: 0.68;
}

.node-detail-panel {
  margin-bottom: 18px;
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
  border-bottom: 1px solid var(--line-soft);
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
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.acl-row,
.alert-row {
  display: grid;
  align-items: center;
  gap: 10px;
  min-height: 54px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.56);
  padding: 10px;
}

.acl-row {
  grid-template-columns: 72px minmax(0, 1fr) minmax(160px, auto) auto;
}

.alert-row {
  grid-template-columns: 82px minmax(0, 1fr) auto;
}

.acl-row strong,
.acl-row span,
.alert-row strong,
.alert-row span,
.alert-row p {
  overflow-wrap: anywhere;
}

.alert-row p {
  margin: 4px 0;
  color: var(--text-dim);
  font-size: 12px;
}

.alert-row span {
  color: var(--text-muted);
  font-size: 12px;
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

  .metric-grid,
  .layout-grid,
  .nav-list {
    grid-template-columns: 1fr;
    width: 100%;
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
