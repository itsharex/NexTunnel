---
layout: home

hero:
  name: NexTunnel
  text: 面向未来的 NAT 穿透工具
  tagline: 桌面可视化 · P2P 直连优先 · Relay 自动降级 · 本机端口扫描
  image:
    src: /logo.png
    alt: NexTunnel
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: 桌面端能力
      link: /desktop/overview
    - theme: alt
      text: CLI 使用
      link: /cli/overview
    - theme: alt
      text: 发布流程
      link: /deploy/release

features:
  - title: 可视化流量反馈
    details: 桌面端实时记录上传和下载增量曲线，帮助用户快速判断隧道是否有真实流量。
  - title: 连接类型透明
    details: 隧道列表直接展示 P2P 直连、中继或待机状态，减少排障时的猜测成本。
  - title: 本机端口扫描
    details: 一键扫描回环地址常用端口，覆盖开发、数据库、软件服务、游戏和远程访问。
  - title: 分类设置界面
    details: 参考现代桌面工具的信息架构，将连接、网络、端口、安全、外观和关于分区管理。
  - title: 安全边界明确
    details: 端口扫描限制在 127.0.0.1/::1，并对高风险端口提供说明，避免误扫外部网络。
  - title: 标准发布流程
    details: v0.4.1-alpha 统一桌面端、CLI、服务端脚本、验证工具和文档站发布入口。
  - title: 统一 CLI
    details: 覆盖服务端安装、状态、日志、远端控制面和桌面端本机控制，适合自动化运维。
  - title: 生产验证工具链
    details: Dashboard、P2P/TUN、eBPF 和 Edge 演练均输出 JSON 报告，便于发布前归档。
---

<style>
:root {
  --vp-c-brand-1: #00d5d5;
  --vp-c-brand-2: #00ffff;
  --vp-c-brand-3: #8a2be2;
  --vp-home-hero-name-color: transparent;
  --vp-home-hero-name-background: -webkit-linear-gradient(120deg, #1da1f2 20%, #8a2be2 72%);
  --vp-home-hero-image-background-image: linear-gradient(135deg, #00ffff 20%, #8a2be2 80%);
  --vp-home-hero-image-filter: blur(48px);
}
</style>
