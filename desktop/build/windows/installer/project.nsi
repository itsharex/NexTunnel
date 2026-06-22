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
  !define WINTUN_SHA256 ""
!endif

# The version information for this two must consist of 4 parts.
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
!define MUI_WELCOMEFINISHPAGE_BITMAP "resources\nextunnel-welcome.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "resources\nextunnel-welcome.bmp"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_WELCOMEPAGE_TITLE "Install NexTunnel"
!define MUI_WELCOMEPAGE_TEXT "NexTunnel will install the desktop client and check the Windows Wintun component required for real virtual networking. Administrator privileges are required for application files and shortcuts."
!define MUI_FINISHPAGE_TITLE "NexTunnel Setup Complete"
!define MUI_FINISHPAGE_TEXT "If wintun.dll was not installed, NexTunnel can still use P2P and Relay features. Real system-route TUN will remain blocked until Wintun is available."

BrandingText "NexTunnel v${INFO_PRODUCTVERSION}"
Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$PROGRAMFILES64\${INFO_PRODUCTNAME}"
ShowInstDetails show
ShowUninstDetails show

Var WintunChoice
Var WintunDetectedPath
Var WintunDownloadRadio
Var WintunManualRadio
Var WintunSkipRadio

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
Page custom WintunPageCreate WintunPageLeave
!insertmacro MUI_PAGE_INSTFILES
Page custom InstallSummaryPageCreate
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

## The following two statements can be enabled by release scripts when signing is configured.
#!uninstfinalize 'signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 "%1"'
#!finalize 'signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 "%1"'

Function .onInit
  StrCpy $WintunChoice "download"
  !insertmacro wails.checkArchitecture
FunctionEnd

Function DetectWintun
  StrCpy $WintunDetectedPath ""

  IfFileExists "$INSTDIR\wintun.dll" 0 +3
    StrCpy $WintunDetectedPath "$INSTDIR\wintun.dll"
    Goto done

  IfFileExists "$SYSDIR\wintun.dll" 0 +3
    StrCpy $WintunDetectedPath "$SYSDIR\wintun.dll"
    Goto done

  IfFileExists "$PROGRAMFILES64\WireGuard\wintun.dll" 0 +3
    StrCpy $WintunDetectedPath "$PROGRAMFILES64\WireGuard\wintun.dll"
    Goto done

  IfFileExists "$PROGRAMFILES64\Wintun\bin\amd64\wintun.dll" 0 +3
    StrCpy $WintunDetectedPath "$PROGRAMFILES64\Wintun\bin\amd64\wintun.dll"
    Goto done

  IfFileExists "$PROGRAMFILES64\Wintun\bin\arm64\wintun.dll" 0 +3
    StrCpy $WintunDetectedPath "$PROGRAMFILES64\Wintun\bin\arm64\wintun.dll"
    Goto done

  done:
FunctionEnd

Function WintunPageCreate
  !insertmacro MUI_HEADER_TEXT "Virtual Network Component" "Check and prepare Windows Wintun"

  Call DetectWintun

  nsDialogs::Create 1018
  Pop $0
  ${If} $0 == error
    Abort
  ${EndIf}

  ${NSD_CreateLabel} 0 0 100% 24u "NexTunnel needs the official wintun.dll for system-level virtual networking. The installer can download the official WireGuard Wintun package or you can install it manually."
  Pop $0

  ${If} $WintunDetectedPath != ""
    ${NSD_CreateLabel} 0 32u 100% 18u "Wintun detected: $WintunDetectedPath"
    Pop $0
    StrCpy $WintunChoice "present"
  ${Else}
    ${NSD_CreateLabel} 0 32u 100% 18u "wintun.dll was not found in the application or system paths."
    Pop $0
  ${EndIf}

  ${NSD_CreateRadioButton} 0 58u 100% 12u "Download and install official Wintun (recommended)"
  Pop $WintunDownloadRadio
  ${NSD_CreateRadioButton} 0 78u 100% 12u "I will install wintun.dll manually"
  Pop $WintunManualRadio
  ${NSD_CreateRadioButton} 0 98u 100% 12u "Skip for now and use P2P/Relay only"
  Pop $WintunSkipRadio

  ${If} $WintunDetectedPath != ""
    ${NSD_SetState} $WintunSkipRadio ${BST_CHECKED}
    EnableWindow $WintunDownloadRadio 0
    EnableWindow $WintunManualRadio 0
    EnableWindow $WintunSkipRadio 0
  ${Else}
    ${NSD_SetState} $WintunDownloadRadio ${BST_CHECKED}
  ${EndIf}

  ${NSD_CreateLabel} 0 122u 100% 28u "Note: creating a Wintun adapter also requires administrator privileges. The Network page will show runtime permission and TUN status after launch."
  Pop $0

  nsDialogs::Show
