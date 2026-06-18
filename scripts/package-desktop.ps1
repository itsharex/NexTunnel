param(
  [string]$Version = "v0.4.1-alpha",
  [string]$Platform = "windows/amd64",
  [string]$WintunDllPath = "",
  [switch]$SkipFrontend
)

$ErrorActionPreference = "Stop"

$repositoryRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$desktopRoot = Join-Path $repositoryRoot "desktop"
$frontendRoot = Join-Path $desktopRoot "frontend"
$distRoot = Join-Path $repositoryRoot "dist"
$releaseVersion = $Version.Trim()
$supportedDesktopPlatform = "windows/amd64"
$wintunMachineAmd64 = 0x8664

if ([string]::IsNullOrWhiteSpace($releaseVersion)) {
  throw "Version 不能为空"
}
if ($releaseVersion -notmatch '^v?[0-9A-Za-z][0-9A-Za-z.\-+]*$') {
  throw "Version 包含非法字符：$releaseVersion"
}
if ($Platform -ne $supportedDesktopPlatform) {
  throw "当前桌面端发布脚本仅支持 $supportedDesktopPlatform"
}

$normalizedVersion = $releaseVersion.TrimStart("v")
$targetName = "nextunnel-$releaseVersion-windows-amd64"
$targetDirectory = Join-Path $distRoot $targetName
$binaryOutputName = "$targetName.exe"
$binarySource = Join-Path $desktopRoot "build\bin\$binaryOutputName"
$binaryTarget = Join-Path $targetDirectory $binaryOutputName
$goCacheRoot = Join-Path $repositoryRoot ".gocache-release"

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

function Assert-UnderDirectory {
  param(
    [string]$ChildPath,
    [string]$ParentPath
  )
  $resolvedParent = [System.IO.Path]::GetFullPath($ParentPath)
  $resolvedChild = [System.IO.Path]::GetFullPath($ChildPath)
  if (-not $resolvedChild.StartsWith($resolvedParent, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "目标路径不在允许目录内：$resolvedChild"
  }
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

function Invoke-WailsBuild {
  Write-Host "打包桌面端 $releaseVersion ($Platform)"
  if (-not (Test-Path $goCacheRoot)) {
    New-Item -ItemType Directory -Path $goCacheRoot | Out-Null
  }
  Push-Location $desktopRoot
  try {
    $previousGoCache = $env:GOCACHE
    $env:GOCACHE = $goCacheRoot
    wails build `
      -m `
      -s `
      -trimpath `
      -platform $Platform `
      -webview2 download `
      -o $binaryOutputName `
      -ldflags "-s -w -X main.AppVersion=$normalizedVersion"
  } finally {
    $env:GOCACHE = $previousGoCache
    Pop-Location
  }
}

function New-DesktopArchive {
  if (-not (Test-Path $binarySource)) {
    throw "未找到 Wails 构建产物：$binarySource"
  }

  Assert-UnderDirectory -ChildPath $targetDirectory -ParentPath $distRoot
  if (Test-Path $targetDirectory) {
    Remove-Item -LiteralPath $targetDirectory -Recurse -Force
  }
  New-Item -ItemType Directory -Path $targetDirectory | Out-Null

  Copy-Item -LiteralPath $binarySource -Destination $binaryTarget
  $wintunSourcePath = Get-WintunDllCandidatePath
  $wintunManifestLine = "Wintun: not bundled; set NEXTUNNEL_WINTUN_DLL or -WintunDllPath to include official wintun.dll"
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
    "WindowsResourceVersion: 0.4.1"
    "Target: $Platform"
    "Binary: $binaryOutputName"
    $wintunManifestLine
    "WebView2: download strategy"
  ) | Set-Content -Path $manifestPath -Encoding UTF8

  $archivePath = Join-Path $distRoot "$targetName.zip"
  if (Test-Path $archivePath) {
    Remove-Item -LiteralPath $archivePath -Force
  }
  Compress-Archive -Path $targetDirectory -DestinationPath $archivePath -Force

  $checksumPath = "$archivePath.sha256"
  $hash = Get-FileHash -Algorithm SHA256 -Path $archivePath
  "$($hash.Hash.ToLowerInvariant())  $(Split-Path -Leaf $archivePath)" | Set-Content -Path $checksumPath -Encoding ASCII

  Write-Host "桌面端发布包已生成：$archivePath"
  Write-Host "SHA256：$checksumPath"
}

Invoke-FrontendBuild
Invoke-WailsBuild
New-DesktopArchive
