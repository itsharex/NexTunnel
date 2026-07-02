Unicode true

####
## NexTunnel NSIS installer for Wails.
## Wails generates wails_tools.nsh during `wails build -nsis`; keep this file
## compatible with the default Wails installer entrypoint.
####

!include "wails_tools.nsh"

!define NEXTUNNEL_COLOR_TEXT 0x111827
!define NEXTUNNEL_COLOR_MUTED 0x475569
!define NEXTUNNEL_COLOR_BG 0xF7FAFC
!define NEXTUNNEL_COLOR_PANEL 0xEAF6FF
!define NEXTUNNEL_COLOR_SURFACE 0xFFFFFF
!define NEXTUNNEL_WINDOW_WIDTH 820
!define NEXTUNNEL_WINDOW_HEIGHT 620
!define NEXTUNNEL_PAGE_X 14
!define NEXTUNNEL_PAGE_Y 14
!define NEXTUNNEL_PAGE_WIDTH 792
!define NEXTUNNEL_PAGE_HEIGHT 512
!define NEXTUNNEL_BUTTON_Y 548
!define NEXTUNNEL_BUTTON_WIDTH 112
!define NEXTUNNEL_BUTTON_HEIGHT 32

!include "LogicLib.nsh"
!include "nsDialogs.nsh"
!include "WinMessages.nsh"
!include "FileFunc.nsh"

!include /NONFATAL "nextunnel_installer_config.local.nsh"
!insertmacro GetRoot
!insertmacro DriveSpace

!ifndef WINTUN_DOWNLOAD_URL
  !define WINTUN_DOWNLOAD_URL "https://www.wintun.net/builds/wintun-0.14.1.zip"
!endif
!ifndef WINTUN_SHA256
  !define WINTUN_SHA256 "07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51"
!endif
!ifndef WINTUN_MODE
  !define WINTUN_MODE "bundled"
!endif
!ifndef REQUIRED_INSTALL_SPACE_MB
  !define REQUIRED_INSTALL_SPACE_MB 512
!endif

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

ManifestDPIAware true

Icon "..\icon.ico"
UninstallIcon "..\icon.ico"
BrandingText "NexTunnel v${INFO_PRODUCTVERSION}"
Caption "${INFO_PRODUCTNAME} 安装"
Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$PROGRAMFILES64\${INFO_PRODUCTNAME}"
ShowInstDetails show
ShowUninstDetails show
ChangeUI all "${NSISDIR}\Contrib\UIs\modern.exe"
InstProgressFlags smooth
InstallColors ${NEXTUNNEL_COLOR_TEXT} ${NEXTUNNEL_COLOR_BG}
CompletedText "NexTunnel 安装完成"

Var Dialog
Var InstallDirText
Var LicenseAgreeCheckbox
Var DesktopShortcutCheckbox
Var StartMenuShortcutCheckbox
Var AutoRunAfterInstallCheckbox
Var WintunChoice
Var WintunDetectedPath
Var WintunBundledRadio
Var WintunDownloadRadio
Var WintunManualRadio
Var WintunSkipRadio
Var WintunResult
Var CreateDesktopShortcut
Var CreateStartMenuShortcut
Var RunAfterInstall
Var InstallDriveRoot
Var InstallFreeSpaceMB
Var InstallerIsDarkMode
Var InstallerTitleFont
Var InstallerSubtitleFont
Var InstallerBodyFont
Var InstallerStrongFont
Var InstallerSmallFont
Var InstallerButtonFont

Page custom WelcomePageCreate
Page custom LicensePageCreate LicensePageLeave
Page custom OptionsPageCreate OptionsPageLeave
Page custom WintunPageCreate WintunPageLeave
Page instfiles InstallFilesPagePre InstallFilesPageShow
Page custom FinishPageCreate FinishPageLeave

UninstPage instfiles

LoadLanguageFile "${NSISDIR}\Contrib\Language files\SimpChinese.nlf"

## The following two statements can be enabled by release scripts when signing is configured.
#!uninstfinalize 'signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 "%1"'
#!finalize 'signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 "%1"'

Function .onInit
  Call ConfigureNativeTheme
  StrCpy $WintunChoice "${WINTUN_MODE}"
  StrCpy $WintunResult "pending"
  StrCpy $CreateDesktopShortcut "1"
  StrCpy $CreateStartMenuShortcut "1"
  StrCpy $RunAfterInstall "1"
  !insertmacro wails.checkArchitecture
