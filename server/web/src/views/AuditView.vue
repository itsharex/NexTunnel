<template>
  <div class="page-stack">
    <n-card
      class="panel"
      :bordered="false"
    >
      <template #header>
        <div class="panel-header">
          <div>
            <p class="eyebrow">
              审计
            </p>
            <h2>操作日志</h2>
          </div>
          <n-tag
            round
            size="small"
            :bordered="false"
          >
            {{ filteredAuditEvents.length }} 条记录
          </n-tag>
        </div>
      </template>

      <div class="audit-filter-grid">
        <n-input
          v-model:value="actorKeyword"
          clearable
          placeholder="搜索用户、资源 ID 或来源 IP"
        />
        <n-select
          v-model:value="resourceFilter"
          :options="resourceOptions"
        />
        <n-select
          v-model:value="actionFilter"
          :options="actionOptions"
        />
        <n-select
          v-model:value="resultFilter"
          :options="resultOptions"
        />
        <n-button
          secondary
          :loading="dashboard.isLoading"
          @click="handleRefresh"
        >
          刷新
        </n-button>
      </div>
    </n-card>

    <n-card
      class="panel"
      :bordered="false"
    >
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>时间</th>
              <th>操作者</th>
              <th>动作</th>
              <th>资源</th>
              <th>结果</th>
              <th>详情</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="event in filteredAuditEvents"
              :key="`${event.timestamp}-${event.actor}-${event.action}-${event.resource}-${event.resource_id}`"
            >
              <td>{{ formatDateTime(event.timestamp) }}</td>
              <td>
                <strong>{{ event.actor || 'system' }}</strong>
              </td>
              <td>{{ actionLabel(event.action) }}</td>
              <td>
                <div class="audit-resource">
                  <strong>{{ event.resource }}</strong>
                  <span>{{ event.resource_id || '全局' }}</span>
                </div>
              </td>
              <td>
                <n-tag
                  round
                  size="small"
                  :type="resultTagType(event.result)"
                  :bordered="false"
                >
                  {{ resultLabel(event.result) }}
                </n-tag>
              </td>
              <td>{{ eventDetails(event) }}</td>
            </tr>
          </tbody>
        </table>
        <n-empty
          v-if="filteredAuditEvents.length === 0"
          description="暂无匹配审计日志"
        />
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { NButton, NCard, NEmpty, NInput, NSelect, NTag, type SelectOption } from 'naive-ui'
import { formatDateTime } from '../formatters'
import { useAuthStore } from '../stores/auth'
import { useDashboardStore } from '../stores/dashboard'
import type { AuditEvent } from '../types'

type TagType = 'default' | 'error' | 'success' | 'warning' | 'info'

const ALL_FILTER = '__all__'
const MAX_DETAILS_LENGTH = 96

const auth = useAuthStore()
const dashboard = useDashboardStore()
const actorKeyword = ref('')
const resourceFilter = ref(ALL_FILTER)
const actionFilter = ref(ALL_FILTER)
const resultFilter = ref(ALL_FILTER)

const resourceOptions = computed<SelectOption[]>(() => [
  { label: '全部资源', value: ALL_FILTER },
  ...Array.from(new Set(dashboard.sortedAuditEvents.map((event) => event.resource).filter(Boolean)))
    .sort()
    .map((resource) => ({ label: resource, value: resource })),
])

const actionOptions: SelectOption[] = [
  { label: '全部动作', value: ALL_FILTER },
  { label: '创建', value: 'create' },
  { label: '更新', value: 'update' },
  { label: '删除', value: 'delete' },
  { label: '登录', value: 'login' },
  { label: '访问', value: 'access' },
  { label: '拒绝', value: 'denied' },
]

const resultOptions: SelectOption[] = [
  { label: '全部结果', value: ALL_FILTER },
  { label: '成功', value: 'success' },
  { label: '拒绝', value: 'denied' },
  { label: '错误', value: 'error' },
]

const filteredAuditEvents = computed(() => {
  const keyword = actorKeyword.value.trim().toLowerCase()
  return dashboard.sortedAuditEvents.filter((event) => {
    if (resourceFilter.value !== ALL_FILTER && event.resource !== resourceFilter.value) return false
    if (actionFilter.value !== ALL_FILTER && event.action !== actionFilter.value) return false
    if (resultFilter.value !== ALL_FILTER && event.result !== resultFilter.value) return false
    if (!keyword) return true
    return [event.actor, event.resource_id, event.source_ip]
      .some((value) => (value ?? '').toLowerCase().includes(keyword))
  })
})

const actionLabel = (action: string): string => {
  const labels: Record<string, string> = {
    create: '创建',
    update: '更新',
    delete: '删除',
    login: '登录',
    logout: '退出',
    access: '访问',
    denied: '拒绝',
  }
  return labels[action] ?? action
}

const resultLabel = (result: string): string => {
  if (result === 'success') return '成功'
  if (result === 'denied') return '拒绝'
  if (result === 'error') return '错误'
  return result || '未知'
}

const resultTagType = (result: string): TagType => {
  if (result === 'success') return 'success'
  if (result === 'denied') return 'warning'
  if (result === 'error') return 'error'
  return 'default'
}

const eventDetails = (event: AuditEvent): string => {
  const details = Object.entries(event.details ?? {}).map(([key, value]) => `${key}=${value}`)
  if (event.source_ip) {
    details.unshift(`ip=${event.source_ip}`)
  }
  const joinedDetails = details.join(' · ')
  if (joinedDetails.length <= MAX_DETAILS_LENGTH) return joinedDetails || '无'
  return `${joinedDetails.slice(0, MAX_DETAILS_LENGTH)}...`
}

const handleRefresh = async (): Promise<void> => {
  await dashboard.loadSnapshot(auth.token, { silent: true })
}
</script>
