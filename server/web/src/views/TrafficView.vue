<template>
  <div class="page-stack">
    <MetricGrid />

    <n-card
      class="panel traffic-panel"
      :bordered="false"
    >
      <template #header>
        <div class="panel-header">
          <div>
            <p class="eyebrow">
              流量
            </p>
            <h2>节点带宽分布</h2>
          </div>
          <n-tag
            round
            size="small"
            :bordered="false"
          >
            {{ dashboard.trafficBars.length }} 条样本
          </n-tag>
        </div>
      </template>

      <div class="traffic-bars">
        <div
          v-for="bar in dashboard.trafficBars"
          :key="bar.label"
          class="traffic-row"
        >
          <div class="traffic-label">
            <strong>{{ bar.label }}</strong>
            <span>{{ bar.detail }}</span>
          </div>
          <div
            class="bar-track"
            aria-hidden="true"
          >
            <span
              class="rx-bar"
              :style="{ width: `${bar.rxPercent}%` }"
            />
            <span
              class="tx-bar"
              :style="{ width: `${bar.txPercent}%` }"
            />
          </div>
        </div>
        <n-empty
          v-if="dashboard.trafficBars.length === 0"
          description="暂无流量样本"
        />
      </div>
    </n-card>

    <n-card
      class="panel traffic-panel"
      :bordered="false"
    >
      <template #header>
        <div class="panel-header">
          <div>
            <p class="eyebrow">
              客户端
            </p>
            <h2>Relay 客户端流量</h2>
          </div>
          <n-tag
            round
            size="small"
            :bordered="false"
          >
            {{ dashboard.clientTrafficBars.length }} 个客户端
          </n-tag>
        </div>
      </template>

      <div class="traffic-bars">
        <div
          v-for="bar in dashboard.clientTrafficBars"
          :key="bar.label"
          class="traffic-row"
        >
          <div class="traffic-label">
            <strong>{{ bar.label }}</strong>
            <span>{{ bar.detail }}</span>
          </div>
          <div
            class="bar-track"
            aria-hidden="true"
          >
            <span
              class="rx-bar"
              :style="{ width: `${bar.rxPercent}%` }"
            />
            <span
              class="tx-bar"
              :style="{ width: `${bar.txPercent}%` }"
            />
          </div>
        </div>
        <n-empty
          v-if="dashboard.clientTrafficBars.length === 0"
          description="暂无客户端流量"
        />
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NCard, NEmpty, NTag } from 'naive-ui'
import MetricGrid from '../components/common/MetricGrid.vue'
import { useDashboardStore } from '../stores/dashboard'

const dashboard = useDashboardStore()
</script>
