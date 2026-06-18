<template>
  <section class="login-surface">
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
          :loading="auth.isSubmitting"
        >
          {{ auth.isSubmitting ? '登录中' : '登录' }}
        </n-button>
      </n-form>

      <n-alert
        v-if="auth.errorMessage || auth.sessionMessage"
        :type="auth.errorMessage ? 'error' : 'info'"
        :bordered="false"
      >
        {{ auth.errorMessage || auth.sessionMessage }}
      </n-alert>
    </n-card>
  </section>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { useRouter } from 'vue-router'
import { NAlert, NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import { useAuthStore } from '../stores/auth'
import { useDashboardStore } from '../stores/dashboard'
import blackLogo from '@shared-logo/black-logo.png'

const router = useRouter()
const auth = useAuthStore()
const dashboard = useDashboardStore()
const loginForm = reactive({ username: 'admin', password: '' })

const handleLogin = async (): Promise<void> => {
  await auth.loginWithPassword(loginForm.username, loginForm.password)
  loginForm.password = ''
  const snapshot = await dashboard.refreshDashboard(auth.token)
  auth.setUserFromSnapshot(snapshot.users)
  await router.push({ name: 'overview' })
}
</script>
