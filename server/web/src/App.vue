<template>
  <n-config-provider
    :theme="darkTheme"
    :theme-overrides="themeOverrides"
  >
    <n-message-provider>
      <n-dialog-provider>
        <div class="dashboard-app">
          <RouterView />
        </div>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router'
import { NConfigProvider, NDialogProvider, NMessageProvider, darkTheme, type GlobalThemeOverrides } from 'naive-ui'
import './styles/dashboard.css'

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#00ffff',
    primaryColorHover: '#33f6f6',
    primaryColorPressed: '#00d5d5',
    primaryColorSuppl: '#8a2be2',
    borderRadius: '8px',
    bodyColor: '#091120',
    cardColor: 'rgba(18, 31, 52, 0.82)',
    modalColor: '#111c2f',
    popoverColor: '#111c2f',
    textColorBase: '#feffff',
    textColor1: '#feffff',
    textColor2: '#d2e0ec',
    textColor3: '#a8a9a9',
  },
  Button: {
    borderRadiusMedium: '8px',
    borderRadiusSmall: '8px',
  },
  Card: {
    borderRadius: '12px',
  },
}
</script>
