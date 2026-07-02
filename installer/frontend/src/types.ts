export interface InstallOptions {
  install_dir: string
  create_desktop_shortcut: boolean
  create_start_menu_shortcut: boolean
  launch_after_install: boolean
}

export interface InstallProgress {
  phase: string
  percent: number
  message: string
  error: string
}

export interface InstallResult {
  success: boolean
  version: string
  app_path: string
  error: string
}

export interface InstallPlan {
  version: string
  target: string
  default_dir: string
  payload_ready: boolean
  payload_sha256: string
  app_executable: string
  requires_admin: boolean
  is_admin: boolean
  webview2_ready: boolean
  webview2_mode: string
  required_space_mb: number
  wintun_included: boolean
  signing: string
  error: string
}
