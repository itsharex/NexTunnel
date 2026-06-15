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

## 国内服务器下载加速

腾讯云等国内服务器访问 GitHub Release 慢时，推荐把 Release 资产同步到腾讯云 COS/CDN，再使用自定义 Release 下载基址：

```bash
sudo NEXTUNNEL_RELEASE_BASE_URL=https://cos.example.com/nextunnel/v0.3.1-alpha \
  ./install.sh install --version v0.3.1-alpha --sha256 <sha256>
```

临时方案可以使用可信自建 GitHub 代理：

```bash
sudo ./install.sh install --version v0.3.1-alpha \
  --github-proxy https://your-proxy.example.com/
```

下载优先级为 `--package-url` / `NEXTUNNEL_PACKAGE_URL`、`--release-base-url` / `NEXTUNNEL_RELEASE_BASE_URL`、`--github-proxy` / `NEXTUNNEL_GITHUB_PROXY`、默认 GitHub Release。生产环境建议总是配合 SHA256 校验。