FunctionEnd

Function .onGUIInit
  Call CreateInstallerFonts
  # NSIS 默认现代 UI 的内容框偏小；这里同步扩大主窗口、页面容器和底部按钮区。
  Call ResizeNativeInstallerChrome
  Call ApplyNativeModernFrame
  Call StyleNativeNavigation
FunctionEnd

Function .onUserAbort
  MessageBox MB_ICONQUESTION|MB_YESNO "确定要取消 NexTunnel 安装吗？" IDYES allowAbort
  Abort

  allowAbort:
FunctionEnd

Function CreateInstallerFonts
  CreateFont $InstallerTitleFont "Microsoft YaHei UI" 17 700
  CreateFont $InstallerSubtitleFont "Microsoft YaHei UI" 9 400
  CreateFont $InstallerBodyFont "Microsoft YaHei UI" 9 400
  CreateFont $InstallerStrongFont "Microsoft YaHei UI" 10 700
  CreateFont $InstallerSmallFont "Microsoft YaHei UI" 8 400
  CreateFont $InstallerButtonFont "Microsoft YaHei UI" 9 500
FunctionEnd

Function ResizeNativeInstallerChrome
  System::Call 'user32::GetSystemMetrics(i 0) i.r0'
  System::Call 'user32::GetSystemMetrics(i 1) i.r1'
  IntOp $2 $0 - ${NEXTUNNEL_WINDOW_WIDTH}
  IntOp $2 $2 / 2
  IntOp $3 $1 - ${NEXTUNNEL_WINDOW_HEIGHT}
  IntOp $3 $3 / 2
  ${If} $2 < 0
    StrCpy $2 0
  ${EndIf}
  ${If} $3 < 0
    StrCpy $3 0
  ${EndIf}
  System::Call 'user32::SetWindowPos(p $HWNDPARENT, p 0, i r2, i r3, i ${NEXTUNNEL_WINDOW_WIDTH}, i ${NEXTUNNEL_WINDOW_HEIGHT}, i 0x0014)'

  GetDlgItem $0 $HWNDPARENT 1018
  ${If} $0 <> 0
    System::Call 'user32::SetWindowPos(p $0, p 0, i ${NEXTUNNEL_PAGE_X}, i ${NEXTUNNEL_PAGE_Y}, i ${NEXTUNNEL_PAGE_WIDTH}, i ${NEXTUNNEL_PAGE_HEIGHT}, i 0x0014)'
  ${EndIf}

  GetDlgItem $0 $HWNDPARENT 2
  ${If} $0 <> 0
    System::Call 'user32::SetWindowPos(p $0, p 0, i 24, i ${NEXTUNNEL_BUTTON_Y}, i ${NEXTUNNEL_BUTTON_WIDTH}, i ${NEXTUNNEL_BUTTON_HEIGHT}, i 0x0014)'
  ${EndIf}

  GetDlgItem $0 $HWNDPARENT 3
  ${If} $0 <> 0
    System::Call 'user32::SetWindowPos(p $0, p 0, i 560, i ${NEXTUNNEL_BUTTON_Y}, i ${NEXTUNNEL_BUTTON_WIDTH}, i ${NEXTUNNEL_BUTTON_HEIGHT}, i 0x0014)'
  ${EndIf}

  GetDlgItem $0 $HWNDPARENT 1
  ${If} $0 <> 0
    System::Call 'user32::SetWindowPos(p $0, p 0, i 684, i ${NEXTUNNEL_BUTTON_Y}, i ${NEXTUNNEL_BUTTON_WIDTH}, i ${NEXTUNNEL_BUTTON_HEIGHT}, i 0x0014)'
  ${EndIf}

  GetDlgItem $0 $HWNDPARENT 1028
  ${If} $0 <> 0
    ShowWindow $0 0
  ${EndIf}
FunctionEnd

Function ConfigureNativeTheme
  ReadRegDWORD $0 HKCU "Software\Microsoft\Windows\CurrentVersion\Themes\Personalize" "AppsUseLightTheme"
  IfErrors useLightTheme

  ${If} $0 == 0
    StrCpy $InstallerIsDarkMode "1"
    Return
  ${EndIf}

  useLightTheme:
    StrCpy $InstallerIsDarkMode "0"
