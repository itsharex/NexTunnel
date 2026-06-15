# 隧道与端口

## 一键扫描本机端口

端口扫描只允许回环地址：

- `127.0.0.1`
- `::1`
- `localhost`

扫描默认使用启用的常用端口，并限制最大端口数量和连接超时，避免影响系统性能或误扫外部网络。

## 常用端口分类

内置端口覆盖以下类型：

| 类型 | 示例 |
| --- | --- |
| 开发 | Next.js、Vite、Spring Boot、Laravel |
| 数据库 | MySQL、PostgreSQL、Redis、MongoDB、Elasticsearch |
| 软件服务 | MinIO、Prometheus、Docker API |
| 游戏 | Minecraft Java、Terraria、Steam/Source、Palworld |
| 远程访问 | SSH、Remote Desktop |
| 标准服务 | HTTP、HTTPS |

## 快速创建隧道

扫描结果或常用端口可以一键填充隧道表单。远端端口默认与本地端口一致，用户可在创建前修改。
