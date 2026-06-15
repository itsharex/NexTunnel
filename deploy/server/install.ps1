param(
    [ValidateSet('install', 'up', 'down', 'restart', 'status', 'logs', 'health', 'update', 'uninstall', 'config')]
    [string]$Action = 'install',
    [string]$PackageUrl = '',
    [string]$Version = '',
    [string]$ReleaseBaseUrl = '',
    [string]$GithubProxy = '',
    [string]$Repository = '',
    [string]$PackageSha256 = '',
    [string]$Architecture = '',
    [string]$InstallDir = '',
    [string]$ConfigDir = '',
    [string]$DataDir = '',
    [string]$PublicHost = '',
    [string]$RelayToken = '',
    [string]$ControlToken = '',
    [int]$RelayPort = 7000,
    [int]$RelayQuicPort = 7443,
    [int]$ControlPlanePort = 9090,
    [int]$DashboardPort = 8080,
    [string]$DashboardSecret = '',
    [string]$DashboardAdmin = 'admin',
    [string]$DashboardPassword = '',
    [string]$DashboardOrigins = '',
    [int]$NatPort = 3478,
    [switch]$DashboardDisabled,
    [switch]$NonInteractive,
    [switch]$Force,
    [switch]$Purge
)

$ErrorActionPreference = 'Stop'

function Write-Step {
    param([string]$Message)
    Write-Host "[NexTunnel] $Message" -ForegroundColor Cyan
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[NexTunnel] $Message" -ForegroundColor Yellow
}

function Assert-Command {
    param([string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Command not found: $Name"
    }
}

function Read-DotEnvFile {
    param([string]$Path)
    $values = @{}
    if (-not (Test-Path -LiteralPath $Path)) {
        return $values
    }
    Get-Content -LiteralPath $Path | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq '' -or $line.StartsWith('#') -or -not $line.Contains('=')) {
            return
        }
        $parts = $line.Split('=', 2)
        $values[$parts[0]] = $parts[1]
    }
    return $values
}

function Get-ConfigValue {
    param(
        [hashtable]$LocalEnv,
        [string]$Name,
        [string]$Fallback
    )
    $environmentValue = [Environment]::GetEnvironmentVariable($Name)
    if (-not [string]::IsNullOrWhiteSpace($environmentValue)) {
        return $environmentValue
    }
    if ($LocalEnv.ContainsKey($Name) -and -not [string]::IsNullOrWhiteSpace($LocalEnv[$Name])) {
        return $LocalEnv[$Name]
    }
    return $Fallback
}

function New-RandomSecret {
    param([int]$Bytes = 32)
    $buffer = New-Object byte[] $Bytes
    $generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $generator.GetBytes($buffer)
    } finally {
        $generator.Dispose()
    }
    return [Convert]::ToBase64String($buffer).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

function Read-Setting {
    param(
        [string]$Prompt,
        [string]$DefaultValue,
        [switch]$Secret
    )
    if ($NonInteractive) {
        return $DefaultValue
    }
    $label = if ($DefaultValue) { $Prompt + ' [' + $DefaultValue + ']' } else { $Prompt }
    if ($Secret) {
        $secure = Read-Host $label -AsSecureString
        if ($secure.Length -eq 0) {
            return $DefaultValue
        }
        $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
        try {
            return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer)
        } finally {
            [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer)
        }
    }
    $value = Read-Host $label
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $DefaultValue
    }
    return $value.Trim()
}

function Assert-Port {
    param([string]$Name, [int]$Value)
    if ($Value -lt 1 -or $Value -gt 65535) {
        throw "$Name must be between 1 and 65535. Current value: $Value"
    }
}

function Convert-ToEnvLine {
    param([string]$Name, [string]$Value)
    if ($Value.Contains("`n") -or $Value.Contains("`r")) {
        throw "$Name cannot contain newline characters."
    }
    return $Name + '=' + $Value
}

function Resolve-Architecture {
    param([string]$ConfiguredArchitecture)
    $rawArchitecture = if ($ConfiguredArchitecture) { $ConfiguredArchitecture } else { [Environment]::GetEnvironmentVariable('PROCESSOR_ARCHITECTURE') }
    switch -Regex ($rawArchitecture.ToLowerInvariant()) {
        '^(amd64|x86_64)$' { return 'amd64' }
        '^(arm64|aarch64)$' { return 'arm64' }
        default { throw "Unsupported architecture: $rawArchitecture. Supported: amd64, arm64." }
    }
}

