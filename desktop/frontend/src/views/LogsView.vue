<template>
  <section class="logs-view">
    <div class="logs-toolbar">
      <div class="filter-row">
        <n-select
          v-model:value="filterLevel"
          class="filter-select"
          :options="levelOptions"
          size="small"
          @update:value="handleFilterChange"
        />
        <n-select
          v-model:value="filterCategory"
          class="filter-select"
          :options="categoryOptions"
          size="small"
          @update:value="handleFilterChange"
        />
      </div>
      <n-space align="center">
        <n-button
          size="small"
          secondary
          :loading="store.isLoadingActivityLogs"
          @click="handleRefresh"
        >
          {{ t('logs.refresh') }}
        </n-button>
        <n-button
          size="small"
          type="error"
          secondary
          :disabled="store.activityLogs.length === 0"
          :loading="store.isLoadingActivityLogs"
          @click="handleClear"
        >
          {{ t('logs.clear') }}
        </n-button>
      </n-space>
    </div>

    <n-card
      class="logs-panel"
      :bordered="false"
    >
      <div
        v-if="store.activityLogs.length > 0"
        class="log-list"
      >
        <article
          v-for="log in store.activityLogs"
          :key="log.id"
          class="log-entry"
        >
          <div
            class="level-rail"
            :class="log.level"
          />
          <div class="log-main">
            <div class="log-heading">
              <div>
                <strong>{{ log.title }}</strong>
                <span>{{ log.message || t('logs.noMessage') }}</span>
              </div>
              <n-tag
                round
                size="small"
                :type="getLevelTagType(log.level)"
                :bordered="false"
              >
                {{ translateLevel(log.level) }}
              </n-tag>
            </div>

            <div class="log-meta">
              <span>{{ formatLogDate(log.created_at) }}</span>
              <span>{{ translateCategory(log.category) }}</span>
              <span>{{ translateAction(log.action) }}</span>
              <span v-if="log.target_type">{{ log.target_type }}{{ log.target_id ? ` · ${log.target_id}` : '' }}</span>
            </div>
          </div>
        </article>
      </div>

      <n-empty
        v-else
        class="empty-state"
        :description="t('logs.empty')"
      />
    </n-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NCard, NEmpty, NSelect, NSpace, NTag, useMessage, type SelectOption } from 'naive-ui'
import { useTunnelStore } from '../stores/tunnel'

type TagType = 'default' | 'error' | 'success' | 'warning' | 'info'

const LOG_LIST_LIMIT = 100
const store = useTunnelStore()
const message = useMessage()
const { t } = useI18n()
const filterLevel = ref('')
const filterCategory = ref('')

const levelOptions = computed<SelectOption[]>(() => [
  { label: t('logs.filters.allLevels'), value: '' },
  { label: t('logs.levels.info'), value: 'info' },
  { label: t('logs.levels.warning'), value: 'warning' },
  { label: t('logs.levels.error'), value: 'error' },
])

const categoryOptions = computed<SelectOption[]>(() => [
  { label: t('logs.filters.allCategories'), value: '' },
  { label: t('logs.categories.operation'), value: 'operation' },
  { label: t('logs.categories.security'), value: 'security' },
  { label: t('logs.categories.error'), value: 'error' },
])

// buildFilter 只传递有效筛选条件，避免后端收到空字符串后产生多余条件。
const buildFilter = () => ({
  level: filterLevel.value || undefined,
  category: filterCategory.value || undefined,
  limit: LOG_LIST_LIMIT,
})

const handleFilterChange = async (): Promise<void> => {
  await store.loadActivityLogs(buildFilter())
}

const handleRefresh = async (): Promise<void> => {
  await store.loadActivityLogs(buildFilter())
}

const handleClear = async (): Promise<void> => {
  try {
    await store.clearActivityLogs()
    message.success(t('logs.clearSuccess'))
  } catch {
    message.error(store.lastError || t('logs.clearFailed'))
  }
}

const translateLevel = (level: string): string => {
  const key = `logs.levels.${level}`
  const translated = t(key)
  return translated === key ? level : translated
}

const translateCategory = (category: string): string => {
  const key = `logs.categories.${category}`
  const translated = t(key)
  return translated === key ? category : translated
}

const translateAction = (action: string): string => {
  const key = `logs.actions.${action}`
  const translated = t(key)
  return translated === key ? action : translated
}

const getLevelTagType = (level: string): TagType => {
  if (level === 'error') return 'error'
  if (level === 'warning') return 'warning'
  if (level === 'info') return 'success'
  return 'default'
}

const formatLogDate = (value: string): string => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--'
  return date.toLocaleString()
}

onMounted(async () => {
  await store.loadActivityLogs(buildFilter())
})
</script>

<style scoped>
.logs-view {
  display: grid;
  gap: 14px;
}

.logs-toolbar,
.logs-panel {
  border: 1px solid var(--line-soft);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), rgba(255, 255, 255, 0.012)),
    var(--surface-bg);
  box-shadow: 0 14px 34px rgba(0, 0, 0, 0.16);
}

.logs-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 12px;
  border-radius: 8px;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.filter-select {
  width: 168px;
}

.log-list {
  display: grid;
  gap: 10px;
}

.log-entry {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
  border: 1px solid rgba(168, 169, 169, 0.12);
  border-radius: 8px;
  background: rgba(9, 17, 32, 0.56);
  transition: transform var(--duration-small) var(--ease-standard), opacity var(--duration-small) var(--ease-standard);
}

.log-entry:hover {
  transform: translateY(-1px);
}

.level-rail {
  width: 4px;
  border-radius: 999px;
  background: var(--nex-cyan);
}

.level-rail.warning {
  background: var(--warning);
}

.level-rail.error {
  background: var(--danger);
}

.log-main {
  min-width: 0;
  display: grid;
  gap: 10px;
}

.log-heading {
  min-width: 0;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.log-heading div {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.log-heading strong {
  overflow: hidden;
  color: var(--text-main);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-heading span {
  color: var(--text-dim);
  font-size: 12px;
  line-height: 1.6;
}

.log-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  color: var(--text-muted);
  font-family: Consolas, 'SFMono-Regular', monospace;
  font-size: 11px;
}

.log-meta span {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  padding: 4px 7px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-state {
  min-height: 360px;
  display: grid;
  place-items: center;
}

@media (max-width: 1180px) {
  .logs-toolbar {
    align-items: stretch;
    flex-direction: column;
  }
}

@media (prefers-reduced-motion: reduce) {
  .log-entry {
    transition: none;
  }

  .log-entry:hover {
    transform: none;
  }
}
</style>
