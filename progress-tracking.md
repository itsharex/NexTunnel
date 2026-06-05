# NexTunnel 项目进度跟踪

> **最后更新时间**：2026-06-04（第三次迭代）  
> **当前口径**：按"可验收 MVP / 原型验证 / 占位设计 / 未接入生产链路"统计，避免把单元测试通过等同于生产完成。  
> **本次更新重点**：Phase 4 全部 5 个功能模块（边缘节点、Anycast、eBPF、SD-WAN、Dashboard）逐项校验并纳入详细跟踪。

---

## 总体进度概览

```
Phase 1 [基础隧道]  ███████████████░░░░░  75%  MVP 可用，仍需强化 E2E 与运维配置
Phase 2 [P2P直连]   ████████░░░░░░░░░░░░  40%  原型验证，真实 TUN/IPAM/路由未完成
Phase 3 [智能调度]  ███████░░░░░░░░░░░░░  35%  调度模块存在，数据面闭环仍在接入
Phase 4 [全球加速]  █████░░░░░░░░░░░░░░░  28%  核心功能实现，5 大模块均具生产能力雏形
─────────────────────────────────────────────
总体进度            █████████░░░░░░░░░░░  45%
```

---

## 阶段里程碑状态

| 阶段 | 名称 | 当前状态 | 完成度 | 说明 |
|:---:|:---|:---:|:---:|:---|
| Phase 1 | 基础隧道 | 🔄 MVP 可用 | 75% | TCP Relay、客户端配置、SQLite、重连和桌面基础流程可用；安全配置、端到端覆盖、运维参数仍需补齐 |
| Phase 2 | P2P 直连 | 🔬 原型验证 | 40% | NAT/STUN/ICE/WireGuard 编排有测试；`netTun` 仍是 userspace 通道，缺少真实系统 TUN、IPAM、路由管理 |
| Phase 3 | 智能调度 | 🔬 原型验证 | 35% | QUIC、探测、调度、迁移模块有单元测试；调度器尚未完整控制 TCP/QUIC/P2P/Relay 数据面切换 |
| Phase 4 | 全球加速 | 🛠️ 核心实现 | 28% | 5 大模块核心功能已实现：QoS 队列+令牌桶、AlertEngine、Edge-CP 集成、GeoIP 接口、eBPF 转发规则；仍缺真实 XDP 程序和 GeoIP 数据库 |

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

## Phase 4 模块详细状态（本次迭代重点）

### P4-T01 边缘节点部署（`server/internal/edge/`）

| 子任务 | 状态 | 说明 |
|:---|:---:|:---|
| P4-T01-1 边缘节点数据模型（EdgeNode/Region/Status） | ✅ 已实现 | `node.go` — EdgeNode 结构体、Region/Status 枚举、原子状态管理、失败计数 |
| P4-T01-2 边缘节点注册表（Registry CRUD + 按区域索引） | ✅ 已实现 | `registry.go` — Register/Deregister/Get/List/ListByRegion/ListHealthy/CountHealthy/Regions，带回调 |
| P4-T01-3 健康检查器（心跳探测、自动摘除） | ✅ 已实现 | `health.go` — TCP 探测、并发检查、连续失败阈值判定、自动摘除与恢复、超时自动注销 |
| P4-T01-4 部署自动化（Docker Compose 模板 + 部署清单） | ✅ 已实现 | `deploy.go` — Go template 生成 Docker Compose、DeployManifest 部署指引 |
| P4-T01-5 Control Plane 集成（边缘节点自动注册） | ✅ 已实现 | `controlplane.go` — ControlPlaneClient HTTP 注册/心跳、ConnectRegistry 回调接入、StartHeartbeatLoop/StopAll |
| P4-T01-x 单元测试 | ✅ 已实现 | `node_test.go`（362 行）+ `controlplane_test.go`（223 行）覆盖 EdgeNode 状态机、Registry CRUD、健康检查、CP 注册/心跳全流程 |