FunctionEnd

Function ApplyNativeModernFrame
  # 仅使用 NSIS 原生 System 插件调用 Windows DWM/WinAPI；旧系统不支持时会静默退回标准窗口。
  SetCtlColors $HWNDPARENT "" ${NEXTUNNEL_COLOR_BG}

  System::Call 'user32::GetWindowLong(p $HWNDPARENT, i -16) i.r0'
  IntOp $0 $0 & 0xFF0BFFFF
  System::Call 'user32::SetWindowLong(p $HWNDPARENT, i -16, i r0) i.r1'
  System::Call 'user32::SetWindowPos(p $HWNDPARENT, p 0, i 0, i 0, i 0, i 0, i 0x0027)'

  ${If} $InstallerIsDarkMode == "1"
    System::Call 'dwmapi::DwmSetWindowAttribute(p $HWNDPARENT, i 19, *i 1, i 4)'
    System::Call 'dwmapi::DwmSetWindowAttribute(p $HWNDPARENT, i 20, *i 1, i 4)'
  ${EndIf}
  System::Call 'dwmapi::DwmSetWindowAttribute(p $HWNDPARENT, i 33, *i 2, i 4)'
  System::Call 'dwmapi::DwmSetWindowAttribute(p $HWNDPARENT, i 38, *i 3, i 4)'
FunctionEnd

Function StyleDialog
  Pop $0
  SetCtlColors $0 "" ${NEXTUNNEL_COLOR_BG}
FunctionEnd

Function SetDarkDialog
  Pop $0
  SetCtlColors $0 ${NEXTUNNEL_COLOR_TEXT} transparent
  SendMessage $0 ${WM_SETFONT} $InstallerBodyFont 1
  Push $0
FunctionEnd

Function SetMutedDialog
  Pop $0
  SetCtlColors $0 ${NEXTUNNEL_COLOR_MUTED} transparent
  SendMessage $0 ${WM_SETFONT} $InstallerSmallFont 1
  Push $0
FunctionEnd

Function SetPanelDialog
  Pop $0
  SetCtlColors $0 ${NEXTUNNEL_COLOR_TEXT} ${NEXTUNNEL_COLOR_SURFACE}
  SendMessage $0 ${WM_SETFONT} $InstallerBodyFont 1
  Push $0
FunctionEnd

Function SetSectionTitleDialog
  Pop $0
  SetCtlColors $0 ${NEXTUNNEL_COLOR_TEXT} transparent
  SendMessage $0 ${WM_SETFONT} $InstallerStrongFont 1
  Push $0
FunctionEnd

Function StyleNativeNavigation
  GetDlgItem $0 $HWNDPARENT 1
  ${If} $0 <> 0
    SendMessage $0 ${WM_SETFONT} $InstallerButtonFont 1
  ${EndIf}
  GetDlgItem $0 $HWNDPARENT 2
  ${If} $0 <> 0
    SendMessage $0 ${WM_SETFONT} $InstallerButtonFont 1
  ${EndIf}
  GetDlgItem $0 $HWNDPARENT 3
  ${If} $0 <> 0
    SendMessage $0 ${WM_SETFONT} $InstallerButtonFont 1
  ${EndIf}
FunctionEnd

Function SetNextButtonText
  Pop $0
  GetDlgItem $1 $HWNDPARENT 1
  SendMessage $1 ${WM_SETTEXT} 0 "STR:$0"
FunctionEnd

Function CreateInstallerTitle
  ${NSD_CreateLabel} 0 0 100% 56u ""
  Call SetPanelDialog
  Pop $0
  SetCtlColors $0 ${NEXTUNNEL_COLOR_TEXT} ${NEXTUNNEL_COLOR_PANEL}

  ${NSD_CreateLabel} 18u 10u 74% 20u "NexTunnel 安装"
  Call SetPanelDialog
  Pop $0
  SetCtlColors $0 ${NEXTUNNEL_COLOR_TEXT} ${NEXTUNNEL_COLOR_PANEL}
  SendMessage $0 ${WM_SETFONT} $InstallerTitleFont 1

  ${NSD_CreateLabel} 18u 34u 84% 13u "安装客户端、WebView2 引导器和真实 TUN 所需组件"
  Call SetPanelDialog
  Pop $0
  SetCtlColors $0 ${NEXTUNNEL_COLOR_MUTED} ${NEXTUNNEL_COLOR_PANEL}
  SendMessage $0 ${WM_SETFONT} $InstallerSubtitleFont 1