function Resolve-ReleaseBaseUrl {
    if ($script:ReleaseBaseUrlValue) {
        return $script:ReleaseBaseUrlValue.TrimEnd('/')
    }
    $githubReleaseUrl = ''
    if ($script:VersionValue -eq 'latest') {
        $githubReleaseUrl = "https://github.com/$script:RepositoryValue/releases/latest/download"
    } else {
        $githubReleaseUrl = "https://github.com/$script:RepositoryValue/releases/download/$script:VersionValue"
    }
    if ([string]::IsNullOrWhiteSpace($script:GithubProxyValue)) {
        return $githubReleaseUrl
    }
    # 仅在用户显式配置时改写 GitHub URL，避免默认使用不可信第三方代理。
    $proxyValue = $script:GithubProxyValue.Trim()
    if ($proxyValue.Contains('{url}')) {
        return $proxyValue.Replace('{url}', $githubReleaseUrl)
    }
    return $proxyValue.TrimEnd('/') + '/' + $githubReleaseUrl
}

function Resolve-PackageUrl {
    if ($script:PackageUrlValue) {
        return $script:PackageUrlValue
    }
    $resolvedArchitecture = Resolve-Architecture $script:ArchitectureValue
    return "$(Resolve-ReleaseBaseUrl)/nextunnel-server-windows-$resolvedArchitecture.zip"
}

function Assert-FileChecksum {
    param([string]$Path)
    if ([string]::IsNullOrWhiteSpace($script:PackageSha256Value)) {
        return
    }
    $actualChecksum = (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualChecksum -ne $script:PackageSha256Value.ToLowerInvariant()) {
        throw "Package SHA256 mismatch. Expected $script:PackageSha256Value, actual $actualChecksum."
    }
}

function Copy-PackageSource {
    param([string]$Source, [string]$Target)
    if ($Source.StartsWith('http://') -or $Source.StartsWith('https://')) {
        Invoke-WebRequest -Uri $Source -OutFile $Target -UseBasicParsing
        return
    }
    if ($Source.StartsWith('file://')) {
        $fileUri = [Uri]$Source
        Copy-Item -LiteralPath $fileUri.LocalPath -Destination $Target -Force
        return
    }
    if (Test-Path -LiteralPath $Source) {
        Copy-Item -LiteralPath $Source -Destination $Target -Force
        return
    }
    throw "Package source not found: $Source"
}

function Expand-ServerPackage {
    param([string]$ArchivePath, [string]$Destination)
    New-Item -ItemType Directory -Force -Path $Destination | Out-Null
    if ($ArchivePath.EndsWith('.zip')) {
        Expand-Archive -LiteralPath $ArchivePath -DestinationPath $Destination -Force
        return
    }
    if ($ArchivePath.EndsWith('.tar.gz') -or $ArchivePath.EndsWith('.tgz')) {
        Assert-Command tar
        tar -xzf $ArchivePath -C $Destination
        return
    }
    throw "Unsupported package format: $ArchivePath"
}

function Find-Binary {
    param([string]$RootPath, [string]$BinaryName)
    $expectedNames = @($BinaryName, "$BinaryName.exe")
    $match = Get-ChildItem -LiteralPath $RootPath -Recurse -File |
        Where-Object { $expectedNames -contains $_.Name } |
        Select-Object -First 1
    if (-not $match) {
        throw "Binary not found in package: $BinaryName"
    }
    return $match.FullName
}

function Find-OptionalBinary {
    param([string]$RootPath, [string]$BinaryName)
    $expectedNames = @($BinaryName, "$BinaryName.exe")
    $match = Get-ChildItem -LiteralPath $RootPath -Recurse -File |
        Where-Object { $expectedNames -contains $_.Name } |
        Select-Object -First 1
    if ($match) {
        return $match.FullName
    }
    return ''
}

function Find-DashboardWebDir {
    param([string]$RootPath)
    $indexFile = Get-ChildItem -LiteralPath $RootPath -Recurse -File -Filter 'index.html' |
        Where-Object { $_.FullName -match '[\\/]web[\\/]dashboard[\\/]index\.html$' } |
        Select-Object -First 1
    if ($indexFile) {
        return $indexFile.Directory.FullName
    }
    return ''
}

