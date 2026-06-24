import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  acknowledgeAlert,
  createACLRule,
  deleteACLRule,
  deleteNode,
  disconnectClient,
  fetchDashboardSnapshot,
  fetchHealth,
} from '../api'
import { formatBandwidth, formatBytes, formatNumber } from '../formatters'
import type {
  ACLFormState,
  ACLRuleView,
  AlertEvent,
  DashboardSnapshot,
  NodeStatus,
  RuntimeConfigStatus,
  TrafficStats,
} from '../types'

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

const createEmptyRuntimeConfigStatus = (): RuntimeConfigStatus => ({
  https_enabled: false,
  audit_log_enabled: false,
  audit_log_queryable: false,
  relay_admin_configured: false,
  relay_admin_available: false,
  allowed_origins: [],
  store_persistent: false,
})

const createEmptySnapshot = (): DashboardSnapshot => ({
  nodes: [],
  stats: [],
  clients: [],
  clientStatus: {
    configured: false,
    available: false,
  },
  aclRules: [],
  alerts: [],
  auditEvents: [],
  configStatus: createEmptyRuntimeConfigStatus(),
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

const describeError = (error: unknown): string => (error instanceof Error ? error.message : '未知错误')

export const useDashboardStore = defineStore('dashboard', () => {
  const snapshot = ref<DashboardSnapshot>(createEmptySnapshot())
  const selectedRegion = ref(ALL_REGIONS)
  const selectedNodeID = ref('')
  const healthStatus = ref('检测中')
  const lastRefreshAt = ref('')
  const isLoading = ref(false)
  const isSubmitting = ref(false)
  const errorMessage = ref('')
  const successMessage = ref('')
  const deletingNodeIDs = ref<Set<string>>(new Set())
  const disconnectingClientIDs = ref<Set<string>>(new Set())
  const deletingACLIDs = ref<Set<string>>(new Set())
  const acknowledgingAlertIDs = ref<Set<string>>(new Set())
  const aclForm = reactive<ACLFormState>(createEmptyACLForm())

  const healthStatusClass = computed(() => (healthStatus.value === 'ok' ? 'online' : 'offline'))
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

  const clientProxyCount = computed(() =>
    snapshot.value.clients.reduce((total, client) => total + client.proxy_count, 0),
  )
  const clientSessionCount = computed(() =>
    snapshot.value.clients.reduce((total, client) => total + client.sessions, 0),
  )
  const clientTrafficBytes = computed(() =>
    snapshot.value.clients.reduce((total, client) => total + client.bytes_in + client.bytes_out, 0),
  )

  const metrics = computed(() => [
    {
      label: '在线节点',
      value: `${onlineNodes.value.length}/${snapshot.value.nodes.length}`,
      detail: `${offlineNodes.value.length} 个离线节点`,
    },
    {
      label: '在线客户端',
      value: formatNumber(snapshot.value.clients.length),
      detail: `${clientProxyCount.value} 个代理 · ${clientSessionCount.value} 会话`,
    },
    {
      label: '实时带宽',
      value: formatBandwidth(aggregateStats.value.rx_bandwidth_bps + aggregateStats.value.tx_bandwidth_bps),
      detail: `${formatBytes(aggregateStats.value.rx_bytes + aggregateStats.value.tx_bytes + clientTrafficBytes.value)} 累计流量`,
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

  const regionSelectOptions = computed(() =>
    regionOptions.value.map((region) => ({ label: region, value: region })),
  )

  const filteredNodes = computed(() => {
    if (selectedRegion.value === ALL_REGIONS) {
      return snapshot.value.nodes
    }
    return snapshot.value.nodes.filter((node) => (node.region || '未分区') === selectedRegion.value)
  })

  const sortedACLRules = computed(() => [...snapshot.value.aclRules].sort((a, b) => a.priority - b.priority))
  const sortedClients = computed(() =>
    [...snapshot.value.clients].sort((a, b) => a.client_id.localeCompare(b.client_id)),
  )
  const recentAlerts = computed(() =>
    [...snapshot.value.alerts]
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .slice(0, 8),
  )
  const sortedAuditEvents = computed(() =>
    [...snapshot.value.auditEvents].sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()),
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

  const clientTrafficBars = computed(() => {
    const maxClientTraffic = Math.max(1, ...snapshot.value.clients.map((client) => client.bytes_in + client.bytes_out))
    return snapshot.value.clients.map((client) => {
      const totalBytes = client.bytes_in + client.bytes_out
      return {
        label: client.client_id,
        detail: `${formatBytes(totalBytes)} · ${client.proxy_count} 代理 · ${client.sessions} 会话`,
        rxPercent: Math.max(4, Math.round((client.bytes_in / maxClientTraffic) * 100)),
        txPercent: Math.max(4, Math.round((client.bytes_out / maxClientTraffic) * 100)),
      }
    })
  })

  const lastRefreshLabel = computed(() => {
    if (!lastRefreshAt.value) return '等待首次刷新'
    return `上次刷新 ${new Date(lastRefreshAt.value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
  })

  const setFeedback = (type: 'error' | 'success', message: string): void => {
    errorMessage.value = type === 'error' ? message : ''
    successMessage.value = type === 'success' ? message : ''
  }

  const mutateBusySet = (target: typeof deletingNodeIDs, id: string, busy: boolean): void => {
    const nextIDs = new Set(target.value)
    if (busy) {
      nextIDs.add(id)
    } else {
      nextIDs.delete(id)
    }
    target.value = nextIDs
  }

  const loadHealth = async (): Promise<void> => {
    try {
      healthStatus.value = await fetchHealth()
    } catch {
      healthStatus.value = '不可用'
    }
  }

  const loadSnapshot = async (token: string, options: { silent?: boolean } = {}): Promise<DashboardSnapshot> => {
    if (!token) return snapshot.value

    isLoading.value = true
    try {
      // 所有核心面板共享同一次快照刷新，避免组件拆分后重复打 API。
      snapshot.value = await fetchDashboardSnapshot(token)
      lastRefreshAt.value = new Date().toISOString()
      if (!options.silent) {
        setFeedback('success', '数据已刷新')
      }
      return snapshot.value
    } catch (error) {
      setFeedback('error', `刷新失败：${describeError(error)}`)
      throw error
    } finally {
      isLoading.value = false
    }
  }

  const refreshDashboard = async (token: string): Promise<DashboardSnapshot> => {
    const [, nextSnapshot] = await Promise.all([loadHealth(), loadSnapshot(token)])
    return nextSnapshot
  }

  const resetSnapshot = (): void => {
    snapshot.value = createEmptySnapshot()
    selectedRegion.value = ALL_REGIONS
    selectedNodeID.value = ''
  }

  const selectNode = (nodeID: string): void => {
    selectedNodeID.value = selectedNodeID.value === nodeID ? '' : nodeID
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

  const submitACLRule = async (token: string): Promise<void> => {
    if (!aclForm.source || !aclForm.target) {
      setFeedback('error', 'ACL 来源和目标不能为空')
      return
    }

    isSubmitting.value = true
    try {
      await createACLRule(token, buildACLRule())
      Object.assign(aclForm, createEmptyACLForm())
      setFeedback('success', 'ACL 规则已添加')
      await loadSnapshot(token)
    } catch (error) {
      setFeedback('error', `添加 ACL 失败：${describeError(error)}`)
      throw error
    } finally {
      isSubmitting.value = false
    }
  }

  const removeACLRule = async (token: string, ruleID: string): Promise<void> => {
    mutateBusySet(deletingACLIDs, ruleID, true)
    try {
      await deleteACLRule(token, ruleID)
      setFeedback('success', 'ACL 规则已删除')
      await loadSnapshot(token)
    } catch (error) {
      setFeedback('error', `删除 ACL 失败：${describeError(error)}`)
      throw error
    } finally {
      mutateBusySet(deletingACLIDs, ruleID, false)
    }
  }

  const removeNode = async (token: string, nodeID: string): Promise<void> => {
    mutateBusySet(deletingNodeIDs, nodeID, true)
    try {
      await deleteNode(token, nodeID)
      setFeedback('success', '节点已删除')
      await loadSnapshot(token)
    } catch (error) {
      setFeedback('error', `删除节点失败：${describeError(error)}`)
      throw error
    } finally {
      mutateBusySet(deletingNodeIDs, nodeID, false)
    }
  }

  const disconnectRelayClient = async (token: string, clientID: string): Promise<void> => {
    mutateBusySet(disconnectingClientIDs, clientID, true)
    try {
      await disconnectClient(token, clientID)
      setFeedback('success', '客户端已断开')
      await loadSnapshot(token)
    } catch (error) {
      setFeedback('error', `断开客户端失败：${describeError(error)}`)
      throw error
    } finally {
      mutateBusySet(disconnectingClientIDs, clientID, false)
    }
  }

  const ackAlert = async (token: string, alert: AlertEvent, ackedBy: string): Promise<void> => {
    mutateBusySet(acknowledgingAlertIDs, alert.id, true)
    try {
      await acknowledgeAlert(token, alert.id, ackedBy)
      setFeedback('success', '告警已确认')
      await loadSnapshot(token)
    } catch (error) {
      setFeedback('error', `确认告警失败：${describeError(error)}`)
      throw error
    } finally {
      mutateBusySet(acknowledgingAlertIDs, alert.id, false)
    }
  }

  return {
    snapshot,
    selectedRegion,
    selectedNodeID,
    healthStatus,
    lastRefreshAt,
    isLoading,
    isSubmitting,
    errorMessage,
    successMessage,
    deletingNodeIDs,
    disconnectingClientIDs,
    deletingACLIDs,
    acknowledgingAlertIDs,
    aclForm,
    healthStatusClass,
    onlineNodes,
    offlineNodes,
    unackedAlerts,
    aggregateStats,
    metrics,
    regionOptions,
    regionSelectOptions,
    filteredNodes,
    sortedACLRules,
    sortedClients,
    recentAlerts,
    sortedAuditEvents,
    trafficBars,
    clientTrafficBars,
    lastRefreshLabel,
    setFeedback,
    loadHealth,
    loadSnapshot,
    refreshDashboard,
    resetSnapshot,
    selectNode,
    nodePosition,
    submitACLRule,
    removeACLRule,
    removeNode,
    disconnectRelayClient,
    ackAlert,
  }
})
