# NexTunnel

NexTunnel 是一款**开源内网穿透 + P2P 直连**的现代化网络工具，提供可视化桌面管理界面。项目采用 Go + Vue 3 + Wails 技术栈构建，旨在超越传统 FRP/NPS 等"客户端→中转服务器"的 TCP 转发模式，打造下一代智能组网方案。

> **项目愿景**：让内网穿透从"能连上"进化为"智能直连"——用户无需理解端口、NAT、UDP、Tunnel 等底层概念，设备自动发现、自动组网、自动加速、自动直连。

---

## 核心特性

| 特性 | 说明 |
|------|------|
| **P2P 优先** | 目标能力：优先尝试直连传输；当前已具备 TUN 抽象、IPAM、路由下发和测试链路，真实 OS TUN 仍需权限验证 |
| **智能链路** | 目标能力：自动检测网络环境，选择最优传输路径；当前调度器数据面闭环已有集成测试覆盖 |
| **安全零信任** | 端到端加密与身份认证；支持 mTLS 双向认证、Bearer Token、bcrypt 密码存储、CORS 白名单 |
| **自动降级** | P2P 不可达时自动切换至中继，保证连通性 |
| **可视化桌面端** | 基于 Wails 的原生桌面应用，支持 Relay 连接、隧道配置与单隧道启停 |
| **跨平台** | 覆盖 Windows / macOS / Linux |

---

## 技术栈

| 组件 | 技术 |
|------|------|
| 客户端核心 | Go 1.25 |
| 桌面框架 | Wails v2 |
| 前端 | Vue 3 + Vite + TypeScript |
| 状态管理 | Pinia |
| 本地存储 | SQLite（modernc.org/sqlite，纯 Go 实现） |
| STUN | pion/stun v2 |
| WireGuard | wireguard-go |
| 中继传输 | QUIC |
| 服务端 | Go + Docker |
| CI/CD | GitHub Actions |

---

## 当前实现状态

当前仓库更准确的定位是：**核心开发链路已补齐，具备可验收 MVP 与多项生产化能力；仍需在真实部署环境完成 eBPF、TUN、反向代理和多地域演练验证**。

| 领域 | 当前状态 | 生产化缺口 |
|------|------|------|
| 基础 Relay | TCP Relay、工作连接、会话桥接、流量统计可用；Relay 已支持共享 token 验证和 mTLS | 仍需生产级限流、配置下发和运维监控联调 |
| QUIC Relay | 服务端 QUIC listener 与 work stream 会话交付已接入；客户端 QUIC work opener 已接入 TCP tunnel E2E，默认 TLS 1.3 且不跳过证书校验 | 仍需生产证书信任链部署、超时/限流策略和真实网络压测 |
| Control Plane | HTTP API：节点注册、心跳、Peer 查询、ACL、密钥、自动 IPAM、节点路由下发；支持 Bearer Token + mTLS + SQLite 持久化恢复 + 审计日志 | 尚未实现 gRPC/proto、OIDC |
| P2P/TUN | NAT/STUN/ICE/WireGuard/Mesh 原型有测试覆盖；TUN 平台抽象层支持 Linux/macOS/Windows；控制面可下发虚拟 IP 和路由配置 | 真实 OS TUN 需 root/驱动权限验证，跨平台系统路由应用需实机验收 |
| 智能调度 | scheduler/relay/migration 已强类型接入 P2P Engine，路径切换闭环有集成测试 | 仍需真实环境多路径切换和弱网压测 |
| Dashboard | 后端 API、Vue 管理台、SQLite 持久化、RBAC、HTTPS、审计日志已接入 | 生产环境仍需 HTTPS 反向代理联调和端到端部署 |

---

## 项目结构

