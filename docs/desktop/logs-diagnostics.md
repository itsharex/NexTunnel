# 日志与诊断

桌面端会把关键操作写入本地活动日志，方便普通用户和运维人员定位问题。

## 日志范围

日志覆盖：

- 连接 Relay
- 断开 Relay
- 创建、更新、启动、停止、删除隧道
- 保存设置
- 应用和重置虚拟网络
- NAT 探测
- 本机端口扫描
- 常用端口保存和删除
- Wintun 修复
- 运行错误
- 清空日志

## 查看日志

进入“日志”页可以：

- 按级别筛选：info、warning、error。
- 按分类筛选：operation、security、error。
- 刷新日志。
- 清空日志。

总览页只显示最近日志，完整排障请进入日志页。

## 错误处理

桌面端后端会把最近错误写入运行状态，前端失败提示会优先显示这个错误。例如：

```text
创建 Windows TUN 适配器 "nextunnel0" 失败：access denied。
请确认 wintun.dll 已就绪，并以管理员身份运行 NexTunnel 后重新应用路由。
```

收到错误后建议：

1. 先看错误文本是否给出明确修复入口。
2. 打开网络页查看平台能力和 Wintun/TUN 阻塞项。
3. 打开日志页按 error 过滤。
4. 使用“关于 -> 收集诊断”生成诊断文本。

## CLI 诊断

桌面端运行后，CLI 可以读取本机控制文件：

```bash
nextunnel desktop status
nextunnel desktop settings get
nextunnel desktop nat detect
nextunnel desktop network apply
```

如果控制文件不在默认位置：

```bash
nextunnel desktop --control-file /path/to/nextunnel-control.json status
```

## 收集诊断

设置页“关于 -> 收集诊断”会生成当前配置、运行状态和环境摘要。诊断输出不应包含原始 Relay Token 或 Control Plane Token。

提交问题时建议附带：

- NexTunnel 版本。
- 操作系统版本。
- Relay/Control Plane/Dashboard 部署方式。
- 相关活动日志。
- 网络页 Wintun/TUN 状态。
- 服务端 `install.sh health` 或 `nextunnel server health` 输出。
