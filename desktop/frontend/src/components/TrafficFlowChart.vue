<template>
  <div class="traffic-chart">
    <div class="chart-header">
      <div>
        <strong>{{ title }}</strong>
        <span>{{ subtitle }}</span>
      </div>
      <div class="chart-legend">
        <span class="legend-item upload">{{ uploadLabel }}</span>
        <span class="legend-item download">{{ downloadLabel }}</span>
      </div>
    </div>

    <div class="chart-stage">
      <div
        ref="chartElement"
        class="uplot-host"
        :aria-label="title"
        role="img"
      />

      <div
        v-if="!hasChartData"
        class="chart-empty"
      >
        {{ emptyText }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import uPlot from 'uplot'
import type { AlignedData } from 'uplot'
import 'uplot/dist/uPlot.min.css'
import type { TrafficSample } from '../stores/tunnel'

const MIN_DELTA_BYTES = 0
const CHART_HEIGHT = 220
const MIN_CHART_WIDTH = 320
const TIME_SCALE_SECONDS = 1000
const TRAFFIC_SERIES_STROKE_WIDTH = 2
const BYTE_UNITS = ['B/s', 'KB/s', 'MB/s', 'GB/s']

const props = defineProps<{
  samples: TrafficSample[]
  title: string
  subtitle: string
  uploadLabel: string
  downloadLabel: string
  emptyText: string
}>()

const chartElement = ref<HTMLElement | null>(null)
let plotInstance: uPlot | null = null
let resizeObserver: ResizeObserver | null = null

// buildRateSeries 将累计流量转换为每秒增量速率，避免累计图表只升不降。
const buildRateSeries = (key: 'bytes_in' | 'bytes_out'): number[] => {
  if (props.samples.length < 2) return []
  return props.samples.slice(1).map((sample, index) => {
    const previous = props.samples[index]
    const deltaBytes = Math.max(MIN_DELTA_BYTES, sample[key] - previous[key])
    const deltaSeconds = Math.max(1, (sample.timestamp - previous.timestamp) / TIME_SCALE_SECONDS)
    return deltaBytes / deltaSeconds
  })
}

const chartData = computed<AlignedData>(() => {
  if (props.samples.length < 2) return [[], [], []]
  const timestamps = props.samples.slice(1).map((sample) => sample.timestamp / TIME_SCALE_SECONDS)
  return [timestamps, buildRateSeries('bytes_out'), buildRateSeries('bytes_in')]
})
const hasChartData = computed(() => chartData.value[0].length > 0)

const formatBytesPerSecond = (value: number): string => {
  if (!Number.isFinite(value) || value <= 0) return `0 ${BYTE_UNITS[0]}`
  const unitIndex = Math.min(Math.floor(Math.log(value) / Math.log(1024)), BYTE_UNITS.length - 1)
  return `${(value / Math.pow(1024, unitIndex)).toFixed(1)} ${BYTE_UNITS[unitIndex]}`
}

const createChartOptions = (width: number): uPlot.Options => {
  return {
    width: Math.max(MIN_CHART_WIDTH, width),
    height: CHART_HEIGHT,
    legend: {
      show: false,
    },
    cursor: {
      drag: {
        setScale: false,
      },
      points: {
        show: true,
      },
    },
    scales: {
      x: {
        time: true,
      },
      y: {
        auto: true,
      },
    },
    axes: [
      {
        stroke: '#708194',
        grid: { stroke: 'rgba(168, 169, 169, 0.12)', width: 1 },
        ticks: { stroke: 'rgba(168, 169, 169, 0.18)', width: 1 },
        size: 28,
      },
      {
        stroke: '#708194',
        grid: { stroke: 'rgba(168, 169, 169, 0.12)', width: 1 },
        ticks: { stroke: 'rgba(168, 169, 169, 0.18)', width: 1 },
        size: 72,
        values: (_self, values) => values.map((value) => formatBytesPerSecond(value)),
      },
    ],
    series: [
      {},
      {
        label: props.uploadLabel,
        stroke: '#00ffff',
        width: TRAFFIC_SERIES_STROKE_WIDTH,
        fill: 'rgba(0, 255, 255, 0.12)',
        points: { show: false },
        value: (_self, value) => formatBytesPerSecond(value),
      },
      {
        label: props.downloadLabel,
        stroke: '#8a2be2',
        width: TRAFFIC_SERIES_STROKE_WIDTH,
        fill: 'rgba(138, 43, 226, 0.1)',
        points: { show: false },
        value: (_self, value) => formatBytesPerSecond(value),
      },
    ],
  }
}

const updateChartSize = (): void => {
  if (!plotInstance || !chartElement.value) return
  plotInstance.setSize({
    width: Math.max(MIN_CHART_WIDTH, chartElement.value.clientWidth),
    height: CHART_HEIGHT,
  })
}

const initChart = async (): Promise<void> => {
  await nextTick()
  if (!chartElement.value || plotInstance) return
  plotInstance = new uPlot(createChartOptions(chartElement.value.clientWidth), chartData.value, chartElement.value)
  resizeObserver = new ResizeObserver(updateChartSize)
  resizeObserver.observe(chartElement.value)
}

watch(chartData, (data) => {
  plotInstance?.setData(data, true)
})

watch(
  () => [props.uploadLabel, props.downloadLabel],
  async () => {
    if (!chartElement.value || !plotInstance) return
    plotInstance.destroy()
    plotInstance = null
    await initChart()
  },
)

onMounted(initChart)

onUnmounted(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
  plotInstance?.destroy()
  plotInstance = null
})
</script>

<style scoped>
.traffic-chart {
  min-height: 260px;
  display: grid;
  gap: 16px;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
}

.chart-header div:first-child {
  display: grid;
  gap: 4px;
}

.chart-header strong {
  color: var(--text-main);
  font-size: 16px;
}

.chart-header span {
  color: var(--text-dim);
  font-size: 12px;
}

.chart-legend {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-dim);
  font-size: 12px;
}

.legend-item::before {
  content: '';
  width: 16px;
  height: 3px;
  border-radius: 999px;
  background: currentColor;
}

.legend-item.upload {
  color: var(--nex-cyan);
}

.legend-item.download {
  color: var(--tunnel-violet);
}

.chart-stage {
  position: relative;
  min-height: 228px;
  overflow: hidden;
  border: 1px solid rgba(0, 255, 255, 0.14);
  border-radius: 8px;
  background:
    linear-gradient(180deg, rgba(9, 17, 32, 0.24), rgba(9, 17, 32, 0.76)),
    rgba(4, 9, 18, 0.58);
}

.uplot-host {
  width: 100%;
  min-height: 228px;
  padding: 10px 10px 4px;
}

.uplot-host :deep(.uplot) {
  background: transparent;
  color: var(--text-dim);
  font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
}

.uplot-host :deep(.u-title),
.uplot-host :deep(.u-legend) {
  display: none;
}

.uplot-host :deep(.u-over),
.uplot-host :deep(.u-under) {
  border-radius: 8px;
}

.uplot-host :deep(.u-cursor-x),
.uplot-host :deep(.u-cursor-y) {
  border-color: rgba(254, 255, 255, 0.36);
}

.uplot-host :deep(.u-cursor-pt) {
  width: 7px;
  height: 7px;
  margin-left: -3.5px;
  margin-top: -3.5px;
  border: 2px solid var(--surface-strong);
  box-shadow: 0 0 14px currentColor;
}

.chart-empty {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: var(--text-muted);
  font-size: 13px;
  pointer-events: none;
}

@media (prefers-reduced-motion: reduce) {
  .uplot-host :deep(*) {
    transition: none;
  }
}
</style>
