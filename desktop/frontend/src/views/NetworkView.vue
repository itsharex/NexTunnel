<template>
  <section class="network-view">
    <div class="network-grid">
      <n-card
        class="network-panel"
        :bordered="false"
      >
        <template #header>
          <div class="panel-title">
            <div>
              <strong>{{ t('network.virtualNetwork') }}</strong>
              <span>{{ t('network.virtualNetworkSubtitle') }}</span>
            </div>
            <n-tag
              round
              :type="virtualNetwork?.applied ? 'success' : 'warning'"
              :bordered="false"
            >
              {{ virtualNetwork?.applied ? t('network.applied') : t('network.notApplied') }}
            </n-tag>
          </div>
        </template>

        <div class="network-facts">
          <div
            v-for="item in virtualNetworkFacts"
            :key="item.label"
            class="fact-row"
          >
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>

        <n-space>
          <n-button
            type="primary"
            :loading="store.isApplyingNetwork"
            @click="handleApplyVirtualNetwork"
          >
            {{ t('network.applyRoutes') }}
          </n-button>
          <n-button
            secondary
            :loading="store.isApplyingNetwork"
            @click="handleResetVirtualNetwork"
          >
            {{ t('network.resetRoutes') }}
          </n-button>
        </n-space>
      </n-card>

      <n-card
        class="network-panel"
        :bordered="false"
      >
        <template #header>
          <div class="panel-title">
            <div>
              <strong>{{ t('network.platform') }}</strong>
              <span>{{ t('network.platformSubtitle') }}</span>
            </div>
          </div>
        </template>

        <div class="capability-list">
          <div
            v-for="item in platformFacts"
            :key="item.label"
            class="capability-item"
          >
            <span>{{ item.label }}</span>
            <n-tag
              round
              size="small"
              :type="item.ok ? 'success' : 'warning'"
              :bordered="false"
            >
              {{ item.value }}
            </n-tag>
          </div>
        </div>
      </n-card>
    </div>

    <div class="network-grid">
      <n-card
        class="network-panel"
        :bordered="false"
      >
        <template #header>
          <div class="panel-title">
            <div>
              <strong>{{ t('network.natDiagnosis') }}</strong>
              <span>{{ t('network.natDiagnosisSubtitle') }}</span>
            </div>
            <n-button
              size="small"
              type="primary"
              :loading="store.isDetectingNAT"
              @click="handleDetectNAT"
            >
              {{ t('network.detectNow') }}
            </n-button>
          </div>
        </template>

        <div class="network-facts">
          <div class="fact-row">
            <span>{{ t('network.natType') }}</span>
            <strong>{{ natDetection?.type || store.natType || t('status.waiting') }}</strong>
          </div>
          <div class="fact-row">
            <span>{{ t('network.publicAddress') }}</span>
            <strong>{{ natDetection?.public_addr || '--' }}</strong>
          </div>
          <div class="fact-row">
            <span>{{ t('network.localAddress') }}</span>
            <strong>{{ natDetection?.local_addr || '--' }}</strong>
          </div>
        </div>
      </n-card>

      <n-card
        class="network-panel"
        :bordered="false"
      >
        <template #header>
          <div class="panel-title compact">
            <div>
              <strong>{{ t('network.lastCommands') }}</strong>
              <span>{{ t('network.lastCommandsSubtitle') }}</span>
            </div>
          </div>
        </template>

        <div class="command-log">
          <div
            v-for="command in lastCommands"
            :key="command"
            class="command-line"
          >
            {{ command }}
          </div>
          <n-empty
            v-if="lastCommands.length === 0"
            :description="t('network.noCommands')"
          />
        </div>
      </n-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NCard, NEmpty, NSpace, NTag, useMessage } from 'naive-ui'
import { useTunnelStore } from '../stores/tunnel'

const store = useTunnelStore()
const message = useMessage()
const { t } = useI18n()

