<template>
  <div class="port-manager">
    <div class="manager-toolbar">
      <div>
        <strong>{{ t('ports.title') }}</strong>
        <span>{{ t('ports.subtitle') }}</span>
      </div>
      <n-space align="center">
        <n-button
          size="small"
          secondary
          :loading="store.isScanningPorts"
          @click="openScanModal"
        >
          {{ t('ports.scanCommon') }}
        </n-button>
        <n-button
          size="small"
          type="primary"
          @click="showEditor = !showEditor"
        >
          {{ showEditor ? t('ports.closeEditor') : t('ports.addFavorite') }}
        </n-button>
      </n-space>
    </div>

    <n-collapse-transition :show="showEditor">
      <n-form
        class="favorite-form"
        label-placement="top"
        :show-feedback="false"
      >
        <n-grid
          :cols="5"
          :x-gap="10"
          :y-gap="10"
          responsive="screen"
        >
          <n-form-item-gi :label="t('ports.name')">
            <n-input v-model:value="favoriteForm.name" />
          </n-form-item-gi>
          <n-form-item-gi :label="t('ports.category')">
            <n-select
              v-model:value="favoriteForm.category"
              :options="categoryOptions"
            />
          </n-form-item-gi>
          <n-form-item-gi :label="t('ports.port')">
            <n-input-number
              v-model:value="favoriteForm.port"
              :min="1"
              :max="65535"
            />
          </n-form-item-gi>
          <n-form-item-gi :label="t('ports.protocol')">
            <n-select
              v-model:value="favoriteForm.protocol"
              :options="protocolOptions"
            />
          </n-form-item-gi>
          <n-form-item-gi label=" ">
            <n-button
              type="primary"
              block
              :disabled="!canSaveFavorite"
              :loading="store.isSavingFavoritePort"
              @click="handleSaveFavorite"
            >
              {{ t('ports.saveFavorite') }}
            </n-button>
          </n-form-item-gi>
        </n-grid>
        <n-input
          v-model:value="favoriteForm.description"
          type="textarea"
          :autosize="{ minRows: 1, maxRows: 3 }"
          :placeholder="t('ports.descriptionPlaceholder')"
        />
      </n-form>
    </n-collapse-transition>

    <div class="scan-summary">
      <div class="summary-item">
        <span>{{ t('ports.openPorts') }}</span>
        <strong>{{ store.openPortCount }}</strong>
      </div>
      <div class="summary-item">
        <span>{{ t('ports.favoriteCount') }}</span>
        <strong>{{ store.favoritePorts.length }}</strong>
      </div>
      <div class="summary-item">
        <span>{{ t('ports.scanHost') }}</span>
        <strong>127.0.0.1</strong>
      </div>
    </div>

    <n-tabs
      v-model:value="activeCategory"
      type="segment"
      animated
    >
      <n-tab-pane
        v-for="category in visibleCategories"
        :key="category"
        :name="category"
        :tab="translateCategory(category)"
      >
        <div class="favorite-list">
          <article
            v-for="port in favoritePortsByCategory[category]"
            :key="port.id"
            class="favorite-item"
          >
            <div class="favorite-main">
              <strong>{{ port.name }}</strong>
              <span>{{ port.description || t('ports.noDescription') }}</span>
            </div>
            <div class="favorite-actions">
              <n-tag
                round
                size="small"
                :type="port.enabled ? 'success' : 'default'"
                :bordered="false"
              >
                {{ port.protocol.toUpperCase() }} {{ port.port }}
              </n-tag>
              <n-switch
                :value="port.enabled"
                size="small"
                @update:value="(enabled) => handleTogglePort(port, enabled)"
              />
              <n-button
                size="tiny"
                secondary
                @click="emitUsePort(port)"
              >
                {{ t('ports.use') }}
              </n-button>
              <n-button
                size="tiny"
                type="error"
                secondary
                @click="handleDeleteFavorite(port.id)"
              >
                {{ t('ports.delete') }}
              </n-button>
            </div>
          </article>
        </div>
      </n-tab-pane>
    </n-tabs>

    <n-modal
      v-model:show="showScanModal"
      preset="card"
      class="port-scan-modal"
      :title="t('ports.scanModalTitle')"
      :bordered="false"
      :segmented="{ content: true, footer: 'soft' }"
    >
      <div class="scan-modal-layout">
        <section
          v-if="scanModalStage === 'radar'"
          class="radar-panel radar-only"
        >
          <div
            class="radar-screen"
            :class="{ active: isRadarVisible }"
            aria-hidden="true"
          >
            <span class="radar-ring ring-a" />
            <span class="radar-ring ring-b" />
            <span class="radar-ring ring-c" />
            <span class="radar-sweep" />
            <span
              v-for="dot in radarDots"
              :key="dot.port"
              class="radar-dot"
              :class="{ open: dot.open }"
              :style="{ left: dot.x, top: dot.y }"
            />
          </div>
          <div class="radar-copy">
            <strong>{{ t('ports.scanning') }}</strong>
            <span>{{ t('ports.scanModalSubtitle') }}</span>
          </div>
        </section>

        <section
          v-else
          class="scan-result-panel"
        >
          <div class="scan-result-header">
            <strong>{{ t('ports.scanResults') }}</strong>
            <div class="scan-result-meta">
              <n-tag
                round
                size="small"
                :type="store.openPortCount > 0 ? 'success' : 'default'"
                :bordered="false"
              >
                {{ t('ports.openPortCount', { count: store.openPortCount }) }}
              </n-tag>
              <n-checkbox
                v-if="openScanResults.length > 0"
                :checked="isAllScanResultsSelected"
                :indeterminate="isPartialScanResultsSelected"
                @update:checked="handleToggleAllScanResults"
              >
                {{ t('ports.selectAll') }}
              </n-checkbox>
            </div>
          </div>

          <n-list
            v-if="openScanResults.length > 0"
            class="scan-result-list"
            bordered
          >
            <n-list-item
              v-for="draft in editableScanPorts"
              :key="draft.source_port"
              class="scan-result-row"
              :class="{ selected: selectedScanPorts.includes(draft.source_port) }"
            >
              <div class="scan-row-layout">
                <n-checkbox
                  :checked="selectedScanPorts.includes(draft.source_port)"
                  @update:checked="(checked) => handleToggleScanResult(draft.source_port, checked)"
                />
                <div class="scan-row-fields">
                  <n-input
                    v-model:value="draft.name"
                    size="small"
                    :placeholder="t('ports.name')"
                  />
                  <n-select
                    v-model:value="draft.category"
                    size="small"
                    :options="categoryOptions"
                  />
                  <n-input-number
                    v-model:value="draft.port"
                    size="small"
                    :min="1"
                    :max="65535"
                  />
                  <n-select
                    v-model:value="draft.protocol"
                    size="small"
                    :options="protocolOptions"
                  />
                  <n-input
                    v-model:value="draft.description"
                    class="scan-row-description"
                    size="small"
                    :placeholder="t('ports.descriptionPlaceholder')"
                  />
                </div>
              </div>
            </n-list-item>
          </n-list>
          <n-empty
            v-else
            class="scan-empty"
            :description="t('ports.noScanResults')"
          />
        </section>
      </div>

      <template #footer>
        <div
          v-if="scanModalStage === 'radar'"
          class="scan-modal-actions"
        >
          <span class="scan-selected-count">{{ t('ports.scanning') }}</span>
          <n-button
            secondary
            disabled
            :loading="isRadarVisible"
          >
            {{ t('ports.rescan') }}
          </n-button>
          <n-button
            type="primary"
            disabled
          >
            {{ t('ports.addSelectedPorts') }}
          </n-button>
        </div>
        <div
          v-else
          class="scan-modal-actions"
        >
          <span class="scan-selected-count">{{ t('ports.selectedCount', { count: selectedScanPorts.length }) }}</span>
          <n-button
            secondary
            :loading="isRadarVisible"
            @click="handleScanEnabledPorts"
          >
            {{ t('ports.rescan') }}
          </n-button>
          <n-button
            type="primary"
            :disabled="!canSaveSelectedScanPorts"
            :loading="store.isSavingFavoritePort"
            @click="handleSaveSelectedScanPorts"
          >
            {{ t('ports.addSelectedPorts') }}
          </n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NCheckbox,
  NCollapseTransition,
  NEmpty,
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NInputNumber,
  NList,
  NListItem,
  NModal,
  NSelect,
  NSpace,
  NSwitch,
  NTabPane,
  NTabs,
  NTag,
  useMessage,
  type SelectOption,
} from 'naive-ui'
import { useTunnelStore } from '../stores/tunnel'
import type { FavoritePortInfo, LocalPortScanResult } from '../api/app'