FunctionEnd

Function DetectWintun
  StrCpy $WintunDetectedPath ""

  IfFileExists "$INSTDIR\wintun.dll" 0 +3
    StrCpy $WintunDetectedPath "$INSTDIR\wintun.dll"
    Goto done

  IfFileExists "$SYSDIR\wintun.dll" 0 +3
    StrCpy $WintunDetectedPath "$SYSDIR\wintun.dll"
    Goto done

  done:
FunctionEnd

Function WelcomePageCreate
  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  Push $Dialog
  Call StyleDialog
  Push "下一步"
  Call SetNextButtonText
  Call CreateInstallerTitle

  ${NSD_CreateLabel} 4u 72u 92% 14u "准备安装"
  Call SetSectionTitleDialog
  Pop $0

  ${NSD_CreateLabel} 4u 96u 92% 56u "安装器将复制 NexTunnel 客户端，按需安装 WebView2 引导器，并配置真实 TUN 所需的 Wintun 组件。安装过程会请求管理员权限，用于写入 Program Files、创建快捷方式和写入卸载信息。"
  Call SetDarkDialog
  Pop $0

  ${NSD_CreateLabel} 4u 168u 92% 42u "当前界面使用 NSIS 原生自定义页面实现，不依赖 nsNiuniuSkin 插件。点击“下一步”后可阅读许可协议，并选择安装位置、快捷方式和启动选项。"
  Call SetMutedDialog
  Pop $0

  nsDialogs::Show
FunctionEnd

Function LicensePageCreate
  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  Push $Dialog
  Call StyleDialog
  Push "下一步"
  Call SetNextButtonText
  Call CreateInstallerTitle

  ${NSD_CreateLabel} 4u 72u 92% 14u "软件许可协议"
  Call SetSectionTitleDialog
  Pop $0

  ${NSD_CreateText} 4u 96u 92% 96u "请在继续安装前阅读并同意 NexTunnel 软件许可条款。$\r$\n$\r$\nNexTunnel 按开源项目方式提供，安装和使用即表示你理解网络穿透、虚拟网卡和本地端口暴露可能带来的安全影响。请仅在你拥有管理权限且可信的设备上安装，并妥善保护 Relay Token、Control Plane Token 等敏感凭据。$\r$\n$\r$\n继续安装表示你同意自行确认部署环境、网络策略和第三方组件许可。"
  Pop $0
  SendMessage $0 ${EM_SETREADONLY} 1 0
  Push $0
  Call SetPanelDialog
  Pop $0

  # 许可勾选必须独立可见，未同意时不允许进入安装选项。
  ${NSD_CreateCheckbox} 4u 210u 92% 14u "我已阅读并同意许可条款"
  Pop $LicenseAgreeCheckbox
  Push $LicenseAgreeCheckbox
  Call SetDarkDialog
  Pop $0

  ${NSD_CreateLabel} 4u 232u 92% 18u "未勾选同意时无法继续安装。"
  Call SetMutedDialog
  Pop $0

  nsDialogs::Show
FunctionEnd

Function LicensePageLeave
  ${NSD_GetState} $LicenseAgreeCheckbox $0
  ${If} $0 != ${BST_CHECKED}
    MessageBox MB_ICONEXCLAMATION "请先勾选同意许可条款后继续。"
    Abort
  ${EndIf}
FunctionEnd

