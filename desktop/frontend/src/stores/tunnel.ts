import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  ApplyVirtualNetwork,
  GetTunnels,
  CreateTunnel,
  DeleteTunnel,
  DetectNAT,
  StartTunnel,
  StopTunnel,
  ConnectServer,
  DisconnectServer,
  GetConnectionStatus,
  GetRuntimeStatus,
  GetServerSettings,
  GetTrafficStats,
  GetP2PStatus,
  GetNATType,
  ResetVirtualNetwork,
  SaveServerSettings,
  type NATDetectionInfo,
  type RuntimeStatus,
  type ServerConfigInput,
  type ServerSettings,
  type VirtualNetworkState,
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
  const serverSettings = ref<ServerSettings>({
    relay_addr: '127.0.0.1:7000',
    relay_token: '',
    control_plane_url: '',
    control_plane_token: '',
    stun_server: 'stun.l.google.com:19302',
    stun_alt_server: 'stun.l.google.com:19302',
  })
  const lastError = ref<string>('')
  const isConnecting = ref<boolean>(false)
  const isApplyingNetwork = ref<boolean>(false)
  const isDetectingNAT = ref<boolean>(false)
  const busyTunnelIds = ref<Set<string>>(new Set())
  const trafficStats = ref<{ bytes_in: number; bytes_out: number; tunnels: number }>({
    bytes_in: 0,
    bytes_out: 0,
    tunnels: 0,
  })
  const p2pStatus = ref<string>('')
  const natType = ref<string>('')
  const runtimeStatus = ref<RuntimeStatus | null>(null)
  const virtualNetwork = ref<VirtualNetworkState | null>(null)
  const lastNATDetection = ref<NATDetectionInfo | null>(null)

  const tunnelCount = computed(() => tunnels.value.length)
  const isConnected = computed(() => connectionStatus.value === 'connected')
  const activeTunnelCount = computed(() =>
    tunnels.value.filter((tunnel) => tunnel.status === 'active' || tunnel.status === 'running').length,
  )

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

  const syncSettingsToRelayForm = (settings: ServerSettings): void => {
    serverAddr.value = settings.relay_addr
    authToken.value = settings.relay_token
  }

  const loadServerSettings = async (): Promise<void> => {
    try {
      const settings = await GetServerSettings()
      serverSettings.value = settings
      syncSettingsToRelayForm(settings)
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    }
  }

  const saveServerSettings = async (settings: ServerSettings): Promise<void> => {
    lastError.value = ''
    try {
      await SaveServerSettings(settings)
      serverSettings.value = { ...settings }
      syncSettingsToRelayForm(settings)
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    }
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
      runtimeStatus.value = await GetRuntimeStatus()
      virtualNetwork.value = runtimeStatus.value.virtual_network
      if (runtimeStatus.value.last_error) {
        lastError.value = runtimeStatus.value.last_error
      }
    } catch {
      connectionStatus.value = 'disconnected'
    }
  }

  const refreshRuntimeStatus = async (): Promise<void> => {
    try {
      runtimeStatus.value = await GetRuntimeStatus()
      connectionStatus.value = runtimeStatus.value.connection_status
      p2pStatus.value = runtimeStatus.value.p2p_status
      natType.value = runtimeStatus.value.nat_type
      trafficStats.value = runtimeStatus.value.traffic_stats
      virtualNetwork.value = runtimeStatus.value.virtual_network
      lastError.value = runtimeStatus.value.last_error
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    }
  }

  const applyVirtualNetwork = async (): Promise<void> => {
    isApplyingNetwork.value = true
    lastError.value = ''
    try {
      virtualNetwork.value = await ApplyVirtualNetwork()
      await refreshRuntimeStatus()
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    } finally {
      isApplyingNetwork.value = false
    }
  }

  const resetVirtualNetwork = async (): Promise<void> => {
    isApplyingNetwork.value = true
    lastError.value = ''
    try {
      virtualNetwork.value = await ResetVirtualNetwork()
      await refreshRuntimeStatus()
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    } finally {
      isApplyingNetwork.value = false
    }
  }

  const detectNAT = async (): Promise<void> => {
    isDetectingNAT.value = true
    lastError.value = ''
    try {
      lastNATDetection.value = await DetectNAT()
      natType.value = lastNATDetection.value.type
      await refreshRuntimeStatus()
    } catch (e) {
      lastError.value = extractErrorMessage(e)
      throw e
    } finally {
      isDetectingNAT.value = false
    }
  }

  return {
    tunnels,
    connectionStatus,
    serverAddr,
    authToken,
    serverSettings,
    lastError,
    isConnecting,
    isApplyingNetwork,
    isDetectingNAT,
    busyTunnelIds,
    trafficStats,
    p2pStatus,
    natType,
    runtimeStatus,
    virtualNetwork,
    lastNATDetection,
    tunnelCount,
    isConnected,
    activeTunnelCount,
    loadServerSettings,
    saveServerSettings,
    loadTunnels,
    createTunnel,
    deleteTunnel,
    connectServer,
    disconnectServer,
    startTunnel,
    stopTunnel,
    refreshStatus,
    refreshRuntimeStatus,
    applyVirtualNetwork,
    resetVirtualNetwork,
    detectNAT,
  }
})
