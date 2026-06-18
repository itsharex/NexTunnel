<template>
  <div class="page-stack">
    <MetricGrid />

    <section class="layout-grid">
      <n-card
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
              v-model:value="dashboard.selectedRegion"
              class="region-select"
              :options="dashboard.regionSelectOptions"
            />
          </div>
        </template>

        <n-skeleton
          v-if="dashboard.isLoading && dashboard.snapshot.nodes.length === 0"
          height="280px"
          :sharp="false"
        />
        <template v-else>
          <div
            class="node-map"
            aria-label="节点区域地图"
          >
            <span
              v-for="(node, index) in dashboard.filteredNodes"
              :key="node.node_id"
              class="map-dot"
              :class="node.online ? 'online' : 'offline'"
              :style="dashboard.nodePosition(node, index)"
              :title="`${node.node_id} · ${node.region}`"
            />
          </div>

          <div class="node-table compact">
            <button
              v-for="node in dashboard.filteredNodes.slice(0, 6)"
              :key="node.node_id"
              class="node-row"
              type="button"
              :class="{ selected: dashboard.selectedNodeID === node.node_id }"
              @click="dashboard.selectNode(node.node_id)"
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
          <n-empty
            v-if="dashboard.filteredNodes.length === 0"
            description="暂无节点"
          />
        </template>
      </n-card>

      <n-card
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
              {{ dashboard.unackedAlerts.length }} 未确认
            </n-tag>
          </div>
        </template>

        <div class="alert-list">
          <div
            v-for="alert in dashboard.recentAlerts.slice(0, 5)"
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
          </div>
          <n-empty
            v-if="dashboard.recentAlerts.length === 0"
            description="暂无告警"
          />
        </div>
      </n-card>
    </section>
  </div>
</template>

<script setup lang="ts">
import { NCard, NEmpty, NSkeleton, NSelect, NTag } from 'naive-ui'
import MetricGrid from '../components/common/MetricGrid.vue'
import { formatDateTime, formatRelativeTime, statusLabel } from '../formatters'
import { useDashboardStore } from '../stores/dashboard'

type TagType = 'default' | 'error' | 'success' | 'warning' | 'info'

const dashboard = useDashboardStore()

const severityTagType = (level: string): TagType => {
  const normalizedLevel = level.toLowerCase()
  if (normalizedLevel === 'critical') return 'error'
  if (normalizedLevel === 'warning') return 'warning'
  return 'info'
}
</script>
