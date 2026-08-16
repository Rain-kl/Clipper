# Clipper

跨设备捕获与分级留存 — 类微信「文件传输助手」的多用户 Web 应用。

[English](./README.md)

[![License: Apache2.0](https://img.shields.io/badge/License-Apache2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org/)
[![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)](https://nextjs.org/)
[![React](https://img.shields.io/badge/React-19-blue.svg)](https://reactjs.org/)

## 项目简介

**Clipper** 用于捕获你需要临时或长期保存的内容：一个想法、一段暂存密码、跨设备文件。内容以 **Item** 入库，按生命周期与重要性分级，在「回顾 / 搜索 / 记录 / 密码本」中浏览，并按策略自动归档或清理。

底层基于生产级 Go + Next.js 栈（认证、任务、上传、管理后台）。

### 产品能力（规划）

- **Clip** — 大输入框投递文本 / 图片 / 附件
- **回顾** — 按天时间线 + 快速分类
- **搜索 / 记录 / 密码本** — 按类型与重要性浏览
- **保留策略** — 未处理自动归档、垃圾箱自动硬删（可配置）
- **后续：** 标签、AI 分类、Telegram/QQ 入口、密码本加密

### 平台基础能力

- 多认证（密码 + OIDC）、访问令牌、用户管理
- 动态系统配置、Asynq Worker、S3 兼容存储
- 可观测性（Zap + OpenTelemetry）、Swagger、可选前端嵌入

## 快速开始

```bash
git clone https://github.com/Rain-kl/Clipper.git
cd Clipper
cp config.example.yaml config.yaml
# 修改 session_secret、数据库与 Redis
```

```bash
createdb -h 127.0.0.1 -p 5432 -U postgres clipper
go run . all
cd frontend && pnpm install && pnpm dev
```

编译产物：`bin/clipper`（`clipper api|worker|scheduler|all`）。

## 配置

见 `config.example.yaml`、`.env.example`。产品默认名：

| 配置 | 默认 |
| --- | --- |
| `app.app_name` | `clipper` |
| `app.session_cookie_name` | `clipper_session_id` |
| 数据库名 / SQLite | `clipper` / `clipper.db` |
| Redis 前缀 | `clipper:` |

种子配置含 `site_name=Clipper`、`update_upstream_repository=Rain-kl/Clipper`。

> Go module 路径暂时仍为 `github.com/Rain-kl/Wavelet`（兼容历史 import）；产品名与仓库为 **Clipper**。

## 文档

- 设计规格：`docs/superpowers/specs/2026-07-22-clipper-mvp-design.md`
- 实现计划：`docs/superpowers/plans/2026-07-22-clipper-mvp.md`
- 部署：`docs/DEPLOYMENT.md`

## 许可证

Apache 2.0 — 见 [LICENSE](./LICENSE) 与 [NOTICE](./NOTICE)。
