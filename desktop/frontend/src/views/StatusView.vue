<template>
  <div class="status-view">
    <div class="status-card">
      <div class="status-indicator online">
        <span class="dot"></span>
        <span>Ready</span>
      </div>
      <h2>NexTunnel Desktop</h2>
      <p class="description">Next-generation tunnel and P2P networking tool</p>
    </div>

    <div class="greet-section">
      <input
        v-model="name"
        type="text"
        placeholder="Enter your name"
        class="input"
      />
      <button class="btn" @click="greet">Test Connection</button>
      <p v-if="greeting" class="greeting">{{ greeting }}</p>
    </div>

    <div class="info-grid">
      <div class="info-card">
        <span class="label">Tunnels</span>
        <span class="value">0</span>
      </div>
      <div class="info-card">
        <span class="label">Connections</span>
        <span class="value">0</span>
      </div>
      <div class="info-card">
        <span class="label">Traffic In</span>
        <span class="value">0 B</span>
      </div>
      <div class="info-card">
        <span class="label">Traffic Out</span>
        <span class="value">0 B</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Greet } from '../api/app'

const name = ref('')
const greeting = ref('')

async function greet() {
  if (!name.value) return
  try {
    greeting.value = await Greet(name.value)
  } catch {
    greeting.value = 'Failed to communicate with backend'
  }
}
</script>

<style scoped>
.status-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
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

.status-indicator.online .dot {
  background-color: #66bb6a;
}

.status-card h2 {
  font-size: 24px;
  margin-bottom: 8px;
}

.description {
  color: var(--color-text-secondary);
  font-size: 14px;
}

.greet-section {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.input {
  flex: 1;
  min-width: 200px;
  padding: 10px 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  background-color: var(--color-surface);
  color: var(--color-text);
  font-size: 14px;
  outline: none;
}

.input:focus {
  border-color: var(--color-primary);
}

.btn {
  padding: 10px 20px;
  border: none;
  border-radius: 6px;
  background-color: var(--color-primary);
  color: #1b2636;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.btn:hover {
  opacity: 0.9;
}

.greeting {
  width: 100%;
  color: var(--color-primary);
  font-size: 14px;
  margin-top: 4px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.info-card {
  background-color: var(--color-surface);
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-card .label {
  font-size: 12px;
  color: var(--color-text-secondary);
  text-transform: uppercase;
}

.info-card .value {
  font-size: 24px;
  font-weight: 600;
}
</style>