function Find-DeployScriptDir {
    param([string]$RootPath)
    $scriptFile = Get-ChildItem -LiteralPath $RootPath -Recurse -File -Filter 'install.ps1' |
        Where-Object { $_.FullName -match '[\\/]deploy[\\/]server[\\/]install\.ps1$' } |
        Select-Object -First 1
    if ($scriptFile) {
        return $scriptFile.Directory.FullName
    }
    return ''
}

function Test-Truthy {
    param([string]$Value)
    if ([string]::IsNullOrWhiteSpace($Value)) {
        return $false
    }
    return -not (@('false', '0', 'no', 'off') -contains $Value.ToLowerInvariant())
}

function Get-BinaryPath {
    param([string]$BinaryName)
    $windowsPath = Join-Path $script:BinDir "$BinaryName.exe"
    if (Test-Path -LiteralPath $windowsPath) {
        return $windowsPath
    }
    return (Join-Path $script:BinDir $BinaryName)
}

function Install-ReleasePackage {
    $resolvedPackageUrl = Resolve-PackageUrl
    $temporaryRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nextunnel-" + [Guid]::NewGuid().ToString('N'))
    $extractPath = Join-Path $temporaryRoot 'extract'
    $extension = if ($resolvedPackageUrl.EndsWith('.tar.gz') -or $resolvedPackageUrl.EndsWith('.tgz')) { '.tar.gz' } else { '.zip' }
    $archivePath = Join-Path $temporaryRoot ("nextunnel-server" + $extension)

    try {
        New-Item -ItemType Directory -Force -Path $temporaryRoot, $extractPath | Out-Null
        Write-Step "Downloading server package: $resolvedPackageUrl"
        Copy-PackageSource -Source $resolvedPackageUrl -Target $archivePath
        Assert-FileChecksum -Path $archivePath

        Write-Step 'Extracting and validating server binaries'
        Expand-ServerPackage -ArchivePath $archivePath -Destination $extractPath
        $relayBinary = Find-Binary -RootPath $extractPath -BinaryName 'relay-server'
        $controlPlaneBinary = Find-Binary -RootPath $extractPath -BinaryName 'control-plane'
        $natDetectorBinary = Find-Binary -RootPath $extractPath -BinaryName 'nat-detector'
        $dashboardBinary = Find-OptionalBinary -RootPath $extractPath -BinaryName 'dashboard'
        $dashboardWebDir = Find-DashboardWebDir -RootPath $extractPath
        $deployScriptDir = Find-DeployScriptDir -RootPath $extractPath

        New-Item -ItemType Directory -Force -Path $script:BinDir, $script:WebDir, $script:DeployDir | Out-Null
        Copy-Item -LiteralPath $relayBinary -Destination (Join-Path $script:BinDir ([IO.Path]::GetFileName($relayBinary))) -Force
        Copy-Item -LiteralPath $controlPlaneBinary -Destination (Join-Path $script:BinDir ([IO.Path]::GetFileName($controlPlaneBinary))) -Force
        Copy-Item -LiteralPath $natDetectorBinary -Destination (Join-Path $script:BinDir ([IO.Path]::GetFileName($natDetectorBinary))) -Force
        if ($dashboardBinary) {
            Copy-Item -LiteralPath $dashboardBinary -Destination (Join-Path $script:BinDir ([IO.Path]::GetFileName($dashboardBinary))) -Force
        } else {
            Write-Warn 'Release package does not include dashboard binary. Core services only.'
        }
        if ($dashboardWebDir) {
            Get-ChildItem -LiteralPath $script:WebDir -Force -ErrorAction SilentlyContinue |
                Remove-Item -Recurse -Force
            Copy-Item -Path (Join-Path $dashboardWebDir '*') -Destination $script:WebDir -Recurse -Force
        } else {
            Write-Warn 'Release package does not include Dashboard web assets.'
        }
        if ($deployScriptDir) {
            Copy-Item -Path (Join-Path $deployScriptDir '*') -Destination $script:DeployDir -Recurse -Force
        } else {
            Write-Warn 'Release package does not include deploy scripts.'
        }
        Write-Step "Installed server binaries: $script:BinDir"
    } finally {
        if (Test-Path -LiteralPath $temporaryRoot) {
            Remove-Item -LiteralPath $temporaryRoot -Recurse -Force
        }
    }
}