**结论**：后端骨架完整，核心数据模型、注册表、健康检查、部署模板、Control Plane 集成均已到位；缺真实多地域部署验证和 GeoIP 数据库接入。

### P4-T02 Anycast 路由（`server/internal/anycast/`）

| 子任务 | 状态 | 说明 |
|:---|:---:|:---|
| P4-T02-1 Anycast 路由器核心逻辑 | ✅ 已实现 | `router.go` — Haversine 距离计算、SelectNearest/SelectNearestWithFailover/SelectByRegion |
| P4-T02-2 GeoDNS 解析引导 | ✅ 已实现 | `router.go` + `geoip.go` — GeoIPProvider 接口、MaxMindAdapter（CIDR 匹配 + 静态映射）、GeoIPRouter 封装 |
| P4-T02-3 故障自动切换 | ✅ 已实现 | SelectNearestWithFailover + GeoIPRouter.SelectNearestWithFailoverForIP 支持多节点故障转移 |
| P4-T02-x 单元测试 | ✅ 已实现 | `router_test.go`（220 行）覆盖距离计算、故障转移、GeoDNS 解析 |

**结论**：路由核心逻辑和 GeoDNS 原型完整，GeoIPProvider 接口已抽象、MaxMindAdapter 支持 CIDR 匹配；生产环境需加载真实 .mmdb 数据库文件。

### P4-T03 eBPF 网络加速（`server/internal/ebpf/`）

| 子任务 | 状态 | 说明 |
|:---|:---:|:---|
| P4-T03-1 eBPF 程序加载器 | 🟡 骨架 | `loader_linux.go` — Loader 结构体、Load/Unload/GetMode 框架，但 Load() 内为模拟逻辑，无真实 BPF 字节码加载 |
| P4-T03-2 XDP 转发程序 | 🟡 框架 | `forwarder.go` — RuleMap + UserspaceForwarder + ProcessPacket 完整实现；内核态 XDP 仍待接入 cilium/ebpf |
| P4-T03-3 优雅降级机制 | ✅ 已实现 | `loader_other.go`（非 Linux）+ linux Loader 默认 ModeUserspace，条件编译 `//go:build linux` |
| P4-T03-4 性能监控 | 🟡 骨架 | StartStats 定期日志输出、RecordForward/RecordDrop 原子计数，但无真实内核态数据来源 |
| P4-T03-x 单元测试 | ✅ 已实现 | `loader_test.go`（87 行）覆盖创建、加载、模式切换、统计 |

**结论**：架构骨架、条件编译和降级机制正确；新增 RuleMap 转发规则和 UserspaceForwarder 可在用户态处理数据包；内核态 XDP 仍待实现。

### P4-T04 SD-WAN 流量策略（`server/internal/sdwan/`）

| 子任务 | 状态 | 说明 |
|:---|:---:|:---|
| P4-T04-1 流量分类器 | ✅ 已实现 | `classifier.go` — Classifier 基于端口/协议映射 AppType（HTTP/SSH/RDP/DNS 等） |
| P4-T04-2 策略引擎 | ✅ 已实现 | `policy.go` — PolicyEngine 规则 CRUD、优先级排序、Evaluate 匹配（AppType/Protocol/SrcNode/DstPort） |
| P4-T04-3 QoS 管理器 | ✅ 已实现 | `qos.go` — 8 级 heap 优先级队列、TokenBucket 令牌桶限速、QoSManager 入队/出队/批量/统计 |
| P4-T04-4 策略热更新 | ✅ 已实现 | UpdateRule 原子替换 + rebuildSorted，规则修改即时生效 |
| P4-T04-x 单元测试 | ✅ 已实现 | `policy_test.go`（227 行）覆盖规则增删改、优先级匹配、Evaluate 全流程 |

**结论**：策略引擎、分类器、QoS 队列和令牌桶限速全部完成；规则 CRUD、匹配、优先级队列、流量整形均已实现。

### P4-T05 管理控制台（`server/internal/dashboard/` + `server/web/`）

