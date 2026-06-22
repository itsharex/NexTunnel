NexTunnel macOS 安装说明

1. 将 NexTunnel.app 拖入 Applications。
2. 首次启动如果 macOS 提示应用来自未验证开发者，请在系统设置的隐私与安全中允许打开。本 alpha 包默认未签名，正式发布可通过 Developer ID 签名和 notarization 钩子生成。
3. NexTunnel 的 P2P/Relay 功能不需要安装内核组件；真实系统路由和 utun 路由注入需要管理员授权、授权 helper 或 LaunchDaemon 支持。
4. 应用内“网络”页面会显示当前权限、utun/TUN 状态和可执行修复建议。

