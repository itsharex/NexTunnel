<template>
  <n-popconfirm
    :positive-text="positiveText"
    negative-text="取消"
    @positive-click="$emit('confirm')"
  >
    <template #trigger>
      <n-button
        v-bind="$attrs"
        :loading="loading"
      >
        <slot />
      </n-button>
    </template>
    {{ message }}
  </n-popconfirm>
</template>

<script setup lang="ts">
import { NButton, NPopconfirm } from 'naive-ui'

defineOptions({
  inheritAttrs: false,
})

withDefaults(
  defineProps<{
    message: string
    loading?: boolean
    positiveText?: string
  }>(),
  {
    loading: false,
    positiveText: '确认',
  },
)

defineEmits<{
  confirm: []
}>()
</script>
