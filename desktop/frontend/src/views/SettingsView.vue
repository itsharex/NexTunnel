<template>
  <section class="settings-view">
    <n-card
      class="settings-panel"
      :bordered="false"
    >
      <template #header>
        <div class="panel-title">
          <div>
            <strong>{{ t('settings.connectionTitle') }}</strong>
            <span>{{ t('settings.connectionSubtitle') }}</span>
          </div>
          <n-button
            type="primary"
            :loading="isSaving"
            @click="handleSave"
          >
            {{ t('settings.save') }}
          </n-button>
        </div>
      </template>

      <n-form
        label-placement="top"
        :show-feedback="false"
      >
        <n-grid
          :cols="2"
          :x-gap="14"
          :y-gap="14"
          responsive="screen"
        >
          <n-form-item-gi :label="t('connection.relayAddress')">
            <n-input v-model:value="form.relay_addr" />
          </n-form-item-gi>
          <n-form-item-gi :label="t('connection.relayToken')">
            <n-input
              v-model:value="form.relay_token"
              type="password"
              show-password-on="click"
            />
          </n-form-item-gi>
          <n-form-item-gi :label="t('settings.controlPlaneURL')">
            <n-input
              v-model:value="form.control_plane_url"
              placeholder="https://control.example.com:9090"
            />
          </n-form-item-gi>
          <n-form-item-gi :label="t('settings.controlPlaneToken')">
            <n-input
              v-model:value="form.control_plane_token"
              type="password"
              show-password-on="click"
            />
          </n-form-item-gi>
          <n-form-item-gi :label="t('settings.stunServer')">
            <n-input v-model:value="form.stun_server" />
          </n-form-item-gi>
          <n-form-item-gi :label="t('settings.stunAltServer')">
            <n-input v-model:value="form.stun_alt_server" />
          </n-form-item-gi>
        </n-grid>
      </n-form>
    </n-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NCard, NForm, NFormItemGi, NGrid, NInput, useMessage } from 'naive-ui'
import { useTunnelStore } from '../stores/tunnel'
import type { ServerSettings } from '../api/app'

const store = useTunnelStore()
const message = useMessage()
const { t } = useI18n()
const isSaving = ref(false)
const form = reactive<ServerSettings>({
  relay_addr: '127.0.0.1:7000',
  relay_token: '',
  control_plane_url: '',
  control_plane_token: '',
  stun_server: 'stun.l.google.com:19302',
  stun_alt_server: 'stun.l.google.com:19302',
})

const fillForm = (settings: ServerSettings): void => {
  Object.assign(form, settings)
}

const handleSave = async (): Promise<void> => {
  isSaving.value = true
  try {
    await store.saveServerSettings({ ...form })
    message.success(t('settings.saveSuccess'))
  } catch {
    message.error(store.lastError || t('settings.saveFailed'))
  } finally {
    isSaving.value = false
  }
}

onMounted(async () => {
  await store.loadServerSettings()
  fillForm(store.serverSettings)
})
</script>

<style scoped>
.settings-view {
  display: grid;
  gap: 18px;
}

.settings-panel {
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

.panel-title div {
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
</style>
