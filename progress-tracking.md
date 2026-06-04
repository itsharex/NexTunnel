# NexTunnel 项目进度跟踪

> **最后更新时间**：2026-06-04  
> **当前口径**：按“可验收 MVP / 原型验证 / 占位设计 / 未接入生产链路”统计，避免把单元测试通过等同于生产完成。

---

## 总体进度概览

```
Phase 1 [基础隧道]  ███████████████░░░░░  75%  MVP 可用，仍需强化 E2E 与运维配置
Phase 2 [P2P直连]   ████████░░░░░░░░░░░░  40%  原型验证，真实 TUN/IPAM/路由未完成
Phase 3 [智能调度]  ███████░░░░░░░░░░░░░  35%  调度模块存在，数据面闭环仍在接入
Phase 4 [全球加速]  ████░░░░░░░░░░░░░░░░  20%  后端模型/测试原型，非生产能力
─────────────────────────────────────────────
总体进度            ████████░░░░░░░░░░░░  42%
```

---

## 阶段里程碑状态

| 阶段 | 名称 | 当前状态 | 完成度 | 说明 |
|:---:|:---|:---:|:---:|:---|
| Phase 1 | 基础隧道 | 🔄 MVP 可用 | 75% | TCP Relay、客户端配置、SQLite、重连和桌面基础流程可用；安全配置、端到端覆盖、运维参数仍需补齐 |
| Phase 2 | P2P 直连 | 🔬 原型验证 | 40% | NAT/STUN/ICE/WireGuard 编排有测试；`netTun` 仍是 userspace 通道，缺少真实系统 TUN、IPAM、路由管理 |
| Phase 3 | 智能调度 | 🔬 原型验证 | 35% | QUIC、探测、调度、迁移模块有单元测试；调度器尚未完整控制 TCP/QUIC/P2P/Relay 数据面切换 |
| Phase 4 | 全球加速 | 🧪 概念原型 | 20% | edge/anycast/ebpf/sdwan/dashboard 为后端模型和测试原型，尚无生产部署和真实网络验证 |

---

## 当前已落地事实

| 领域 | 已实现 | 当前限制 |
|:---|:---|:---|
| Relay TCP 数据面 | 控制连接、工作连接、代理监听、会话桥接、基础统计 | 生产部署仍需完整认证策略、限流、审计、配置管理 |
| Relay QUIC 数据面 | QUIC listener 可启动，work stream 握手后接入既有会话桥接 | 客户端 QUIC Relay 工作连接仍需完整接入和证书信任配置 |
| 控制面 | HTTP API 支持节点注册、心跳、Peer 查询、ACL、密钥接口；支持可选 Bearer Token | 尚未实现 gRPC/proto、持久化数据库、OIDC/mTLS、实时 ACL 下发 |
| Dashboard 后端 | REST API、bcrypt 密码、可配置 CORS、启动期弱配置拒绝 | 内存 token/内存数据存储，尚无前端 Dashboard 和持久化 |
| 桌面端 | 连接/断开服务器、创建/删除配置、启动/停止单隧道、状态刷新 | 仍需更完整的错误可视化、Relay/STUN 配置持久化、P2P 操作流 |
| P2P/TUN | NAT/STUN/ICE/WireGuard/Mesh 测试原型 | `netTun` 非真实 OS TUN，Peer IP 规划/IPAM/路由未落地 |
| 调度器 | scheduler/relay/migration setter 已替换为强类型接口 | `Session.quicT/prober` 仍是未接入占位，路径切换闭环未完成 |
| 质量门禁 | Go 版本统一到 1.25.0；Go 测试范围避免扫描 `frontend/node_modules`；前端补 `test` 脚本 | 当前机器无可用 `npm`，已用本地 `vue-tsc`/ESLint CLI 完成等价验证 |

---

## 近期 TODO

| 优先级 | 任务 | 验收口径 |
|:---:|:---|:---|
| P0 | Relay/Control/Dashboard 安全基线 | Relay 必须配置认证令牌；Dashboard 禁止默认弱密码；控制面生产环境启用 Bearer Token/mTLS/OIDC |
| P0 | 质量门禁稳定化 | `make test-go`、CI Go 测试、前端类型检查在标准开发环境稳定通过 |
| P0 | Control Plane 最小持久化 | Store 接口接入 SQLite 或 PostgreSQL，MemoryStore 仅用于测试 |
| P1 | QUIC Relay E2E | 客户端可通过 QUIC work stream 完成一条真实隧道 E2E |
| P1 | P2P/TUN 架构落地 | 引入真实 TUN 抽象或 userspace netstack MVP，明确平台边界 |
| P1 | 调度器闭环 | 调度器能真实触发 TCP Relay/QUIC Relay/P2P 路径切换并有集成测试 |
| P2 | Dashboard 持久化与权限 | 用户、token、节点、ACL、告警持久化；补 RBAC 测试 |

---

## 验证记录

| 日期 | 验证项 | 结果 | 备注 |
|:---:|:---|:---:|:---|
| 2026-06-04 | `go test ./...`（server） | ✅ 通过 | 包含 controlplane、dashboard、relay 等 |
| 2026-06-04 | `go test ./...`（pkg） | ✅ 通过 | protocol、crypto 通过 |
| 2026-06-04 | `go test`（desktop，过滤 `frontend/node_modules`，工作区 GOCACHE） | ✅ 通过 | 默认 GOCACHE 受沙箱限制，已改用工作区缓存验证 |
| 2026-06-04 | `node node_modules/vue-tsc/bin/vue-tsc.js --noEmit` | ✅ 通过 | 等价于当前 `npm run test` 脚本；普通沙箱无法启动 node，已提权运行本地 CLI |
| 2026-06-04 | `node node_modules/eslint/bin/eslint.js . --ext .vue,.js,.jsx,.ts,.tsx` | ✅ 通过 | 0 error，仍有 Vue 模板格式 warning |

---

## 进度更新日志

| 日期 | 更新内容 | 操作人 |
|:---:|:---|:---:|
| 2026-06-04 | 核心代码与架构审查后修正进度口径：Phase 1 为 MVP 可用，Phase 2-4 为原型/概念验证；补质量门禁、安全基线、控制面 HTTP API、Relay QUIC work stream、桌面端连接/启停流程 | Codex |
| 2026-06-03 | 原记录显示 Phase 3/4 全完成，经代码审查后调整为“模块原型与单元测试完成”，不再作为生产验收完成依据 | - |
| 2026-06-01 | 原记录显示 Phase 2 全完成，经代码审查后调整为“P2P 核心链路原型完成”，真实 TUN/IPAM/路由仍待实现 | - |
| 2026-05-29 | Phase 1 基础 TCP Relay 原型、SQLite 配置、桌面基础 UI 与集成测试形成 MVP 基线 | - |
