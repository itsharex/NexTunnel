<template>
  <main class="installer-window" :class="`state-${viewState}`">
    <header class="title-bar" aria-label="窗口控制">
      <div class="title-caption">
        <span class="title-dot" aria-hidden="true"></span>
        <span>NexTunnel Installer</span>
      </div>
      <div class="window-controls">
        <button class="win-btn" type="button" title="最小化" @click="handleMinimize">
          <svg viewBox="0 0 14 14" aria-hidden="true">
            <path d="M2 7h10" />
          </svg>
          <span class="sr-only">最小化</span>
        </button>
        <button
          class="win-btn close"
          type="button"
          :title="isBusy ? '取消当前操作' : '关闭'"
          @click="handleClose"
        >
          <svg viewBox="0 0 14 14" aria-hidden="true">
            <path d="m3 3 8 8M11 3l-8 8" />
          </svg>
          <span class="sr-only">{{ isBusy ? '取消当前操作' : '关闭' }}</span>
        </button>
      </div>
    </header>

    <section class="left-panel" aria-label="NexTunnel 品牌">
      <div class="cyber-grid" aria-hidden="true"></div>
      <div class="left-glow primary" aria-hidden="true"></div>
      <div class="left-glow secondary" aria-hidden="true"></div>

      <div class="brand-core">
        <div class="logo-plate">
          <img class="logo-image" :src="logoUrl" alt="NexTunnel" />
        </div>
        <div class="brand-text">
          <p class="brand-kicker">NAT TUNNEL CLIENT</p>
          <h1>NexTunnel</h1>
          <p>安全部署桌面端、驱动组件和系统集成入口。</p>
        </div>
      </div>

      <div class="brand-bottom">
        <dl class="brand-meta" aria-label="安装包信息">
          <div>
            <dt>版本</dt>
            <dd>{{ versionText }}</dd>
          </div>
          <div>
            <dt>平台</dt>
            <dd>{{ targetText }}</dd>
          </div>
        </dl>
        <ul class="brand-points" aria-label="安装器能力">
          <li>Payload 校验</li>
          <li>回滚保护</li>
          <li>系统快捷方式</li>
        </ul>
      </div>
    </section>

    <section class="right-panel" aria-label="安装器操作">
      <nav class="flow-steps" aria-label="安装步骤">
        <div v-for="step in flowSteps" :key="step.key" :class="['flow-step', step.status]">
          <span>{{ step.index }}</span>
          <strong>{{ step.label }}</strong>
        </div>
      </nav>

      <div class="view-stack">
        <section v-if="viewState === 'prepare'" class="view-step prepare-view">
          <div class="view-content">
            <div class="section-heading">
              <p class="eyebrow">准备安装</p>
              <h2>安装 NexTunnel 客户端</h2>
              <p class="version-sub">版本 {{ versionText }} · 默认安装到 {{ conciseInstallDir }}</p>
            </div>

            <div class="overview-layout">
              <article class="install-hero">
                <span class="hero-label">推荐</span>
                <h3>快速安装</h3>
                <p>使用默认目录和常用快捷方式，安装器会自动完成校验、替换和系统集成。</p>
                <div :class="['hero-status', canInstall ? 'ok' : 'blocked']">
                  <span aria-hidden="true"></span>
                  <strong>{{ canInstall ? '环境已就绪' : blockedReason }}</strong>
                </div>
              </article>

              <div class="overview-cards" aria-label="安装前状态摘要">
                <article v-for="card in overviewCards" :key="card.key" :class="['overview-card', card.state]">
                  <span class="status-led" aria-hidden="true"></span>
                  <div>
                    <strong>{{ card.value }}</strong>
                    <small>{{ card.label }}</small>
                  </div>
                </article>
              </div>
            </div>

            <p v-if="planNotice" :class="['inline-alert', planNoticeState]">{{ planNotice }}</p>
          </div>

          <footer class="action-area">
            <button class="main-btn" type="button" :disabled="!canInstall" @click="handleInstall">
              立即安装
            </button>
            <div class="action-footer">
              <button class="secondary-inline" type="button" @click="goToCustomize">
                自定义安装
              </button>
              <button class="text-button danger" type="button" :disabled="isBusy" @click="handleUninstall">
                卸载已安装版本
              </button>
            </div>
          </footer>
        </section>

        <section v-else-if="viewState === 'customize'" class="view-step customize-view">
          <div class="view-content">
            <div class="section-heading compact">
              <p class="eyebrow">安装选项</p>
              <h2>选择安装位置和快捷方式</h2>
              <p class="version-sub">自定义选项不会改变安装器后端行为，只调整已支持的安装参数。</p>
            </div>

            <div class="settings-layout">
              <section class="settings-panel" aria-label="安装选项">
                <label class="field">
                  <span>安装目录</span>
                  <div class="path-input-group">
                    <input v-model="options.install_dir" type="text" :disabled="isBusy" spellcheck="false" />
                    <button class="browse-button" type="button" :disabled="isBusy" @click="handleBrowseDir">
                      浏览
                    </button>
                  </div>
                </label>

                <div class="option-list">
                  <label v-for="option in shortcutOptions" :key="option.key" class="check-option">
                    <input v-model="option.model.value" type="checkbox" :disabled="isBusy" />
                    <span>
                      <strong>{{ option.label }}</strong>
                      <small>{{ option.detail }}</small>
                    </span>
                  </label>
                </div>
              </section>

              <aside class="checks-panel" aria-label="前置检查">
                <div class="checks-heading">
                  <strong>前置检查</strong>
                  <button class="ghost-icon" type="button" title="刷新状态" @click="refreshPlan">
                    <svg viewBox="0 0 20 20" aria-hidden="true">
                      <path d="M16 7a6 6 0 1 0 1 4M16 7V3m0 4h-4" />
                    </svg>
                    <span class="sr-only">刷新状态</span>
                  </button>
                </div>

                <div class="check-list">
                  <article v-for="card in readinessCards" :key="card.key" :class="['check-row', card.state]">
                    <span class="status-led" aria-hidden="true"></span>
                    <div>
                      <strong>{{ card.label }}</strong>
                      <small>{{ card.value }} · {{ card.detail }}</small>
                    </div>
                  </article>
                  <article v-for="item in integrityItems" :key="item.key" :class="['check-row', item.state]">
                    <span class="status-led" aria-hidden="true"></span>
                    <div>
                      <strong>{{ item.label }}</strong>
                      <small>{{ item.value }}</small>
                    </div>
                  </article>
                </div>
              </aside>
            </div>

            <p v-if="planNotice" :class="['inline-alert', planNoticeState]">{{ planNotice }}</p>
          </div>

          <footer class="action-area split">
            <button class="secondary-btn" type="button" :disabled="isBusy" @click="viewState = 'prepare'">
              返回
            </button>
            <button class="main-btn" type="button" :disabled="!canInstall" @click="handleInstall">
              按当前选项安装
            </button>
          </footer>
        </section>

        <section v-else-if="isProgressView" class="view-step progress-view" aria-live="polite">
          <div class="view-content">
            <div class="section-heading">
              <p class="eyebrow">{{ progressEyebrow }}</p>
              <h2>{{ progressTitle }}</h2>
              <p class="version-sub">{{ progressSubtitle }}</p>
            </div>

            <div class="stage-list" aria-label="当前阶段">
              <div v-for="step in progressSteps" :key="step.key" :class="['stage-item', step.status]">
                <span aria-hidden="true"></span>
                <strong>{{ step.label }}</strong>
              </div>
            </div>

            <div class="progress-card">
              <div class="progress-meta">
                <strong>{{ progressMessage }}</strong>
                <span>{{ progressPercent }}%</span>
              </div>
              <div class="progress-track">
                <span :style="{ width: `${progressPercent}%` }"></span>
              </div>
              <p v-if="cancelNotice" class="progress-hint">{{ cancelNotice }}</p>
              <p v-else-if="progress.error" class="progress-hint error">{{ progress.error }}</p>
              <p v-else class="progress-hint">{{ progressPhaseLabel }}</p>
            </div>
          </div>

          <footer class="action-area">
            <button class="secondary-btn" type="button" :disabled="cancelRequested" @click="handleCancel">
              {{ cancelRequested ? '正在取消' : '取消操作' }}
            </button>
          </footer>
        </section>

        <section v-else-if="viewState === 'finished'" class="view-step result-view" aria-live="polite">
          <div class="view-content result-content">
            <div :class="['result-icon', result?.error ? 'warning' : 'success']">
              <svg v-if="!result?.error" viewBox="0 0 28 28" aria-hidden="true">
                <path d="m7 14 5 5L22 9" />
              </svg>
              <svg v-else viewBox="0 0 28 28" aria-hidden="true">
                <path d="M14 6v10m0 5v1" />
              </svg>
            </div>
            <p class="eyebrow">{{ finishedEyebrow }}</p>
            <h2>{{ finishedTitle }}</h2>
            <p class="version-sub">{{ finishedMessage }}</p>

            <div class="result-summary">
              <div>
                <span>{{ resultPathLabel }}</span>
                <strong>{{ resultPath }}</strong>
              </div>
              <div>
                <span>版本</span>
                <strong>{{ result?.version || versionText }}</strong>
              </div>
              <div v-if="lastAction === 'install'">
                <span>启动状态</span>
                <strong>{{ launchStateText }}</strong>
              </div>
            </div>
          </div>

          <footer class="action-area split">
            <button class="secondary-btn" type="button" @click="returnToPrepare">返回安装选项</button>
            <button class="main-btn" type="button" @click="handleClose">关闭安装器</button>
          </footer>
        </section>

        <section v-else class="view-step result-view failed-view" aria-live="assertive">
          <div class="view-content result-content">
            <div class="result-icon failed">
              <svg viewBox="0 0 28 28" aria-hidden="true">
                <path d="m8 8 12 12M20 8 8 20" />
              </svg>
            </div>
            <p class="eyebrow">{{ lastAction === 'uninstall' ? '卸载失败' : '安装失败' }}</p>
            <h2>操作未完成</h2>
            <p class="version-sub">{{ failureMessage }}</p>

            <div class="diagnostic-panel">
              <span>当前阶段</span>
              <strong>{{ progressPhaseLabel }}</strong>
              <small>{{ progressMessage }}</small>
            </div>
          </div>

          <footer class="action-area split">
            <button class="secondary-btn" type="button" @click="returnToPrepare">返回设置</button>
            <button class="main-btn" type="button" @click="retryLastAction">{{ retryText }}</button>
          </footer>
        </section>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, toRef } from 'vue'