function Write-EnvFile {
    if ((Test-Path -LiteralPath $script:EnvPath) -and -not $Force) {
        Write-Warn "Config already exists. Keep current config: $script:EnvPath. Use -Force to regenerate."
        return
    }

    $resolvedPublicHost = if ($PublicHost) { $PublicHost } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_PUBLIC_HOST' '127.0.0.1' }
    $resolvedRelayToken = if ($RelayToken) { $RelayToken } else { Get-ConfigValue $script:LocalEnv 'RELAY_AUTH_TOKEN' (New-RandomSecret) }
    $resolvedControlToken = if ($ControlToken) { $ControlToken } else { Get-ConfigValue $script:LocalEnv 'CONTROL_PLANE_API_TOKEN' (New-RandomSecret) }
    $resolvedRelayPort = [int](Get-ConfigValue $script:LocalEnv 'RELAY_CONTROL_PORT' "$RelayPort")
    $resolvedRelayQuicPort = [int](Get-ConfigValue $script:LocalEnv 'RELAY_QUIC_PORT' "$RelayQuicPort")
    $resolvedControlPlanePort = [int](Get-ConfigValue $script:LocalEnv 'CONTROL_PLANE_PORT' "$ControlPlanePort")
    $resolvedDashboardEnabled = if ($DashboardDisabled) { 'false' } else { Get-ConfigValue $script:LocalEnv 'DASHBOARD_ENABLED' 'true' }
    $resolvedDashboardPort = [int](Get-ConfigValue $script:LocalEnv 'DASHBOARD_PORT' "$DashboardPort")
    $resolvedDashboardSecret = if ($DashboardSecret) { $DashboardSecret } else { Get-ConfigValue $script:LocalEnv 'DASHBOARD_SECRET_KEY' (New-RandomSecret) }
    $resolvedDashboardAdmin = if ($DashboardAdmin) { $DashboardAdmin } else { Get-ConfigValue $script:LocalEnv 'DASHBOARD_ADMIN_USER' 'admin' }
    $resolvedDashboardPassword = if ($DashboardPassword) { $DashboardPassword } else { Get-ConfigValue $script:LocalEnv 'DASHBOARD_ADMIN_PASSWORD' (New-RandomSecret) }
    $resolvedDashboardOrigins = if ($DashboardOrigins) { $DashboardOrigins } else { Get-ConfigValue $script:LocalEnv 'DASHBOARD_ALLOWED_ORIGINS' 'http://127.0.0.1:8080,http://localhost:8080' }
    $resolvedNatPort = [int](Get-ConfigValue $script:LocalEnv 'NAT_PORT' "$NatPort")

    $resolvedPublicHost = Read-Setting -Prompt 'Public IP or domain' -DefaultValue $resolvedPublicHost
    $resolvedRelayPort = [int](Read-Setting -Prompt 'Relay TCP port' -DefaultValue "$resolvedRelayPort")
    $resolvedRelayQuicPort = [int](Read-Setting -Prompt 'Relay QUIC UDP port' -DefaultValue "$resolvedRelayQuicPort")
    $resolvedControlPlanePort = [int](Read-Setting -Prompt 'Control Plane HTTP port' -DefaultValue "$resolvedControlPlanePort")
    if (Test-Truthy $resolvedDashboardEnabled) {
        $resolvedDashboardPort = [int](Read-Setting -Prompt 'Dashboard HTTP port' -DefaultValue "$resolvedDashboardPort")
    }
    $resolvedNatPort = [int](Read-Setting -Prompt 'NAT Detector UDP port' -DefaultValue "$resolvedNatPort")
    $resolvedRelayToken = Read-Setting -Prompt 'Relay auth token' -DefaultValue $resolvedRelayToken -Secret
    $resolvedControlToken = Read-Setting -Prompt 'Control Plane Bearer Token' -DefaultValue $resolvedControlToken -Secret
    if (Test-Truthy $resolvedDashboardEnabled) {
        $resolvedDashboardPassword = Read-Setting -Prompt 'Dashboard admin password' -DefaultValue $resolvedDashboardPassword -Secret
    }

    if ([string]::IsNullOrWhiteSpace($resolvedRelayToken) -or [string]::IsNullOrWhiteSpace($resolvedControlToken)) {
        throw 'RelayToken and ControlToken cannot be empty.'
    }
    if ((Test-Truthy $resolvedDashboardEnabled) -and ([string]::IsNullOrWhiteSpace($resolvedDashboardSecret) -or [string]::IsNullOrWhiteSpace($resolvedDashboardPassword))) {
        throw 'DashboardSecret and DashboardAdminPassword cannot be empty.'
    }
    Assert-Port 'RELAY_CONTROL_PORT' $resolvedRelayPort
    Assert-Port 'RELAY_QUIC_PORT' $resolvedRelayQuicPort
    Assert-Port 'CONTROL_PLANE_PORT' $resolvedControlPlanePort
    Assert-Port 'DASHBOARD_PORT' $resolvedDashboardPort
    Assert-Port 'NAT_PORT' $resolvedNatPort

    New-Item -ItemType Directory -Force -Path $script:ConfigDir, $script:DataDir, $script:LogDir, $script:RunDir, $script:WebDir | Out-Null
    $lines = @(
        '# NexTunnel server runtime config generated by deploy/server/install.ps1',
        (Convert-ToEnvLine 'NEXTUNNEL_PUBLIC_HOST' $resolvedPublicHost),
        (Convert-ToEnvLine 'RELAY_BIND' (Get-ConfigValue $script:LocalEnv 'RELAY_BIND' '0.0.0.0')),
        (Convert-ToEnvLine 'RELAY_CONTROL_PORT' "$resolvedRelayPort"),
        (Convert-ToEnvLine 'RELAY_QUIC_PORT' "$resolvedRelayQuicPort"),
        (Convert-ToEnvLine 'RELAY_AUTH_TOKEN' $resolvedRelayToken),
        (Convert-ToEnvLine 'RELAY_REQUIRE_AUTH' (Get-ConfigValue $script:LocalEnv 'RELAY_REQUIRE_AUTH' 'true')),
        (Convert-ToEnvLine 'RELAY_STATS_INTERVAL' (Get-ConfigValue $script:LocalEnv 'RELAY_STATS_INTERVAL' '30s')),
        (Convert-ToEnvLine 'CONTROL_PLANE_LISTEN' "0.0.0.0:$resolvedControlPlanePort"),
        (Convert-ToEnvLine 'CONTROL_PLANE_PORT' "$resolvedControlPlanePort"),
        (Convert-ToEnvLine 'CONTROL_PLANE_API_TOKEN' $resolvedControlToken),
        (Convert-ToEnvLine 'CONTROL_PLANE_STORE_PATH' (Join-Path $script:DataDir 'control-plane.db')),
        (Convert-ToEnvLine 'DASHBOARD_ENABLED' $resolvedDashboardEnabled),
        (Convert-ToEnvLine 'DASHBOARD_LISTEN' "0.0.0.0:$resolvedDashboardPort"),
        (Convert-ToEnvLine 'DASHBOARD_PORT' "$resolvedDashboardPort"),
        (Convert-ToEnvLine 'DASHBOARD_SECRET_KEY' $resolvedDashboardSecret),
        (Convert-ToEnvLine 'DASHBOARD_ADMIN_USER' $resolvedDashboardAdmin),
        (Convert-ToEnvLine 'DASHBOARD_ADMIN_PASSWORD' $resolvedDashboardPassword),
        (Convert-ToEnvLine 'DASHBOARD_ALLOWED_ORIGINS' "$resolvedDashboardOrigins,http://${resolvedPublicHost}:$resolvedDashboardPort"),
        (Convert-ToEnvLine 'DASHBOARD_STORE_PATH' (Join-Path $script:DataDir 'dashboard.db')),
        (Convert-ToEnvLine 'DASHBOARD_STATIC_DIR' $script:WebDir),
        (Convert-ToEnvLine 'NAT_PRIMARY_ADDR' (Get-ConfigValue $script:LocalEnv 'NAT_PRIMARY_ADDR' '0.0.0.0')),
        (Convert-ToEnvLine 'NAT_ALT_ADDR' (Get-ConfigValue $script:LocalEnv 'NAT_ALT_ADDR' '127.0.0.1')),
        (Convert-ToEnvLine 'NAT_PORT' "$resolvedNatPort"),
        (Convert-ToEnvLine 'NAT_REALM' (Get-ConfigValue $script:LocalEnv 'NAT_REALM' 'nextunnel.local')),
        (Convert-ToEnvLine 'NEXTUNNEL_DATA_DIR' $script:DataDir)
    )
    Set-Content -LiteralPath $script:EnvPath -Value $lines -Encoding UTF8
    Write-Step "Generated config: $script:EnvPath"
}

