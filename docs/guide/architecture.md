# 架构说明

NexTunnel 按客户端、控制面、Relay 和公共协议分层。

## 客户端

- Vue 3 + Wails 负责桌面交互。
- `desktop/internal/tunnel` 管理 TCP/HTTP 隧道。
- `desktop/internal/p2p` 提供 NAT、ICE、WireGuard 和 Mesh 组网能力。
- `desktop/internal/config` 使用 SQLite 保存隧道、设置和常用端口。

## 服务端

- Relay Server 负责客户端注册、工作连接和流量中继。
- Control Plane 负责节点、ACL、路由和密钥元数据。
- NAT Detector 提供 STUN 探测。
- Dashboard 提供 Web 管理界面。

## v0.3.3-alpha 桌面链路

桌面端会记录流量快照，通过 `uPlot` Canvas 时序图展示上传和下载增量。隧道列表会展示连接类型：P2P 引擎明确 connected 时显示 P2P 直连；运行中但未进入直连态时显示中继；未运行时显示待机。

CLI 与安装脚本共享服务端运行配置。Linux 部署会把 `NEXTUNNEL_SERVICE_PREFIX`、安装目录和数据目录写入包内 `deploy/server/.env`，后续 CLI 或脚本查看状态、启动、停止、重启和读取日志时会使用同一组路径。
