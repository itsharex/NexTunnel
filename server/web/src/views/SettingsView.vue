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
import { NCard, NEmpty, NTag } from 'naive-ui'
import { useAuthStore } from '../stores/auth'
import { useDashboardStore } from '../stores/dashboard'

type TagType = 'default' | 'error' | 'success' | 'warning' | 'info'

const auth = useAuthStore()
const dashboard = useDashboardStore()

const roleTagType = (role: string): TagType => {
  if (role === 'admin') return 'error'
  if (role === 'operator') return 'warning'
  return 'info'
}
</script>
