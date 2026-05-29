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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function call(method: string, ...args: any[]): Promise<any> {
  return (window as any).go.main.App[method](...args)
}

export function GetVersion(): Promise<string> {
  return call('GetVersion')
}

export function GetTunnels(): Promise<TunnelInfo[]> {
  return call('GetTunnels')
}

export function CreateTunnel(input: CreateTunnelInput): Promise<TunnelInfo> {
  return call('CreateTunnel', input)
}

export function DeleteTunnel(id: string): Promise<void> {
  return call('DeleteTunnel', id)
}

export function GetConnectionStatus(): Promise<string> {
  return call('GetConnectionStatus')
}

export function GetTrafficStats(): Promise<{ bytes_in: number; bytes_out: number; tunnels: number }> {
  return call('GetTrafficStats')
}