```
NexTunnel/
├── desktop/                          # Wails 桌面客户端（Go + Vue）
│   ├── main.go                       # 主程序入口
│   ├── app.go                        # Wails 应用入口
│   ├── frontend/                     # Vue 3 前端
│   │   ├── src/
│   │   │   ├── api/                  # Wails binding 调用
│   │   │   ├── components/           # UI 组件
│   │   │   ├── views/                # 页面视图
│   │   │   ├── stores/               # Pinia 状态管理
│   │   │   ├── App.vue               # 根组件
│   │   │   └── main.ts               # 前端入口
│   │   ├── wailsjs/                  # Wails Go↔JS 自动绑定
│   │   ├── package.json
│   │   ├── vite.config.ts
│   │   └── tsconfig.json
│   ├── internal/                     # 客户端内部模块
│   │   ├── auth/                     # 认证模块（Token 管理）
│   │   ├── config/                   # 本地配置持久化（SQLite）
│   │   ├── nat/                      # NAT 穿透（STUN 检测、类型识别）
│   │   ├── oidc/                     # OIDC 认证客户端
│   │   ├── p2p/                      # P2P 连接引擎（ICE、WireGuard、Mesh）
│   │   ├── relay/                    # Relay 中继客户端
│   │   ├── scheduler/                # 链路调度器
│   │   └── tunnel/                   # 隧道核心（TCP/HTTP Tunnel、重连）
│   ├── go.mod
│   └── wails.json
│
├── server/                           # 服务端
│   ├── cmd/
│   │   ├── control-plane/            # 控制面入口（节点管理、认证、ACL）
│   │   ├── relay/                    # Relay 中继服务入口
│   │   ├── nat-detector/             # NAT 检测服务入口
│   │   └── dashboard/                # Dashboard 管理控制台入口
│   ├── web/                          # Dashboard Vue 前端
│   ├── internal/
│   │   ├── controlplane/             # 控制面逻辑
│   │   ├── relay/                    # Relay 服务逻辑（会话管理、代理路由）
│   │   ├── natdetect/                # NAT 检测逻辑
│   │   └── common/                   # 公共组件
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
│
├── pkg/                              # 公共包（客户端与服务端共享）
│   ├── protocol/                     # 协议定义（消息编解码）
│   ├── crypto/                       # 加密工具（密钥管理）
│   ├── types/                        # 共享类型定义
│   └── go.mod
│
├── scripts/                          # 构建与部署脚本
├── docs/                             # 项目文档
├── .github/workflows/ci.yml          # CI 流水线配置
├── go.work                           # Go 工作区配置
├── docker-compose.yml                # 服务端容器编排
└── Makefile                          # 构建命令
```

---

## 架构设计

### 系统架构

```
  ┌──────────────┐                              ┌──────────────┐
  │  Client A    │                              │  Client B    │
  │ (Wails App)  │                              │ (Wails App)  │
  └──────┬───────┘                              └──────┬───────┘
         │                                             │
         │  ←─────── P2P Direct (WireGuard/QUIC) ───→ │
         │                                             │
         │         ┌─────────────────────┐             │
         ├────────→│   Control Plane     │←────────────┤
         │         │  - 节点注册/认证     │             │
         │         │  - 密钥交换          │             │
         │         │  - ACL/路由下发      │             │
         │         └─────────────────────┘             │
         │                                             │
         │         ┌─────────────────────┐             │
         ├────────→│   STUN/TURN 服务    │←────────────┤
         │         │  - NAT 类型检测      │             │
         │         │  - 打洞协助          │             │
         │         └─────────────────────┘             │
         │                                             │
         │         ┌─────────────────────┐             │
         └────────→│   Relay 中继节点    │←────────────┘
                   │  (QUIC Relay)       │
                   │  - 降级中继          │             │
                   └─────────────────────┘
```

### 客户端分层

```
┌─────────────────────────────────────────┐
│       Vue 3 + Vite (Desktop UI)         │  表现层
├─────────────────────────────────────────┤
│    Wails Bridge (Go ↔ Frontend IPC)     │  应用层
├─────────────────────────────────────────┤
│  Tunnel Manager │ P2P Engine │ Config   │  业务逻辑层
├─────────────────────────────────────────┤
│ WireGuard Tunnel │ ICE/STUN │ Link      │  P2P 引擎层
├─────────────────────────────────────────┤
│   QUIC Transport │ UDP │ TCP            │  传输层
├─────────────────────────────────────────┤
│          SQLite (Local Config)          │  存储层
└─────────────────────────────────────────┘
```