function Read-EnvFile {
    return Read-DotEnvFile $script:EnvPath
}

function Set-ProcessEnvironment {
    $envMap = Read-EnvFile
    if ($envMap.Count -eq 0) {
        throw 'server.env not found. Run install or config first.'
    }
    foreach ($key in $envMap.Keys) {
        [Environment]::SetEnvironmentVariable($key, $envMap[$key], 'Process')
    }
    return $envMap
}

function Get-PidPath {
    param([string]$Name)
    return (Join-Path $script:RunDir "$Name.pid")
}

function Test-ManagedProcess {
    param([string]$Name)
    $pidPath = Get-PidPath $Name
    if (-not (Test-Path -LiteralPath $pidPath)) {
        return $false
    }
    $processId = [int](Get-Content -LiteralPath $pidPath -Raw)
    return [bool](Get-Process -Id $processId -ErrorAction SilentlyContinue)
}

function Start-ManagedProcess {
    param(
        [string]$Name,
        [string]$FilePath,
        [string[]]$Arguments
    )
    if (Test-ManagedProcess $Name) {
        Write-Warn "$Name is already running."
        return
    }
    if (-not (Test-Path -LiteralPath $FilePath)) {
        throw "Binary not found: $FilePath"
    }
    New-Item -ItemType Directory -Force -Path $script:LogDir, $script:RunDir | Out-Null
    $stdoutPath = Join-Path $script:LogDir "$Name.log"
    $stderrPath = Join-Path $script:LogDir "$Name.err.log"
    # 通过当前进程环境传入敏感配置，避免把 token 固定写到启动脚本里。
    $process = Start-Process -FilePath $FilePath -ArgumentList $Arguments -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath -WindowStyle Hidden -PassThru
    Set-Content -LiteralPath (Get-PidPath $Name) -Value $process.Id -Encoding ASCII
    Write-Step "Started $Name pid=$($process.Id)"
}