const runtime = computed(() => store.runtimeStatus)
const virtualNetwork = computed(() => store.virtualNetwork)
const natDetection = computed(() => store.lastNATDetection)
const lastCommands = computed(() => virtualNetwork.value?.last_commands ?? [])

const virtualNetworkFacts = computed(() => [
  { label: t('network.interface'), value: virtualNetwork.value?.interface || '--' },
  { label: t('network.virtualIP'), value: virtualNetwork.value?.virtual_ip || '--' },
  { label: t('network.subnet'), value: virtualNetwork.value?.subnet || '--' },
  { label: t('network.gateway'), value: virtualNetwork.value?.gateway || '--' },
  { label: t('network.mtu'), value: `${virtualNetwork.value?.mtu || 1420}` },
])

const platformFacts = computed(() => {
  const tun = runtime.value?.tun
  return [
    { label: t('network.platformName'), value: tun?.PlatformName || '--', ok: Boolean(tun?.PlatformName) },
    { label: t('network.kernelTun'), value: tun?.HasKernelTUN ? t('network.available') : t('network.unavailable'), ok: Boolean(tun?.HasKernelTUN) },
    { label: t('network.userspaceTun'), value: tun?.HasUserspaceNetstack ? t('network.available') : t('network.unavailable'), ok: Boolean(tun?.HasUserspaceNetstack) },
    { label: t('network.adminRequired'), value: tun?.NeedsAdminPrivilege ? t('network.required') : t('network.notRequired'), ok: !tun?.NeedsAdminPrivilege },
  ]
})

const handleApplyVirtualNetwork = async (): Promise<void> => {
  try {
    await store.applyVirtualNetwork()
    message.success(t('network.applySuccess'))
  } catch {
    message.error(store.lastError || t('network.applyFailed'))
  }
}

const handleResetVirtualNetwork = async (): Promise<void> => {
  try {
    await store.resetVirtualNetwork()
    message.success(t('network.resetSuccess'))
  } catch {
    message.error(store.lastError || t('network.resetFailed'))
  }
}

const handleDetectNAT = async (): Promise<void> => {
  try {
    await store.detectNAT()
    message.success(t('network.detectSuccess'))
  } catch {
    message.error(store.lastError || t('network.detectFailed'))
  }
}

onMounted(async () => {
  await store.refreshRuntimeStatus()
})
</script>

<style scoped>
.network-view {
  display: grid;
  gap: 18px;
}

.network-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(360px, 0.8fr);
  gap: 18px;
}

.network-panel {
  border: 1px solid var(--line-soft);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.012)),
    var(--surface-bg);
  box-shadow: 0 18px 44px rgba(0, 0, 0, 0.2);
}

.panel-title {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
}

.panel-title div,
.panel-title.compact {
  display: grid;
  gap: 4px;
}

.panel-title strong {
  color: var(--text-main);
  font-size: 16px;
}

.panel-title span {
  color: var(--text-dim);
  font-size: 12px;
}

.network-facts,
.capability-list {
  display: grid;
  gap: 10px;
  margin-bottom: 16px;
}

.fact-row,
.capability-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  min-height: 42px;
  padding: 10px 12px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.56);
}

.fact-row span,
.capability-item span {
  color: var(--text-dim);
  font-size: 12px;
}

.fact-row strong {
  color: var(--text-main);
  font-family: Consolas, 'SFMono-Regular', monospace;
  font-size: 13px;
  overflow-wrap: anywhere;
  text-align: right;
}

.command-log {
  min-height: 180px;
  max-height: 260px;
  overflow: auto;
  padding: 12px;
  border: 1px solid rgba(0, 255, 255, 0.16);
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.72);
}

.command-line {
  color: #24e6a1;
  font-family: Consolas, 'SFMono-Regular', monospace;
  font-size: 12px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

@media (max-width: 1180px) {
  .network-grid {
    grid-template-columns: 1fr;
  }
}
</style>