### 链路调度策略

```
检测 NAT 类型 → 评估 RTT/丢包 → 选择最优路径:
  ├── Full Cone NAT     → UDP P2P (直连)
  ├── Restricted NAT    → QUIC P2P (打洞)
  ├── Symmetric NAT     → TCP P2P (尝试)
  ├── P2P 全部失败      → Nearby Relay (就近中继)
  └── 全部不可用        → Global Relay (全球中继)
```

---

## 模块说明

### 桌面端（`desktop/`）

基于 **Wails v2** 构建的桌面应用，Go 后端 + Vue 3 前端通过 Wails Bridge 进行 IPC 通信。

- **`internal/tunnel`** — 隧道核心：TCP/HTTP 端口转发，支持多路复用与断线自动重连
- **`internal/p2p`** — P2P 引擎：ICE 协商、WireGuard 隧道、Mesh 组网
- **`internal/nat`** — NAT 穿透：STUN 绑定、NAT 类型检测（Full Cone / Restricted / Symmetric）
- **`internal/config`** — 配置管理：SQLite 持久化存储，支持数据库迁移
- **`internal/auth`** — 认证模块：Token 管理，支持过期与刷新
- **`internal/oidc`** — OIDC 客户端：对接第三方身份提供商
- **`internal/scheduler`** — 链路调度：基于网络指标动态选择最优链路
- **`internal/relay`** — Relay 客户端：P2P 失败时的降级中继路径

### 服务端（`server/`）

独立部署的四个服务进程：

| 服务 | 入口 | 端口 | 职责 |
|------|------|------|------|
| **Relay Server** | `cmd/relay/` | 7000/TCP、7443/QUIC | TCP/QUIC 中继转发，客户端注册与认证，流量统计 |
| **Control Plane** | `cmd/control-plane/` | 9090/HTTP | 节点管理、ACL 规则、密钥交换、Peer 查询 |
| **NAT Detector** | `cmd/nat-detector/` | 3478/UDP | NAT 类型检测服务，支持高并发请求 |
| **Dashboard** | `cmd/dashboard/` + `server/web/` | 8080/HTTP | Web 管理台、节点/流量/ACL/告警查看与操作，SQLite 持久化 |

### 公共包（`pkg/`）

客户端与服务端共享的基础库：

- **`protocol`** — 自定义协议的消息编解码
- **`crypto`** — 加密工具与密钥管理
- **`types`** — 项目共享类型定义

---

## 快速开始

### 环境要求

| 工具 | 版本 |
|------|------|
| Go | >= 1.25 |
| Node.js | >= 18 |
| Wails CLI | v2.x |
| Docker & Docker Compose | （服务端部署可选） |

### 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 克隆项目

```bash
git clone https://github.com/nextunnel/nextunnel.git
cd NexTunnel
```

### 安装依赖

```bash
make install-deps
```

或手动安装：

```bash
# Go 依赖
cd desktop && go mod tidy && cd ..
cd server  && go mod tidy && cd ..
cd pkg     && go mod tidy && cd ..

# 前端依赖
cd desktop/frontend && npm install && cd ../..
```

### 开发模式

启动 Wails 开发服务器（支持前端热更新）：

```bash
make dev
```

### 构建桌面应用

```bash
make build
```

构建产物位于 `desktop/build/bin/`。

### 构建服务端

```bash
make build-server
```

生成四个二进制文件至 `build/` 目录：
- `control-plane` — 控制面服务
- `relay-server` — 中继服务
- `nat-detector` — NAT 检测服务
- `dashboard` — Web 管理控制台

### 本地运行服务端

```bash
# Relay：非本地环境必须配置强随机 auth token
cd server
go run ./cmd/relay -bind 127.0.0.1 -control-port 7000 -quic-port 7443 -auth-token <strong-token>

# Control Plane：生产环境应配置 Bearer Token，可启用 mTLS
go run ./cmd/control-plane -listen 127.0.0.1:9090 -api-token <strong-token>

# Dashboard：生产环境必须配置强 secret 和管理员密码，可启用 HTTPS
go run ./cmd/dashboard -listen 127.0.0.1:8080 -secret-key <strong-secret> -admin-password <strong-password> -store-path ./data/dashboard.db -static-dir ./web/dist
```