function Stop-ManagedProcess {
    param([string]$Name)
    $pidPath = Get-PidPath $Name
    if (-not (Test-Path -LiteralPath $pidPath)) {
        Write-Warn "$Name pid file not found."
        return
    }
    $processId = [int](Get-Content -LiteralPath $pidPath -Raw)
    $process = Get-Process -Id $processId -ErrorAction SilentlyContinue
    if ($process) {
        Stop-Process -Id $processId -Force
        Write-Step "Stopped $Name pid=$processId"
    }
    Remove-Item -LiteralPath $pidPath -Force
}

function Start-Stack {
    $envMap = Set-ProcessEnvironment
    Start-ManagedProcess -Name 'control-plane' -FilePath (Get-BinaryPath 'control-plane') -Arguments @(
        '--listen', $envMap['CONTROL_PLANE_LISTEN'],
        '--api-token', $envMap['CONTROL_PLANE_API_TOKEN'],
        '--store-path', $envMap['CONTROL_PLANE_STORE_PATH']
    )
    Start-ManagedProcess -Name 'relay-server' -FilePath (Get-BinaryPath 'relay-server') -Arguments @(
        '--bind', $envMap['RELAY_BIND'],
        '--control-port', $envMap['RELAY_CONTROL_PORT'],
        '--quic-port', $envMap['RELAY_QUIC_PORT'],
        '--auth-token', $envMap['RELAY_AUTH_TOKEN'],
        '--require-auth',
        '--stats-interval', $envMap['RELAY_STATS_INTERVAL']
    )
    Start-ManagedProcess -Name 'nat-detector' -FilePath (Get-BinaryPath 'nat-detector') -Arguments @(
        '--primary-addr', $envMap['NAT_PRIMARY_ADDR'],
        '--alt-addr', $envMap['NAT_ALT_ADDR'],
        '--port', $envMap['NAT_PORT'],
        '--realm', $envMap['NAT_REALM']
    )
    if ((Test-Truthy $envMap['DASHBOARD_ENABLED']) -and (Test-Path -LiteralPath (Get-BinaryPath 'dashboard'))) {
        Start-ManagedProcess -Name 'dashboard' -FilePath (Get-BinaryPath 'dashboard') -Arguments @(
            '--listen', $envMap['DASHBOARD_LISTEN'],
            '--secret-key', $envMap['DASHBOARD_SECRET_KEY'],
            '--admin-user', $envMap['DASHBOARD_ADMIN_USER'],
            '--admin-password', $envMap['DASHBOARD_ADMIN_PASSWORD'],
            '--allowed-origins', $envMap['DASHBOARD_ALLOWED_ORIGINS'],
            '--store-path', $envMap['DASHBOARD_STORE_PATH'],
            '--static-dir', $envMap['DASHBOARD_STATIC_DIR']
        )
    } elseif (Test-Truthy $envMap['DASHBOARD_ENABLED']) {
        Write-Warn 'Dashboard is enabled, but dashboard binary is not installed. Core services are running only.'
    }
}

