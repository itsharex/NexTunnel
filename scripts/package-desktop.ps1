param(
  [string]$Version = "v0.1.1-alpha",
  [string]$Platform = "windows/amd64",
  [switch]$SkipFrontend
)

$ErrorActionPreference = "Stop"

$repositoryRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$desktopRoot = Join-Path $repositoryRoot "desktop"
$frontendRoot = Join-Path $desktopRoot "frontend"
$distRoot = Join-Path $repositoryRoot "dist"
$releaseVersion = $Version.Trim()

if ([string]::IsNullOrWhiteSpace($releaseVersion)) {
  throw "Version 不能为空"
}
if ($releaseVersion -notmatch '^v?[0-9A-Za-z][0-9A-Za-z.\-+]*$') {
  throw "Version 包含非法字符：$releaseVersion"
}
if ($Platform -ne "windows/amd64") {
  throw "当前桌面端发布脚本仅支持 windows/amd64"
}

$normalizedVersion = $releaseVersion.TrimStart("v")
$targetName = "nextunnel-$releaseVersion-windows-amd64"
$targetDirectory = Join-Path $distRoot $targetName
$binaryOutputName = "$targetName.exe"
$binarySource = Join-Path $desktopRoot "build\bin\$binaryOutputName"
$binaryTarget = Join-Path $targetDirectory $binaryOutputName
$goCacheRoot = Join-Path $repositoryRoot ".gocache-release"

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
  $manifestPath = Join-Path $targetDirectory "MANIFEST.txt"
  @(
    "NexTunnel desktop package"
    "Version: $releaseVersion"
    "ApplicationVersion: $normalizedVersion"
    "WindowsResourceVersion: 0.1.1"
    "Target: $Platform"
    "Binary: $binaryOutputName"
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
