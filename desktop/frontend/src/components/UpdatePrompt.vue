<template>
  <Transition name="update-prompt">
    <section
      v-if="shouldShowPrompt"
      class="update-prompt"
      role="status"
      aria-live="polite"
    >
      <button
        class="prompt-close"
        type="button"
        :title="t('updatePrompt.dismiss')"
        @click="store.dismissUpdatePrompt"
      >
        <X :size="16" />
      </button>

      <div class="prompt-heading">
        <Download :size="19" />
        <div>
          <strong>{{ t('updatePrompt.title') }}</strong>
          <span>{{ versionLabel }}</span>
        </div>
      </div>

      <p v-if="changelogSummary">
        {{ changelogSummary }}
      </p>

      <div class="prompt-actions">
        <n-button
          type="primary"
          size="small"
          :loading="store.isInstallingUpdate"
          @click="handlePrimaryAction"
        >
          <template #icon>
            <Download :size="15" />
          </template>
          {{ canInstallDirectly ? t('updatePrompt.install') : t('updatePrompt.openRelease') }}
        </n-button>
        <n-button
          size="small"
          secondary
          @click="openUpdateURL"
        >
          <template #icon>
            <ExternalLink :size="15" />
          </template>
          {{ t('updatePrompt.details') }}
        </n-button>
      </div>
    </section>
  </Transition>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Download, ExternalLink, X } from 'lucide-vue-next'
import { NButton, useMessage } from 'naive-ui'
import { useTunnelStore } from '../stores/tunnel'

const MAX_CHANGELOG_SUMMARY_LENGTH = 180
const WINDOWS_INSTALLER_PATTERN = /\.exe(?:$|\?)/i

const store = useTunnelStore()
const message = useMessage()
const { t } = useI18n()

const shouldShowPrompt = computed(() => Boolean(store.updateInfo?.available && !store.hasShownUpdatePrompt))
const canInstallDirectly = computed(() => WINDOWS_INSTALLER_PATTERN.test(store.updateInfo?.url || ''))
const versionLabel = computed(() =>
  t('updatePrompt.versionLabel', {
    current: store.updateInfo?.current_version || '--',
    latest: store.updateInfo?.latest_version || '--',
  }),
)
const changelogSummary = computed(() => {
  const changelog = (store.updateInfo?.changelog || '').replace(/\s+/g, ' ').trim()
  if (changelog.length <= MAX_CHANGELOG_SUMMARY_LENGTH) return changelog
  return `${changelog.slice(0, MAX_CHANGELOG_SUMMARY_LENGTH)}...`
})

const openUpdateURL = (): void => {
  const url = store.updateInfo?.url
  if (!url) return
  window.open(url, '_blank', 'noopener,noreferrer')
}

const handlePrimaryAction = async (): Promise<void> => {
  const url = store.updateInfo?.url
  if (!url) return
  if (!canInstallDirectly.value) {
    openUpdateURL()
    return
  }
  try {
    const result = await store.installUpdate(url)
    if (result.error) {
      message.error(result.error)
      return
    }
    message.success(t('updatePrompt.installStarted'))
  } catch {
    message.error(store.lastError || t('updatePrompt.installFailed'))
  }
}
</script>

<style scoped>
.update-prompt {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 20;
  width: min(420px, calc(100vw - 48px));
  display: grid;
  gap: 12px;
  padding: 16px;
  border: 1px solid color-mix(in srgb, var(--nex-cyan) 38%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, var(--surface-strong) 94%, #ffffff 6%);
  box-shadow: 0 18px 42px rgba(0, 0, 0, 0.26);
}

.prompt-close {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text-dim);
  cursor: pointer;
}

.prompt-close:hover,
.prompt-close:focus-visible {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-main);
}

.prompt-heading {
  min-width: 0;
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  padding-right: 30px;
}

.prompt-heading > svg {
  color: var(--nex-cyan);
  margin-top: 2px;
}

.prompt-heading div {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.prompt-heading strong {
  color: var(--text-main);
  font-size: 15px;
}

.prompt-heading span,
.update-prompt p {
  color: var(--text-dim);
  font-size: 12px;
  line-height: 1.6;
}

.update-prompt p {
  margin: 0;
}

.prompt-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.update-prompt-enter-active,
.update-prompt-leave-active {
  transition:
    opacity var(--duration-small) var(--ease-standard),
    transform var(--duration-small) var(--ease-standard);
}

.update-prompt-enter-from,
.update-prompt-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@media (max-width: 720px) {
  .update-prompt {
    right: 12px;
    bottom: 12px;
    width: calc(100vw - 24px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .update-prompt-enter-active,
  .update-prompt-leave-active {
    transition: none;
  }

  .update-prompt-enter-from,
  .update-prompt-leave-to {
    transform: none;
  }
}
</style>