Function OptionsPageCreate
  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  Push $Dialog
  Call StyleDialog
  Push "下一步"
  Call SetNextButtonText
  Call CreateInstallerTitle

  ${NSD_CreateLabel} 4u 72u 92% 14u "安装选项"
  Call SetSectionTitleDialog
  Pop $0

  ${NSD_CreateLabel} 4u 96u 92% 12u "安装位置"
  Call SetMutedDialog
  Pop $0

  ${NSD_CreateText} 4u 114u 70% 16u "$INSTDIR"
  Pop $InstallDirText
  Push $InstallDirText
  Call SetPanelDialog
  Pop $0

  ${NSD_CreateBrowseButton} 77% 113u 19% 18u "浏览..."
  Pop $0
  SendMessage $0 ${WM_SETFONT} $InstallerButtonFont 1
  ${NSD_OnClick} $0 SelectInstallDirectory

  ${NSD_CreateCheckbox} 4u 154u 92% 14u "创建桌面快捷方式"
  Pop $DesktopShortcutCheckbox
  Push $DesktopShortcutCheckbox
  Call SetDarkDialog
  Pop $0
  ${If} $CreateDesktopShortcut == "1"
    ${NSD_SetState} $DesktopShortcutCheckbox ${BST_CHECKED}
  ${Else}
    ${NSD_SetState} $DesktopShortcutCheckbox ${BST_UNCHECKED}
  ${EndIf}

  ${NSD_CreateCheckbox} 4u 178u 92% 14u "创建开始菜单快捷方式"
  Pop $StartMenuShortcutCheckbox
  Push $StartMenuShortcutCheckbox
  Call SetDarkDialog
  Pop $0
  ${If} $CreateStartMenuShortcut == "1"
    ${NSD_SetState} $StartMenuShortcutCheckbox ${BST_CHECKED}
  ${Else}
    ${NSD_SetState} $StartMenuShortcutCheckbox ${BST_UNCHECKED}
  ${EndIf}

  ${NSD_CreateCheckbox} 4u 202u 92% 14u "安装完成后自动启动 NexTunnel"
  Pop $AutoRunAfterInstallCheckbox
  Push $AutoRunAfterInstallCheckbox
  Call SetDarkDialog
  Pop $0
  ${If} $RunAfterInstall == "1"
    ${NSD_SetState} $AutoRunAfterInstallCheckbox ${BST_CHECKED}
  ${Else}
    ${NSD_SetState} $AutoRunAfterInstallCheckbox ${BST_UNCHECKED}
  ${EndIf}

  ${NSD_CreateLabel} 4u 232u 92% 18u "安装器会验证路径并检查安装盘剩余空间，至少需要 ${REQUIRED_INSTALL_SPACE_MB} MB 可用空间。"
  Call SetMutedDialog
  Pop $0

  nsDialogs::Show
FunctionEnd

Function SelectInstallDirectory
  nsDialogs::SelectFolderDialog "选择 NexTunnel 安装位置" "$INSTDIR"
  Pop $0
  ${If} $0 != error
    StrCpy $INSTDIR $0
    ${NSD_SetText} $InstallDirText "$INSTDIR"
  ${EndIf}
FunctionEnd

Function OptionsPageLeave
  ${NSD_GetText} $InstallDirText $INSTDIR
  ${If} $INSTDIR == ""
    MessageBox MB_ICONEXCLAMATION "安装位置不能为空。"
    Abort
  ${EndIf}
  Call ValidateInstallDirectory

  ${NSD_GetState} $DesktopShortcutCheckbox $0
  ${If} $0 == ${BST_CHECKED}
    StrCpy $CreateDesktopShortcut "1"
  ${Else}
    StrCpy $CreateDesktopShortcut "0"
  ${EndIf}

  ${NSD_GetState} $StartMenuShortcutCheckbox $0
  ${If} $0 == ${BST_CHECKED}
    StrCpy $CreateStartMenuShortcut "1"
  ${Else}
    StrCpy $CreateStartMenuShortcut "0"
  ${EndIf}

  ${NSD_GetState} $AutoRunAfterInstallCheckbox $0
  ${If} $0 == ${BST_CHECKED}
    StrCpy $RunAfterInstall "1"
  ${Else}
    StrCpy $RunAfterInstall "0"
  ${EndIf}
FunctionEnd

Function ValidateInstallDirectory
  ${GetRoot} "$INSTDIR" $InstallDriveRoot
  ${If} $InstallDriveRoot == ""
    MessageBox MB_ICONEXCLAMATION "安装路径无效，请选择本机磁盘上的目录。"
    Abort
  ${EndIf}

  ${DriveSpace} "$InstallDriveRoot" "/D=F /S=M" $InstallFreeSpaceMB
  ${If} $InstallFreeSpaceMB == ""
    MessageBox MB_ICONEXCLAMATION "无法检查安装盘剩余空间，请选择其他安装位置。"
    Abort
  ${EndIf}
  ${If} $InstallFreeSpaceMB < ${REQUIRED_INSTALL_SPACE_MB}
    MessageBox MB_ICONEXCLAMATION "安装盘剩余空间不足。至少需要 ${REQUIRED_INSTALL_SPACE_MB} MB，当前可用 $InstallFreeSpaceMB MB。"
    Abort
  ${EndIf}
