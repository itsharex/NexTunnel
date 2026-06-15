# 快速开始

NexTunnel 使用 Go + Vue 3 + Wails 构建桌面端，服务端提供 Relay、Control Plane、NAT Detector 和 Dashboard。

## 环境要求

| 工具 | 版本 |
| --- | --- |
| Go | >= 1.25 |
| Node.js | >= 18 |
| Wails CLI | v2.x |
| Docker Compose | 服务端部署可选 |

## 本地开发

```bash
make install-deps
make dev
```

桌面端开发模式会启动 Wails，并自动连接前端 Vite 开发服务器。

## 构建桌面端

```bash
make build
```

构建产物位于 `desktop/build/bin/`。

## 运行服务端

```bash
docker-compose up -d
```

默认会启动 Relay、Control Plane、Dashboard 和 NAT Detector。生产环境必须配置强随机 token、HTTPS 或 mTLS。
