[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateNotNullOrEmpty()]
    [string]$Target = "help",

    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$RemainingArgs = @()
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$DefaultVersion = "v0.5.0-alpha"
$DefaultMacHost = "10.160.166.44"
$DefaultMacUser = "lizhigang"
$DefaultMacPort = "22"
$DefaultDashboardUser = "root"
$DefaultDashboardRemotePort = "8080"
$IsWindowsHost = [System.Environment]::OSVersion.Platform -eq [System.PlatformID]::Win32NT

$TargetDescriptions = [ordered]@{
    "help"                 = "显示帮助"
    "dev"                  = "启动 Wails 桌面开发服务器"
    "dev-server-web"       = "启动服务端管理 Web 控制台"
    "build"                = "构建 Wails 桌面应用"
    "build-server"         = "构建服务端与 CLI 二进制"
    "package-desktop"      = "构建 Windows 桌面发布包"
    "package-macos"        = "构建 macOS 桌面 DMG 发布包"
    "package-cli"          = "构建 CLI 发布包"
    "package-server"       = "构建服务端发布包"
    "lint"                 = "运行全部代码检查"
    "lint-go"              = "运行 Go 代码检查"
    "lint-frontend"        = "运行前端代码检查"
    "test"                 = "运行全部测试"
    "test-go"              = "运行 Go 测试"
    "test-frontend"        = "运行前端测试"
    "verify-dashboard"     = "验证 Dashboard 生产 API"
    "verify-dashboard-ssh" = "通过 SSH 隧道验证 Dashboard"
    "verify-tun"           = "运行本地 TUN 验证"
    "verify-p2p-tun"       = "运行 Windows/macOS P2P TUN 验证"
    "verify-edge"          = "运行 Edge/Anycast 演练"
    "verify-ebpf-linux"    = "在 Linux 上编译并挂载 XDP"
    "clean"                = "清理构建产物"
    "install-deps"         = "安装全部依赖"
}

function Show-Help {
    Write-Host "NexTunnel Build System"
    Write-Host ""
    Write-Host "Usage: .\make.ps1 <target>"
    Write-Host ""
    Write-Host "Targets:"
    foreach ($targetName in $TargetDescriptions.Keys) {
        Write-Host ("  {0,-22} {1}" -f $targetName, $TargetDescriptions[$targetName])
    }
    Write-Host ""
    Write-Host "提示：Windows PowerShell 不会自动提供 make 命令；如需裸命令 make dev，请安装 GNU Make 并加入 PATH。"
}

function Get-SettingValue {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $true)]
        [AllowEmptyString()]
        [string]$DefaultValue
    )

    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $DefaultValue
    }

    return $value
}

function Assert-SettingValue {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $true)]
        [AllowEmptyString()]
        [string]$Value
    )

    if ([string]::IsNullOrWhiteSpace($Value)) {
        throw "目标 '$Target' 需要先设置环境变量 $Name。"
    }
}

function Invoke-RequiredCommand {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command,

        [string[]]$Arguments = @()
    )

    if (-not (Get-Command $Command -ErrorAction SilentlyContinue)) {
        throw "未找到命令 '$Command'。请先安装并确认它已加入 PATH。"
    }

    & $Command @Arguments
}

function Invoke-InDirectory {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,

        [Parameter(Mandatory = $true)]
        [scriptblock]$Script
    )

    $absolutePath = Join-Path $RepoRoot $Path
    if (-not (Test-Path -LiteralPath $absolutePath -PathType Container)) {
        throw "目录不存在：$Path"
    }

    Push-Location $absolutePath
    try {
        & $Script
    }
    finally {
        Pop-Location
    }
}

function Get-PowerShellExecutable {
    if (Get-Command "pwsh" -ErrorAction SilentlyContinue) {
        return "pwsh"
    }

    return "powershell"
}