function Stop-Stack {
    Stop-ManagedProcess 'dashboard'
    Stop-ManagedProcess 'nat-detector'
    Stop-ManagedProcess 'relay-server'
    Stop-ManagedProcess 'control-plane'
}

function Show-Status {
    foreach ($name in @('control-plane', 'relay-server', 'nat-detector', 'dashboard')) {
        $pidPath = Get-PidPath $name
        if (-not (Test-Path -LiteralPath $pidPath)) {
            Write-Host "$name stopped"
            continue
        }
        $processId = [int](Get-Content -LiteralPath $pidPath -Raw)
        $process = Get-Process -Id $processId -ErrorAction SilentlyContinue
        if ($process) {
            Write-Host "$name running pid=$processId"
        } else {
            Write-Host "$name stale pid=$processId"
        }
    }
}

function Show-Logs {
    $paths = @(
        (Join-Path $script:LogDir 'control-plane.log'),
        (Join-Path $script:LogDir 'relay-server.log'),
        (Join-Path $script:LogDir 'nat-detector.log'),
        (Join-Path $script:LogDir 'dashboard.log')
    ) | Where-Object { Test-Path -LiteralPath $_ }
    if ($paths.Count -eq 0) {
        throw "No log files found in $script:LogDir"
    }
    Get-Content -LiteralPath $paths -Tail 200 -Wait
}

function Show-ConnectionInfo {
    $envMap = Read-EnvFile
    if ($envMap.Count -eq 0) {
        throw 'server.env not found. Run install or config first.'
    }
    $hostName = $envMap['NEXTUNNEL_PUBLIC_HOST']
    Write-Host ''
    Write-Host 'Connection info:' -ForegroundColor Green
    Write-Host "  Relay TCP:       ${hostName}:$($envMap['RELAY_CONTROL_PORT'])"
    Write-Host "  Relay QUIC UDP:  ${hostName}:$($envMap['RELAY_QUIC_PORT'])"
    Write-Host "  NAT UDP:         ${hostName}:$($envMap['NAT_PORT'])"
    Write-Host "  Control Plane:   http://${hostName}:$($envMap['CONTROL_PLANE_PORT'])"
    if (Test-Truthy $envMap['DASHBOARD_ENABLED']) {
        Write-Host "  Dashboard:       http://${hostName}:$($envMap['DASHBOARD_PORT'])"
        Write-Host "  Dashboard User:  $($envMap['DASHBOARD_ADMIN_USER'])"
    }
    Write-Host "  Relay Token:     $($envMap['RELAY_AUTH_TOKEN'])"
    Write-Host "  Control Token:   $($envMap['CONTROL_PLANE_API_TOKEN'])"
}

