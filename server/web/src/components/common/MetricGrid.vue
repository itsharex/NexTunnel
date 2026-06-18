<template>
  <section
    class="metric-grid"
    aria-label="Dashboard metrics"
  >
    <n-card
      v-for="metric in dashboard.metrics"
      :key="metric.label"
      class="metric-card"
      :bordered="false"
    >
      <n-statistic
        :label="metric.label"
        :value="metric.value"
      />
      <span>{{ metric.detail }}</span>
    </n-card>
  </section>
</template>

<script setup lang="ts">
import { NCard, NStatistic } from 'naive-ui'
import { useDashboardStore } from '../../stores/dashboard'

const dashboard = useDashboardStore()
</script>
