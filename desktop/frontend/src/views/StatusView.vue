<template>
  <div class="status-view">
    <div class="status-card">
      <div class="status-indicator" :class="store.connectionStatus">
        <span class="dot"></span>
        <span>{{ statusLabel }}</span>
      </div>
      <h2>NexTunnel Desktop</h2>
      <p class="description">Next-generation tunnel and P2P networking tool</p>
    </div>

    <div class="info-grid">
      <div class="info-card">
        <span class="label">Tunnels</span>
        <span class="value">{{ store.tunnelCount }}</span>
      </div>
      <div class="info-card">
        <span class="label">Traffic In</span>
        <span class="value">{{ formatBytes(store.trafficStats.bytes_in) }}</span>
      </div>
      <div class="info-card">
        <span class="label">Traffic Out</span>
        <span class="value">{{ formatBytes(store.trafficStats.bytes_out) }}</span>
      </div>
    </div>

    <div class="section">
      <div class="section-header">
        <h3>Tunnels</h3>
        <button class="btn btn-sm" @click="showForm = !showForm">
          {{ showForm ? 'Cancel' : '+ New Tunnel' }}
        </button>
      </div>

      <div v-if="showForm" class="tunnel-form">
        <input v-model="form.name" placeholder="Name" class="input" />
        <select v-model="form.proxy_type" class="input">
          <option value="tcp">TCP</option>
          <option value="http">HTTP</option>
        </select>
        <input v-model="form.local_addr" placeholder="Local Address" class="input" />
        <input v-model.number="form.local_port" type="number" placeholder="Local Port" class="input" />
        <input v-model.number="form.remote_port" type="number" placeholder="Remote Port" class="input" />
        <button class="btn" @click="handleCreate">Create</button>
      </div>

      <div v-if="store.tunnels.length === 0 && !showForm" class="empty-state">
        <p>No tunnels configured. Click "+ New Tunnel" to create one.</p>
      </div>

      <div v-for="t in store.tunnels" :key="t.id" class="tunnel-item">
        <div class="tunnel-info">
          <span class="tunnel-name">{{ t.name }}</span>
          <span class="tunnel-type">{{ t.proxy_type.toUpperCase() }}</span>
          <span class="tunnel-addr">{{ t.local_addr }}:{{ t.local_port }} → :{{ t.remote_port }}</span>
        </div>
        <div class="tunnel-actions">
          <span class="tunnel-status" :class="t.status">{{ t.status }}</span>
          <button class="btn btn-sm btn-danger" @click="handleDelete(t.id)">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useTunnelStore } from '../stores/tunnel'

const store = useTunnelStore()
const showForm = ref(false)
const form = ref({
  name: '',
  proxy_type: 'tcp',
  local_addr: '127.0.0.1',
  local_port: 8080,
  remote_port: 80,
})

const statusLabel = computed(() => {
  switch (store.connectionStatus) {
    case 'connected': return 'Connected'
    case 'reconnecting': return 'Reconnecting...'
    default: return 'Disconnected'
  }
})

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

async function handleCreate() {
  if (!form.value.name) return
  try {
    await store.createTunnel(form.value)
    showForm.value = false
    form.value = { name: '', proxy_type: 'tcp', local_addr: '127.0.0.1', local_port: 8080, remote_port: 80 }
  } catch {
    // error handled in store
  }
}

async function handleDelete(id: string) {
  await store.deleteTunnel(id)
}

let interval: ReturnType<typeof setInterval>

onMounted(async () => {
  await store.loadTunnels()
  await store.refreshStatus()
  interval = setInterval(() => store.refreshStatus(), 3000)
})

onUnmounted(() => {
  if (interval) clearInterval(interval)
})
</script>

<style scoped>
.status-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.status-card {
  background-color: var(--color-surface);
  border-radius: 8px;
  padding: 24px;
  text-align: center;
}

.status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 14px;
}

.status-indicator .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #ef5350;
}

.status-indicator.connected .dot { background-color: #66bb6a; }
.status-indicator.reconnecting .dot { background-color: #ffa726; }

.status-card h2 { font-size: 24px; margin-bottom: 8px; }
.description { color: var(--color-text-secondary); font-size: 14px; }

.info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.info-card {
  background-color: var(--color-surface);
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-card .label { font-size: 11px; color: var(--color-text-secondary); text-transform: uppercase; }
.info-card .value { font-size: 20px; font-weight: 600; }

.section { display: flex; flex-direction: column; gap: 12px; }
.section-header { display: flex; justify-content: space-between; align-items: center; }
.section-header h3 { font-size: 16px; }

.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  background-color: var(--color-primary);
  color: #1b2636;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.btn:hover { opacity: 0.9; }
.btn-sm { padding: 4px 12px; font-size: 12px; }
.btn-danger { background-color: #ef5350; color: white; }

.input {
  padding: 8px 12px;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 6px;
  background-color: var(--color-surface);
  color: var(--color-text);
  font-size: 13px;
  outline: none;
}
.input:focus { border-color: var(--color-primary); }

.tunnel-form {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  background-color: var(--color-surface);
  padding: 16px;
  border-radius: 8px;
}
.tunnel-form .input { flex: 1; min-width: 100px; }

.empty-state {
  text-align: center;
  padding: 24px;
  color: var(--color-text-secondary);
  font-size: 14px;
}

.tunnel-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: var(--color-surface);
  padding: 12px 16px;
  border-radius: 8px;
}

.tunnel-info { display: flex; align-items: center; gap: 12px; }
.tunnel-name { font-weight: 600; font-size: 14px; }
.tunnel-type {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  background-color: rgba(79, 195, 247, 0.2);
  color: var(--color-primary);
}
.tunnel-addr { font-size: 12px; color: var(--color-text-secondary); font-family: monospace; }

.tunnel-actions { display: flex; align-items: center; gap: 12px; }
.tunnel-status {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
}
.tunnel-status.active, .tunnel-status.running { color: #66bb6a; }
.tunnel-status.stopped, .tunnel-status.inactive { color: var(--color-text-secondary); }
.tunnel-status.error { color: #ef5350; }
</style>