type PortLike = FavoritePortInfo | LocalPortScanResult
type ScanPortDraft = {
  source_port: number
  name: string
  category: string
  port: number
  protocol: string
  description: string
  enabled: boolean
}
type ScanModalStage = 'radar' | 'results'

const DEFAULT_CATEGORY = 'development'
const DEFAULT_PROTOCOL = 'tcp'
const DEFAULT_PORT = 3000
const SCAN_RADAR_MIN_DURATION_MS = 7000

const emit = defineEmits<{
  usePort: [port: PortLike]
}>()

const store = useTunnelStore()
const message = useMessage()
const { t } = useI18n()
const showEditor = ref(false)
const showScanModal = ref(false)
const scanModalStage = ref<ScanModalStage>('radar')
const activeCategory = ref(DEFAULT_CATEGORY)
const favoriteForm = reactive({
  name: '',
  category: DEFAULT_CATEGORY,
  port: DEFAULT_PORT,
  protocol: DEFAULT_PROTOCOL,
  description: '',
  enabled: true,
})
const selectedScanPorts = ref<number[]>([])
const editableScanPorts = ref<ScanPortDraft[]>([])

const categoryOptions = computed<SelectOption[]>(() => [
  { label: t('ports.categories.development'), value: 'development' },
  { label: t('ports.categories.database'), value: 'database' },
  { label: t('ports.categories.software'), value: 'software' },
  { label: t('ports.categories.game'), value: 'game' },
  { label: t('ports.categories.remote'), value: 'remote' },
  { label: t('ports.categories.service'), value: 'service' },
  { label: t('ports.categories.custom'), value: 'custom' },
])

