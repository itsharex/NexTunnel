# NexTunnel 项目进度跟踪

> **最后更新时间**：2026-06-09（服务端 Release 包部署与 1Panel 教程补齐）
> **当前口径**：按"可验收 MVP / 原型验证 / 占位设计 / 未接入生产链路"统计，避免把单元测试通过等同于生产完成。
> **本次更新重点**：确认 Phase 4 迭代 1/2/3 已完成；服务端一键部署切换为 Release 包安装模式，并补齐 1Panel 部署配置教程。

---

## 总体进度概览

```
Phase 1 [基础隧道]  █████████████████░░░  85%  MVP+安全基线+持久化已就绪
Phase 2 [P2P直连]   ██████████░░░░░░░░░░  50%  TUN抽象+QUIC E2E就绪，IPAM/路由待完善
Phase 3 [智能调度]  █████████░░░░░░░░░░░  45%  调度器数据面闭环已验证，三接口驱动真实切换
Phase 4 [全球加速]  ██████████████░░░░░░  70%  迭代1+2+3开发完成，待Linux生产基准和真实部署联调
─────────────────────────────────────────────
总体进度            █████████████░░░░░░░  63%
```

---

## 阶段里程碑状态

| 阶段 | 名称 | 当前状态 | 完成度 | 说明 |
|:---:|:---|:---:|:---:|:---|
| Phase 1 | 基础隧道 | 🔄 MVP+安全基线 | 85% | TCP Relay、客户端配置、SQLite、重连和桌面基础流程可用；安全配置、持久化已就绪 |
| Phase 2 | P2P 直连 | 🔬 原型+TUN抽象 | 50% | NAT/STUN/ICE/WireGuard 编排有测试；TUNDevice 接口抽象完成，QUIC E2E 就绪；IPAM/路由待实现 |
| Phase 3 | 智能调度 | 🔄 数据面闭环 | 45% | 调度器+RelayManager+Migrator 三接口驱动真实路径切换已验证；QUIC work stream E2E 就绪 |
| Phase 4 | 全球加速 | 🔄 迭代3开发完成 | 70% | 迭代1(安全+持久化+CI)+迭代2(QUIC E2E+TUN抽象+调度闭环+Edge↔CP)+迭代3(Dashboard持久化+GeoIP真实加载+网络场景测试+eBPF XDP+Dashboard前端)已完成；服务端 Release 包部署脚本和 1Panel 教程已补齐，待 Linux 生产基准和真实部署联调 |

---

## 当前已落地事实

| 领域 | 已实现 | 当前限制 |
|:---|:---|:---|
| Relay TCP 数据面 | 控制连接、工作连接、代理监听、会话桥接、基础统计 | 生产部署仍需完整认证策略、限流、审计、配置管理 |
| Relay QUIC 数据面 | QUIC listener 可启动，work stream 握手后接入既有会话桥接 | 客户端 QUIC Relay 工作连接仍需完整接入和证书信任配置 |
| 控制面 | HTTP API 支持节点注册、心跳、Peer 查询、ACL、密钥接口；支持可选 Bearer Token | 尚未实现 gRPC/proto、持久化数据库、OIDC/mTLS、实时 ACL 下发 |
| Dashboard 后端/前端 | REST API、bcrypt 密码、可配置 CORS、启动期弱配置拒绝、SQLiteDashboardStore、Vue Dashboard 登录/节点/流量/ACL/告警界面 | 默认启动链路仍需配置化接入持久化存储，生产环境需补 RBAC 策略和端到端部署联调 |
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
| P4-T03-1 eBPF 程序加载器 | ✅ 开发完成 | `loader_linux.go` — 接入 `github.com/cilium/ebpf`，支持加载 XDP ELF 对象、`link.AttachXDP` 挂载、`RequireKernelMode` 强制内核态和失败降级 |
| P4-T03-2 XDP 转发程序 | ✅ 开发完成 | `xdp_forwarder.c` + `kernel_rules.go` — 实现 IPv4 TCP/UDP 目标端口级 PASS/DROP/REDIRECT fast path，复杂 CIDR/源端口/优先级遮挡规则保留用户态 |
| P4-T03-3 优雅降级机制 | ✅ 已实现 | `loader_other.go`（非 Linux）+ Linux 内核态加载失败自动回落 ModeUserspace；强制内核态配置下返回明确错误 |
| P4-T03-4 性能监控 | ✅ 开发完成 | StartStats 定期日志、用户态原子计数与 `xdp_stats_map` 内核统计合并读取 |
| P4-T03-x 单元测试 | ✅ 已实现 | `loader_test.go` + `kernel_rules_test.go` 覆盖创建、加载、模式切换、统计、XDP L4 规则编码和优先级遮挡 |

