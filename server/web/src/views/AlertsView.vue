<template>
  <div class="page-stack">
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
          v-for="alert in dashboard.recentAlerts"
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
            :loading="dashboard.acknowledgingAlertIDs.has(alert.id)"
            :disabled="alert.acked"
            @click="handleAckAlert(alert)"
          >
            {{ alert.acked ? '已确认' : '确认' }}
          </n-button>
        </div>
        <n-empty
          v-if="dashboard.recentAlerts.length === 0"
          description="暂无告警"
        />
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NButton, NCard, NEmpty, NTag } from 'naive-ui'
import { formatDateTime } from '../formatters'
import { useAuthStore } from '../stores/auth'
import { useDashboardStore } from '../stores/dashboard'
import type { AlertEvent } from '../types'

type TagType = 'default' | 'error' | 'success' | 'warning' | 'info'

const auth = useAuthStore()
const dashboard = useDashboardStore()

const severityTagType = (level: string): TagType => {
  const normalizedLevel = level.toLowerCase()
  if (normalizedLevel === 'critical') return 'error'
  if (normalizedLevel === 'warning') return 'warning'
  return 'info'
}

const handleAckAlert = async (alert: AlertEvent): Promise<void> => {
  await dashboard.ackAlert(auth.token, alert, auth.user?.username ?? 'dashboard')
}
</script>