const protocolOptions: SelectOption[] = [{ label: 'TCP', value: 'tcp' }]

const favoritePortsByCategory = computed<Record<string, FavoritePortInfo[]>>(() => {
  return store.favoritePorts.reduce<Record<string, FavoritePortInfo[]>>((groups, port) => {
    const category = port.category || 'custom'
    groups[category] = groups[category] || []
    groups[category].push(port)
    return groups
  }, {})
})

const visibleCategories = computed(() => {
  const categories = Object.keys(favoritePortsByCategory.value)
  return categories.length > 0 ? categories : [DEFAULT_CATEGORY]
})

const canSaveFavorite = computed(() => favoriteForm.name.trim().length > 0 && favoriteForm.port > 0 && favoriteForm.port <= 65535)
const canSaveSelectedScanPorts = computed(() => {
  return selectedScanPorts.value.some((port) => {
    const draft = editableScanPorts.value.find((item) => item.source_port === port)
    return Boolean(draft && draft.name.trim().length > 0 && draft.port > 0 && draft.port <= 65535)
  })
})
const openScanResults = computed(() => store.portScanResults.filter((result) => result.open))
const isRadarVisible = computed(() => scanModalStage.value === 'radar' || store.isScanningPorts)
const isAllScanResultsSelected = computed(() => {
  return openScanResults.value.length > 0 && selectedScanPorts.value.length === openScanResults.value.length
})
const isPartialScanResultsSelected = computed(() => {
  return selectedScanPorts.value.length > 0 && selectedScanPorts.value.length < openScanResults.value.length
})
const radarDots = computed(() => {
  return store.portScanResults.slice(0, 18).map((result, index) => {
    const angle = (index * 137.5 * Math.PI) / 180
    const radius = 18 + (index % 5) * 13
    return {
      port: result.port,
      open: result.open,
      x: `${50 + Math.cos(angle) * radius}%`,
      y: `${50 + Math.sin(angle) * radius}%`,
    }
  })
})

