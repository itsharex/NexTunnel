param(
  [string]$Version = "v0.5.2-alpha",
  [string]$Platform = "windows/amd64",
  [ValidateSet("none", "nsis")]
  [string]$Installer = "nsis",
  [ValidateSet("bundled", "download", "manual")]
  [string]$WintunMode = "bundled",
  [string]$WintunDllPath = "",
  [string]$WintunDownloadUrl = "https://www.wintun.net/builds/wintun-0.14.1.zip",
  [string]$WintunSha256 = "07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51",
  [switch]$SkipFrontend,
  [switch]$SkipZip
)

$ErrorActionPreference = "Stop"

$repositoryRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$desktopRoot = Join-Path $repositoryRoot "desktop"
$frontendRoot = Join-Path $desktopRoot "frontend"
$distRoot = Join-Path $repositoryRoot "dist"
$releaseVersion = $Version.Trim()
$supportedDesktopPlatform = "windows/amd64"
$wintunMachineAmd64 = 0x8664
$wintunArchiveRelativeDllPath = "wintun\bin\amd64\wintun.dll"
$installerConfigPath = Join-Path $desktopRoot "build\windows\installer\nextunnel_installer_config.local.nsh"
$installerBundledWintunPath = Join-Path $desktopRoot "build\windows\installer\resources\wintun-amd64.dll"
$goCacheRoot = Join-Path $repositoryRoot ".gocache-release"
$wintunDownloadRoot = Join-Path $repositoryRoot ".tmp\wintun-release"
$preparedWintunDllPath = ""

if ([string]::IsNullOrWhiteSpace($releaseVersion)) {
  throw "Version 不能为空"
}
if ($releaseVersion -notmatch '^v?[0-9A-Za-z][0-9A-Za-z.\-+]*$') {
  throw "Version 包含非法字符：$releaseVersion"
}
if ($Platform -ne $supportedDesktopPlatform) {
  throw "当前桌面端发布脚本仅支持 $supportedDesktopPlatform"
}
if ($Installer -eq "none" -and $SkipZip) {
  throw "Installer=none 时不能同时使用 -SkipZip，否则不会生成任何桌面发布资产"
}
if (($WintunMode -eq "bundled" -or $WintunMode -eq "download") -and [string]::IsNullOrWhiteSpace($WintunSha256)) {
  throw "WintunMode=$WintunMode 时必须提供 WintunSha256，避免发布不可校验的 DLL 来源"
}

$normalizedVersion = $releaseVersion.TrimStart("v")
$windowsResourceVersion = "$normalizedVersion.0"
$targetName = "nextunnel-$releaseVersion-windows-amd64"
$targetDirectory = Join-Path $distRoot $targetName
$binaryOutputName = "$targetName.exe"
$binarySource = Join-Path $desktopRoot "build\bin\$binaryOutputName"
$binaryTarget = Join-Path $targetDirectory $binaryOutputName
$installerSource = Join-Path $desktopRoot "build\bin\NexTunnel-amd64-installer.exe"
$installerTarget = Join-Path $distRoot "$targetName-installer.exe"

function New-DirectoryIfMissing {
  param([string]$Path)
  if (-not (Test-Path $Path)) {
    New-Item -ItemType Directory -Path $Path | Out-Null
  }
}