| 子任务 | 状态 | 说明 |
|:---|:---:|:---|
| P4-T05-1 HTTP API 服务器 | ✅ 已实现 | `server.go` — 路由注册、CORS 白名单中间件、JSON 响应封装、优雅启停 |
| P4-T05-2 管理员认证 | ✅ 已实现 | `auth.go` — JWT 签发/验证、bcrypt 密码、Login/Refresh/角色、弱密码拒绝 |
| P4-T05-3 节点管理 API | ✅ 已实现 | GET/GET-by-ID/DELETE /api/v1/nodes |
| P4-T05-4 流量统计 API | ✅ 已实现 | GET /api/v1/stats、GET /api/v1/stats/{node_id} |
| P4-T05-5 ACL 管理 API | ✅ 已实现 | GET/POST/DELETE /api/v1/acl |
| P4-T05-6 告警通知 API | ✅ 已实现 | `alerts.go` — AlertEngine 规则引擎、Evaluate/冷却/节点过滤、LogNotifier + WebhookNotifier、POST /api/v1/metrics 指标摄入、POST /api/v1/alert-rules CRUD |
| P4-T05-x Web 前端 | 🟡 骨架 | `server/web/src/App.vue`（1051 行）已有单文件组件；`main.ts` 入口存在 |
| P4-T05-y 单元测试 | ✅ 已实现 | `server_test.go`（246 行）覆盖登录、节点/ACL/告警 CRUD、CORS、认证中间件 |

**结论**：后端 API 完整，认证、节点、ACL、统计、告警规则引擎、Webhook 通知、指标摄入端点全部可用；DataStore 仍为内存实现，需接入持久化；前端 App.vue 已有骨架但需完善。

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
| P1 | Edge ↔ Control Plane 集成 | 边缘节点 Registry 回调接入 Control Plane HTTP API，实现自动注册/注销 |
| P2 | Dashboard 持久化与权限 | 用户、token、节点、ACL、告警持久化；补 RBAC 测试 |
| P2 | eBPF XDP 内核态接入 | 编写 eBPF C 程序 + cilium/ebpf 加载，替换 Load() 模拟逻辑，Linux 环境基准测试 |
| P2 | GeoIP 数据库加载 | MaxMindAdapter 加载真实 .mmdb 文件，替换静态映射 |
| P2 | Dashboard 前端完善 | App.vue 完善节点地图、流量图表、ACL 编辑表单等交互 |

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
| 2026-06-04 | **Phase 4 实现迭代**：新增 SD-WAN QoS 队列+令牌桶（qos.go）、Dashboard AlertEngine+Webhook 通知（alerts.go）、Edge ControlPlane 注册/心跳集成（controlplane.go）、Anycast GeoIPProvider 接口+MaxMind 适配器（geoip.go）、eBPF 转发规则+用户态转发（forwarder.go）；全部模块新增单元测试 | Agent |
| 2026-06-04 | **Phase 4 专项迭代**：逐模块校验 edge/anycast/ebpf/sdwan/dashboard 代码实现状态，新增 Phase 4 详细子任务跟踪表；近期 TODO 增加 Edge↔Control Plane 集成、eBPF XDP 真实转发、SD-WAN QoS 队列、Anycast GeoIP、Dashboard 前端 5 项 P2 任务 | Agent |
| 2026-06-04 | 核心代码与架构审查后修正进度口径：Phase 1 为 MVP 可用，Phase 2-4 为原型/概念验证；补质量门禁、安全基线、控制面 HTTP API、Relay QUIC work stream、桌面端连接/启停流程 | Codex |
| 2026-06-03 | 原记录显示 Phase 3/4 全完成，经代码审查后调整为“模块原型与单元测试完成”，不再作为生产验收完成依据 | - |
| 2026-06-01 | 原记录显示 Phase 2 全完成，经代码审查后调整为“P2P 核心链路原型完成”，真实 TUN/IPAM/路由仍待实现 | - |
| 2026-05-29 | Phase 1 基础 TCP Relay 原型、SQLite 配置、桌面基础 UI 与集成测试形成 MVP 基线 | - |