const translateCategory = (category: string): string => {
  const key = `ports.categories.${category}`
  const translated = t(key)
  return translated === key ? category : translated
}

const resetFavoriteForm = (): void => {
  favoriteForm.name = ''
  favoriteForm.category = DEFAULT_CATEGORY
  favoriteForm.port = DEFAULT_PORT
  favoriteForm.protocol = DEFAULT_PROTOCOL
  favoriteForm.description = ''
  favoriteForm.enabled = true
}

const wait = (durationMs: number): Promise<void> => new Promise((resolve) => {
  window.setTimeout(resolve, durationMs)
})

const createScanPortDraft = (port: LocalPortScanResult): ScanPortDraft => ({
  source_port: port.port,
  name: port.name || `local-${port.port}`,
  category: port.category || DEFAULT_CATEGORY,
  port: port.port,
  protocol: port.protocol || DEFAULT_PROTOCOL,
  description: port.description || '',
  enabled: true,
})

// syncEditableScanPorts 保留用户已编辑内容，并补齐新扫描出的开放端口。
const syncEditableScanPorts = (): void => {
  const previousDrafts = new Map(editableScanPorts.value.map((draft) => [draft.source_port, draft]))
  editableScanPorts.value = openScanResults.value.map((result) => previousDrafts.get(result.port) ?? createScanPortDraft(result))
  const openPorts = new Set(openScanResults.value.map((result) => result.port))
  selectedScanPorts.value = selectedScanPorts.value.filter((port) => openPorts.has(port))
}

const openScanModal = (): void => {
  showScanModal.value = true
  if (store.portScanResults.length > 0) {
    scanModalStage.value = 'results'
    syncEditableScanPorts()
    selectedScanPorts.value = openScanResults.value.map((result) => result.port)
    return
  }
  void handleScanEnabledPorts()
}

const handleToggleScanResult = (port: number, checked: boolean): void => {
  if (checked) {
    selectedScanPorts.value = Array.from(new Set([...selectedScanPorts.value, port]))
    return
  }
  selectedScanPorts.value = selectedScanPorts.value.filter((item) => item !== port)
}

const handleToggleAllScanResults = (checked: boolean): void => {
  selectedScanPorts.value = checked ? openScanResults.value.map((result) => result.port) : []
}

const handleScanEnabledPorts = async (): Promise<void> => {
  const scanStartedAt = Date.now()
  scanModalStage.value = 'radar'
  selectedScanPorts.value = []

  try {
    await store.scanLocalPorts()
    const remainingRadarDuration = Math.max(0, SCAN_RADAR_MIN_DURATION_MS - (Date.now() - scanStartedAt))
    await wait(remainingRadarDuration)
    syncEditableScanPorts()
    selectedScanPorts.value = openScanResults.value.map((result) => result.port)
    scanModalStage.value = 'results'
    message.success(t('ports.scanSuccess'))
  } catch {
    const remainingRadarDuration = Math.max(0, SCAN_RADAR_MIN_DURATION_MS - (Date.now() - scanStartedAt))
    await wait(remainingRadarDuration)
    scanModalStage.value = 'results'
    message.error(store.lastError || t('ports.scanFailed'))
  }
}

const handleSaveSelectedScanPorts = async (): Promise<void> => {
  if (!canSaveSelectedScanPorts.value) return
  const selectedDrafts = editableScanPorts.value.filter((draft) => selectedScanPorts.value.includes(draft.source_port))
  let successCount = 0
  let failedCount = 0

  for (const draft of selectedDrafts) {
    if (draft.name.trim().length === 0 || draft.port < 1 || draft.port > 65535) {
      failedCount += 1
      continue
    }
    try {
      await store.saveFavoritePort({
        name: draft.name.trim(),
        category: draft.category,
        port: draft.port,
        protocol: draft.protocol,
        description: draft.description,
        enabled: draft.enabled,
      })
      successCount += 1
    } catch {
      failedCount += 1
    }
  }

  if (successCount > 0 && failedCount === 0) {
    message.success(t('ports.batchSaveSuccess', { count: successCount }))
    showScanModal.value = false
    selectedScanPorts.value = []
    return
  }
  if (successCount > 0) {
    message.warning(t('ports.batchSavePartial', { success: successCount, failed: failedCount }))
    return
  }
  message.error(store.lastError || t('ports.saveFailed'))
}

