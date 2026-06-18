# 生产验证手册

本文档覆盖 Beta 发布前剩余的生产验收项。所有命令都会生成 JSON 报告到 `dist/verification/`，失败时返回非零退出码。

## v0.4.1-alpha 验证进度

截至 v0.4.1-alpha 发布前，工程侧已完成生产验证入口、报告结构和故障前置检查，剩余实机验收以环境准备为主：

| 项目 | 状态 | 说明 |
| --- | --- | --- |
| Dashboard 端到端部署联调 | 待公网 HTTPS 环境复验 | 验证脚本已覆盖健康检查、登录、token、CORS、节点、ACL 和告警接口。 |
| Windows/macOS P2P 直连 | 已通过局域网双端验证 | Mac 回调 Windows 不可达时，验证器可由 Windows 主动推/拉候选完成直连链路。 |
| Windows 真实 TUN | 环境阻塞 | 当前环境缺少匹配架构 `wintun.dll`；生产可用前必须随包或安装器提供，并以管理员权限运行。 |
| macOS 真实 TUN | 权限阻塞 | 当前非 sudo/root 运行会失败；生产可用前需要授权 helper、LaunchDaemon 或 `sudo -n` 验证链路。 |
| Linux eBPF XDP | 待隔离 Linux 节点复验 | 验证脚本和 `ebpf-verify` 已入包；需真实网卡、clang 和 root/CAP 权限。 |
| 多地域 Edge/Anycast | 待真实多地域复验 | 本地演练脚本已覆盖注册、故障、恢复和 GeoIP 路由偏移。 |

发布边界：v0.4.1-alpha 可以声明生产验证工具链和故障诊断能力已齐备，P2P 直连链路已验证；真实系统 TUN、eBPF XDP 和多地域拓扑仍需在具备权限和依赖的生产或隔离环境完成最终验收。

## Dashboard 端到端部署联调

前置条件：

- Dashboard 已部署到真实 HTTP/HTTPS 地址。
- 已配置强 `DASHBOARD_SECRET_KEY`、管理员密码和 CORS 白名单。
- 若通过反向代理提供 HTTPS，外部访问地址应使用 HTTPS URL。

执行：

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-dashboard.ps1 `
  -BaseUrl "https://dashboard.example.com" `
  -Username "admin" `
  -Password "<强管理员密码>" `
  -AllowedOrigin "https://dashboard.example.com" `
  -ReportPath "dist/verification/dashboard-report.json"
```

验收点：

- `/api/v1/health` 返回 `ok`。
- 管理员可登录，合法 token 能访问节点、统计、ACL、告警接口。
- 无效 token 返回 401，前端应清理本地 token 并回到登录态。
- CORS 只允许白名单 Origin。
- 验证脚本会创建并删除临时 ACL 与告警规则。

## P2P/TUN 生产化验证

前置条件：

- Windows 使用管理员权限并安装/允许 Wintun。
- macOS 用户 `lizhigang` 先完成一次临时公钥接入。
- 执行前确认验证路由不会与当前办公/VPN 网段冲突。

执行：

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-p2p-tun.ps1 `
  -MacHost "10.160.166.44" `
  -MacUser "lizhigang" `
  -MacUseSudo `
  -WintunDllPath "D:\path\to\wintun.dll"
```

首次运行若 Mac 还没有临时公钥，会输出一次性 bootstrap 命令。先在本机执行该命令把公钥写入 Mac 的 `authorized_keys`，再重新运行上面的验证脚本。

验收点：

- Windows 侧真实 Wintun 创建成功，设备名不能是 `netTun`。
- Windows 侧路由注入成功，结束后无残留路由。
- macOS 侧 utun 创建成功，路由应用与清理成功。
- 双端 ICE 候选交换成功，直连优先，必要时可回落 Relay。
- 临时 SSH 公钥、远端临时文件和临时路由全部清理。
- 报告会输出 Windows 与 macOS 两端的 JSON 汇总到 `dist/verification/`。
- 若报告出现 `wintun_dll_missing`、`privilege_required`、`linux_dev_net_tun_missing` 等阻塞项，应先修复环境，不要用 `netTun` 回退结果替代生产验收。

验证完成后清理临时 SSH 公钥和远端临时目录：

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-p2p-tun.ps1 `
  -MacHost "10.160.166.44" `
  -MacUser "lizhigang" `
  -CleanupOnly
```

## eBPF Linux 生产验证

前置条件：

- 只能在 Linux 节点执行。
- 使用 root 或等价 `CAP_BPF`、`CAP_NET_ADMIN` 权限。
- 已安装 `clang` 和 Go。
- 在隔离节点或维护窗口执行，避免 XDP 挂载影响生产流量。

执行：

```bash
sudo INTERFACE_NAME=eth0 XDP_MODE=skb REPORT_PATH=dist/verification/ebpf-linux-report.json \
  bash scripts/verify-ebpf-linux.sh
```

验收点：

- `load_xdp` 显示 `mode=kernel`。
- 临时 DROP 规则可以同步到内核 map。
- `stats_read` 能读取内核/用户态统计。
- `unload_xdp` 成功卸载，执行后确认网卡无残留 XDP 程序。

## 多地域边缘部署演练

本地模拟演练：

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-edge-rehearsal.ps1
```

连真实 Control Plane 演练：

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-edge-rehearsal.ps1 `
  -ControlUrl "https://control.example.com" `
  -ControlToken "<control-plane-token>" `
  -RegisterRemote
```

验收点：

- Edge Registry 能注册 3 个区域节点。
- Anycast 能按客户端坐标选择最近节点。
- 节点故障后能切换到备用节点。
- GeoIP 静态映射能触发路由偏移。
- 真实 Control Plane 模式会注册、心跳并清理临时节点。

## 发布前清理

发布前执行：

```powershell
git status --short --branch
Get-FileHash -Algorithm MD5 desktop/frontend/package.json
Get-Content desktop/frontend/package.json.md5
```

`desktop/frontend/package.json.md5` 必须等于当前 `package.json` 的 MD5 小写值。若没有修改 `package.json`，不要单独提交无来源的 md5 变化。
