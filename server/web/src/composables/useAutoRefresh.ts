import { onMounted, onUnmounted } from 'vue'

interface AutoRefreshOptions {
  intervalMs: number
  refresh: () => Promise<void>
}

export const useAutoRefresh = ({ intervalMs, refresh }: AutoRefreshOptions): void => {
  let refreshTimer: number | undefined

  const clearRefreshTimer = (): void => {
    if (refreshTimer !== undefined) {
      window.clearInterval(refreshTimer)
      refreshTimer = undefined
    }
  }

  const runRefresh = async (): Promise<void> => {
    if (document.visibilityState === 'hidden') return
    await refresh()
  }

  const startRefreshTimer = (): void => {
    clearRefreshTimer()
    refreshTimer = window.setInterval(() => {
      void runRefresh()
    }, intervalMs)
  }

  const handleVisibilityChange = (): void => {
    if (document.visibilityState === 'hidden') {
      clearRefreshTimer()
      return
    }
    void runRefresh()
    startRefreshTimer()
  }

  onMounted(() => {
    void runRefresh()
    startRefreshTimer()
    document.addEventListener('visibilitychange', handleVisibilityChange)
  })

  onUnmounted(() => {
    clearRefreshTimer()
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  })
}
