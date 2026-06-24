# 服务端部署

服务端可以通过 Docker Compose 或发布包部署。

## Docker Compose

```bash
docker-compose up -d
```

生产环境建议：

- Relay 配置强随机 `auth-token`。
- Relay 管理 API 配置强随机 `admin-token`，并保持本机或容器内网监听。
- Dashboard 配置强 `secret-key` 和管理员密码。
- Control Plane API 配置 Bearer Token。
- 公开入口使用 HTTPS 或 mTLS。

## 端口

| 服务 | 默认端口 |
| --- | --- |
| Relay 控制端口 | 7000/TCP |
| Relay QUIC | 7443/UDP |
| Relay 管理 API | 7001/TCP，仅本机/内网 |
| Control Plane | 9090/TCP |
| Dashboard | 8080/TCP |
| NAT Detector | 3478/UDP |

## 国内服务器下载加速

腾讯云等国内服务器访问 GitHub Release 慢时，推荐把 Release 资产同步到腾讯云 COS/CDN，再使用自定义 Release 下载基址：

```bash
sudo NEXTUNNEL_RELEASE_BASE_URL=https://cos.example.com/nextunnel/v0.6.0-beta \
  ./install.sh install --version v0.6.0-beta --sha256 <sha256>
```

临时方案可以使用可信自建 GitHub 代理：

```bash
sudo ./install.sh install --version v0.6.0-beta \
  --github-proxy https://your-proxy.example.com/
```

下载优先级为 `--package-url` / `NEXTUNNEL_PACKAGE_URL`、`--release-base-url` / `NEXTUNNEL_RELEASE_BASE_URL`、`--github-proxy` / `NEXTUNNEL_GITHUB_PROXY`、默认 GitHub Release。生产环境建议总是配合 SHA256 校验。

## 手动上传包安装

服务器已手动上传服务端包时，直接指定本地包路径即可。生产环境需要显式指定公网 IP 或域名：

```bash
sudo /tmp/nextunnel-install.sh install \
  --package-url /tmp/nextunnel-server-linux-amd64.tar.gz \
  --public-host <腾讯云公网IP或域名>
```

安装脚本会把 CLI 安装到 `/opt/nextunnel/bin/nextunnel`，并默认创建 `/usr/local/bin/nextunnel` 软链接。旧脚本安装后如出现 `nextunnel: command not found`，可手动补建：

```bash
sudo ln -sfn /opt/nextunnel/bin/nextunnel /usr/local/bin/nextunnel
```

无法连接时先执行：

```bash
sudo /opt/nextunnel/deploy/server/install.sh health
sudo /opt/nextunnel/deploy/server/install.sh logs --no-log-follow --log-lines 80
```

安装完成后，包内脚本会在 `/opt/nextunnel/deploy/server/.env` 保存本次安装路径和服务前缀。后续查看状态、启动服务或重启服务不需要重复传参：

```bash
sudo /opt/nextunnel/deploy/server/install.sh status
sudo /opt/nextunnel/deploy/server/install.sh up
sudo /opt/nextunnel/deploy/server/install.sh restart
```

`up` 和 `restart` 会等待 systemd 服务进入 active 状态，并自动执行 Control Plane、Relay 和 Dashboard 健康检查。

腾讯云安全组和服务器防火墙至少需要放行 `7000/tcp`；启用 QUIC/NAT 检测时还需要 `7443/udp`、`3478/udp`，Dashboard 需要 `8080/tcp`。Relay 管理 API 默认用于 Dashboard 客户端监控，不应对公网开放。

## 生产验收

服务启动和健康检查通过后，继续按 [生产验证手册](./production-verification.md) 执行 Dashboard HTTPS/CORS/鉴权、P2P/TUN、eBPF Linux 和多地域 Edge 演练。验证报告默认写入 `dist/verification/`，可作为 Beta 发布前验收附件。

## 同机测试和 WSL 验证

同一台机器已有 NexTunnel 服务时，可以使用独立服务前缀和端口做隔离测试：

```bash
sudo ./install.sh install \
  --package-url /tmp/nextunnel-server-linux-amd64.tar.gz \
  --install-dir /opt/nextunnel-test \
  --config-dir /etc/nextunnel-test \
  --data-dir /var/lib/nextunnel-test \
  --service-prefix nextunnel-test \
  --relay-port 27000 \
  --relay-quic-port 27443 \
  --control-plane-port 29090 \
  --dashboard-port 28080 \
  --nat-port 23478
```

不要把 systemd 模式安装目录放在 `/tmp`、`/var/tmp` 或 `/dev/shm` 下。服务启用了隔离和自动重启，临时目录会导致二进制或数据目录在服务命名空间中不可见。
