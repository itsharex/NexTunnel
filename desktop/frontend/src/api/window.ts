type WindowRuntimeMethod = () => void

interface WailsWindowRuntime {
  runtime?: {
    WindowMinimise?: WindowRuntimeMethod
    WindowToggleMaximise?: WindowRuntimeMethod
    Quit?: WindowRuntimeMethod
  }
}

const getWindowRuntime = (): WailsWindowRuntime['runtime'] | undefined => {
  return (window as unknown as WailsWindowRuntime).runtime
}

// minimiseWindow 使用 Wails 运行时控制无边框窗口，普通浏览器预览时安全忽略。
export const minimiseWindow = (): void => {
  getWindowRuntime()?.WindowMinimise?.()
}

// toggleMaximiseWindow 在自定义标题栏中切换最大化状态。
export const toggleMaximiseWindow = (): void => {
  getWindowRuntime()?.WindowToggleMaximise?.()
}

// closeWindow 关闭桌面应用窗口；预览环境不会执行任何破坏性动作。
export const closeWindow = (): void => {
  getWindowRuntime()?.Quit?.()
}
