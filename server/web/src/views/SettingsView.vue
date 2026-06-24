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
              系统设置
            </p>
            <h2>账户与控制台状态</h2>
          </div>
          <n-tag
            round
            size="small"
            :bordered="false"
          >
            {{ dashboard.snapshot.users.length }} 个用户
          </n-tag>
        </div>
      </template>

      <div class="settings-list">
        <div class="settings-row">
          <div>
            <strong>当前会话</strong>
            <span>{{ auth.activeUserLabel }}</span>
          </div>
          <n-tag
            round
            type="success"
            :bordered="false"
          >
            已认证
          </n-tag>
        </div>
        <div class="settings-row">
          <div>
            <strong>数据刷新</strong>
            <span>页面可见时每 30 秒自动刷新，恢复可见后立即刷新。</span>
          </div>
          <n-tag
            round
            type="info"
            :bordered="false"
          >
            自动
          </n-tag>
        </div>
        <div
          v-for="item in runtimeConfigRows"
          :key="item.label"
          class="settings-row"
        >
          <div>
            <strong>{{ item.label }}</strong>
            <span>{{ item.detail }}</span>
          </div>
          <n-tag
            round
            :type="item.type"
            :bordered="false"
          >
            {{ item.state }}
          </n-tag>
        </div>
      </div>
    </n-card>

    <n-card
      class="panel"
      :bordered="false"
    >
      <template #header>
        <div class="panel-header">
          <div>
            <p class="eyebrow">
              用户
            </p>
            <h2>Dashboard 用户列表</h2>
          </div>
        </div>
      </template>

      <div class="user-list">
        <div
          v-for="user in dashboard.snapshot.users"
          :key="user.id || user.username"
          class="user-row"
        >
          <div>
            <strong>{{ user.username }}</strong>
            <span>{{ user.email || '未配置邮箱' }}</span>
          </div>
          <n-tag
            round
            :type="roleTagType(user.role)"
            :bordered="false"
          >
            {{ user.role }}
          </n-tag>
        </div>
        <n-empty
          v-if="dashboard.snapshot.users.length === 0"
          description="暂无用户数据"
        />
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NCard, NEmpty, NTag } from 'naive-ui'
import { useAuthStore } from '../stores/auth'
import { useDashboardStore } from '../stores/dashboard'

type TagType = 'default' | 'error' | 'success' | 'warning' | 'info'
interface RuntimeConfigRow {
  label: string
  detail: string
  state: string
  type: TagType
}

const auth = useAuthStore()
const dashboard = useDashboardStore()

const configStatus = computed(() => dashboard.snapshot.configStatus)

const runtimeConfigRows = computed<RuntimeConfigRow[]>(() => [
  {
    label: 'Dashboard HTTPS',
    detail: configStatus.value.https_enabled ? '已由 Dashboard 进程直接启用 TLS。' : '未在 Dashboard 进程内启用，生产环境应由 Nginx/OpenResty 或证书参数补齐 HTTPS。',
    state: configStatus.value.https_enabled ? '已启用' : '未启用',
    type: configStatus.value.https_enabled ? 'success' : 'warning',
  },
  {
    label: 'Relay Admin API',
    detail: configStatus.value.relay_admin_error || configStatus.value.relay_admin_url || '未配置 Relay 管理 API 地址。',
    state: configStatus.value.relay_admin_available ? '可用' : configStatus.value.relay_admin_configured ? '异常' : '未配置',
    type: configStatus.value.relay_admin_available ? 'success' : configStatus.value.relay_admin_configured ? 'warning' : 'default',
  },
  {
    label: 'CORS 白名单',
    detail: configStatus.value.allowed_origins.length > 0 ? configStatus.value.allowed_origins.join('，') : '未配置允许来源。',
    state: `${configStatus.value.allowed_origins.length} 项`,
    type: configStatus.value.allowed_origins.length > 0 ? 'info' : 'warning',
  },
  {
    label: '审计日志',
    detail: configStatus.value.audit_log_error || configStatus.value.audit_log_path || '未启用审计日志文件。',
    state: configStatus.value.audit_log_enabled && configStatus.value.audit_log_queryable ? '可查询' : configStatus.value.audit_log_enabled ? '不可查询' : '未启用',
    type: configStatus.value.audit_log_enabled && configStatus.value.audit_log_queryable ? 'success' : 'warning',
  },
  {
    label: '存储路径',
    detail: configStatus.value.store_path || '当前使用内存存储，重启后运行数据不会保留。',
    state: configStatus.value.store_persistent ? '持久化' : '内存',
    type: configStatus.value.store_persistent ? 'success' : 'warning',
  },
  {
    label: '当前版本',
    detail: configStatus.value.version || '随当前 Dashboard 二进制发布。',
    state: configStatus.value.version || '内置',
    type: 'info',
  },
])

const roleTagType = (role: string): TagType => {
  if (role === 'admin') return 'error'
  if (role === 'operator') return 'warning'
  return 'info'
}
</script>