**结论**：eBPF XDP 内核态加载路径、L4 fast path 规则同步、用户态降级和统计读取已完成开发；当前 Windows 环境无法生成/挂载 BPF 对象，生产验收需在 Linux 节点用 clang/llc 编译 `xdp_forwarder.c` 为 `xdp_forwarder_bpfel.o` 并完成真实网卡基准测试。

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
| P4-T05-x Web 前端 | ✅ 开发完成 | `server/web/src/App.vue` 接入登录、健康检查、节点地图、流量图表、节点表、ACL 新增/删除、告警确认；拆分 `api.ts`、`types.ts`、`formatters.ts` |
| P4-T05-y 单元测试 | ✅ 已实现 | `server_test.go`（246 行）覆盖登录、节点/ACL/告警 CRUD、CORS、认证中间件 |

**结论**：后端 API 完整，认证、节点、ACL、统计、告警规则引擎、Webhook 通知、指标摄入端点全部可用；前端 Dashboard 已从静态原型升级为可操作控制台。默认启动链路仍需配置化接入 SQLiteDashboardStore，并在真实部署中验证 CORS、反向代理和 token 刷新策略。

---

## Phase 4 迭代开发计划（2026-06-08 启动）

基于 Phase 3 验收通过和 Phase 4 当前模块状态，制定以下迭代路线：

> **完成确认（2026-06-09）**：迭代 1「生产基线加固」、迭代 2「数据面闭环」、迭代 3「生产能力完善」均已完成开发验收；I3-02 eBPF XDP 和 I3-04 Dashboard 前端已在本次补齐，剩余为 Linux 生产基准、真实网卡挂载和 Dashboard 端到端部署联调。

### 迭代 1：生产基线加固（P0，预计 2 周）

| 编号 | 任务 | 验收标准 | 涉及模块 |
|:---:|:---|:---|:---|
| I1-01 | Control Plane 持久化 | Store 接口接入 SQLite 或 PostgreSQL；MemoryStore 仅用于测试 | `server/internal/controlplane/` |
| I1-02 | Relay 安全基线 | 控制/工作连接必须配置认证 token；生产环境禁止空 token | `server/internal/relay/` |
| I1-03 | Dashboard 安全加固 | 启动期拒绝弱密码和空 secret；密码 bcrypt 存储；CORS 白名单 | `server/internal/dashboard/` |
| I1-04 | 质量门禁 CI 化 | `go test ./...`、`go vet ./...`、`go build ./...` 在 CI Pipeline 稳定通过 | `.github/workflows/` |

### 迭代 2：数据面闭环（P1，预计 3 周）

| 编号 | 任务 | 验收标准 | 涉及模块 |
|:---:|:---|:---|:---|
| I2-01 | QUIC Relay E2E | 客户端通过 QUIC work stream 完成一条 TCP 隧道端到端转发 | `desktop/internal/quic/` + `server/internal/relay/` |
| I2-02 | P2P/TUN 平台抽象 | 引入真实 TUN 接口或 userspace netstack MVP，明确 Win/Mac/Linux 实现计划 | `desktop/internal/p2p/` |
| I2-03 | 调度器数据面闭环 | PathManager/RelaySelector/MigrationController 能真实触发 TCP Relay/QUIC/P2P 路径切换，并有集成测试 | `desktop/internal/scheduler/` + `p2p/engine.go` |
| I2-04 | Edge↔CP 集成 | 边缘节点 Registry 回调接入 Control Plane HTTP API，实现自动注册/注销 | `server/internal/edge/` + `controlplane/` |

### 迭代 3：生产能力完善（P2，预计 4 周）