import {
  cancelInstall,
  getInstallPlan,
  onInstallProgress,
  selectInstallDir,
  startInstall,
  startUninstall,
} from './api'
import type { InstallOptions, InstallPlan, InstallProgress, InstallResult } from './types'
import logoUrl from '../../../desktop/frontend/src/assets/logo.png'

type ViewState = 'prepare' | 'customize' | 'installing' | 'finished' | 'failed' | 'uninstalling'
type ActionKind = 'install' | 'uninstall'
type StatusState = 'ok' | 'pending' | 'blocked' | 'neutral'
type StepStatus = 'waiting' | 'active' | 'done' | 'error'

interface StatusCard {
  key: string
  label: string
  value: string
  detail: string
  state: StatusState
}

interface FlowStep {
  key: string
  index: string
  label: string
  status: StepStatus
}

interface IntegrityItem {
  key: string
  label: string
  value: string
  state: StatusState
}

interface ProgressStep {
  key: string
  label: string
  status: StepStatus
}

const installPhaseOrder = ['preparing', 'validating', 'extracting', 'replacing', 'integrating', 'complete']
const installPhaseLabels: Record<string, string> = {
  preparing: '准备环境',
  validating: '校验安装包',
  extracting: '展开文件',
  replacing: '替换版本',
  integrating: '写入系统',
  rollback: '回滚变更',
  uninstalling: '移除组件',
  complete: '完成',
}

