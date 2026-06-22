# 发布流程

v0.5.0-alpha 使用统一版本号发布桌面端安装器、CLI、服务端和文档。

## 本地打包

```bash
make package-desktop VERSION=v0.5.0-alpha
make package-macos VERSION=v0.5.0-alpha
make package-cli VERSION=v0.5.0-alpha
make package-server VERSION=v0.5.0-alpha
```

Windows 推荐发布 NSIS 安装包和 zip 便携包：

```powershell
.\scripts\package-desktop.ps1 -Version v0.5.0-alpha -Installer nsis
```

NSIS 安装包会检测 `wintun.dll`。缺失时，安装器可以从 WireGuard 官方资源下载 Wintun ZIP、可选校验 SHA256，并复制匹配架构的 DLL 到应用目录。发布时建议配置校验值：

```powershell
.\scripts\package-desktop.ps1 `
  -Version v0.5.0-alpha `
  -Installer nsis `
  -WintunSha256 "<official-wintun-zip-sha256>"
```

zip 便携包仍支持把官方 DLL 随包放入应用目录：

```powershell
$env:NEXTUNNEL_WINTUN_DLL="D:\path\to\wintun.dll"
make package-desktop VERSION=v0.5.0-alpha
```

macOS DMG 只能在 macOS 本机或 macOS runner 构建：

```bash
bash scripts/package-macos.sh --version v0.5.0-alpha
```

alpha 默认生成未签名 DMG。配置 `MACOS_DEVELOPER_ID_APPLICATION` 后可加 `--sign`，配置 `MACOS_NOTARY_APPLE_ID`、`MACOS_NOTARY_TEAM_ID`、`MACOS_NOTARY_PASSWORD` 后可加 `--notarize`。

## GitHub Release

推送 `v0.5.0-alpha` 标签会触发 `.github/workflows/release.yml`：

- Windows NSIS 安装包、Windows zip 便携包、manifest 和 SHA256。
- macOS universal DMG、manifest 和 SHA256。
- Linux/Windows CLI 包和 SHA256。
- Linux/Windows 服务端包和 SHA256。
- Linux `install.sh` 和 Windows `install.ps1` 一键安装脚本。
- VitePress 文档站压缩包和 SHA256，并同步发布到 GitHub Pages。

首次发布文档站前，需要在仓库 Pages 中启用 `GitHub Actions` 发布模式。当前站点地址为 `https://lee-zg.github.io/NexTunnel/`。

## Release 资产清单

```text
nextunnel-v0.5.0-alpha-windows-amd64-installer.exe
nextunnel-v0.5.0-alpha-windows-amd64-installer.exe.sha256
nextunnel-v0.5.0-alpha-windows-amd64-installer.MANIFEST.txt
nextunnel-v0.5.0-alpha-windows-amd64.zip
nextunnel-v0.5.0-alpha-windows-amd64.zip.sha256
nextunnel-v0.5.0-alpha-darwin-universal.dmg
nextunnel-v0.5.0-alpha-darwin-universal.dmg.sha256
nextunnel-v0.5.0-alpha-darwin-universal.MANIFEST.txt
nextunnel-cli-v0.5.0-alpha-linux-amd64.tar.gz
nextunnel-cli-v0.5.0-alpha-linux-arm64.tar.gz
nextunnel-cli-v0.5.0-alpha-windows-amd64.zip
nextunnel-server-linux-amd64.tar.gz
nextunnel-server-linux-arm64.tar.gz
nextunnel-server-windows-amd64.zip
nextunnel-docs-v0.5.0-alpha.tar.gz
install.sh
install.ps1
*.sha256
```

发布资源只上传最终压缩包、安装器、manifest、校验文件和一键安装脚本；不上传 unpacked 目录、构建缓存、旧版本 exe、日志或临时下载资源。

## 发布前检查

```bash
go test ./...
cd desktop/frontend && npm run build
cd server/web && npm run build
cd docs && npm run docs:build
```

发布前还应按 `docs/deploy/production-verification.md` 更新生产验证状态。真实 TUN 验证需要 Windows 管理员权限和匹配架构 `wintun.dll`；macOS 需要授权 helper、LaunchDaemon 或可用的 `sudo -n`。Dashboard 若没有可用 HTTPS 域名，应使用 `scripts/verify-dashboard-ssh.ps1` 通过 SSH 隧道完成 API 验收，不能把管理员密码发送到公网 HTTP。

发布后需要确认：

- Release 页面存在所有安装器、压缩包和校验文件。
- Windows 安装器可启动并显示 Wintun 检测/下载/跳过路径。
- macOS DMG 可挂载并包含 `NexTunnel.app`、Applications 链接、README 和 manifest。
- `install.sh` 可以通过 `releases/download/v0.5.0-alpha/install.sh` 下载。
- `nextunnel-docs-v0.5.0-alpha.tar.gz` 可下载，且校验文件匹配。
- 文档站可访问并显示 `v0.5.0-alpha` 更新日志。
