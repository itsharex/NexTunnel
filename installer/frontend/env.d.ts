/// <reference types="vite/client" />

import type { InstallOptions, InstallPlan, InstallProgress, InstallResult } from './src/types'

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<object, object, unknown>
  export default component
}

interface WailsInstallerAPI {
  GetInstallPlan: () => Promise<InstallPlan>
  StartInstall: (options: InstallOptions) => Promise<InstallResult>
  CancelInstall: () => Promise<void>
  SelectInstallDir: (currentDir: string) => Promise<string>
  StartUninstall: () => Promise<InstallResult>
}

interface WailsRuntimeAPI {
  EventsOn: (eventName: string, callback: (data: InstallProgress) => void) => () => void
  WindowMinimise?: () => void
  Quit?: () => void
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: WailsInstallerAPI
      }
    }
    runtime?: WailsRuntimeAPI
  }
}
