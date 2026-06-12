// API layer - calls Wails runtime bindings directly

export interface TunnelInfo {
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

export interface ServerConfigInput {
  server_addr: string
  auth_token: string
}

export interface ServerSettings {
  relay_addr: string
  relay_token: string
  control_plane_url: string
  control_plane_token: string
  stun_server: string
  stun_alt_server: string
}

export interface PlatformCapabilities {
  HasKernelTUN: boolean
  HasUserspaceNetstack: boolean
  NeedsAdminPrivilege: boolean
  PlatformName: string
}

export interface VirtualNetworkRoute {
  destination: string
  gateway: string
  interface: string
  metric: number
}

export interface VirtualNetworkState {
  applied: boolean
  interface: string
  virtual_ip: string
  subnet: string
  gateway: string
  mtu: number
  routes: VirtualNetworkRoute[]
  last_error: string
  last_commands: string[]
}

export interface RuntimeStatus {
  connection_status: string
  p2p_status: string
  nat_type: string
  tun: PlatformCapabilities
  virtual_network: VirtualNetworkState
  traffic_stats: { bytes_in: number; bytes_out: number; tunnels: number }
  last_error: string
}

export interface NATDetectionInfo {
  type: string
  public_addr: string
  mapped_port: number
  local_addr: string
}

type WailsMethod = (...args: unknown[]) => Promise<unknown> | unknown

interface WailsRuntimeWindow extends Window {
  go?: {
    main?: {
      App?: Record<string, WailsMethod>
    }
  }
}

const PREVIEW_VERSION = 'preview'
const PREVIEW_TUNNEL_STATUS = 'stopped'
const PREVIEW_TRAFFIC_STATS = { bytes_in: 0, bytes_out: 0, tunnels: 0 }
const PREVIEW_SETTINGS: ServerSettings = {
  relay_addr: '127.0.0.1:7000',
  relay_token: '',
  control_plane_url: '',
  control_plane_token: '',
  stun_server: 'stun.l.google.com:19302',
  stun_alt_server: 'stun.l.google.com:19302',
}
const PREVIEW_VIRTUAL_NETWORK: VirtualNetworkState = {
  applied: false,
  interface: '',
  virtual_ip: '',
  subnet: '',
  gateway: '',
  mtu: 1420,
  routes: [],
  last_error: '',
  last_commands: [],
}
const PREVIEW_TUN: PlatformCapabilities = {
  HasKernelTUN: false,
  HasUserspaceNetstack: true,
  NeedsAdminPrivilege: false,
  PlatformName: 'preview',
}

// createPreviewBinding 在普通浏览器预览时提供安全空实现，避免设计页面因 Wails 未注入而报错。
const createPreviewBinding = (): Record<string, WailsMethod> => ({
  GetVersion: () => PREVIEW_VERSION,
  GetTunnels: () => [],
  CreateTunnel: (input: unknown) => {
    const tunnelInput = input as CreateTunnelInput
    return {
      id: `preview-${Date.now()}`,
      ...tunnelInput,
      status: PREVIEW_TUNNEL_STATUS,
    }
  },
  DeleteTunnel: () => undefined,
  StartTunnel: () => undefined,
  StopTunnel: () => undefined,
  ConnectServer: () => undefined,
  DisconnectServer: () => undefined,
  GetConnectionStatus: () => 'disconnected',
  GetTrafficStats: () => PREVIEW_TRAFFIC_STATS,
  GetP2PStatus: () => '',
  GetNATType: () => '',
  GetServerSettings: () => PREVIEW_SETTINGS,
  SaveServerSettings: () => undefined,
  GetRuntimeStatus: () => ({
    connection_status: 'disconnected',
    p2p_status: '',
    nat_type: '',
    tun: PREVIEW_TUN,
    virtual_network: PREVIEW_VIRTUAL_NETWORK,
    traffic_stats: PREVIEW_TRAFFIC_STATS,
    last_error: '',
  }),
  ApplyVirtualNetwork: () => PREVIEW_VIRTUAL_NETWORK,
  ResetVirtualNetwork: () => PREVIEW_VIRTUAL_NETWORK,
  DetectNAT: () => ({
    type: 'preview',
    public_addr: '',
    mapped_port: 0,
    local_addr: '',
  }),
})

// getAppBinding 统一读取 Wails 注入对象，避免业务 API 直接依赖全局结构。
const getAppBinding = (): Record<string, WailsMethod> => {
  const wailsWindow = window as WailsRuntimeWindow
  return wailsWindow.go?.main?.App ?? createPreviewBinding()
}

const call = async <T>(method: string, ...args: unknown[]): Promise<T> => {
  const app = getAppBinding()
  const target = app[method]
  if (!target) {
    throw new Error(`Wails method not found: ${method}`)
  }
  return (await target(...args)) as T
}

export const GetVersion = (): Promise<string> => {
  return call<string>('GetVersion')
}

export const GetTunnels = (): Promise<TunnelInfo[]> => {
  return call<TunnelInfo[]>('GetTunnels')
}

export const CreateTunnel = (input: CreateTunnelInput): Promise<TunnelInfo> => {
  return call<TunnelInfo>('CreateTunnel', input)
}

export const DeleteTunnel = (id: string): Promise<void> => {
  return call<void>('DeleteTunnel', id)
}

export const StartTunnel = (id: string): Promise<void> => {
  return call<void>('StartTunnel', id)
}

export const StopTunnel = (id: string): Promise<void> => {
  return call<void>('StopTunnel', id)
}

export const ConnectServer = (input: ServerConfigInput): Promise<void> => {
  return call<void>('ConnectServer', input)
}

export const DisconnectServer = (): Promise<void> => {
  return call<void>('DisconnectServer')
}

export const GetConnectionStatus = (): Promise<string> => {
  return call<string>('GetConnectionStatus')
}

export const GetTrafficStats = (): Promise<{ bytes_in: number; bytes_out: number; tunnels: number }> => {
  return call<{ bytes_in: number; bytes_out: number; tunnels: number }>('GetTrafficStats')
}

export const GetP2PStatus = (): Promise<string> => {
  return call<string>('GetP2PStatus')
}

export const GetNATType = (): Promise<string> => {
  return call<string>('GetNATType')
}

export const GetServerSettings = (): Promise<ServerSettings> => {
  return call<ServerSettings>('GetServerSettings')
}

export const SaveServerSettings = (settings: ServerSettings): Promise<void> => {
  return call<void>('SaveServerSettings', settings)
}

export const GetRuntimeStatus = (): Promise<RuntimeStatus> => {
  return call<RuntimeStatus>('GetRuntimeStatus')
}

export const ApplyVirtualNetwork = (): Promise<VirtualNetworkState> => {
  return call<VirtualNetworkState>('ApplyVirtualNetwork')
}

export const ResetVirtualNetwork = (): Promise<VirtualNetworkState> => {
  return call<VirtualNetworkState>('ResetVirtualNetwork')
}

export const DetectNAT = (): Promise<NATDetectionInfo> => {
  return call<NATDetectionInfo>('DetectNAT')
}
