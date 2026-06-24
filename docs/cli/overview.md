# CLI 使用指南

`nextunnel` 是 NexTunnel 的统一命令行入口，覆盖本机服务端部署、远端 Control Plane / Dashboard 操作、桌面端本机控制和环境诊断。

## 获取方式

Release 提供两类 CLI 来源：

- 服务端包内置 `bin/nextunnel`，Linux 一键安装后默认创建 `/usr/local/bin/nextunnel` 软链接。
- 独立 CLI 包适合运维机、CI 或无需安装服务端的管理终端。

```bash
nextunnel version
nextunnel doctor
nextunnel --help
```

CLI 配置默认保存到当前用户目录。配置文件可能包含 token，写入时会使用当前用户私有权限。

## 服务端管理

服务端子命令用于调用发布包内置的一键安装脚本，适合 Linux systemd 部署和 Windows 本地进程管理。

```bash
nextunnel server paths
nextunnel server install --version v0.6.0-beta --non-interactive --public-host <公网IP或域名>
nextunnel server status
nextunnel server health
nextunnel server logs --follow
nextunnel server restart
nextunnel server down
nextunnel server up
nextunnel server update --version v0.6.0-beta
nextunnel server uninstall --purge
```

常用安装参数：

| 参数 | 说明 |
| --- | --- |
| `--version` | GitHub Release 版本，例如 `v0.6.0-beta` |
| `--package-url` | 服务端包 URL、本地路径或 `file://` 路径 |
| `--release-base-url` | 自定义 Release 下载基址，适合 COS/CDN 镜像 |
| `--github-proxy` | 可信自建 GitHub 下载代理 |
| `--public-host` | 客户端访问的公网 IP 或域名 |
| `--relay-token` | Relay 客户端认证 token |
| `--control-token` | Control Plane Bearer Token |
| `--dashboard-password` | Dashboard 管理员密码 |
| `--dashboard-disabled` | 仅部署核心服务，不启动 Dashboard |

## 同机隔离测试

同一台服务器已有 NexTunnel 服务时，可以使用独立目录、端口和 systemd 服务名前缀，避免覆盖生产服务。

```bash
sudo nextunnel server install \
  --package-url /tmp/nextunnel-server-linux-amd64.tar.gz \
  --install-dir /opt/nextunnel-test \
  --config-dir /etc/nextunnel-test \
  --data-dir /var/lib/nextunnel-test \
  --service-prefix nextunnel-test \
  --relay-port 27000 \
  --relay-quic-port 27443 \
  --control-plane-port 29090 \
  --dashboard-port 28080 \
  --nat-port 23478 \
  --non-interactive
```

安装脚本会把路径和 `NEXTUNNEL_SERVICE_PREFIX` 写入包内 `deploy/server/.env`，后续 `status`、`up`、`down`、`restart`、`logs` 会按该前缀查找服务。

不要把 systemd 模式安装目录放在 `/tmp`、`/var/tmp` 或 `/dev/shm` 下。服务启用隔离后，临时目录可能在服务命名空间内不可见。

## 远端上下文

远端命令依赖当前上下文。Control Plane 和 Dashboard 可以分别配置。

```bash
nextunnel config path
nextunnel config set-context prod \
  --server http://127.0.0.1:9090 \
  --token <control-token> \
  --dashboard http://127.0.0.1:8080 \
  --dashboard-token <dashboard-token>
nextunnel config use-context prod
nextunnel config current-context
nextunnel config get-contexts
```

Dashboard 支持通过账号密码登录并自动保存 token：

```bash
nextunnel remote login \
  --dashboard http://127.0.0.1:8080 \
  --username admin \
  --password <password> \
  --context prod
```

## 远端管理

```bash
nextunnel remote health
nextunnel remote health --target control-plane
nextunnel remote health --target dashboard
nextunnel remote node list
nextunnel remote node inspect <node_id>
nextunnel remote acl list
nextunnel remote alert list
nextunnel remote alert ack <alert_id>
```

`node` 和 `acl` 命令需要当前上下文配置 Control Plane 地址；`alert` 命令需要配置 Dashboard 地址。缺少地址时，CLI 会提示应使用的 `config` 或 `remote login` 命令。

## 桌面端控制

桌面端运行后会创建本机控制文件，CLI 通过该控制文件访问本机控制 API。

```bash
nextunnel desktop open
nextunnel desktop status
nextunnel desktop settings get
nextunnel desktop settings set --relay 127.0.0.1:7000 --relay-token <token>
nextunnel desktop connect --relay 127.0.0.1:7000 --token <token>
nextunnel desktop nat detect
nextunnel desktop network apply
nextunnel desktop network reset
nextunnel desktop disconnect
```

如果桌面端控制文件不在默认位置，可以显式指定：

```bash
nextunnel desktop --control-file /path/to/nextunnel-control.json status
```

## 输出格式

默认输出面向终端阅读。自动化脚本建议使用 JSON：

```bash
nextunnel server status --output json
nextunnel remote node list -o json
```

## 排障建议

```bash
nextunnel doctor
nextunnel server health
nextunnel server logs --follow
```

国内服务器下载 GitHub Release 慢时，优先使用 `--package-url` 指向手动上传的包，或把 Release 资产同步到 COS/CDN 后使用 `--release-base-url`。生产环境下载远端资产时建议同时配置 SHA256 校验。