FunctionEnd

Function WintunPageCreate
  Call DetectWintun

  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  Push $Dialog
  Call StyleDialog
  Push "安装"
  Call SetNextButtonText
  Call CreateInstallerTitle

  ${If} $WintunDetectedPath != ""
    ${NSD_CreateLabel} 4u 72u 92% 24u "已检测到 Wintun：$WintunDetectedPath"
    Call SetDarkDialog
    Pop $0
    StrCpy $WintunChoice "present"
  ${Else}
    ${NSD_CreateLabel} 4u 72u 92% 24u "未检测到 wintun.dll。建议安装器将官方匹配架构 DLL 放到 NexTunnel.exe 同目录。"
    Call SetDarkDialog
    Pop $0
  ${EndIf}

  !ifdef WINTUN_BUNDLED_DLL
    ${NSD_CreateRadioButton} 4u 108u 10u 10u ""
    Pop $WintunBundledRadio
    ${NSD_CreateLabel} 20u 105u 88% 14u "内置官方 Wintun DLL（推荐，离线可用）"
    Call SetDarkDialog
    Pop $0
  !else
    ${NSD_CreateRadioButton} 4u 108u 10u 10u ""
    Pop $WintunBundledRadio
    ${NSD_CreateLabel} 20u 105u 88% 14u "本安装包未内置 Wintun DLL"
    Call SetMutedDialog
    Pop $0
    EnableWindow $WintunBundledRadio 0
  !endif

  ${NSD_CreateRadioButton} 4u 134u 10u 10u ""
  Pop $WintunDownloadRadio
  ${NSD_CreateLabel} 20u 131u 88% 14u "在线下载官方 Wintun ZIP，并校验 SHA256"
  Call SetDarkDialog
  Pop $0

  ${NSD_CreateRadioButton} 4u 160u 10u 10u ""
  Pop $WintunManualRadio
  ${NSD_CreateLabel} 20u 157u 88% 14u "稍后手动放置 wintun.dll"
  Call SetDarkDialog
  Pop $0

  ${NSD_CreateRadioButton} 4u 186u 10u 10u ""
  Pop $WintunSkipRadio
  ${NSD_CreateLabel} 20u 183u 88% 14u "暂时跳过，仅使用 P2P/Relay 能力"
  Call SetDarkDialog
  Pop $0

  ${NSD_CreateLabel} 4u 218u 92% 20u "如跳过 Wintun，真实系统路由 TUN 不可用；安装后仍可在网络页修复。"
  Call SetMutedDialog
  Pop $0

  ${If} $WintunDetectedPath != ""
    ${NSD_SetState} $WintunSkipRadio ${BST_CHECKED}
    EnableWindow $WintunBundledRadio 0
    EnableWindow $WintunDownloadRadio 0
    EnableWindow $WintunManualRadio 0
    EnableWindow $WintunSkipRadio 0
  ${Else}
    !ifdef WINTUN_BUNDLED_DLL
      ${If} $WintunChoice == "download"
        ${NSD_SetState} $WintunDownloadRadio ${BST_CHECKED}
      ${ElseIf} $WintunChoice == "manual"
        ${NSD_SetState} $WintunManualRadio ${BST_CHECKED}
      ${ElseIf} $WintunChoice == "skip"
        ${NSD_SetState} $WintunSkipRadio ${BST_CHECKED}
      ${Else}
        ${NSD_SetState} $WintunBundledRadio ${BST_CHECKED}
      ${EndIf}
    !else
      ${If} $WintunChoice == "manual"
        ${NSD_SetState} $WintunManualRadio ${BST_CHECKED}
      ${ElseIf} $WintunChoice == "skip"
        ${NSD_SetState} $WintunSkipRadio ${BST_CHECKED}
      ${Else}
        ${NSD_SetState} $WintunDownloadRadio ${BST_CHECKED}
      ${EndIf}
    !endif
  ${EndIf}

  nsDialogs::Show
FunctionEnd