function Test-Health {
    $envMap = Read-EnvFile
    if ($envMap.Count -eq 0) {
        throw 'server.env not found. Run install first.'
    }
    Write-Step 'Checking Control Plane health'
    Invoke-WebRequest -Uri "http://127.0.0.1:$($envMap['CONTROL_PLANE_PORT'])/healthz" -UseBasicParsing | Select-Object StatusCode, StatusDescription
    Write-Step 'Checking Relay TCP port'
    $tcpClient = New-Object Net.Sockets.TcpClient
    try {
        $tcpClient.Connect('127.0.0.1', [int]$envMap['RELAY_CONTROL_PORT'])
    } finally {
        $tcpClient.Close()
    }
    Write-Host "Relay TCP $($envMap['RELAY_CONTROL_PORT']) is reachable" -ForegroundColor Green
    if ((Test-Truthy $envMap['DASHBOARD_ENABLED']) -and (Test-Path -LiteralPath (Get-BinaryPath 'dashboard'))) {
        Write-Step 'Checking Dashboard health'
        Invoke-WebRequest -Uri "http://127.0.0.1:$($envMap['DASHBOARD_PORT'])/api/v1/health" -UseBasicParsing | Select-Object StatusCode, StatusDescription
    }
}

function Invoke-Install {
    Write-EnvFile
    Install-ReleasePackage
    Start-Stack
    Show-ConnectionInfo
}

$script:ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$script:LocalEnvPath = Join-Path $script:ScriptDir '.env'
$script:LocalEnv = Read-DotEnvFile $script:LocalEnvPath

$programDataRoot = if ($env:ProgramData) { Join-Path $env:ProgramData 'NexTunnel' } else { Join-Path $script:ScriptDir 'runtime' }
$script:InstallDir = if ($InstallDir) { $InstallDir } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_INSTALL_DIR' (Join-Path $programDataRoot 'server') }
$script:ConfigDir = if ($ConfigDir) { $ConfigDir } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_CONFIG_DIR' (Join-Path $programDataRoot 'config') }
$script:DataDir = if ($DataDir) { $DataDir } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_DATA_DIR' (Join-Path $programDataRoot 'data') }
$script:BinDir = Join-Path $script:InstallDir 'bin'
$script:WebDir = Join-Path (Join-Path $script:InstallDir 'web') 'dashboard'
$script:DeployDir = Join-Path (Join-Path $script:InstallDir 'deploy') 'server'
$script:LogDir = Join-Path $script:InstallDir 'logs'
$script:RunDir = Join-Path $script:InstallDir 'run'
$script:EnvPath = Join-Path $script:ConfigDir 'server.env'
$script:RepositoryValue = if ($Repository) { $Repository } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_REPOSITORY' 'Lee-zg/NexTunnel' }
$script:VersionValue = if ($Version) { $Version } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_VERSION' 'latest' }
$script:ReleaseBaseUrlValue = if ($ReleaseBaseUrl) { $ReleaseBaseUrl } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_RELEASE_BASE_URL' '' }
$script:GithubProxyValue = if ($GithubProxy) { $GithubProxy } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_GITHUB_PROXY' '' }
$script:PackageUrlValue = if ($PackageUrl) { $PackageUrl } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_PACKAGE_URL' '' }
$script:PackageSha256Value = if ($PackageSha256) { $PackageSha256 } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_PACKAGE_SHA256' '' }
$script:ArchitectureValue = if ($Architecture) { $Architecture } else { Get-ConfigValue $script:LocalEnv 'NEXTUNNEL_ARCH' '' }

switch ($Action) {
    'install' { Invoke-Install }
    'up' { Start-Stack; Show-ConnectionInfo }
    'down' { Stop-Stack }
    'restart' { Stop-Stack; Start-Stack; Show-ConnectionInfo }
    'status' { Show-Status }
    'logs' { Show-Logs }
    'health' { Test-Health }
    'update' { Stop-Stack; Install-ReleasePackage; Start-Stack; Show-ConnectionInfo }
    'uninstall' {
        Stop-Stack
        if ($Purge) {
            foreach ($path in @($script:InstallDir, $script:ConfigDir, $script:DataDir)) {
                if (Test-Path -LiteralPath $path) {
                    Remove-Item -LiteralPath $path -Recurse -Force
                }
            }
            Write-Warn 'Removed install, config and data directories.'
        } else {
            Write-Warn "Processes stopped. Keep $script:InstallDir, $script:ConfigDir and $script:DataDir by default. Use -Purge to remove them."
        }
    }
    'config' { Write-EnvFile; Show-ConnectionInfo }
}