const plan = ref<InstallPlan | null>(null)
const result = ref<InstallResult | null>(null)
const viewState = ref<ViewState>('prepare')
const lastAction = ref<ActionKind>('install')
const errorMessage = ref('')
const isBusy = ref(false)
const cancelRequested = ref(false)
const cancelNotice = ref('')
const hasInitializedDir = ref(false)

const progress = reactive<InstallProgress>({
  phase: 'preparing',
  percent: 0,
  message: '',
  error: '',
})

const options = reactive<InstallOptions>({
  install_dir: '',
  create_desktop_shortcut: true,
  create_start_menu_shortcut: true,
  launch_after_install: true,
})

const shortcutOptions = [
  {
    key: 'desktop',
    label: '桌面快捷方式',
    detail: '在桌面创建 NexTunnel 启动入口',
    model: toRef(options, 'create_desktop_shortcut'),
  },
  {
    key: 'start-menu',
    label: '开始菜单快捷方式',
    detail: '写入 Windows 开始菜单',
    model: toRef(options, 'create_start_menu_shortcut'),
  },
  {
    key: 'launch',
    label: '完成后启动',
    detail: '安装成功后尝试启动客户端',
    model: toRef(options, 'launch_after_install'),
  },
]

let unsubscribeProgress: (() => void) | null = null

