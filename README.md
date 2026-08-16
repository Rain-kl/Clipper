# Clipper

跨设备捕获与分级留存 — 类微信「文件传输助手」的多用户 Web 应用。

[中文](./README_zh.md)

[![License: Apache2.0](https://img.shields.io/badge/License-Apache2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org/)
[![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)](https://nextjs.org/)
[![React](https://img.shields.io/badge/React-19-blue.svg)](https://reactjs.org/)

## Introduction

**Clipper** lets you capture anything you need temporarily or long-term: a thought, a password snippet, or files between devices. Captures are stored as **Items**, classified by lifecycle and importance, browsed in Review / Search / Library / Vault, and automatically archived or purged by policy.

Built on a production Go + Next.js stack (auth, tasks, uploads, admin).

### Product capabilities (roadmap)

- **Clip** — large composer for text / image / file capture
- **Review** — day-grouped timeline with quick classify
- **Search / Library / Vault** — find and browse by type or importance
- **Retention** — pending auto-archive, trash auto-purge (configurable)
- **Later:** tags, AI classify, Telegram/QQ providers, vault encryption

### Platform foundations

- Multi-auth (password + OIDC), access tokens, user admin
- Dynamic system config, Asynq workers, S3-compatible storage
- Observability (Zap + OpenTelemetry), Swagger, embedded frontend option

## Architecture

```
┌─────────────────┐    ┌─────────────────────────────┐    ┌─────────────────┐
│   Frontend      │    │          Backend             │    │   Database      │
│   (Next.js)     │◄──►│           (Go)               │◄──►│  (PostgreSQL)   │
│  Clip / Review  │    │  Item domain + platform      │    │  Redis / Asynq  │
└─────────────────┘    └─────────────────────────────┘    └─────────────────┘
                              │
                   api | worker | scheduler
```

## Tech stack

**Backend:** Go 1.25+, Gin, GORM, PostgreSQL, Redis, Asynq, Cobra/Viper, OTel, Swaggo, AWS SDK v2  
**Frontend:** Next.js App Router, React 19, TypeScript, Tailwind 4, shadcn/ui

## Requirements

- Go >= 1.25
- Node.js >= 18, pnpm >= 8
- PostgreSQL >= 14
- Redis >= 6

## Quick start

```bash
git clone https://github.com/Rain-kl/Clipper.git
cd Clipper
cp config.example.yaml config.yaml
# edit config.yaml (session_secret, database, redis)
```

```bash
# DB (example)
createdb -h 127.0.0.1 -p 5432 -U postgres clipper
```

```bash
# Backend
go run . all
# or: go run . api / worker / scheduler

# Frontend (dev)
cd frontend && pnpm install && pnpm dev
```

Default API `:8000`, frontend typically `http://localhost:3000` with rewrites to the API.

Binary after build: `bin/clipper` (CLI use: `clipper api|worker|scheduler|all`).

## Configuration

See `config.example.yaml` and `.env.example`. Notable defaults for this product:

| Key | Default |
| --- | --- |
| `app.app_name` | `clipper` |
| `app.session_cookie_name` | `clipper_session_id` |
| `database.database` / SQLite path | `clipper` / `clipper.db` |
| `redis.key_prefix` | `clipper:` |

Seeded system config includes `site_name=Clipper` and `update_upstream_repository=Rain-kl/Clipper`.

> Go module path remains `github.com/Rain-kl/Wavelet` for historical compatibility; product name and repo are **Clipper**.

## Docker

```bash
docker compose up -d
# image name (CI): clipper
```

## License

Apache 2.0 — see [LICENSE](./LICENSE) and [NOTICE](./NOTICE).
