Unicode true

####
## NexTunnel NSIS installer for Wails.
## Wails generates wails_tools.nsh during `wails build -nsis`; keep this file
## compatible with the default Wails installer entrypoint.
####

!include "wails_tools.nsh"
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "nsDialogs.nsh"
!include "WinMessages.nsh"

!include /NONFATAL "nextunnel_installer_config.local.nsh"

!ifndef WINTUN_DOWNLOAD_URL
  !define WINTUN_DOWNLOAD_URL "https://www.wintun.net/builds/wintun-0.14.1.zip"
!endif
!ifndef WINTUN_SHA256
  !define WINTUN_SHA256 "07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51"
!endif
!ifndef WINTUN_MODE
  !define WINTUN_MODE "bundled"
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

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_NOAUTOCLOSE

BrandingText "NexTunnel v${INFO_PRODUCTVERSION}"
Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$PROGRAMFILES64\${INFO_PRODUCTNAME}"
ShowInstDetails show
ShowUninstDetails show

Var Dialog
Var InstallDirText
Var DesktopShortcutCheckbox
Var RunAfterInstallCheckbox
Var WintunChoice
Var WintunDetectedPath
Var WintunBundledRadio
Var WintunDownloadRadio
Var WintunManualRadio
Var WintunSkipRadio
Var WintunResult
Var CreateDesktopShortcut
Var RunAfterInstall

Page custom WelcomePageCreate
Page custom OptionsPageCreate OptionsPageLeave
Page custom WintunPageCreate WintunPageLeave
!insertmacro MUI_PAGE_INSTFILES
Page custom FinishPageCreate FinishPageLeave

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

## The following two statements can be enabled by release scripts when signing is configured.
#!uninstfinalize 'signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 "%1"'
#!finalize 'signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 "%1"'

Function .onInit
  StrCpy $WintunChoice "${WINTUN_MODE}"
  StrCpy $WintunResult "pending"
  StrCpy $CreateDesktopShortcut "1"
  StrCpy $RunAfterInstall "1"
  !insertmacro wails.checkArchitecture
FunctionEnd

Function SetDarkDialog
  Pop $0
  SetCtlColors $0 0xEAF7FF 0x111827
  Push $0
FunctionEnd

Function CreateInstallerTitle
  ${NSD_CreateLabel} 0 0 100% 18u "NexTunnel 桌面端安装向导"
  Call SetDarkDialog
  Pop $0
  CreateFont $1 "Microsoft YaHei UI" 12 700
  SendMessage $0 ${WM_SETFONT} $1 1

  ${NSD_CreateLabel} 0 22u 100% 14u "安装客户端、WebView2 引导器，并准备真实 TUN 所需的官方 Wintun 组件。"
  Call SetDarkDialog
  Pop $0
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

  SetCtlColors $Dialog 0xEAF7FF 0x111827
  Call CreateInstallerTitle

  ${NSD_CreateLabel} 0 52u 100% 62u "本安装器使用 Wails 官方 NSIS 流程构建，采用可审计的原生 NSIS 自定义界面。$\r$\n$\r$\n安装过程会请求管理员权限，用于写入 Program Files、创建快捷方式，并把官方 wintun.dll 放到 NexTunnel.exe 同目录。安装 DLL 只解决组件缺失，首次创建虚拟网卡仍需要管理员权限。"
  Call SetDarkDialog
  Pop $0

  ${NSD_CreateLabel} 0 128u 100% 28u "点击“下一步”继续选择安装位置、快捷方式和 Wintun 处理方式。"
  Call SetDarkDialog
  Pop $0

  nsDialogs::Show
FunctionEnd

