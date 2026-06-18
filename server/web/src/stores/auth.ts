import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { clearStoredToken, login, readStoredToken } from '../api'
import type { User } from '../types'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(readStoredToken())
  const user = ref<User | null>(null)
  const isSubmitting = ref(false)
  const errorMessage = ref('')
  const sessionMessage = ref('')

  const isAuthenticated = computed(() => token.value.length > 0)
  const activeUserLabel = computed(() => {
    if (!user.value) return '已认证'
    return `${user.value.username} · ${user.value.role}`
  })

  const loginWithPassword = async (username: string, password: string): Promise<void> => {
    if (!username || !password) {
      errorMessage.value = '请输入用户名和密码'
      return
    }

    isSubmitting.value = true
    errorMessage.value = ''
    try {
      const response = await login(username, password)
      token.value = response.token
      user.value = response.user
      sessionMessage.value = '登录成功'
    } catch (error) {
      errorMessage.value = `登录失败：${error instanceof Error ? error.message : '未知错误'}`
      throw error
    } finally {
      isSubmitting.value = false
    }
  }

  const setUserFromSnapshot = (users: User[]): void => {
    if (!user.value && users.length > 0) {
      user.value = users[0]
    }
  }

  const expireSession = (message = '会话已过期，请重新登录'): void => {
    clearStoredToken()
    token.value = ''
    user.value = null
    sessionMessage.value = message
  }

  const logout = (): void => {
    clearStoredToken()
    token.value = ''
    user.value = null
    sessionMessage.value = '已退出登录'
  }

  return {
    token,
    user,
    isSubmitting,
    errorMessage,
    sessionMessage,
    isAuthenticated,
    activeUserLabel,
    loginWithPassword,
    setUserFromSnapshot,
    expireSession,
    logout,
  }
})
