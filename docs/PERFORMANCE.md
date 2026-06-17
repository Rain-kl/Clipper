# Wavelet 系统性能分析与优化建议

> 分析日期：2026-06-17  
> 范围：Go 后端 + Next.js 前端  
> 目标：识别可能在生产环境真实出现的性能问题，并给出高 ROI 优化路线

---

## 目录

- [架构概览与核心瓶颈](#架构概览与核心瓶颈)
- [Critical — 高概率生产问题](#critical--高概率生产问题)
- [Medium — 中等风险](#medium--中等风险)
- [高价值优化路线图](#高价值优化路线图)
- [已做得好的设计](#已做得好的设计)
- [场景风险矩阵](#场景风险矩阵)
- [优先行动清单](#优先行动清单)

---

## 架构概览与核心瓶颈

```mermaid
flowchart LR
  subgraph frontend["前端 (Static Export)"]
    A[HTML 静态壳] --> B[Hydrate]
    B --> C["UserProvider.getUserInfo()"]
    C --> D[页面数据请求]
    D --> E[渲染]
  end

  subgraph backend["后端热点路径"]
    F["/f/{id}?quality=..."] --> G[DB 查 upload]
    G --> H[迁移状态 DB 查询]
    H --> I[白名单 Redis/DB]
    I --> J{WebP 缓存命中?}
    J -->|否| K["全量读文件 + 编码 + 磁盘缓存(全局锁)"]
    J -->|是| L[返回]
  end

  C -.->|串行阻塞| D
```

当前最大的结构性问题：

1. **前端**：全客户端渲染 + 全局认证瀑布流，所有业务数据请求被 `getUserInfo` 串行阻塞。
2. **后端**：文件服务路径（`/f/{id}`）是最高频热点，WebP 缓存未命中时在请求线程内做重 CPU/IO 工作，且磁盘缓存使用全局互斥锁。

---

## Critical — 高概率生产问题

### 1. 图片 WebP 服务：请求路径阻塞 + 全局锁串行化

**涉及文件**：

- `internal/apps/upload/file_server.go`
- `pkg/cache/disk/cache.go`

**问题描述**：

缓存未命中时，在 HTTP 请求 goroutine 内执行：

1. `io.ReadAll` 将原始文件全量读入内存
2. 进程内 WebP 解码 + 编码
3. 写入磁盘缓存

同时，磁盘缓存 `Get`/`Set` 使用**全局 `sync.Mutex`**，所有并发图片请求在缓存层完全串行。

```go
// file_server.go — 缓存 miss 时的重操作
origBytes, err := getOriginalFileBytes(ctx, upload)  // io.ReadAll
webpBytes, err = CompressImageToWebP(bytes.NewReader(origBytes), quality)
cache.Set(cacheKey, webpBytes, diskcache.NoExpiration)

// pkg/cache/disk/cache.go — 全局互斥锁
func (c *Cache) Get(key string) ([]byte, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    // ...
}
```

**生产表现**：

- 首次访问或缓存淘汰后，P99 延迟从几十毫秒飙升到数秒
- 并发图片请求形成「隐形队列」
- 大文件全量读入带来内存尖峰，可能触发 OOM 或 GC 停顿

**优化价值**：⭐⭐⭐⭐⭐

**建议**：

- [ ] 磁盘缓存改用 `RWMutex`，读路径不互斥
- [ ] 对同一 cache key 使用 `singleflight` 合并并发 miss
- [ ] 部署后强制执行 `upload:warm_image_cache` 异步预热任务
- [ ] 考虑 miss 时先返回原图，后台异步生成 WebP

---

### 2. 文件访问路径：每次请求多次 DB/Redis 查询

**涉及文件**：

- `internal/apps/upload/storage_ops.go`
- `internal/apps/upload/file_server.go`

**问题描述**：

存储迁移状态**无进程内缓存**，每次文件操作都查询 `w_task_executions`：

```go
// storage_ops.go
func StorageReadOnly(ctx context.Context) bool {
    execution, ok, err := latestStorageMigrationExecution(ctx)
    // ...
}

func backendForStoredDriver(ctx context.Context, driver storage.Driver) (storage.Backend, error) {
    // 可能再次调用 currentMigrationTargetConfig → 又一次相同 DB 查询
}
```

公开文件白名单每次走 Redis/DB：

```go
// file_server.go
func isFilePublic(ctx context.Context, uploadType string) bool {
    sc.GetByKey(ctx, model.ConfigKeyFileAccessWhitelist)
    // JSON 解析 + 遍历
}
```

对比：`storage.Active()` 已有 5 秒内存缓存 + Redis pub/sub 失效机制，迁移状态却未复用该模式。

**生产表现**：

- 每个 `/f/{id}` 请求额外 2–4 次 DB/Redis 往返
- 图片站/CDN 场景下 QPS 放大后 PostgreSQL 连接池压力明显

**优化价值**：⭐⭐⭐⭐⭐

**建议**：

- [ ] 为 `StorageReadOnly` / `latestStorageMigrationExecution` 增加 5s TTL 进程内缓存
- [ ] 配置变更或迁移状态变化时通过 Redis pub/sub 失效
- [ ] `file_access_whitelist` 增加进程内缓存，复用 `GetByKey` 的失效机制

---

### 3. Admin 文件统计：无界全表扫描

**涉及文件**：`internal/apps/upload/stats.go`

**问题描述**：

```go
err = db.DB(ctx).Model(&model.Upload{}).
    Select("extension, mime_type, file_size").
    Where("status != ?", model.UploadStatusDeleted).
    Scan(&fileRaws).Error
// 然后在 Go 中遍历全量结果做分类统计
```

**生产表现**：

- 10 万+ 文件时，管理端「文件统计」接口耗时数秒
- 占用数百 MB 内存，可能拖垮 admin API

**优化价值**：⭐⭐⭐⭐

**建议**：

- [ ] 改为 SQL `GROUP BY` + `CASE WHEN` 聚合
- [ ] 或维护增量统计表，上传/删除时更新计数

---

### 4. `w_uploads` 索引缺口

**涉及文件**：`internal/db/migrator/goose/postgres/202606090001_initial_schema.sql`

**当前索引**：`user_id`, `file_path`, `hash`, `type`

**缺失的高频查询索引**：

| 查询场景 | 建议索引 |
|----------|----------|
| 清理任务 `status + created_at` | `(status, created_at)` |
| 存储迁移 `storage_driver + status` | `(storage_driver, status)` |
| 秒传去重 `hash + file_size + status` | `(hash, file_size, status)` |

**生产表现**：

- 数据量增长后，清理 worker、迁移任务、上传去重退化为顺序扫描
- 后台任务积压，admin 操作变慢

**优化价值**：⭐⭐⭐⭐

**建议**：

- [ ] 通过 goose migration 新增上述复合索引（PostgreSQL + SQLite 双方言）

---

### 5. 批量 ZIP 下载：无上限 + 同步阻塞

**涉及文件**：`internal/apps/upload/routers.go` — `BatchDownloadFiles`

**问题描述**：

- `req.IDs` 无数量上限
- 在请求 goroutine 内串行打开每个文件并 `io.Copy` 到 ZIP
- 远端 S3 场景下单个文件就可能耗时数秒

**生产表现**：

- 网关超时、连接耗尽
- Admin 批量下载操作卡死

**优化价值**：⭐⭐⭐⭐

**建议**：

- [ ] 限制单次批量数量（如 max 50）
- [ ] 或改为 Asynq 后台任务生成 ZIP，前端轮询下载链接

---

### 6. 前端全局认证瀑布流

**涉及文件**：

- `frontend/contexts/user-context.tsx`
- `frontend/app/(main)/layout.tsx`

**问题描述**：

```tsx
// user-context.tsx — 挂载时获取用户
useEffect(() => {
  fetchUser()
}, [fetchUser])

// layout.tsx — 阻塞所有子页面渲染
if (loading || !user) {
  return <LoadingPage text="登录状态" badgeText="Auth" />
}
```

**生产表现**：

- 每次进入 `/home`、`/files`、`/admin/*` 都先等 `getUserInfo`（约 200–800ms）
- 页面级数据请求无法并行启动，TTI 被硬性拉长

**优化价值**：⭐⭐⭐⭐⭐

**建议**：

- [ ] Layout 不阻塞渲染，子页面自行处理未登录状态
- [ ] 或 Server Component 通过 cookie 预取 session，消除客户端首屏等待
- [ ] `/login`、`/register` 跳过 `getUserInfo`

---

### 7. 实时日志面板：2000 行 DOM 无虚拟化

**涉及文件**：`frontend/components/common/admin/app-logs.tsx`

**问题描述**：

- 日志上限 2000 行（内存有界，但 DOM 无界）
- 每行渲染完整 `<div>`，无虚拟滚动
- `@tanstack/react-virtual` 已在 `package.json` 但未使用

**生产表现**：

- 管理员开着日志 Tab 时 CPU/内存持续升高
- 滚动卡顿，长时间运行拖慢整台机器

**优化价值**：⭐⭐⭐⭐

**建议**：

- [ ] 使用 `useVirtualizer` 只渲染可视区域行
- [ ] 行组件 `React.memo` 避免无效重渲染

---

## Medium — 中等风险

| # | 问题 | 位置 | 影响 |
|---|------|------|------|
| 1 | 公共配置接口无 Redis 缓存 | `internal/model/system_configs.go` — `ListVisibleSystemConfigs` | 每次前端启动/登录直查 PostgreSQL |
| 2 | CAPTCHA 每次 5 次独立 `GetByKey` | `internal/apps/cap/manager.go` | 登录高峰 Redis 压力 |
| 3 | OIDC 每次 `oidc.NewProvider` 无缓存 | `internal/apps/oauth/sources.go:164` | 登录发起/回调多一次外部 HTTP |
| 4 | CORS 每次跨域查 `server_address` 配置 | `internal/router/middlewares.go:75` | 预检请求放大 |
| 5 | 推送通知无界 goroutine + 逐 target DB 查询 | `internal/apps/admin/push/events.go:102` | 通知风暴时 goroutine/DB 双压 |
| 6 | 上传清理：每文件一个事务 | `internal/apps/upload/cleanup.go` | 大量 pending 文件时 commit 风暴 |
| 7 | ClickHouse 风控：每请求 `json.Marshal` 全部 headers | `internal/apps/risk_control/middleware.go:58` | 高 QPS 时 CPU 开销（写入本身已异步批处理） |
| 8 | 存储迁移日志大量写 Redis | `internal/apps/upload/storage_migration_task.go` | 迁移期间 Redis CPU/内存压力 |
| 9 | 存储迁移后二次 SHA 全量读取验证 | `storage_migration_task.go` | 迁移期间对象 I/O 翻倍 |
| 10 | Admin 状态页 5s 轮询 | `frontend/components/common/admin/status.tsx` | Tab 常驻时持续打后端 |
| 11 | 路由切换 500ms fade 动画 | `frontend/app/(main)/layout.tsx:53-60` | 即使数据已缓存，感知仍慢 |
| 12 | 无 `next/dynamic` 代码分割 | 全项目 | Admin 首包 300–450KB+ |
| 13 | 19/24 个 `page.tsx` 为 `"use client"` | 各路由 | 无法 RSC 预取，bundle 偏大 |
| 14 | Admin 部分页面用 `useEffect` 而非 React Query | `access-logs.tsx`, `task-executions.tsx` 等 | 无缓存去重，重复请求 |
| 15 | 登录页 OIDC sources 等待 public config | `frontend/components/auth/login-form.tsx` | 多 1 次 RTT 瀑布 |
| 16 | Users 表每行嵌套 3 个 `TooltipProvider` | `frontend/app/(main)/admin/users/page.tsx` | 50+ 行时不必要重渲染 |
| 17 | 缩略图用原生 `<img>` 无 lazy loading | `file-list.tsx`, `file-manager.tsx` | 文件管理页初始解码压力大 |
| 18 | `@/lib/services` barrel 导入 | ~40 个文件 | 单路由 bundle 膨胀 10–30KB |
| 19 | SQLite 模式无连接池调优 | `internal/db/postgres.go` | 默认 SQLite 写锁瓶颈 |
| 20 | Session Redis 仅用第一个地址 | `internal/router/router.go` | Sentinel/Cluster 场景不一致 |

---

## 高价值优化路线图

### P0 — 立即做（1–2 周，收益最大）

| # | 优化项 | 涉及模块 | 预期收益 | 复杂度 |
|---|--------|----------|----------|--------|
| 1 | WebP：`singleflight` + `RWMutex` + 强制预热 | `file_server.go`, `pkg/cache/disk/` | 图片 P99 ↓ 80%+，并发吞吐 ↑ 5–10x | 中 |
| 2 | 缓存 `StorageReadOnly` / 迁移状态 | `storage_ops.go` | 每文件请求减少 1–3 次 DB | 低 |
| 3 | 内存缓存 `file_access_whitelist` | `file_server.go` | 每公开文件请求减少 1 次 Redis | 低 |
| 4 | `GetFileStats` 改为 SQL 聚合 | `stats.go` | Admin 统计从 O(n) → O(1) | 低 |
| 5 | 新增 `w_uploads` 复合索引 | goose migration | 清理/迁移/秒传全面加速 | 低 |
| 6 | 前端日志虚拟化 | `app-logs.tsx` | Admin 日志 Tab 流畅度质变 | 低 |
| 7 | Admin 重模块 `dynamic()` 懒加载 | `database/page.tsx`, `logs/page.tsx`, `settings/page.tsx` 等 | 首包 JS ↓ 150–300KB | 低 |

### P1 — 短期（2–4 周）

| # | 优化项 | 预期收益 |
|---|--------|----------|
| 8 | 认证并行化：layout 不阻塞 / Server 预取 session | TTI ↓ 200–800ms |
| 9 | `ListVisibleSystemConfigs` 加 Redis 缓存 | 前端冷启动加速 |
| 10 | CAPTCHA 配置快照（一次加载 5 个 key） | 验证码路径 Redis ops ↓ 80% |
| 11 | OIDC Provider/JWKS 进程内缓存（TTL 1h） | 登录延迟 ↓ 100–500ms |
| 12 | 批量下载限制（max 50）或异步任务 | 消除网关超时风险 |
| 13 | Admin `useEffect` 数据获取迁移到 React Query | 去重、缓存、后台刷新 |
| 14 | 登录页并行请求 public config + auth sources | 登录页 ↓ 100–300ms |
| 15 | 状态轮询在 `document.hidden` 时暂停 | 降低后台 + 客户端负载 |

### P2 — 中期架构演进

| # | 优化项 | 预期收益 |
|---|--------|----------|
| 16 | 批量 ZIP 改为 Asynq 后台任务 | 彻底解耦长耗时操作 |
| 17 | 存储迁移日志降噪 + 跳过已验证文件二次 SHA | 迁移期间 Redis/I/O ↓ 50% |
| 18 | 推送通知 target 批量解析（`WHERE id IN ?`） | 通知风暴 DB 查询 ↓ N 倍 |
| 19 | 上传清理改为批量 UPDATE + 异步存储删除 | 减少 DB commit 频率 |
| 20 | 路由动画 0.5s → 0.15s 或纯 CSS | 导航感知速度 ↑ |
| 21 | 服务导入收窄（直接 import 具体 Service） | 每路由 bundle ↓ 10–30KB |
| 22 | Admin 路由级 `loading.tsx` + Suspense | 渐进式渲染体验 |
| 23 | 缩略图 `loading="lazy"` + 固定尺寸 | 文件管理页初始 paint 加速 |

---

## 已做得好的设计

以下设计说明团队已有性能意识，优化应在此基础上增量改进，**不必重复造轮子**：

| # | 设计 | 位置 |
|---|------|------|
| 1 | 系统配置单 key Redis Hash 缓存 | `internal/model/system_configs.go` — `GetByKey` |
| 2 | Storage Backend 单例 + 5s TTL + pub/sub 失效 | `internal/storage/storage.go` — `Active()` |
| 3 | 推送事件/渠道 24h Redis 缓存 + GORM hook 失效 | `internal/model/push_event.go`, `push_channel.go` |
| 4 | 风控日志异步批写 ClickHouse（1 万缓冲 + 1000 条/1s + 429 背压） | `internal/apps/risk_control/` |
| 5 | HTTP 连接池统一（`httppool` + OTel） | `pkg/httppool/` |
| 6 | DB/Redis 连接池显式配置 | `config.yaml`, `internal/db/` |
| 7 | 游标分批处理（`id > ? LIMIT n`） | `cleanup.go`, image warmup |
| 8 | 存储迁移并发上限 `errgroup.SetLimit(10)` | `storage_migration_task.go` |
| 9 | 邮件/推送走 Asynq，不在 HTTP 路径同步发送 | `user/logics.go`, `push/events.go` |
| 10 | 文件服务 ETag/304 + 原图 `DataFromReader` 流式返回 | `file_server.go` |
| 11 | 无 GORM `Preload` 滥用 | 全项目 |
| 12 | 前端 API 请求去重（`pendingRequests` Map） | `frontend/lib/services/core/api-client.ts` |
| 13 | React Query 全局 30s `staleTime` | `frontend/components/providers/query-provider.tsx` |
| 14 | React Compiler 已启用 | `frontend/next.config.ts` |
| 15 | 读副本支持（`dbresolver`） | `internal/db/postgres.go` |
| 16 | 任务执行日志 Redis 缓冲 + 批量回写 | `internal/model/task_execution.go` |

---

## 场景风险矩阵

| 场景 | 最可能爆的点 | 对应优先级 |
|------|-------------|-----------|
| 图片站 / 公开相册 | WebP miss + 磁盘锁 + 白名单 Redis | P0 #1, #2, #3 |
| 文件量 10 万+ | 统计全表扫描 + 索引缺失 + 清理慢 | P0 #4, #5 |
| 管理端日常使用 | 认证瀑布 + 大 bundle + 日志 DOM | P0 #6, #7 |
| 存储迁移进行中 | 迁移状态重复查 + Redis 日志风暴 | P1 #2, P2 #17 |
| 登录高峰 | CAPTCHA 5×Redis + OIDC discovery | P1 #10, #11 |
| 多租户 / 跨域前端 | CORS 配置查询 + 公共配置无缓存 | P1 #9 |
| 批量文件操作 | ZIP 同步打包无上限 | P0 #5, P1 #12 |

---

## 优先行动清单

如果只选 **3 件事** 先做（预计用户感知延迟降低 50–70%）：

1. **WebP 路径解耦** — `singleflight` + `RWMutex` + 部署后预热
2. **文件路径查询缓存** — 迁移状态 + 白名单进程内缓存
3. **前端认证与首屏并行化** — 消除全局 auth gate + Admin 代码分割

### 实施检查清单

```
P0 后端
[ ] disk cache RWMutex + singleflight
[ ] StorageReadOnly 5s 缓存 + pub/sub 失效
[ ] file_access_whitelist 进程内缓存
[ ] GetFileStats SQL 聚合改写
[ ] w_uploads 复合索引 migration
[ ] 批量下载数量上限

P0 前端
[ ] app-logs.tsx 虚拟滚动
[ ] SQLConsole / Recharts / Settings Tabs dynamic import
[ ] 认证 gate 并行化

P1
[ ] ListVisibleSystemConfigs Redis 缓存
[ ] CAPTCHA 配置快照
[ ] OIDC Provider 缓存
[ ] Admin useEffect → React Query 统一
[ ] 状态轮询 visibility 感知
```

---

## 附录：关键代码路径索引

| 路径 | 文件 | 说明 |
|------|------|------|
| 图片服务 | `internal/apps/upload/file_server.go` | `/f/{id}` 热点 |
| 磁盘缓存 | `pkg/cache/disk/cache.go` | 全局 Mutex |
| 迁移状态 | `internal/apps/upload/storage_ops.go` | 无缓存 DB 查询 |
| 文件统计 | `internal/apps/upload/stats.go` | 全表扫描 |
| 批量下载 | `internal/apps/upload/routers.go` | 同步 ZIP |
| 上传索引 | `internal/db/migrator/goose/*/202606090001_initial_schema.sql` | 缺失复合索引 |
| 认证 gate | `frontend/app/(main)/layout.tsx` | 阻塞渲染 |
| 用户上下文 | `frontend/contexts/user-context.tsx` | 挂载时 fetch |
| 实时日志 | `frontend/components/common/admin/app-logs.tsx` | 无虚拟化 |
| API 去重 | `frontend/lib/services/core/api-client.ts` | 已有，可复用模式 |