| 编号 | 任务 | 验收标准 | 涉及模块 |
|:---:|:---|:---|:---|
| I3-01 | Dashboard 持久化 | 用户/token/节点/ACL/告警接入 SQLite/PostgreSQL；补 RBAC 测试 | `server/internal/dashboard/` |
| I3-02 | eBPF XDP 内核态 | ✅ 开发完成：eBPF C 程序 + cilium/ebpf 加载 + XDP L4 fast path + 用户态降级；Linux 生产基准待部署环境执行 | `server/internal/ebpf/` |
| I3-03 | GeoIP 数据库 | MaxMindAdapter 加载真实 .mmdb 文件，替换静态映射 | `server/internal/anycast/` |
| I3-04 | Dashboard 前端 | ✅ 开发完成：App.vue 接入节点地图、流量图表、节点表、ACL 编辑表单、告警确认和 API 客户端 | `server/web/` |
| I3-05 | 网络场景测试 | localhost 双节点 Docker、Relay 降级、连接中断重连、QUIC 证书失败路径 | 测试套件 |

---

## 近期 TODO

| 优先级 | 任务 | 验收口径 |
|:---:|:---|:---|
| P0 | 标准质量门禁固化 | 将本地已通过的 Go/前端检查纳入统一脚本或 CI，避免依赖手工命令 |
| P0 | 服务端 Release 首发验证 | 推送 `v*` tag 生成服务端发布包，在 Linux 服务器用 `install.sh --version` 完成 systemd 启动、健康检查和客户端最小连接 |
| P0 | Dashboard 持久化启动链路 | 默认 Server 配置可选择 SQLiteDashboardStore，MemoryStore 仅用于测试/开发 |
| P1 | eBPF Linux 生产验证 | 在 Linux 节点编译 `xdp_forwarder.c` 为 BPF 对象，挂载真实网卡并完成吞吐/延迟基准 |
| P1 | Dashboard 端到端部署联调 | 连接真实 Dashboard API，验证反向代理、CORS、token 刷新、错误提示和移动端布局 |
| P1 | 多地域边缘部署演练 | 用真实或容器化多节点验证 Edge 注册、Anycast/GeoIP 路由、故障切换和观测指标 |
| P2 | P2P/TUN 生产化 | 完成真实 OS TUN、IPAM、路由下发和跨平台权限边界设计 |

---

## 验证记录

| 日期 | 验证项 | 结果 | 备注 |
|:---:|:---|:---:|:---|
| 2026-06-09 | Phase 4 I3-02/I3-04 补齐验证 | ✅ 通过 | `go mod tidy`；`go test ./internal/ebpf ./internal/dashboard ./internal/anycast`；`GOOS=linux GOARCH=amd64 go test -c ./internal/ebpf`；`vue-tsc --noEmit`；ESLint；Vite production build。Windows 环境缺 `clang/llc`，未生成/挂载 BPF 对象 |
| 2026-06-09 | 服务端 Release 包部署脚本验证 | 🔄 部分通过 | `install.ps1` 语法解析通过；`install.ps1 -Action config -NonInteractive` 使用工作区临时目录生成 `server.env` 和连接信息通过；本机 Docker/WSL/systemd 不可用，`install.sh` 的下载、systemd 启动和健康检查需在 Linux 服务器验证 |
| 2026-06-08 | Phase 4 迭代3 生产能力完善 | ✅ 通过 | Dashboard SQLite持久化(5测试)+GeoIP真实加载(23测试)+网络场景(7测试)=35新测试全PASS。全量回归desktop(11)+server(8)+pkg(2)零失败。go vet+build零警告 |
| 2026-06-08 | Phase 4 迭代2 数据面闭环 | ✅ 通过 | QUIC E2E 1测试+TUN抽象 3测试+调度器闭环 1测试+Edge↔CP 6测试=11新测试全PASS。全量回归desktop(11)+server(8)+pkg(2)零失败。go vet+build零警告 |
| 2026-06-08 | Phase 4 迭代1 生产基线加固 | ✅ 通过 | SQLite Store 5测试+Relay安全 5测试+Dashboard安全 4测试=14新测试全PASS。全量回归desktop(11)+server(8)+pkg(2)零失败。go vet+build零警告。CI新增desktop build+go vet步骤 |
| 2026-06-08 | Phase 3 系统性测试验证 | ✅ 通过 | 6模块37个单元测试全PASS，全量回归零失败，go vet零警告，go build三模块成功。修复edge模块flaky test（改用原子安全的ConsecutiveOKs，新增DirectProbe确定性测试） |
| 2026-06-04 | `go test ./...`（server） | ✅ 通过 | 包含 controlplane、dashboard、relay 等 |
| 2026-06-04 | `go test ./...`（pkg） | ✅ 通过 | protocol、crypto 通过 |
| 2026-06-04 | `go test`（desktop，过滤 `frontend/node_modules`，工作区 GOCACHE） | ✅ 通过 | 默认 GOCACHE 受沙箱限制，已改用工作区缓存验证 |
| 2026-06-04 | `node node_modules/vue-tsc/bin/vue-tsc.js --noEmit` | ✅ 通过 | 等价于当前 `npm run test` 脚本；普通沙箱无法启动 node，已提权运行本地 CLI |
| 2026-06-04 | `node node_modules/eslint/bin/eslint.js . --ext .vue,.js,.jsx,.ts,.tsx` | ✅ 通过 | 0 error，仍有 Vue 模板格式 warning |