Function WintunPageLeave
  ${NSD_GetState} $WintunBundledRadio $0
  ${If} $0 == ${BST_CHECKED}
    StrCpy $WintunChoice "bundled"
    Return
  ${EndIf}

  ${NSD_GetState} $WintunDownloadRadio $0
  ${If} $0 == ${BST_CHECKED}
    StrCpy $WintunChoice "download"
    Return
  ${EndIf}

  ${NSD_GetState} $WintunManualRadio $0
  ${If} $0 == ${BST_CHECKED}
    StrCpy $WintunChoice "manual"
    Return
  ${EndIf}

  ${NSD_GetState} $WintunSkipRadio $0
  ${If} $0 == ${BST_CHECKED}
    StrCpy $WintunChoice "skip"
    Return
  ${EndIf}
FunctionEnd

Function InstallBundledWintun
  !ifdef WINTUN_BUNDLED_DLL
    SetOutPath "$INSTDIR"
    File "/oname=wintun.dll" "${WINTUN_BUNDLED_DLL}"
    IfFileExists "$INSTDIR\wintun.dll" done failed

    failed:
      StrCpy $WintunResult "failed"
      DetailPrint "Wintun: failed to copy bundled DLL"
      MessageBox MB_ICONEXCLAMATION "无法复制内置 wintun.dll。安装会继续，你可以稍后在网络页修复。"
      Return

    done:
      StrCpy $WintunResult "bundled"
      DetailPrint "Wintun: bundled DLL installed to $INSTDIR\wintun.dll"
      Return
  !else
    StrCpy $WintunResult "missing"
    DetailPrint "Wintun: no bundled DLL in installer"
  !endif
FunctionEnd

Function DownloadAndInstallWintun
  InitPluginsDir

  DetailPrint "Wintun: downloading official package"
  nsExec::ExecToLog 'powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference = ''Stop''; [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; $$zip = Join-Path ''$PLUGINSDIR'' ''wintun.zip''; $$extract = Join-Path ''$PLUGINSDIR'' ''wintun''; Invoke-WebRequest -Uri ''${WINTUN_DOWNLOAD_URL}'' -UseBasicParsing -OutFile $$zip; $$expected = ''${WINTUN_SHA256}''.ToLowerInvariant(); if ([string]::IsNullOrWhiteSpace($$expected)) { throw ''Wintun SHA256 is required'' }; $$actual = (Get-FileHash -Algorithm SHA256 -Path $$zip).Hash.ToLowerInvariant(); if ($$actual -ne $$expected) { throw ''Wintun SHA256 mismatch'' }; Expand-Archive -Path $$zip -DestinationPath $$extract -Force"'
  Pop $0
  ${If} $0 != 0
    StrCpy $WintunResult "download_failed"
    DetailPrint "Wintun: download or extraction failed with exit code $0"
    MessageBox MB_ICONEXCLAMATION "自动下载 Wintun 失败。NexTunnel 主程序已继续安装，可稍后在网络页修复或手动放置官方 wintun.dll。"
    Return
  ${EndIf}

  ${If} ${IsNativeARM64}
    StrCpy $1 "$PLUGINSDIR\wintun\wintun\bin\arm64\wintun.dll"
  ${Else}
    StrCpy $1 "$PLUGINSDIR\wintun\wintun\bin\amd64\wintun.dll"
  ${EndIf}

  IfFileExists "$1" 0 missing
    CopyFiles /SILENT "$1" "$INSTDIR\wintun.dll"
    IfFileExists "$INSTDIR\wintun.dll" done failed

  missing:
    StrCpy $WintunResult "download_missing_dll"
    DetailPrint "Wintun: extracted DLL not found: $1"
    MessageBox MB_ICONEXCLAMATION "官方 Wintun 包已下载，但没有找到当前架构的 DLL。请稍后在网络页修复。"
    Return

  failed:
    StrCpy $WintunResult "copy_failed"
    DetailPrint "Wintun: failed to copy into $INSTDIR"
    MessageBox MB_ICONEXCLAMATION "无法复制 wintun.dll 到安装目录。请检查权限，或以管理员身份在网络页修复。"
    Return

  done:
    StrCpy $WintunResult "downloaded"
    DetailPrint "Wintun: installed to $INSTDIR\wintun.dll"
FunctionEnd

