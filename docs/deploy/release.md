# 发布流程

v0.4.1-alpha 使用统一版本号发布桌面端、CLI、服务端和文档。

## 本地打包

```bash
make package-desktop VERSION=v0.4.1-alpha
make package-cli VERSION=v0.4.1-alpha
make package-server VERSION=v0.4.1-alpha
```

## GitHub Release

推送 `v0.4.1-alpha` 标签会触发 `.github/workflows/release.yml`：

- Windows 桌面端 zip 和 SHA256。
- Linux/Windows CLI 包和 SHA256。
- Linux/Windows 服务端包和 SHA256。
- Linux `install.sh` 和 Windows `install.ps1` 一键安装脚本。
- VitePress 文档站压缩包和 SHA256，并同步发布到 GitHub Pages。
- 服务端包内置 Dashboard、Edge rehearsal、eBPF verify 验证入口和生产验证脚本。

首次发布文档站前，需要在仓库 Pages 中启用 `GitHub Actions` 发布模式。当前站点地址为 `https://lee-zg.github.io/NexTunnel/`。

## Release 资产清单

```text
nextunnel-v0.4.1-alpha-windows-amd64.zip
nextunnel-v0.4.1-alpha-windows-amd64.zip.sha256
nextunnel-cli-v0.4.1-alpha-linux-amd64.tar.gz
nextunnel-cli-v0.4.1-alpha-linux-arm64.tar.gz
nextunnel-cli-v0.4.1-alpha-windows-amd64.zip
nextunnel-server-linux-amd64.tar.gz
nextunnel-server-linux-arm64.tar.gz
nextunnel-server-windows-amd64.zip
nextunnel-docs-v0.4.1-alpha.tar.gz
install.sh
install.ps1
*.sha256
```

## 发布前检查

```bash
go test ./...
cd desktop/frontend && npm run build
cd server/web && npm run build
cd docs && npm run docs:build
```

发布前还应按 `docs/deploy/production-verification.md` 更新生产验证状态。真实 TUN 验证需要 Windows 管理员权限和匹配架构 `wintun.dll`，macOS 需要 sudo/root 权限。

发布后需要确认：

- Release 页面存在所有包和校验文件。
- `install.sh` 可以通过 `releases/download/v0.4.1-alpha/install.sh` 下载。
- `nextunnel-docs-v0.4.1-alpha.tar.gz` 可下载，且校验文件匹配。
- 文档站可访问并显示 `v0.4.1-alpha` 更新日志。