function Invoke-RepoScript {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ScriptPath,

        [string[]]$Arguments = @()
    )

    $absoluteScriptPath = Join-Path $RepoRoot $ScriptPath
    if (-not (Test-Path -LiteralPath $absoluteScriptPath -PathType Leaf)) {
        throw "脚本不存在：$ScriptPath"
    }

    $powerShellExecutable = Get-PowerShellExecutable
    $scriptArguments = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $absoluteScriptPath) + $Arguments
    Invoke-RequiredCommand -Command $powerShellExecutable -Arguments $scriptArguments
}

function Get-BinaryOutputName {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if ($IsWindowsHost) {
        return "$Name.exe"
    }

    return $Name
}

function Invoke-GoBuild {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ModulePath,

        [Parameter(Mandatory = $true)]
        [string]$PackagePath,

        [Parameter(Mandatory = $true)]
        [string]$OutputName
    )

    $outputPath = Join-Path "..\build" (Get-BinaryOutputName -Name $OutputName)
    Invoke-InDirectory $ModulePath {
        Invoke-RequiredCommand -Command "go" -Arguments @("build", "-o", $outputPath, $PackagePath)
    }
}

function Remove-RepoPath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $absoluteRepoRoot = [System.IO.Path]::GetFullPath($RepoRoot).TrimEnd(
        [System.IO.Path]::DirectorySeparatorChar,
        [System.IO.Path]::AltDirectorySeparatorChar
    )
    $absoluteTargetPath = [System.IO.Path]::GetFullPath((Join-Path $RepoRoot $Path))
    $repoPathPrefix = $absoluteRepoRoot + [System.IO.Path]::DirectorySeparatorChar

    # 清理命令只允许删除仓库内的已知构建产物，避免误删用户数据。
    if ($absoluteTargetPath -ne $absoluteRepoRoot -and -not $absoluteTargetPath.StartsWith($repoPathPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "拒绝清理仓库外路径：$Path"
    }

    if (Test-Path -LiteralPath $absoluteTargetPath) {
        Remove-Item -LiteralPath $absoluteTargetPath -Recurse -Force
    }
}

