import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetTunnels, CreateTunnel, DeleteTunnel, GetConnectionStatus, GetTrafficStats, GetP2PStatus, GetNATType } from '../api/app'

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
  const trafficStats = ref<{ bytes_in: number; bytes_out: number; tunnels: number }>({
    bytes_in: 0,
    bytes_out: 0,
    tunnels: 0,
  })
  const p2pStatus = ref<string>('')
  const natType = ref<string>('')

  const tunnelCount = computed(() => tunnels.value.length)

  async function loadTunnels() {
    try {
      tunnels.value = (await GetTunnels()) as Tunnel[]
    } catch (e) {
      console.error('Failed to load tunnels:', e)
    }
  }

  async function createTunnel(input: CreateTunnelInput) {
    try {
      const t = (await CreateTunnel(input)) as Tunnel
      tunnels.value.push(t)
      return t
    } catch (e) {
      console.error('Failed to create tunnel:', e)
      throw e
    }
  }

  async function deleteTunnel(id: string) {
    try {
      await DeleteTunnel(id)
      tunnels.value = tunnels.value.filter((t) => t.id !== id)
    } catch (e) {
      console.error('Failed to delete tunnel:', e)
      throw e
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
    trafficStats,
    p2pStatus,
    natType,
    tunnelCount,
    loadTunnels,
    createTunnel,
    deleteTunnel,
    refreshStatus,
  }
})
