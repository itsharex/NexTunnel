# 服务端部署

服务端可以通过 Docker Compose 或发布包部署。

## Docker Compose

```bash
docker-compose up -d
```

生产环境建议：

- Relay 配置强随机 `auth-token`。
- Dashboard 配置强 `secret-key` 和管理员密码。
- Control Plane API 配置 Bearer Token。
- 公开入口使用 HTTPS 或 mTLS。

## 端口

| 服务 | 默认端口 |
| --- | --- |
| Relay 控制端口 | 7000/TCP |
| Relay QUIC | 7443/UDP |
| Control Plane | 9090/TCP |
| Dashboard | 8080/TCP |
| NAT Detector | 3478/UDP |