Function OptionsPageCreate
  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  SetCtlColors $Dialog 0xEAF7FF 0x111827
  Call CreateInstallerTitle

  ${NSD_CreateLabel} 0 52u 100% 12u "安装位置"
  Call SetDarkDialog
  Pop $0

  ${NSD_CreateText} 0 68u 76% 14u "$INSTDIR"
  Pop $InstallDirText

  ${NSD_CreateBrowseButton} 79% 67u 21% 16u "浏览..."
  Pop $0
  ${NSD_OnClick} $0 SelectInstallDirectory

  ${NSD_CreateCheckbox} 0 98u 100% 14u "创建桌面快捷方式"
  Pop $DesktopShortcutCheckbox
  ${If} $CreateDesktopShortcut == "1"
    ${NSD_SetState} $DesktopShortcutCheckbox ${BST_CHECKED}
  ${Else}
    ${NSD_SetState} $DesktopShortcutCheckbox ${BST_UNCHECKED}
  ${EndIf}

  ${NSD_CreateLabel} 0 122u 100% 28u "开始菜单快捷方式会默认创建。若安装目录位于 Program Files，当前安装器会使用管理员权限完成写入。"
  Call SetDarkDialog
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

  ${NSD_GetState} $DesktopShortcutCheckbox $0
  ${If} $0 == ${BST_CHECKED}
    StrCpy $CreateDesktopShortcut "1"
  ${Else}
    StrCpy $CreateDesktopShortcut "0"
  ${EndIf}
FunctionEnd

Function WintunPageCreate
  Call DetectWintun

  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  SetCtlColors $Dialog 0xEAF7FF 0x111827
  Call CreateInstallerTitle

  ${If} $WintunDetectedPath != ""
    ${NSD_CreateLabel} 0 52u 100% 24u "已检测到 Wintun：$WintunDetectedPath"
    Call SetDarkDialog
    Pop $0
    StrCpy $WintunChoice "present"
  ${Else}
    ${NSD_CreateLabel} 0 52u 100% 24u "未检测到 wintun.dll。建议安装器将官方匹配架构 DLL 放到 NexTunnel.exe 同目录。"
    Call SetDarkDialog
    Pop $0
  ${EndIf}

  !ifdef WINTUN_BUNDLED_DLL
    ${NSD_CreateRadioButton} 0 86u 100% 12u "使用安装包内置官方 Wintun DLL（推荐，离线可用）"
    Pop $WintunBundledRadio
  !else
    ${NSD_CreateRadioButton} 0 86u 100% 12u "安装包未内置 Wintun DLL"
    Pop $WintunBundledRadio
    EnableWindow $WintunBundledRadio 0
  !endif

  ${NSD_CreateRadioButton} 0 106u 100% 12u "联网下载官方 Wintun ZIP 并校验 SHA256"
  Pop $WintunDownloadRadio
  ${NSD_CreateRadioButton} 0 126u 100% 12u "我稍后手动安装 wintun.dll"
  Pop $WintunManualRadio
  ${NSD_CreateRadioButton} 0 146u 100% 12u "暂时跳过，仅使用 P2P/Relay 能力"
  Pop $WintunSkipRadio

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

  SetCtlColors $Dialog 0xEAF7FF 0x111827
  Call CreateInstallerTitle

  ${NSD_CreateLabel} 0 52u 100% 32u "NexTunnel 已安装到：$INSTDIR"
  Call SetDarkDialog
  Pop $0

  ${If} $WintunDetectedPath != ""
    ${NSD_CreateLabel} 0 88u 100% 28u "Wintun 状态：已就绪，路径为 $WintunDetectedPath。创建虚拟网卡仍需要管理员权限。"
    Call SetDarkDialog
    Pop $0
  ${Else}
    ${NSD_CreateLabel} 0 88u 100% 36u "Wintun 状态：未就绪。NexTunnel 可继续使用 P2P/Relay；如需真实系统路由 TUN，请在网络页执行修复或手动放置官方 wintun.dll。"
    Call SetDarkDialog
    Pop $0
  ${EndIf}

  ${NSD_CreateCheckbox} 0 136u 100% 14u "立即运行 NexTunnel"
  Pop $RunAfterInstallCheckbox
  ${If} $RunAfterInstall == "1"
    ${NSD_SetState} $RunAfterInstallCheckbox ${BST_CHECKED}
  ${Else}
    ${NSD_SetState} $RunAfterInstallCheckbox ${BST_UNCHECKED}
  ${EndIf}

  nsDialogs::Show
FunctionEnd

Function FinishPageLeave
  ${NSD_GetState} $RunAfterInstallCheckbox $0
  ${If} $0 == ${BST_CHECKED}
    ExecShell "open" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  ${EndIf}
FunctionEnd

Section
  !insertmacro wails.setShellContext

  !insertmacro wails.webview2runtime

  SetOutPath $INSTDIR

  !insertmacro wails.files
  Call InstallWintunIfRequired

  CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  ${If} $CreateDesktopShortcut == "1"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  ${EndIf}

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