---

## 进度更新日志

| 日期 | 更新内容 | 操作人 |
|:---:|:---|:---:|
| 2026-06-09 | **服务端 Release 包部署与 1Panel 教程补齐**：将 `deploy/server/install.sh` 从 Docker Compose 源码构建改为 GitHub Release 包下载、SHA256 可选校验、二进制安装、`/etc/nextunnel/server.env` 配置生成和 systemd 三服务管理；`install.ps1` 改为下载 Windows Release 包并以 PID/日志管理本地进程；`.env.example` 增加 `NEXTUNNEL_PACKAGE_URL`、`NEXTUNNEL_VERSION`、`NEXTUNNEL_RELEASE_BASE_URL`、`NEXTUNNEL_PACKAGE_SHA256` 等多源配置；`docker-compose.yml` 去除源码 build，保留为后续官方镜像和 1Panel 容器编排模板；新增 `.github/workflows/release.yml` 在 `v*` tag 生成服务端 linux-amd64、linux-arm64、windows-amd64 发布包和校验文件；README 补齐命令行、环境变量、配置文件、离线包、1Panel 终端/文件/计划任务/容器编排多种部署方式 | Agent |
| 2026-06-09 | **服务端一键部署能力补齐**：新增 `deploy/server/` 生产部署目录，提供 Linux `install.sh`、Windows/PowerShell `install.ps1`、`.env.example`、Docker Compose 编排和部署说明；支持 install/up/down/restart/status/logs/health/update/uninstall，多方式配置 Relay、Control Plane、NAT Detector。修正服务端 Dockerfile 和根 Compose 构建上下文，保证 `server` 依赖 `../pkg` 时可在干净环境构建；公网 Relay 默认强制认证 token | Agent |
| 2026-06-09 | **Phase 4 剩余开发任务补齐**：I3-02 eBPF XDP 内核态转发完成（cilium/ebpf 加载、XDP attach、L4 fast path 规则编码、优先级遮挡保护、内核统计读取、用户态降级、xdp_forwarder.c）。I3-04 Dashboard 前端完成（登录、健康检查、节点地图、流量图表、节点表、ACL 新增/删除、告警确认、API/types/formatters 模块拆分）。验证通过 Go 测试、Linux 目标编译、vue-tsc、ESLint、Vite build；Linux 真实网卡基准待部署环境执行 | Agent |
| 2026-06-08 | **Phase 4 迭代3 生产能力完善完成**：I3-01 Dashboard持久化(SQLiteDashboardStore接口+实现，Node/ACL/Alert/User CRUD，5测试PASS)。I3-03 GeoIP数据库(MaxMindAdapter接入oschwald/maxminddb-golang纯Go库，真实.mdb文件加载+CIDR匹配+回退映射，23测试PASS)。I3-05 网络场景测试(QUIC证书失败+TCP不可达+Relay降级+重连，7测试PASS)。新增文件：store_sqlite.go、store_sqlite_test.go、network_scenario_test.go。修改文件：geoip.go、go.mod。全量回归零失败 | Agent |
| 2026-06-08 | **Phase 4 迭代2 数据面闭环完成**：I2-01 QUIC Relay E2E(WorkConnOpener接口+QUICWorkConnOpener实现+QUIC stream转发，E2E测试PASS)。I2-02 P2P/TUN抽象(TUNDevice接口+TUNConfig+PlatformCapabilities+netTun实现ReadPacket/WritePacket，3测试PASS)。I2-03 调度器数据面闭环(PathManager+RelaySelector+MigrationController三接口驱动真实路径切换，集成测试PASS)。I2-04 Edge↔CP集成(ControlPlaneClient+Registry回调自动注册/注销+HeartbeatLoop，6测试PASS)。修复Manager日志初始化nil writer bug。新增文件：workconn.go、workconn_quic.go、tun.go、tun_test.go、quic_relay_e2e_test.go、scheduler_integration_test.go。修改文件：manager.go、tunnel.go、net_tun.go。全量回归零失败 | Agent |
| 2026-06-08 | **Phase 4 迭代1 生产基线加固完成**：I1-01 Control Plane SQLite持久化(store_sqlite.go+factory+config 5个新测试全PASS)。I1-02 Relay安全基线(非localhost部署强制要求auth-token，5个配置验证测试全PASS)。I1-03 Dashboard安全加固验证(bcrypt密码+弱密码拒绝+空密钥拒绝+CORS白名单，4个安全测试全PASS)。I1-04 CI质量门禁(添加desktop Go构建+go vet全三模块)。全量回归desktop(11)+server(8)+pkg(2)全PASS，go vet零警告，go build成功。新增文件：store_sqlite.go、store_factory.go、store_sqlite_test.go。修改文件：config.go、main.go(control-plane+relay)、server_test.go(relay)、server_test.go(dashboard)、ci.yml | Agent |
| 2026-06-08 | **Phase 3 系统性验收 + Phase 4 迭代计划启动**：逐模块运行Phase 3全部6个功能模块测试（QUIC 5+Probe 6+Scheduler 9+Relay 4+Migration 5+ControlPlane 8=37个单元测试全PASS）。全量回归desktop(11模块)+server(7模块)+pkg(2模块)零失败。go vet三模块零警告，go build三模块成功。集成验证：协议7种Phase 3消息类型完整，Engine强类型接口(PathManager/RelaySelector/MigrationController)集成正常，P2P全链路(ICE+Punch+WireGuard)测试PASS。UAT验收对照task-plan.md需求全覆盖。修复edge HealthChecker时序flaky test。更新进度跟踪文档，制定Phase 4迭代开发计划 | Agent |
| 2026-06-04 | **Phase 4 实现迭代**：新增 SD-WAN QoS 队列+令牌桶（qos.go）、Dashboard AlertEngine+Webhook 通知（alerts.go）、Edge ControlPlane 注册/心跳集成（controlplane.go）、Anycast GeoIPProvider 接口+MaxMind 适配器（geoip.go）、eBPF 转发规则+用户态转发（forwarder.go）；全部模块新增单元测试 | Agent |
| 2026-06-04 | **Phase 4 专项迭代**：逐模块校验 edge/anycast/ebpf/sdwan/dashboard 代码实现状态，新增 Phase 4 详细子任务跟踪表；近期 TODO 增加 Edge↔Control Plane 集成、eBPF XDP 真实转发、SD-WAN QoS 队列、Anycast GeoIP、Dashboard 前端 5 项 P2 任务 | Agent |
| 2026-06-04 | 核心代码与架构审查后修正进度口径：Phase 1 为 MVP 可用，Phase 2-4 为原型/概念验证；补质量门禁、安全基线、控制面 HTTP API、Relay QUIC work stream、桌面端连接/启停流程 | Codex |
| 2026-06-03 | 原记录显示 Phase 3/4 全完成，经代码审查后调整为“模块原型与单元测试完成”，不再作为生产验收完成依据 | - |
| 2026-06-01 | 原记录显示 Phase 2 全完成，经代码审查后调整为“P2P 核心链路原型完成”，真实 TUN/IPAM/路由仍待实现 | - |
| 2026-05-29 | Phase 1 基础 TCP Relay 原型、SQLite 配置、桌面基础 UI 与集成测试形成 MVP 基线 | - |