Function InstallWintunIfRequired
  Call DetectWintun
  ${If} $WintunDetectedPath != ""
    StrCpy $WintunResult "present"
    DetailPrint "Wintun: ready ($WintunDetectedPath)"
    Return
  ${EndIf}

  ${If} $WintunChoice == "bundled"
    Call InstallBundledWintun
    Call DetectWintun
    ${If} $WintunDetectedPath == ""
      DetailPrint "Wintun: bundled install unavailable, trying official download fallback"
      Call DownloadAndInstallWintun
    ${EndIf}
  ${ElseIf} $WintunChoice == "download"
    Call DownloadAndInstallWintun
  ${ElseIf} $WintunChoice == "manual"
    StrCpy $WintunResult "manual"
    DetailPrint "Wintun: manual installation selected"
  ${Else}
    StrCpy $WintunResult "skipped"
    DetailPrint "Wintun: skipped; real TUN remains unavailable until wintun.dll is installed"
  ${EndIf}
FunctionEnd

Function FinishPageCreate
  Call DetectWintun

  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  Push $Dialog
  Call StyleDialog
  Push "完成"
  Call SetNextButtonText
  Call CreateInstallerTitle

  ${NSD_CreateLabel} 4u 72u 92% 14u "安装完成"
  Call SetSectionTitleDialog
  Pop $0

  ${NSD_CreateLabel} 4u 98u 92% 32u "NexTunnel 已安装到：$INSTDIR"
  Call SetDarkDialog
  Pop $0

  ${If} $WintunDetectedPath != ""
    ${NSD_CreateLabel} 4u 142u 92% 28u "Wintun 状态：已就绪，路径为 $WintunDetectedPath。创建虚拟网卡仍需要管理员权限。"
    Call SetDarkDialog
    Pop $0
  ${Else}
    ${NSD_CreateLabel} 4u 142u 92% 42u "Wintun 状态：未就绪。NexTunnel 可继续使用 P2P/Relay；如需真实系统路由 TUN，请在网络页执行修复或手动放置官方 wintun.dll。"
    Call SetDarkDialog
    Pop $0
  ${EndIf}

  ${If} $RunAfterInstall == "1"
    ${NSD_CreateLabel} 4u 206u 92% 14u "点击“完成”后将自动启动 NexTunnel。"
  ${Else}
    ${NSD_CreateLabel} 4u 206u 92% 14u "点击“完成”退出安装向导。"
  ${EndIf}
  Call SetDarkDialog
  Pop $0

  nsDialogs::Show
FunctionEnd

Function InstallFilesPagePre
  Push "安装中"
  Call SetNextButtonText
FunctionEnd

Function InstallFilesPageShow
  Call ResizeNativeInstallerChrome
  Call StyleNativeNavigation

  GetDlgItem $0 $HWNDPARENT 1016
  ${If} $0 <> 0
    SetCtlColors $0 ${NEXTUNNEL_COLOR_TEXT} ${NEXTUNNEL_COLOR_BG}
    SendMessage $0 ${WM_SETFONT} $InstallerBodyFont 1
  ${EndIf}

  GetDlgItem $0 $HWNDPARENT 1004
  ${If} $0 <> 0
    SendMessage $0 ${WM_SETFONT} $InstallerBodyFont 1
  ${EndIf}
FunctionEnd

Function FinishPageLeave
  ${If} $RunAfterInstall == "1"
    ExecShell "open" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  ${EndIf}
FunctionEnd

Section
  !insertmacro wails.setShellContext

  DetailPrint "准备安装 WebView2 运行时"
  !insertmacro wails.webview2runtime

  SetOutPath $INSTDIR

  DetailPrint "复制 NexTunnel 应用文件"
  !insertmacro wails.files
  DetailPrint "检查并安装 Wintun 组件"
  Call InstallWintunIfRequired

  ${If} $CreateStartMenuShortcut == "1"
    DetailPrint "创建开始菜单快捷方式"
    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  ${EndIf}
  ${If} $CreateDesktopShortcut == "1"
    DetailPrint "创建桌面快捷方式"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  ${EndIf}

  DetailPrint "写入文件关联和卸载信息"
  !insertmacro wails.associateFiles
  !insertmacro wails.associateCustomProtocols

  !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
  !insertmacro wails.setShellContext

  RMDir /r "$AppData\${PRODUCT_EXECUTABLE}"
  RMDir /r "$INSTDIR"

  Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
  Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

  !insertmacro wails.unassociateFiles
  !insertmacro wails.unassociateCustomProtocols

  !insertmacro wails.deleteUninstaller
SectionEnd
