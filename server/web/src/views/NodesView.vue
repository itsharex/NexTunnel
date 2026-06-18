<template>
  <div class="page-stack">
    <MetricGrid />

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
          <n-select
            v-model:value="dashboard.selectedRegion"
            class="region-select"
            :options="dashboard.regionSelectOptions"
          />
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
              v-for="node in dashboard.filteredNodes"
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
                <ConfirmButton
                  size="small"
                  type="error"
                  secondary
                  :loading="dashboard.deletingNodeIDs.has(node.node_id)"
                  :message="`确认删除节点 ${node.node_id}？`"
                  @confirm="handleRemoveNode(node.node_id)"
                >
                  删除
                </ConfirmButton>
              </td>
            </tr>
          </tbody>
        </table>
        <n-empty
          v-if="dashboard.filteredNodes.length === 0"
          description="暂无节点"
        />
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NCard, NEmpty, NSelect, NTag } from 'naive-ui'
import ConfirmButton from '../components/common/ConfirmButton.vue'
import MetricGrid from '../components/common/MetricGrid.vue'
import { formatBytes, formatRelativeTime, statusLabel } from '../formatters'
import { useAuthStore } from '../stores/auth'
import { useDashboardStore } from '../stores/dashboard'

const auth = useAuthStore()
const dashboard = useDashboardStore()

const handleRemoveNode = async (nodeID: string): Promise<void> => {
  await dashboard.removeNode(auth.token, nodeID)
}
</script>