const handleSaveFavorite = async (): Promise<void> => {
  if (!canSaveFavorite.value) return
  try {
    await store.saveFavoritePort({ ...favoriteForm })
    message.success(t('ports.saveSuccess'))
    resetFavoriteForm()
    showEditor.value = false
  } catch {
    message.error(store.lastError || t('ports.saveFailed'))
  }
}

const handleTogglePort = async (port: FavoritePortInfo, enabled: boolean): Promise<void> => {
  try {
    await store.saveFavoritePort({
      id: port.id,
      name: port.name,
      category: port.category,
      port: port.port,
      protocol: port.protocol,
      description: port.description,
      enabled,
    })
  } catch {
    message.error(store.lastError || t('ports.saveFailed'))
  }
}

const handleDeleteFavorite = async (id: string): Promise<void> => {
  try {
    await store.deleteFavoritePort(id)
    message.success(t('ports.deleteSuccess'))
  } catch {
    message.error(store.lastError || t('ports.deleteFailed'))
  }
}

const emitUsePort = (port: PortLike): void => {
  emit('usePort', port)
}

watch(visibleCategories, (categories) => {
  if (!categories.includes(activeCategory.value)) {
    activeCategory.value = categories[0] || DEFAULT_CATEGORY
  }
})

watch(openScanResults, () => {
  if (!showScanModal.value) return
  syncEditableScanPorts()
})

onMounted(async () => {
  if (store.favoritePorts.length === 0) {
    await store.loadFavoritePorts()
  }
})
</script>

<style scoped>
.port-manager {
  display: grid;
  gap: 16px;
}

.manager-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
}

.manager-toolbar > div:first-child {
  display: grid;
  gap: 4px;
}

.manager-toolbar strong {
  color: var(--text-main);
  font-size: 16px;
}

.manager-toolbar span {
  color: var(--text-dim);
  font-size: 12px;
}

.favorite-form {
  display: grid;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.58);
}

.scan-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.summary-item {
  display: grid;
  gap: 4px;
  padding: 10px 12px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.5);
}

.summary-item span {
  color: var(--text-dim);
  font-size: 12px;
}

.summary-item strong {
  color: var(--text-main);
  font-family: Consolas, 'SFMono-Regular', monospace;
  font-size: 15px;
}

.favorite-list {
  display: grid;
  gap: 10px;
}

.favorite-item {
  width: 100%;
  min-height: 58px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 11px 12px;
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.52);
}