FunctionEnd

Function WintunPageLeave
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

Function DownloadAndInstallWintun
  InitPluginsDir

  DetailPrint "Wintun: downloading official package"
  nsExec::ExecToLog 'powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference = ''Stop''; [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; Invoke-WebRequest -Uri ''${WINTUN_DOWNLOAD_URL}'' -OutFile ''$PLUGINSDIR\wintun.zip''; if (''${WINTUN_SHA256}'' -ne '''') { $$expected = ''${WINTUN_SHA256}''.ToLowerInvariant(); $$actual = (Get-FileHash -Algorithm SHA256 -Path ''$PLUGINSDIR\wintun.zip'').Hash.ToLowerInvariant(); if ($$actual -ne $$expected) { throw ''Wintun SHA256 mismatch'' } }; Expand-Archive -Path ''$PLUGINSDIR\wintun.zip'' -DestinationPath ''$PLUGINSDIR\wintun'' -Force"'
  Pop $0
  ${If} $0 != 0
    DetailPrint "Wintun: download or extraction failed with exit code $0"
    MessageBox MB_ICONEXCLAMATION "Automatic Wintun installation failed. NexTunnel installation will continue. You can later install the official wintun.dll manually and place it beside the application."
    Return
  ${EndIf}

  DetailPrint "Wintun: copying architecture-matched wintun.dll"
  ${If} ${IsNativeARM64}
    StrCpy $1 "$PLUGINSDIR\wintun\wintun\bin\arm64\wintun.dll"
  ${Else}
    StrCpy $1 "$PLUGINSDIR\wintun\wintun\bin\amd64\wintun.dll"
  ${EndIf}

  IfFileExists "$1" 0 missing
    CopyFiles /SILENT "$1" "$INSTDIR\wintun.dll"
    IfFileExists "$INSTDIR\wintun.dll" done failed

  missing:
    DetailPrint "Wintun: extracted wintun.dll not found: $1"
    MessageBox MB_ICONEXCLAMATION "The official Wintun package was downloaded, but the DLL for this architecture was not found. Please install it manually."
    Return

  failed:
    DetailPrint "Wintun: failed to copy into $INSTDIR"
    MessageBox MB_ICONEXCLAMATION "Failed to copy wintun.dll. Please check install directory permissions or copy it beside NexTunnel.exe manually."
    Return

  done:
    DetailPrint "Wintun: installed to $INSTDIR\wintun.dll"
FunctionEnd

Function InstallWintunIfRequired
  Call DetectWintun
  ${If} $WintunDetectedPath != ""
    DetailPrint "Wintun: ready ($WintunDetectedPath)"
    Return
  ${EndIf}

  ${If} $WintunChoice == "download"
    Call DownloadAndInstallWintun
  ${ElseIf} $WintunChoice == "manual"
    DetailPrint "Wintun: manual installation selected, skipping automatic download"
  ${Else}
    DetailPrint "Wintun: skipped; real TUN remains unavailable until wintun.dll is installed"
  ${EndIf}
FunctionEnd

Function InstallSummaryPageCreate
  !insertmacro MUI_HEADER_TEXT "Setup Result" "NexTunnel is installed; network component status follows"

  Call DetectWintun

  nsDialogs::Create 1018
  Pop $0
  ${If} $0 == error
    Abort
  ${EndIf}

  ${NSD_CreateLabel} 0 0 100% 24u "Application files, WebView2 bootstrapper, and shortcuts have been installed."
  Pop $0

  ${If} $WintunDetectedPath != ""
    ${NSD_CreateLabel} 0 34u 100% 28u "Wintun status: detected at $WintunDetectedPath. Creating a virtual network still requires running NexTunnel with administrator privileges."
    Pop $0
  ${Else}
    ${NSD_CreateLabel} 0 34u 100% 38u "Wintun status: not installed. NexTunnel can continue with P2P/Relay. For real system-route TUN, place the official wintun.dll beside NexTunnel.exe."
    Pop $0
  ${EndIf}

  ${NSD_CreateLabel} 0 84u 100% 28u "After launch, open the Network page to check wintun.dll, administrator privileges, and virtual network status."
  Pop $0

  nsDialogs::Show
FunctionEnd

Section
  !insertmacro wails.setShellContext

  !insertmacro wails.webview2runtime

  SetOutPath $INSTDIR

  !insertmacro wails.files
  Call InstallWintunIfRequired

  CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

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