const versionText = computed(() => plan.value?.version || result.value?.version || '检测中')
const targetText = computed(() => plan.value?.target || 'windows/amd64')
const progressPercent = computed(() => clamp(progress.percent, 0, 100))
const isProgressView = computed(() => viewState.value === 'installing' || viewState.value === 'uninstalling')
const conciseInstallDir = computed(() => options.install_dir || plan.value?.default_dir || '默认目录')

const flowSteps = computed<FlowStep[]>(() => {
  const currentIndex = currentFlowIndex.value
  return [
    { key: 'prepare', index: '01', label: '准备', status: flowStepStatus(0, currentIndex) },
    { key: 'customize', index: '02', label: '选项', status: flowStepStatus(1, currentIndex) },
    { key: 'execute', index: '03', label: lastAction.value === 'uninstall' ? '卸载' : '安装', status: flowStepStatus(2, currentIndex) },
    { key: 'finish', index: '04', label: '完成', status: flowStepStatus(3, currentIndex) },
  ]
})

const currentFlowIndex = computed(() => {
  switch (viewState.value) {
    case 'customize':
      return 1
    case 'installing':
    case 'uninstalling':
    case 'failed':
      return 2
    case 'finished':
      return 3
    default:
      return 0
  }
})

const overviewCards = computed<StatusCard[]>(() => {
  const componentBlocked = !plan.value?.webview2_ready || !plan.value?.wintun_included
  return [
    {
      key: 'admin',
      label: '权限',
      value: plan.value?.is_admin ? '管理员就绪' : '需要管理员',
      detail: '',
      state: plan.value?.is_admin ? 'ok' : 'blocked',
    },
    {
      key: 'payload',
      label: '安装包',
      value: plan.value?.payload_ready ? 'Payload 已校验' : 'Payload 缺失',
      detail: '',
      state: plan.value?.payload_ready ? 'ok' : 'blocked',
    },
    {
      key: 'components',
      label: '组件',
      value: componentBlocked ? '待安装/修复' : '运行组件就绪',
      detail: '',
      state: componentBlocked ? 'pending' : 'ok',
    },
  ]
})

const readinessCards = computed<StatusCard[]>(() => {
  const currentPlan = plan.value
  return [
    {
      key: 'admin',
      label: '管理员权限',
      value: currentPlan?.is_admin ? '已获取' : '需要管理员',
      detail: currentPlan?.is_admin ? '可写入 Program Files 与 HKLM' : '请使用管理员权限启动',
      state: currentPlan?.is_admin ? 'ok' : 'blocked',
    },
    {
      key: 'payload',
      label: '安装包 Payload',
      value: currentPlan?.payload_ready ? '已内置' : '不可用',
      detail: currentPlan?.payload_ready ? shortHash(currentPlan.payload_sha256) : '请重新构建安装器',
      state: currentPlan?.payload_ready ? 'ok' : 'blocked',
    },
    {
      key: 'webview2',
      label: 'WebView2 Runtime',
      value: currentPlan?.webview2_ready ? '已就绪' : '待引导',
      detail: currentPlan?.webview2_ready ? '系统运行时可用' : currentPlan?.webview2_mode || '内置引导器',
      state: currentPlan?.webview2_ready ? 'ok' : 'pending',
    },
    {
      key: 'wintun',
      label: 'Wintun DLL',
      value: currentPlan?.wintun_included ? '随包提供' : '未随包',
      detail: currentPlan?.wintun_included ? '网络组件可部署' : '安装后需修复组件',
      state: currentPlan?.wintun_included ? 'ok' : 'pending',
    },
  ]
})

const integrityItems = computed<IntegrityItem[]>(() => {
  const signing = plan.value?.signing || 'checking'
  const isUnsigned = /unsigned/i.test(signing)
  return [
    {
      key: 'signing',
      label: '签名状态',
      value: signing,
      state: isUnsigned ? 'pending' : 'ok',
    },
    {
      key: 'space',
      label: '所需空间',
      value: `${plan.value?.required_space_mb ?? 512} MB`,
      state: 'neutral',
    },
  ]
})

const blockedReason = computed(() => {
  if (!plan.value) return '正在读取安装计划'
  if (plan.value.error) return plan.value.error
  if (!plan.value.is_admin) return '需要管理员权限'
  if (!plan.value.payload_ready) return '安装包 payload 不完整'
  if (!options.install_dir.trim()) return '请选择安装目录'
  return ''
})