.favorite-main {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.favorite-main strong {
  overflow: hidden;
  color: var(--text-main);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.favorite-main span {
  color: var(--text-dim);
  font-size: 12px;
  line-height: 1.45;
}

.favorite-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

:global(.port-scan-modal) {
  width: min(1040px, calc(100vw - 48px));
}

.scan-modal-layout {
  display: grid;
  gap: 14px;
}

.radar-panel,
.scan-result-panel {
  border: 1px solid var(--line-soft);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.58);
}

.radar-panel {
  min-height: 420px;
  display: grid;
  align-content: center;
  justify-items: center;
  gap: 12px;
  padding: 18px 14px;
}

.scan-result-panel {
  min-height: 360px;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
}

.radar-screen {
  position: relative;
  width: min(58vw, 300px);
  aspect-ratio: 1;
  place-self: center;
  overflow: hidden;
  border: 1px solid rgba(0, 255, 255, 0.18);
  border-radius: 50%;
  background:
    radial-gradient(circle at 50% 50%, rgba(0, 255, 255, 0.16) 0 4%, transparent 5%),
    radial-gradient(circle, rgba(0, 255, 255, 0.1), transparent 62%),
    rgba(4, 10, 20, 0.72);
  box-shadow:
    inset 0 0 42px rgba(0, 255, 255, 0.12),
    0 0 36px rgba(0, 255, 255, 0.1);
}

.radar-ring,
.radar-sweep,
.radar-dot {
  position: absolute;
  display: block;
  pointer-events: none;
}

.radar-ring {
  border: 1px solid rgba(0, 255, 255, 0.16);
  border-radius: 50%;
  opacity: 0.72;
  transform: scale(0.7);
}

.ring-a {
  inset: 14%;
}

.ring-b {
  inset: 27%;
}

.ring-c {
  inset: 40%;
}

.radar-sweep {
  inset: 0;
  border-radius: 50%;
  background:
    conic-gradient(from 18deg, rgba(0, 255, 255, 0.34), rgba(0, 255, 255, 0.08) 18%, transparent 38%),
    radial-gradient(circle, transparent 0 10%, rgba(0, 255, 255, 0.08) 11%, transparent 42%);
  transform-origin: 50% 50%;
  opacity: 0.72;
}

.radar-screen.active .radar-sweep {
  animation: radarSweep 1800ms linear infinite;
}

.radar-screen.active .ring-a {
  animation: findPulse 1800ms var(--ease-standard) infinite;
}

.radar-screen.active .ring-b {
  animation: findPulse 1800ms var(--ease-standard) infinite 220ms;
}

.radar-screen.active .ring-c {
  animation: findPulse 1800ms var(--ease-standard) infinite 440ms;
}

.radar-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: rgba(168, 169, 169, 0.46);
  opacity: 0.62;
  transform: translate(-50%, -50%) scale(0.86);
}

.radar-dot.open {
  background: var(--nex-cyan);
  opacity: 1;
  box-shadow:
    0 0 0 5px rgba(0, 255, 255, 0.08),
    0 0 14px rgba(0, 255, 255, 0.76);
}

.radar-copy {
  display: grid;
  gap: 5px;
  justify-items: center;
  text-align: center;
}

.radar-copy strong,
.scan-result-header strong {
  color: var(--text-main);
  font-size: 14px;
}

.radar-copy span {
  color: var(--text-dim);
  font-size: 12px;
  line-height: 1.5;
}

.scan-result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
}

.scan-result-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.scan-result-list {
  max-height: 420px;
  overflow: auto;
  border-color: rgba(168, 169, 169, 0.12);
  background: rgba(4, 10, 20, 0.2);
}

.scan-result-row {
  transition:
    background var(--duration-small) var(--ease-standard),
    box-shadow var(--duration-small) var(--ease-standard);
}

.scan-result-row.selected {
  background: linear-gradient(90deg, rgba(0, 255, 255, 0.1), rgba(138, 43, 226, 0.06));
  box-shadow: inset 3px 0 0 var(--nex-cyan);
}

.scan-row-layout {
  width: 100%;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}

.scan-row-fields {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(130px, 1.1fr) minmax(120px, 0.8fr) minmax(96px, 0.6fr) minmax(92px, 0.55fr) minmax(160px, 1.2fr);
  gap: 8px;
}

.scan-row-description {
  min-width: 0;
}

.scan-empty {
  min-height: 240px;
  display: grid;
  place-items: center;
}

.scan-modal-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.scan-selected-count {
  margin-right: auto;
  color: var(--text-dim);
  font-size: 12px;
}

@keyframes radarSweep {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

@keyframes findPulse {
  0% {
    opacity: 0;
    transform: scale(0.45);
  }

  28% {
    opacity: 0.82;
  }

  100% {
    opacity: 0;
    transform: scale(1.18);
  }
}

@media (max-width: 1180px) {
  .scan-summary {
    grid-template-columns: 1fr;
  }

  .manager-toolbar,
  .favorite-item {
    grid-template-columns: 1fr;
  }

  .manager-toolbar,
  .favorite-item {
    align-items: stretch;
    flex-direction: column;
  }

  .favorite-actions {
    justify-content: flex-start;
  }

  .scan-row-fields {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (prefers-reduced-motion: reduce) {
  .scan-result-row {
    transition: none;
  }

  .radar-sweep,
  .radar-ring {
    animation: none !important;
  }
}
</style>
