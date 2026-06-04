import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  GetTunnels,
  CreateTunnel,
  DeleteTunnel,
  StartTunnel,
  StopTunnel,
  ConnectServer,
  DisconnectServer,
  GetConnectionStatus,
  GetTrafficStats,
  GetP2PStatus,
  GetNATType,
  type ServerConfigInput,
} from '../api/app'

export interface Tunnel {
  id: string
  name: string
  proxy_type: string
  local_addr: string
  local_port: number
  remote_port: number
  status: string
}

export interface CreateTunnelInput {
  name: string
  proxy_type: string
  local_addr: string
  local_port: number
  remote_port: number
}

export const useTunnelStore = defineStore('tunnels', () => {
  const tunnels = ref<Tunnel[]>([])
  const connectionStatus = ref<string>('disconnected')
  const serverAddr = ref<string>('127.0.0.1:7000')
  const authToken = ref<string>('')
  const lastError = ref<string>('')
  const isConnecting = ref<boolean>(false)
  const busyTunnelIds = ref<Set<string>>(new Set())
  const trafficStats = ref<{ bytes_in: number; bytes_out: number; tunnels: number }>({
    bytes_in: 0,
    bytes_out: 0,
    tunnels: 0,
  })
  const p2pStatus = ref<string>('')
  const natType = ref<string>('')

  const tunnelCount = computed(() => tunnels.value.length)
  const isConnected = computed(() => connectionStatus.value === 'connected')

  // extractErrorMessage 将 Wails/JS 异常统一转换为可展示的短错误信息。
  const extractErrorMessage = (error: unknown): string => {
    if (error instanceof Error) return error.message
    if (typeof error === 'string') return error
    return 'operation failed'
  }

  // setTunnelBusy 使用替换 Set 的方式触发 Vue 响应式更新。
  const setTunnelBusy = (id: string, busy: boolean): void => {
    const next = new Set(busyTunnelIds.value)
    if (busy) {
      next.add(id)
    } else {
      next.delete(id)
    }
    busyTunnelIds.value = next
  }

  async function loadTunnels() {
    try {
      tunnels.value = (await GetTunnels()) as Tunnel[]
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      console.error('Failed to load tunnels:', e)
    }
  }

  async function createTunnel(input: CreateTunnelInput) {
    try {
      const t = (await CreateTunnel(input)) as Tunnel
      tunnels.value.push(t)
      return t
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      console.error('Failed to create tunnel:', e)
      throw e
    }
  }

  async function deleteTunnel(id: string) {
    try {
      await DeleteTunnel(id)
      tunnels.value = tunnels.value.filter((t) => t.id !== id)
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      console.error('Failed to delete tunnel:', e)
      throw e
    }
  }

  const connectServer = async (input?: ServerConfigInput): Promise<void> => {
    isConnecting.value = true
    lastError.value = ''
    try {
      const cfg = input ?? { server_addr: serverAddr.value, auth_token: authToken.value }
      serverAddr.value = cfg.server_addr
      authToken.value = cfg.auth_token
      await ConnectServer(cfg)
      await refreshStatus()
      await loadTunnels()
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    } finally {
      isConnecting.value = false
    }
  }

  const disconnectServer = async (): Promise<void> => {
    lastError.value = ''
    try {
      await DisconnectServer()
      await refreshStatus()
      await loadTunnels()
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    }
  }

  const startTunnel = async (id: string): Promise<void> => {
    setTunnelBusy(id, true)
    lastError.value = ''
    try {
      await StartTunnel(id)
      await loadTunnels()
      await refreshStatus()
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    } finally {
      setTunnelBusy(id, false)
    }
  }

  const stopTunnel = async (id: string): Promise<void> => {
    setTunnelBusy(id, true)
    lastError.value = ''
    try {
      await StopTunnel(id)
      await loadTunnels()
      await refreshStatus()
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    } finally {
      setTunnelBusy(id, false)
    }
  }

  async function refreshStatus() {
    try {
      connectionStatus.value = await GetConnectionStatus()
      trafficStats.value = (await GetTrafficStats()) as typeof trafficStats.value
      p2pStatus.value = await GetP2PStatus()
      natType.value = await GetNATType()
    } catch {
      connectionStatus.value = 'disconnected'
    }
  }

  return {
    tunnels,
    connectionStatus,
    serverAddr,
    authToken,
    lastError,
    isConnecting,
    busyTunnelIds,
    trafficStats,
    p2pStatus,
    natType,
    tunnelCount,
    isConnected,
    loadTunnels,
    createTunnel,
    deleteTunnel,
    connectServer,
    disconnectServer,
    startTunnel,
    stopTunnel,
    refreshStatus,
  }
})
