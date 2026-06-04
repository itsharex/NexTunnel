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
