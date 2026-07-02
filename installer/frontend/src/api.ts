import type { InstallOptions, InstallPlan, InstallProgress, InstallResult } from './types'

const getInstallerAPI = () => {
  const api = window.go?.main?.App
  if (!api) {
    throw new Error('安装器桥接尚未就绪。')
  }
  return api
}

export const getInstallPlan = (): Promise<InstallPlan> => {
  return getInstallerAPI().GetInstallPlan()
}

export const startInstall = (options: InstallOptions): Promise<InstallResult> => {
  return getInstallerAPI().StartInstall(options)
}

export const cancelInstall = (): Promise<void> => {
  return getInstallerAPI().CancelInstall()
}

export const selectInstallDir = (currentDir: string): Promise<string> => {
  return getInstallerAPI().SelectInstallDir(currentDir)
}

export const startUninstall = (): Promise<InstallResult> => {
  return getInstallerAPI().StartUninstall()
}

export const onInstallProgress = (callback: (progress: InstallProgress) => void): (() => void) => {
  return window.runtime?.EventsOn('installer:progress', callback) ?? (() => undefined)
}