### mTLS 双向认证（可选）

Control Plane 和 Relay 均支持 mTLS 双向证书认证，作为 Bearer Token 的安全升级选项：

```bash
# 使用 pkg/tlsutil 工具生成 CA 和证书，然后配置服务端：
go run ./cmd/control-plane -listen 0.0.0.0:9090 \
  -tls-ca /path/to/ca.pem -tls-cert /path/to/server.pem -tls-key /path/to/server-key.pem

go run ./cmd/relay -bind 0.0.0.0 -control-port 7000 \
  -tls-ca /path/to/ca.pem -tls-cert /path/to/server.pem -tls-key /path/to/server-key.pem
```

### Dashboard HTTPS 与 RBAC

Dashboard 支持 HTTPS 和基于角色的访问控制（admin/operator/viewer）：

```bash
go run ./cmd/dashboard -listen 0.0.0.0:8080 \
  -secret-key <secret> -admin-password <password> \
  -tls-cert /path/to/cert.pem -tls-key /path/to/key.pem \
  -audit-log /var/log/nextunnel/audit.jsonl
```

**RBAC 权限矩阵**：

| 资源 | admin | operator | viewer |
|------|:-----:|:--------:|:------:|
| 节点管理 | 读写删 | 读写删 | 只读 |
| ACL 规则 | 读写删 | 读写删 | 只读 |
| 告警 | 读写删 | 读写 | 只读 |
| 告警规则 | 读写删 | 只读 | 只读 |
| 用户管理 | 读写删 | — | — |
| 审计日志 | 只读 | — | — |

---

## 部署

### Docker Compose 部署服务端

```bash
docker-compose up -d
```

默认启动以下服务：

| 服务 | 端口 | 说明 |
|------|------|------|
| Relay Server | `7000` | 中继服务（控制端口） |
| Dashboard | `8080` | Web 管理控制台 |
| Control Plane | `9090` | 控制面 API |
| NAT Detector | `3478/UDP` | NAT 类型检测（STUN） |

如需自定义 NAT Detector 的 IP 地址，设置环境变量：

```bash
PRIMARY_IP=<主IP> ALT_IP=<备用IP> docker-compose up -d
```

### Docker 单独构建

```bash
cd server
docker build -t nextunnel-server .
```

---

## 常用命令

```bash
make help            # 查看所有可用命令
make dev             # 启动桌面开发服务器
make build           # 构建桌面应用
make build-server    # 构建服务端二进制
make test            # 运行所有测试（Go + 前端）
make test-go         # 运行 Go 测试
make test-frontend   # 运行前端测试
make lint            # 代码检查（Go + 前端）
make clean           # 清理构建产物
make install-deps    # 安装所有依赖
```

---

## 开发路线

| 阶段 | 目标 | 状态 |
|------|------|------|
| **Phase 1** | 基础隧道：TCP/HTTP Tunnel + Relay 中继 + 桌面 UI | MVP 可用，需继续生产化 |
| **Phase 2** | P2P 直连：UDP 打洞 + WireGuard Mesh + OIDC 认证 | 原型验证，真实 TUN/IPAM/路由未完成 |
| **Phase 3** | 智能调度：QUIC 传输 + 智能路由 + 多 Relay 节点 | 原型验证，数据面切换闭环未完成 |
| **Phase 4** | 全球加速：边缘节点 + eBPF 加速 + SD-WAN | 概念原型，非生产能力 |

近期优先级：质量门禁、安全基线、Control Plane 最小持久化、QUIC Relay E2E、P2P/TUN 平台抽象。更详细的真实进度请以 [progress-tracking.md](progress-tracking.md) 为准。

---

## 贡献指南

欢迎贡献！请遵循以下流程：

1. Fork 本仓库
2. 创建功能分支：`git checkout -b feature/your-feature`
3. 提交代码前运行测试：`make test`
4. 确保代码检查通过：`make lint`
5. 提交 Pull Request

---

## 许可证

本项目开源，具体许可证信息请参阅 [LICENSE](LICENSE) 文件。
