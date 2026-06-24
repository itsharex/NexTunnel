<template>
  <div class="page-stack">
    <MetricGrid />

    <n-card
      class="panel client-status-panel"
      :bordered="false"
    >
      <template #header>
        <div class="panel-header">
          <div>
            <p class="eyebrow">
              客户端
            </p>
            <h2>Relay 连接监控</h2>
          </div>
          <n-tag
            round
            size="small"
            :type="clientSourceTagType"
            :bordered="false"
          >
            {{ clientSourceLabel }}
          </n-tag>
        </div>
      </template>

      <div class="client-summary-grid">
        <div class="detail-row">
          <div>
            <strong>{{ dashboard.snapshot.clients.length }}</strong>
            <span>在线客户端</span>
          </div>
        </div>
        <div class="detail-row">
          <div>
            <strong>{{ proxyCount }}</strong>
            <span>活跃代理</span>
          </div>
        </div>
        <div class="detail-row">
          <div>
            <strong>{{ sessionCount }}</strong>
            <span>累计会话</span>
          </div>
        </div>
        <div class="detail-row">
          <div>
            <strong>{{ formatBytes(trafficBytes) }}</strong>
            <span>客户端流量</span>
          </div>
        </div>
      </div>

      <n-alert
        v-if="dashboard.snapshot.clientStatus.error"
        class="feedback-message"
        type="warning"
        :bordered="false"
      >
        {{ dashboard.snapshot.clientStatus.error }}
      </n-alert>
    </n-card>

    <n-card
      class="panel"
      :bordered="false"
    >
      <template #header>
        <div class="panel-header">
          <div>
            <p class="eyebrow">
              实时连接
            </p>
            <h2>客户端列表</h2>
          </div>
          <n-tag
            round
            size="small"
            :bordered="false"
          >
            5 秒刷新
          </n-tag>
        </div>
      </template>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>客户端</th>
              <th>远端地址</th>
              <th>代理</th>
              <th>会话</th>
              <th>流量</th>
              <th>最后活跃</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="client in dashboard.sortedClients"
              :key="client.client_id"
            >
              <td>
                <n-tag
                  round
                  size="small"
                  type="success"
                  :bordered="false"
                >
                  在线
                </n-tag>
                <strong>{{ client.client_id }}</strong>
              </td>
              <td>{{ client.remote_addr || '未知' }}</td>
              <td>
                <div class="client-proxy-stack">
                  <strong>{{ client.proxy_count }}</strong>
                  <span>{{ proxyNames(client) }}</span>
                </div>
              </td>
              <td>{{ formatNumber(client.sessions) }}</td>
              <td>{{ formatBytes(client.bytes_in + client.bytes_out) }}</td>
              <td>{{ formatRelativeTime(client.last_seen) }}</td>
              <td>
                <ConfirmButton
                  size="small"
                  type="error"
                  secondary
                  :disabled="!dashboard.snapshot.clientStatus.available"
                  :loading="dashboard.disconnectingClientIDs.has(client.client_id)"
                  :message="`确认断开客户端 ${client.client_id}？`"
                  @confirm="handleDisconnectClient(client.client_id)"
                >
                  断开
                </ConfirmButton>
              </td>
            </tr>
          </tbody>
        </table>
        <n-empty
          v-if="dashboard.sortedClients.length === 0"
          description="暂无在线客户端"
        />
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NAlert, NCard, NEmpty, NTag } from 'naive-ui'
import ConfirmButton from '../components/common/ConfirmButton.vue'
import MetricGrid from '../components/common/MetricGrid.vue'
import { useAutoRefresh } from '../composables/useAutoRefresh'
import { formatBytes, formatNumber, formatRelativeTime } from '../formatters'
import { useAuthStore } from '../stores/auth'
import { useDashboardStore } from '../stores/dashboard'
import type { ClientSnapshot } from '../types'

type TagType = 'default' | 'error' | 'success' | 'warning' | 'info'

const CLIENT_REFRESH_INTERVAL_MS = 5_000
const MAX_PROXY_NAMES = 3

const auth = useAuthStore()
const dashboard = useDashboardStore()

const proxyCount = computed(() =>
  dashboard.snapshot.clients.reduce((total, client) => total + client.proxy_count, 0),
)
const sessionCount = computed(() =>
  dashboard.snapshot.clients.reduce((total, client) => total + client.sessions, 0),
)
const trafficBytes = computed(() =>
  dashboard.snapshot.clients.reduce((total, client) => total + client.bytes_in + client.bytes_out, 0),
)
const clientSourceTagType = computed<TagType>(() => {
  if (dashboard.snapshot.clientStatus.available) return 'success'
  if (dashboard.snapshot.clientStatus.configured) return 'warning'
  return 'default'
})
const clientSourceLabel = computed(() => {
  if (dashboard.snapshot.clientStatus.available) return 'Relay 管理 API 已连接'
  if (dashboard.snapshot.clientStatus.configured) return 'Relay 管理 API 不可用'
  return '未配置 Relay 管理 API'
})

const proxyNames = (client: ClientSnapshot): string => {
  if (client.proxies.length === 0) return '未注册代理'
  const names = client.proxies.slice(0, MAX_PROXY_NAMES).map((proxy) => proxy.proxy_name)
  const suffix = client.proxies.length > MAX_PROXY_NAMES ? ` +${client.proxies.length - MAX_PROXY_NAMES}` : ''
  return `${names.join('、')}${suffix}`
}

const handleDisconnectClient = async (clientID: string): Promise<void> => {
  await dashboard.disconnectRelayClient(auth.token, clientID)
}

const refreshClients = async (): Promise<void> => {
  await dashboard.loadSnapshot(auth.token, { silent: true })
}

useAutoRefresh({
  intervalMs: CLIENT_REFRESH_INTERVAL_MS,
  refresh: refreshClients,
})
</script>