function Assert-UnderDirectory {
  param(
    [string]$ChildPath,
    [string]$ParentPath
  )
  $resolvedParent = [System.IO.Path]::GetFullPath($ParentPath).TrimEnd(
    [System.IO.Path]::DirectorySeparatorChar,
    [System.IO.Path]::AltDirectorySeparatorChar
  )
  $resolvedChild = [System.IO.Path]::GetFullPath($ChildPath)
  $repoPathPrefix = $resolvedParent + [System.IO.Path]::DirectorySeparatorChar
  if ($resolvedChild -ne $resolvedParent -and -not $resolvedChild.StartsWith($repoPathPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "目标路径不在允许目录内：$resolvedChild"
  }
}

function New-Sha256File {
  param([string]$Path)
  if (-not (Test-Path $Path)) {
    throw "无法生成 SHA256，文件不存在：$Path"
  }
  $hash = Get-FileHash -Algorithm SHA256 -Path $Path
  $checksumPath = "$Path.sha256"
  "$($hash.Hash.ToLowerInvariant())  $(Split-Path -Leaf $Path)" | Set-Content -Path $checksumPath -Encoding ASCII
  return $checksumPath
}

function Assert-Sha256File {
  param(
    [string]$Path,
    [string]$ExpectedSha256
  )
  if ([string]::IsNullOrWhiteSpace($ExpectedSha256)) {
    throw "缺少 SHA256，无法校验文件：$Path"
  }
  $actualHash = (Get-FileHash -Algorithm SHA256 -Path $Path).Hash.ToLowerInvariant()
  $expectedHash = $ExpectedSha256.Trim().ToLowerInvariant()
  if ($actualHash -ne $expectedHash) {
    throw "SHA256 校验失败：$Path expected=$expectedHash actual=$actualHash"
  }
}

function Get-WintunDllCandidatePath {
  $candidatePaths = [System.Collections.Generic.List[string]]::new()
  if (-not [string]::IsNullOrWhiteSpace($WintunDllPath)) {
    $candidatePaths.Add($WintunDllPath) | Out-Null
  }
  if (-not [string]::IsNullOrWhiteSpace($env:NEXTUNNEL_WINTUN_DLL)) {
    $candidatePaths.Add($env:NEXTUNNEL_WINTUN_DLL) | Out-Null
  }
  $candidatePaths.Add((Join-Path $desktopRoot "wintun.dll")) | Out-Null
  $candidatePaths.Add((Join-Path $repositoryRoot "wintun.dll")) | Out-Null
  if (-not [string]::IsNullOrWhiteSpace($env:ProgramFiles)) {
    $candidatePaths.Add((Join-Path $env:ProgramFiles "WireGuard\wintun.dll")) | Out-Null
    $candidatePaths.Add((Join-Path $env:ProgramFiles "Wintun\bin\amd64\wintun.dll")) | Out-Null
  }
  $programFilesX86 = ${env:ProgramFiles(x86)}
  if (-not [string]::IsNullOrWhiteSpace($programFilesX86)) {
    $candidatePaths.Add((Join-Path $programFilesX86 "Wintun\bin\amd64\wintun.dll")) | Out-Null
  }
  foreach ($candidatePath in $candidatePaths) {
    if (Test-Path $candidatePath) {
      return (Resolve-Path $candidatePath).Path
    }
  }
  return ""
}

function Get-PreparedWintunDllPath {
  if (-not [string]::IsNullOrWhiteSpace($preparedWintunDllPath) -and (Test-Path $preparedWintunDllPath)) {
    return $preparedWintunDllPath
  }
  return Get-WintunDllCandidatePath
}

function Get-PEDllMachine {
  param([string]$Path)
  $stream = [System.IO.File]::OpenRead($Path)
  try {
    if ($stream.Length -lt 0x40) {
      throw "DLL is too small: $Path"
    }
    $reader = [System.IO.BinaryReader]::new($stream)
    try {
      $stream.Seek(0x3C, [System.IO.SeekOrigin]::Begin) | Out-Null
      $peOffset = $reader.ReadInt32()
      if ($peOffset -lt 0 -or ($peOffset + 6) -gt $stream.Length) {
        throw "invalid PE header offset: $Path"
      }
      $stream.Seek($peOffset, [System.IO.SeekOrigin]::Begin) | Out-Null
      $signature = $reader.ReadUInt32()
      if ($signature -ne 0x00004550) {
        throw "invalid PE signature: $Path"
      }
      return $reader.ReadUInt16()
    } finally {
      $reader.Close()
    }
  } finally {
    $stream.Close()
  }
}

function Get-ExpectedWintunDllMachine {
  # PE Machine 值用于阻止错误架构的 wintun.dll 被打进发布包。
  if ($Platform -eq "windows/amd64") {
    return $wintunMachineAmd64
  }
  throw "当前平台不支持 Wintun 架构校验：$Platform"
}

function Assert-WintunDllArchitecture {
  param([string]$Path)
  $expectedMachine = Get-ExpectedWintunDllMachine
  $actualMachine = Get-PEDllMachine -Path $Path
  if ($actualMachine -ne $expectedMachine) {
    throw ("wintun.dll 架构不匹配：path={0} expected=0x{1:X4} actual=0x{2:X4}" -f $Path, $expectedMachine, $actualMachine)
  }
}

function Save-OfficialWintunDll {
  if ($WintunMode -ne "bundled") {
    return
  }

  $existingWintunPath = Get-WintunDllCandidatePath
  if (-not [string]::IsNullOrWhiteSpace($existingWintunPath)) {
    Assert-WintunDllArchitecture -Path $existingWintunPath
    New-DirectoryIfMissing -Path (Split-Path -Parent $installerBundledWintunPath)
    Copy-Item -LiteralPath $existingWintunPath -Destination $installerBundledWintunPath -Force
    $script:preparedWintunDllPath = $installerBundledWintunPath
    Write-Host "已使用本地官方 Wintun DLL：$existingWintunPath"
    return
  }

  if ($WintunDownloadUrl -notmatch '^https://') {
    throw "WintunDownloadUrl 必须使用 HTTPS：$WintunDownloadUrl"
  }
  New-DirectoryIfMissing -Path $wintunDownloadRoot
  New-DirectoryIfMissing -Path (Split-Path -Parent $installerBundledWintunPath)

  $wintunZipPath = Join-Path $wintunDownloadRoot "wintun.zip"
  $wintunExtractRoot = Join-Path $wintunDownloadRoot "extract"
  if (Test-Path $wintunExtractRoot) {
    Remove-Item -LiteralPath $wintunExtractRoot -Recurse -Force
  }

  Write-Host "下载官方 Wintun 包：$WintunDownloadUrl"
  Invoke-WebRequest -Uri $WintunDownloadUrl -OutFile $wintunZipPath
  Assert-Sha256File -Path $wintunZipPath -ExpectedSha256 $WintunSha256

  Expand-Archive -Path $wintunZipPath -DestinationPath $wintunExtractRoot -Force
  $extractedDllPath = Join-Path $wintunExtractRoot $wintunArchiveRelativeDllPath
  if (-not (Test-Path $extractedDllPath)) {
    throw "官方 Wintun 包中未找到预期 DLL：$wintunArchiveRelativeDllPath"
  }
  Assert-WintunDllArchitecture -Path $extractedDllPath
  Copy-Item -LiteralPath $extractedDllPath -Destination $installerBundledWintunPath -Force
  $script:preparedWintunDllPath = $installerBundledWintunPath
  Write-Host "已准备内置 Wintun DLL：$installerBundledWintunPath"
}

function Invoke-FrontendBuild {
  if ($SkipFrontend) {
    Write-Host "跳过桌面端前端构建"
    return
  }

  Write-Host "构建桌面端前端"
  $npmCommand = Get-Command npm -ErrorAction SilentlyContinue
  if ($npmCommand) {
    Push-Location $frontendRoot
    try {
      npm run build
    } finally {
      Pop-Location
    }
    return
  }

  $bundledNode = Join-Path $env:USERPROFILE ".cache\codex-runtimes\codex-primary-runtime\dependencies\node\bin\node.exe"
  if (-not (Test-Path $bundledNode)) {
    throw "未找到 npm，也未找到 bundled Node：$bundledNode"
  }

  Push-Location $frontendRoot
  try {
    & $bundledNode "node_modules\vue-tsc\bin\vue-tsc.js" "--noEmit"
    & $bundledNode "node_modules\vite\bin\vite.js" "build"
  } finally {
    Pop-Location
  }
}

function Set-InstallerConfig {
  if ($Installer -ne "nsis") {
    return
  }
  if ($WintunDownloadUrl -notmatch '^https://') {
    throw "WintunDownloadUrl 必须使用 HTTPS：$WintunDownloadUrl"
  }
  if (-not [string]::IsNullOrWhiteSpace($WintunSha256) -and $WintunSha256 -notmatch '^[0-9a-fA-F]{64}$') {
    throw "WintunSha256 必须是 64 位十六进制 SHA256"
  }
  $installerConfigDirectory = Split-Path -Parent $installerConfigPath
  New-DirectoryIfMissing -Path $installerConfigDirectory

  # 本地 NSIS 配置由发布脚本生成，避免把临时下载地址或校验值写进模板。
  @(
    "!define WINTUN_DOWNLOAD_URL `"$WintunDownloadUrl`""
    "!define WINTUN_SHA256 `"$($WintunSha256.Trim())`""
    "!define WINTUN_MODE `"$WintunMode`""
  ) | Set-Content -Path $installerConfigPath -Encoding UTF8
  if ($WintunMode -eq "bundled" -and (Test-Path $installerBundledWintunPath)) {
    Add-Content -Path $installerConfigPath -Encoding UTF8 -Value "!define WINTUN_BUNDLED_DLL `"resources\wintun-amd64.dll`""
  }
}

function Add-NSISToPath {
  if ($Installer -ne "nsis") {
    return
  }

  $existingMakensis = Get-Command makensis -ErrorAction SilentlyContinue
  if ($existingMakensis) {
    Write-Host "NSIS 已可用：$($existingMakensis.Source)"
    return
  }

  $candidatePaths = [System.Collections.Generic.List[string]]::new()
  if (-not [string]::IsNullOrWhiteSpace($env:ProgramFiles)) {
    $candidatePaths.Add((Join-Path $env:ProgramFiles "NSIS\makensis.exe")) | Out-Null
  }
  $programFilesX86 = ${env:ProgramFiles(x86)}
  if (-not [string]::IsNullOrWhiteSpace($programFilesX86)) {
    $candidatePaths.Add((Join-Path $programFilesX86 "NSIS\makensis.exe")) | Out-Null
  }
  if (-not [string]::IsNullOrWhiteSpace($env:ChocolateyInstall)) {
    $candidatePaths.Add((Join-Path $env:ChocolateyInstall "bin\makensis.exe")) | Out-Null
    $candidatePaths.Add((Join-Path $env:ChocolateyInstall "lib\nsis\tools\makensis.exe")) | Out-Null
  }

  foreach ($candidatePath in $candidatePaths) {
    if (Test-Path $candidatePath) {
      $nsisDirectory = Split-Path -Parent (Resolve-Path $candidatePath).Path
      $env:PATH = "$nsisDirectory;$env:PATH"
      Write-Host "已加入 NSIS 路径：$nsisDirectory"
      return
    }
  }

  throw "未找到 makensis.exe。请先安装 NSIS，或把 makensis.exe 所在目录加入 PATH。"
}

function Invoke-WailsBuild {
  Write-Host "打包桌面端 $releaseVersion ($Platform)"
  New-DirectoryIfMissing -Path $goCacheRoot
  Push-Location $desktopRoot
  try {
    $previousGoCache = $env:GOCACHE
    $env:GOCACHE = $goCacheRoot
    $buildArguments = @(
      "build",
      "-m",
      "-s",
      "-trimpath",
      "-platform", $Platform,
      "-webview2", "download",
      "-o", $binaryOutputName,
      "-ldflags", "-s -w -X main.AppVersion=$normalizedVersion"
    )
    if ($Installer -eq "nsis") {
      $buildArguments += "-nsis"
    }
    & wails @buildArguments
    if ($LASTEXITCODE -ne 0) {
      throw "Wails 构建失败，退出码：$LASTEXITCODE"
    }
  } finally {
    $env:GOCACHE = $previousGoCache
    Pop-Location
  }
}

function New-PortableArchive {
  if ($SkipZip) {
    Write-Host "跳过 Windows 便携 zip"
    return
  }
  if (-not (Test-Path $binarySource)) {
    throw "未找到 Wails 构建产物：$binarySource"
  }

  Assert-UnderDirectory -ChildPath $targetDirectory -ParentPath $distRoot
  if (Test-Path $targetDirectory) {
    Remove-Item -LiteralPath $targetDirectory -Recurse -Force
  }
  New-Item -ItemType Directory -Path $targetDirectory | Out-Null

  Copy-Item -LiteralPath $binarySource -Destination $binaryTarget
  $wintunSourcePath = Get-PreparedWintunDllPath
  $wintunManifestLine = "Wintun: missing; zip users must place official wintun.dll beside NexTunnel.exe or use NEXTUNNEL_WINTUN_DLL"
  if (-not [string]::IsNullOrWhiteSpace($wintunSourcePath)) {
    Assert-WintunDllArchitecture -Path $wintunSourcePath
    Copy-Item -LiteralPath $wintunSourcePath -Destination (Join-Path $targetDirectory "wintun.dll") -Force
    $wintunManifestLine = "Wintun: bundled; architecture verified for $Platform"
  }
  $manifestPath = Join-Path $targetDirectory "MANIFEST.txt"
  @(
    "NexTunnel desktop package"
    "Version: $releaseVersion"
    "ApplicationVersion: $normalizedVersion"
    "WindowsResourceVersion: $windowsResourceVersion"
    "Target: $Platform"
    "Installer: zip"
    "Binary: $binaryOutputName"
    $wintunManifestLine
    "Signing: unsigned-alpha"
    "PrunedResources: true"
    "WebView2: download strategy"
  ) | Set-Content -Path $manifestPath -Encoding UTF8

  $archivePath = Join-Path $distRoot "$targetName.zip"
  if (Test-Path $archivePath) {
    Remove-Item -LiteralPath $archivePath -Force
  }
  Compress-Archive -Path (Join-Path $targetDirectory "*") -DestinationPath $archivePath -Force
  $checksumPath = New-Sha256File -Path $archivePath

  Write-Host "桌面端便携包已生成：$archivePath"
  Write-Host "SHA256：$checksumPath"
}

function Rename-NSISInstaller {
  if ($Installer -ne "nsis") {
    return
  }
  if (-not (Test-Path $installerSource)) {
    throw "未找到 NSIS 安装器产物：$installerSource"
  }
  if (Test-Path $installerTarget) {
    Remove-Item -LiteralPath $installerTarget -Force
  }
  Move-Item -LiteralPath $installerSource -Destination $installerTarget -Force
  $checksumPath = New-Sha256File -Path $installerTarget

  $manifestPath = Join-Path $distRoot "$targetName-installer.MANIFEST.txt"
  $wintunMode = switch ($WintunMode) {
    "bundled" { "bundled; fallback download sha256 pinned" }
    "download" { "download-on-install; sha256 pinned" }
    default { "manual" }
  }
  @(
    "NexTunnel desktop installer"
    "Version: $releaseVersion"
    "ApplicationVersion: $normalizedVersion"
    "WindowsResourceVersion: $windowsResourceVersion"
    "Target: $Platform"
    "Installer: nsis"
    "Binary: $(Split-Path -Leaf $installerTarget)"
    "Wintun: $wintunMode"
    "WintunDownloadUrl: $WintunDownloadUrl"
    "Signing: unsigned-alpha"
    "PrunedResources: true"
    "WebView2: download strategy"
  ) | Set-Content -Path $manifestPath -Encoding UTF8

  Write-Host "桌面端 NSIS 安装包已生成：$installerTarget"
  Write-Host "SHA256：$checksumPath"
}

function Remove-InstallerConfig {
  if (Test-Path $installerConfigPath) {
    Remove-Item -LiteralPath $installerConfigPath -Force
  }
}

New-DirectoryIfMissing -Path $distRoot
Save-OfficialWintunDll
Set-InstallerConfig
try {
  Invoke-FrontendBuild
  Add-NSISToPath
  Invoke-WailsBuild
  New-PortableArchive
  Rename-NSISInstaller
} finally {
  Remove-InstallerConfig
}