const canInstall = computed(() => !isBusy.value && blockedReason.value === '')
const planNotice = computed(() => blockedReason.value || errorMessage.value)
const planNoticeState = computed(() => (canInstall.value ? 'ok' : 'blocked'))

const progressEyebrow = computed(() => (viewState.value === 'uninstalling' ? '卸载中' : '安装中'))
const progressTitle = computed(() => {
  if (viewState.value === 'uninstalling') return '正在移除 NexTunnel'
  return progress.phase === 'rollback' ? '正在回滚变更' : '正在部署核心组件'
})
const progressSubtitle = computed(() => {
  if (cancelRequested.value) return '已请求取消，等待当前安全步骤完成。'
  return viewState.value === 'uninstalling'
    ? '停止进程、移除快捷方式和卸载项，然后清理安装目录。'
    : '校验 payload、解压 staging、替换旧版本并写入系统集成信息。'
})
const progressMessage = computed(() => progress.message || (isBusy.value ? '正在等待后端进度事件' : '等待开始'))
const progressPhaseLabel = computed(() => installPhaseLabels[progress.phase] || progress.phase || '准备环境')

const progressSteps = computed<ProgressStep[]>(() => {
  if (lastAction.value === 'uninstall') {
    return uninstallProgressSteps(progressPercent.value)
  }
  if (progress.phase === 'rollback') {
    return rollbackProgressSteps()
  }
  return installProgressSteps(progress.phase)
})

const finishedEyebrow = computed(() => (lastAction.value === 'uninstall' ? '卸载完成' : '安装完成'))
const finishedTitle = computed(() => {
  if (lastAction.value === 'uninstall') return 'NexTunnel 已移除'
  return result.value?.error ? '安装完成，但需要留意' : 'NexTunnel 已准备就绪'
})
const finishedMessage = computed(() => {
  if (result.value?.error) return result.value.error
  if (lastAction.value === 'uninstall') return '系统集成信息和默认安装目录已清理。'
  return '主程序、卸载入口和快捷方式已按选项写入。'
})
const resultPathLabel = computed(() => (lastAction.value === 'uninstall' ? '原安装位置' : '安装位置'))
const resultPath = computed(() => result.value?.app_path || options.install_dir || plan.value?.default_dir || '--')
const launchStateText = computed(() => {
  if (!options.launch_after_install) return '已按选项跳过'
  if (result.value?.error) return '启动失败'
  return '已尝试启动'
})
const failureMessage = computed(() => {
  return normalizeErrorMessage(errorMessage.value || progress.error || result.value?.error || '未知错误')
})
const retryText = computed(() => (lastAction.value === 'uninstall' ? '重试卸载' : '重试安装'))

onMounted(async () => {
  unsubscribeProgress = onInstallProgress((nextProgress) => {
    Object.assign(progress, nextProgress)
    if (nextProgress.error) {
      errorMessage.value = nextProgress.error
    }
  })
  await refreshPlan()
})

onUnmounted(() => {
  unsubscribeProgress?.()
})

const refreshPlan = async (): Promise<void> => {
  try {
    errorMessage.value = ''
    const nextPlan = await getInstallPlan()
    plan.value = nextPlan
    if (!hasInitializedDir.value || !options.install_dir.trim()) {
      options.install_dir = nextPlan.default_dir
      hasInitializedDir.value = true
    }
    if (nextPlan.error) {
      errorMessage.value = nextPlan.error
    }
  } catch (error) {
    errorMessage.value = normalizeErrorMessage(error)
  }
}

const goToCustomize = (): void => {
  errorMessage.value = ''
  viewState.value = 'customize'
}

const handleBrowseDir = async (): Promise<void> => {
  if (isBusy.value) return
  try {
    const selectedDir = await selectInstallDir(options.install_dir)
    if (selectedDir.trim()) {
      options.install_dir = selectedDir.trim()
    }
  } catch (error) {
    errorMessage.value = normalizeErrorMessage(error)
  }
}

const handleInstall = async (): Promise<void> => {
  if (!canInstall.value) return
  lastAction.value = 'install'
  viewState.value = 'installing'
  resetOperationState('preparing', '正在启动安装')
  try {
    const installResult = await startInstall({ ...options, install_dir: options.install_dir.trim() })
    result.value = installResult
    if (installResult.success) {
      viewState.value = 'finished'
    } else {
      errorMessage.value = installResult.error
      viewState.value = 'failed'
    }
  } catch (error) {
    errorMessage.value = normalizeErrorMessage(error)
    viewState.value = 'failed'
  } finally {
    isBusy.value = false
    cancelRequested.value = false
  }
}

