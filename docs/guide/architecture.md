# 架构说明

NexTunnel 按客户端、控制面、Relay 和公共协议分层。

## 客户端

- Vue 3 + Wails 负责桌面交互。
- `desktop/internal/tunnel` 管理 TCP/HTTP 隧道。
- `desktop/internal/p2p` 提供 NAT、ICE、WireGuard 和 Mesh 原型能力。
- `desktop/internal/config` 使用 SQLite 保存隧道、设置和常用端口。

## 服务端

- Relay Server 负责客户端注册、工作连接和流量中继。
- Control Plane 负责节点、ACL、路由和密钥元数据。
- NAT Detector 提供 STUN 探测。
- Dashboard 提供 Web 管理界面。

## v0.3.1-alpha 桌面链路

当前桌面端新增了流量增量图、连接类型展示和本机端口扫描。真实 P2P 数据面闭环仍需要继续在多 NAT 环境中验证；在 P2P 未明确连接时，运行中隧道会展示为中继路径。
