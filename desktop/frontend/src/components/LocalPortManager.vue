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
          @click="handleScanEnabledPorts"
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

    <div
      v-if="store.portScanResults.length > 0"
      class="scan-results"
    >
      <button
        v-for="result in store.portScanResults"
        :key="result.port"
        class="scan-result"
        :class="{ open: result.open }"
        type="button"
        :disabled="!result.open"
        @click="emitUsePort(result)"
      >
        <span>
          <strong>{{ result.name || `${result.protocol.toUpperCase()} ${result.port}` }}</strong>
          <small>{{ result.category ? translateCategory(result.category) : t('ports.custom') }}</small>
        </span>
        <n-tag
          size="small"
          round
          :type="result.open ? 'success' : 'default'"
          :bordered="false"
        >
          {{ result.open ? t('ports.open') : t('ports.closed') }}
        </n-tag>
      </button>
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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NCollapseTransition,
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NInputNumber,
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

const DEFAULT_CATEGORY = 'development'
const DEFAULT_PROTOCOL = 'tcp'
const DEFAULT_PORT = 3000

const emit = defineEmits<{
  usePort: [port: PortLike]
}>()

const store = useTunnelStore()
const message = useMessage()
const { t } = useI18n()
const showEditor = ref(false)
const activeCategory = ref(DEFAULT_CATEGORY)
const favoriteForm = reactive({
  name: '',
  category: DEFAULT_CATEGORY,
  port: DEFAULT_PORT,
  protocol: DEFAULT_PROTOCOL,
  description: '',
  enabled: true,
})

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

const handleScanEnabledPorts = async (): Promise<void> => {
  try {
    await store.scanLocalPorts()
    message.success(t('ports.scanSuccess'))
  } catch {
    message.error(store.lastError || t('ports.scanFailed'))
  }
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

.scan-results,
.favorite-list {
  display: grid;
  gap: 10px;
}

.scan-result,
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

.scan-result {
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: transform 150ms cubic-bezier(0.4, 0, 0.2, 1);
}

.scan-result.open:hover {
  transform: translateY(-1px);
  border-color: rgba(0, 255, 255, 0.34);
  background: rgba(0, 255, 255, 0.08);
}

.scan-result:disabled {
  cursor: not-allowed;
  opacity: 0.66;
}

.scan-result span,
.favorite-main {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.scan-result strong,
.favorite-main strong {
  overflow: hidden;
  color: var(--text-main);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scan-result small,
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

@media (max-width: 1180px) {
  .scan-summary {
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
}

@media (prefers-reduced-motion: reduce) {
  .scan-result {
    transition: none;
  }
}
</style>