const handleUninstall = async (): Promise<void> => {
  if (isBusy.value) return
  lastAction.value = 'uninstall'
  viewState.value = 'uninstalling'
  resetOperationState('uninstalling', '正在启动卸载')
  try {
    const uninstallResult = await startUninstall()
    result.value = uninstallResult
    if (uninstallResult.success) {
      viewState.value = 'finished'
    } else {
      errorMessage.value = uninstallResult.error
      viewState.value = 'failed'
    }
  } catch (error) {
    errorMessage.value = normalizeErrorMessage(error)
    viewState.value = 'failed'
  } finally {
    isBusy.value = false
    cancelRequested.value = false
  }
}

const handleCancel = async (): Promise<void> => {
  if (!isBusy.value || cancelRequested.value) return
  cancelRequested.value = true
  cancelNotice.value = '正在请求取消，安装器会等待后端完成当前安全步骤。'
  try {
    await cancelInstall()
  } catch (error) {
    errorMessage.value = normalizeErrorMessage(error)
  }
}

const retryLastAction = async (): Promise<void> => {
  if (lastAction.value === 'uninstall') {
    await handleUninstall()
    return
  }
  await handleInstall()
}

const returnToPrepare = async (): Promise<void> => {
  if (isBusy.value) return
  result.value = null
  errorMessage.value = ''
  cancelNotice.value = ''
  cancelRequested.value = false
  viewState.value = 'prepare'
  await refreshPlan()
}

const handleMinimize = (): void => {
  window.runtime?.WindowMinimise?.()
}

const handleClose = async (): Promise<void> => {
  // 安装或卸载进行中不直接退出，先交给后端取消流程收尾。
  if (isBusy.value) {
    await handleCancel()
    return
  }
  window.runtime?.Quit?.()
}

const resetOperationState = (phase: string, message: string): void => {
  isBusy.value = true
  result.value = null
  errorMessage.value = ''
  cancelRequested.value = false
  cancelNotice.value = ''
  Object.assign(progress, { phase, percent: 0, message, error: '' })
}

const installProgressSteps = (currentPhase: string): ProgressStep[] => {
  const activeIndex = installPhaseOrder.indexOf(currentPhase)
  return installPhaseOrder.map((phase, index) => ({
    key: phase,
    label: installPhaseLabels[phase],
    status: phaseStepStatus(index, activeIndex),
  }))
}

const rollbackProgressSteps = (): ProgressStep[] => {
  return [
    ...installPhaseOrder.slice(0, -1).map((phase) => ({
      key: phase,
      label: installPhaseLabels[phase],
      status: 'done' as StepStatus,
    })),
    { key: 'rollback', label: installPhaseLabels.rollback, status: 'error' },
  ]
}

const uninstallProgressSteps = (percent: number): ProgressStep[] => {
  const steps = [
    { key: 'stop', label: '停止进程', doneAt: 45 },
    { key: 'integrate', label: '移除集成', doneAt: 75 },
    { key: 'files', label: '删除文件', doneAt: 98 },
    { key: 'complete', label: '完成', doneAt: 100 },
  ]
  const activeIndex = steps.findIndex((step) => percent < step.doneAt)
  return steps.map((step, index) => ({
    key: step.key,
    label: step.label,
    status: percent >= step.doneAt ? 'done' : index === activeIndex ? 'active' : 'waiting',
  }))
}

const flowStepStatus = (index: number, activeIndex: number): StepStatus => {
  if (index < activeIndex) return 'done'
  if (index === activeIndex) {
    return viewState.value === 'failed' && index === activeIndex ? 'error' : 'active'
  }
  return 'waiting'
}

const phaseStepStatus = (index: number, activeIndex: number): StepStatus => {
  if (activeIndex < 0) return 'waiting'
  if (index < activeIndex) return 'done'
  if (index === activeIndex) return 'active'
  return 'waiting'
}

const clamp = (value: number, min: number, max: number): number => {
  return Math.min(max, Math.max(min, Number.isFinite(value) ? value : min))
}

const shortHash = (hash: string): string => {
  if (!hash) return '等待 manifest'
  return `SHA256 ${hash.slice(0, 10)}...${hash.slice(-6)}`
}

const normalizeErrorMessage = (error: unknown): string => {
  const message = error instanceof Error ? error.message : String(error)
  if (/context canceled|context cancelled/i.test(message)) {
    return '操作已取消，系统未完成本次更改。'
  }
  return message
}
</script>