function Invoke-Target {
    param(
        [Parameter(Mandatory = $true)]
        [string]$TargetName
    )

    switch ($TargetName.ToLowerInvariant()) {
        "all" {
            Invoke-Target "build"
        }
        "help" {
            Show-Help
        }
        "dev" {
            Invoke-InDirectory "desktop" {
                Invoke-RequiredCommand -Command "wails" -Arguments (@("dev") + $RemainingArgs)
            }
        }
        "dev-server-web" {
            Invoke-InDirectory "server\web" {
                Invoke-RequiredCommand -Command "npm" -Arguments (@("run", "dev") + $RemainingArgs)
            }
        }
        "build" {
            Invoke-InDirectory "desktop" {
                Invoke-RequiredCommand -Command "wails" -Arguments (@("build") + $RemainingArgs)
            }
        }
        "package-desktop" {
            $version = Get-SettingValue -Name "VERSION" -DefaultValue $DefaultVersion
            $wintunDll = Get-SettingValue -Name "WINTUN_DLL" -DefaultValue ""
            $wintunSha256 = Get-SettingValue -Name "WINTUN_SHA256" -DefaultValue ""
            Invoke-RepoScript -ScriptPath "scripts\package-desktop.ps1" -Arguments @("-Version", $version, "-WintunDllPath", $wintunDll, "-WintunSha256", $wintunSha256)
        }
        "package-macos" {
            $version = Get-SettingValue -Name "VERSION" -DefaultValue $DefaultVersion
            Invoke-RequiredCommand -Command "bash" -Arguments @("scripts/package-macos.sh", "--version", $version)
        }
        "package-cli" {
            $version = Get-SettingValue -Name "VERSION" -DefaultValue $DefaultVersion
            Invoke-RepoScript -ScriptPath "scripts\package-cli.ps1" -Arguments @("-Version", $version)
        }
        "package-server" {
            $version = Get-SettingValue -Name "VERSION" -DefaultValue $DefaultVersion
            Invoke-RepoScript -ScriptPath "scripts\package-server.ps1" -Arguments @("-Version", $version)
        }
        "build-server" {
            New-Item -ItemType Directory -Path (Join-Path $RepoRoot "build") -Force | Out-Null
            Invoke-GoBuild -ModulePath "server" -PackagePath "./cmd/control-plane" -OutputName "control-plane"
            Invoke-GoBuild -ModulePath "server" -PackagePath "./cmd/relay" -OutputName "relay-server"
            Invoke-GoBuild -ModulePath "server" -PackagePath "./cmd/nat-detector" -OutputName "nat-detector"
            Invoke-GoBuild -ModulePath "server" -PackagePath "./cmd/dashboard" -OutputName "dashboard"
            Invoke-GoBuild -ModulePath "server" -PackagePath "./cmd/edge-rehearsal" -OutputName "edge-rehearsal"
            Invoke-GoBuild -ModulePath "server" -PackagePath "./cmd/ebpf-verify" -OutputName "ebpf-verify"
            Invoke-GoBuild -ModulePath "cli" -PackagePath "." -OutputName "nextunnel"
        }
        "lint" {
            Invoke-Target "lint-go"
            Invoke-Target "lint-frontend"
        }
        "lint-go" {
            Invoke-InDirectory "desktop" {
                Invoke-RequiredCommand -Command "golangci-lint" -Arguments @("run", "./...")
            }
            Invoke-InDirectory "server" {
                Invoke-RequiredCommand -Command "golangci-lint" -Arguments @("run", "./...")
            }
        }
        "lint-frontend" {
            Invoke-InDirectory "desktop\frontend" {
                Invoke-RequiredCommand -Command "npm" -Arguments @("run", "lint")
            }
            Invoke-InDirectory "server\web" {
                Invoke-RequiredCommand -Command "npm" -Arguments @("run", "lint")
            }
        }
        "test" {
            Invoke-Target "test-go"
            Invoke-Target "test-frontend"
        }
        "test-go" {
            Invoke-InDirectory "desktop" {
                Invoke-RequiredCommand -Command "go" -Arguments @("test", "./...")
            }
            Invoke-InDirectory "server" {
                Invoke-RequiredCommand -Command "go" -Arguments @("test", "./...")
            }
            Invoke-InDirectory "cli" {
                Invoke-RequiredCommand -Command "go" -Arguments @("test", "./...")
            }
            Invoke-InDirectory "pkg" {
                Invoke-RequiredCommand -Command "go" -Arguments @("test", "./...")
            }
        }
        "test-frontend" {
            Invoke-InDirectory "desktop\frontend" {
                Invoke-RequiredCommand -Command "npm" -Arguments @("run", "test")
            }
            Invoke-InDirectory "server\web" {
                Invoke-RequiredCommand -Command "npm" -Arguments @("run", "test")
            }
        }
        "verify-dashboard" {
            $dashboardUrl = Get-SettingValue -Name "DASHBOARD_URL" -DefaultValue ""
            $dashboardPassword = Get-SettingValue -Name "DASHBOARD_PASSWORD" -DefaultValue ""
            $dashboardOrigin = Get-SettingValue -Name "DASHBOARD_ORIGIN" -DefaultValue ""
            Assert-SettingValue -Name "DASHBOARD_URL" -Value $dashboardUrl
            Assert-SettingValue -Name "DASHBOARD_PASSWORD" -Value $dashboardPassword
            Invoke-RepoScript -ScriptPath "scripts\verify-dashboard.ps1" -Arguments @(
                "-BaseUrl", $dashboardUrl,
                "-Password", $dashboardPassword,
                "-AllowedOrigin", $dashboardOrigin,
                "-ReportPath", "dist\verification\dashboard-report.json"
            )
        }
        "verify-dashboard-ssh" {
            $dashboardHost = Get-SettingValue -Name "DASHBOARD_HOST" -DefaultValue ""
            $dashboardUser = Get-SettingValue -Name "DASHBOARD_USER" -DefaultValue $DefaultDashboardUser
            $dashboardIdentity = Get-SettingValue -Name "DASHBOARD_IDENTITY" -DefaultValue ""
            $dashboardRemotePort = Get-SettingValue -Name "DASHBOARD_REMOTE_PORT" -DefaultValue $DefaultDashboardRemotePort
            $dashboardOrigin = Get-SettingValue -Name "DASHBOARD_ORIGIN" -DefaultValue ""
            Assert-SettingValue -Name "DASHBOARD_HOST" -Value $dashboardHost
            Invoke-RepoScript -ScriptPath "scripts\verify-dashboard-ssh.ps1" -Arguments @(
                "-SshHost", $dashboardHost,
                "-User", $dashboardUser,
                "-IdentityFile", $dashboardIdentity,
                "-RemoteDashboardPort", $dashboardRemotePort,
                "-AllowedOrigin", $dashboardOrigin,
                "-ReportPath", "dist\verification\dashboard-ssh-report.json"
            )
        }
        "verify-tun" {
            Invoke-RepoScript -ScriptPath "scripts\verify-tun.ps1"
        }
        "verify-p2p-tun" {
            $macHost = Get-SettingValue -Name "MAC_HOST" -DefaultValue $DefaultMacHost
            $macUser = Get-SettingValue -Name "MAC_USER" -DefaultValue $DefaultMacUser
            $macPort = Get-SettingValue -Name "MAC_PORT" -DefaultValue $DefaultMacPort
            $relayAddr = Get-SettingValue -Name "RELAY_ADDR" -DefaultValue ""
            $relayToken = Get-SettingValue -Name "RELAY_TOKEN" -DefaultValue ""
            Invoke-RepoScript -ScriptPath "scripts\verify-p2p-tun.ps1" -Arguments @(
                "-MacHost", $macHost,
                "-MacUser", $macUser,
                "-MacPort", $macPort,
                "-RelayAddr", $relayAddr,
                "-RelayToken", $relayToken
            )
        }
        "verify-edge" {
            $controlUrl = Get-SettingValue -Name "CONTROL_URL" -DefaultValue ""
            $controlToken = Get-SettingValue -Name "CONTROL_TOKEN" -DefaultValue ""
            $registerRemote = Get-SettingValue -Name "REGISTER_REMOTE" -DefaultValue ""
            $verifyEdgeArguments = @("-ControlUrl", $controlUrl, "-ControlToken", $controlToken)
            if ($registerRemote.Equals("true", [System.StringComparison]::OrdinalIgnoreCase)) {
                $verifyEdgeArguments += "-RegisterRemote"
            }
            Invoke-RepoScript -ScriptPath "scripts\verify-edge-rehearsal.ps1" -Arguments $verifyEdgeArguments
        }
        "verify-ebpf-linux" {
            Invoke-RequiredCommand -Command "bash" -Arguments @("scripts/verify-ebpf-linux.sh")
        }
        "clean" {
            Remove-RepoPath "desktop\build\bin"
            Remove-RepoPath "desktop\frontend\dist"
            Remove-RepoPath "server\web\dist"
            Remove-RepoPath "build"
        }
        "install-deps" {
            Invoke-InDirectory "desktop" {
                Invoke-RequiredCommand -Command "go" -Arguments @("mod", "tidy")
            }
            Invoke-InDirectory "server" {
                Invoke-RequiredCommand -Command "go" -Arguments @("mod", "tidy")
            }
            Invoke-InDirectory "pkg" {
                Invoke-RequiredCommand -Command "go" -Arguments @("mod", "tidy")
            }
            Invoke-InDirectory "desktop\frontend" {
                Invoke-RequiredCommand -Command "npm" -Arguments @("install")
            }
            Invoke-InDirectory "server\web" {
                Invoke-RequiredCommand -Command "npm" -Arguments @("install")
            }
        }
        default {
            throw "未知目标 '$TargetName'。运行 '.\make.ps1 help' 查看可用命令。"
        }
    }
}

Invoke-Target $Target
