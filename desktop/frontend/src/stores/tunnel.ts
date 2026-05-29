import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface Tunnel {
  id: string
  name: string
  type: 'tcp' | 'http' | 'udp'
  localAddr: string
  localPort: number
  remotePort: number
  status: 'stopped' | 'running' | 'error'
}

export const useTunnelStore = defineStore('tunnels', () => {
  const tunnels = ref<Tunnel[]>([])

  function addTunnel(tunnel: Tunnel) {
    tunnels.value.push(tunnel)
  }

  function removeTunnel(id: string) {
    tunnels.value = tunnels.value.filter((t) => t.id !== id)
  }

  function updateTunnel(id: string, updates: Partial<Tunnel>) {
    const index = tunnels.value.findIndex((t) => t.id === id)
    if (index !== -1) {
      tunnels.value[index] = { ...tunnels.value[index], ...updates }
    }
  }

  return { tunnels, addTunnel, removeTunnel, updateTunnel }
})
