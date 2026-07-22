# AGENTS.md — Wavelet AI 助手工作操作手册

本文件面向 AI 开发助手，定义其职责与操作规范。

## Git 提交规范

遵循 Conventional Commits：`<type>(<scope>): <subject>`（例：`feat(auth): support email login`）。

## 务必阅读匹配的 Skill

| Skill | 何时使用 |
| :--- | :--- |
| `new-api` | 添加或修改自定义业务 API、Handler、服务层逻辑、自定义路由注册 |
| `new-async-task` | 添加或修改 Asynq 任务、定时任务、TaskHandler、任务元数据 |
| `new-setting` | 添加或修改系统/业务/公开设置、`/admin/system` 参数或 `/admin/settings` 图形化设置 |
| `database-migration` | 数据库表结构变更、goose SQL 迁移（PG/SQLite/ClickHouse）、seed 数据 |
| `clickhouse-batchwriter` | ClickHouse 批量写入、`internal/db/batchwriter` 接入、分析表异步 flush、背压与写入路径改造 |
| `file-upload` | 业务上传文件、Worker 程序化摄取、`upload.Ingest` 策略选型、文件访问与 `w_uploads` / 统计排查 |
| `cache-framework` | 新增或修改业务缓存（RAM/Redis/DB 三层读路径）、缓存失效、多节点 pub/sub 同步、评估高频读是否应接入缓存 |
| `push-notification` | 系统通知推送事件、统一触发器投递、带消息推送的业务功能 |
| `release-guide` | 根据自上一正式版本 Tag 以来的提交整理 Version Bump 提交信息以触发双语 Release |
| `shadcn` | 添加、修改或组合 shadcn/ui 组件 |

## 严格遵循事项 (Guardrails)

- 切勿删除 `frontend/node_modules`。
- 保持 `internal/util/` 绝对纯净，禁止导入 Gin、GORM、sessions 等 Web/数据库框架包。
- 测试用例禁止硬编码相对路径创建临时目录，统一使用 Go 内置 `t.TempDir()`。
- 所有 HTTP 路由仅在 `internal/router/router.go` 中作为高层分发注册。
- 修改 API Handler 后运行 `make swagger`，完成代码开发后必须依次运行 `make code-check` 与 `make format`。
- 业务模块必须复用平台缓存/文件服务：文件摄取统一用 `upload.Ingest`，删除用 `upload.Remove`/`upload.RemoveOwned`；禁止直接写 `w_uploads` 或操作底层 storage。
- 禁止在 `init()` 中注册跨模块集成（任务 Handler、推送事件、域事件监听器等），统一在 `internal/bootstrap` 显式装配并在 `internal/cmd` 入口调用。
- 核心业务模块（`oauth`、`user`）禁止直接 import `push` 或 `custom_events` 触发通知，须通过 `internal/listener` 发射域事件。
- API 错误响应必须通过 `response.Abort*` 中断请求，由 `ErrorHandlerMiddleware` 统一写出 JSON 并记录 Trace；禁止在 Handler/中间件中直接 `c.JSON(status, response.Err(...))` 或 `200` 返回 `error_msg`。

## 技术栈与项目目录结构

### 技术栈
- **后端**：Go 1.25+、Gin、GORM、PostgreSQL、ClickHouse、Redis、Asynq、Cobra、Viper、Swaggo、OpenTelemetry、Zap、AWS SDK v2。
- **前端**：Next.js (App Router)、TypeScript、Tailwind CSS、pnpm、shadcn/ui。

### 顶层目录
- `main.go`：程序入口，委派给 `internal/cmd`。
- `config.example.yaml` / `config.yaml`：配置文件模板与本地配置。
- `docker/`：容器化部署 Dockerfile。
- `docs/`：自动生成的 Swagger 文档（请勿手动编辑）。
- `frontend/`：Next.js 前端应用。
- `internal/`：后端核心私有代码。
- `pkg/`：公共通用 Go 工具库（不包含具体业务）。
- `scripts/`：本地开发与 CI 脚本。
- `support-files/`：部署与 SQL/环境辅助文件。
- `bin/` / `data/` / `uploads/`：编译二进制产物、本地数据文件与上传存储目录。

### 后端目录 (`internal/`)
- `internal/cmd/`：Cobra CLI 命令入口（API/Worker/Scheduler）。
- `internal/bootstrap/`：应用装配根，集中注册 Task、推送订阅、域事件监听器及进程级初始化。
- `internal/config/`：Viper 配置加载与映射结构体。
- `internal/router/`：全局唯一 HTTP 路由注册点。
- `internal/apps/`：按功能（Feature-based）划分模块的 Handler 与业务逻辑（管理端位于 `admin/`）。
- `internal/apps/upload/`：文件上传服务、访问控制与 WebP 压缩。
- `internal/model/`：GORM 数据模型定义与模型层方法。
- `internal/db/`：PostgreSQL/Redis/ClickHouse 连接池与 goose SQL 迁移文件（`db/migrator/goose/`）。
- `internal/diskcache/`：平台级磁盘字节缓存（`diskcache.GetGlobalCache()`）。
- `internal/storage/`：S3 对象存储适配器。
- `internal/task/`：Asynq 异步任务定义。
- `internal/common/`：全局共享模型、统一响应（`response`）、绑定助手与通用错误。
- `internal/util/`：纯底层无框架依赖工具函数。
- `internal/listener/`：域事件分发层（解耦业务域与运维/推送模块）。
- `internal/otel_trace/`：OpenTelemetry 链路追踪助手。
- `internal/testhelper/`：后端测试共享 Helper。
- `internal/buildinfo/`：编译与构建元数据。

### 公共底层包 (`pkg/`)
- `pkg/cache/disk/`：纯底层磁盘缓存引擎。
- `pkg/cap/`：通用验证码库。
- `pkg/httppool/`：带 OTel 链路追踪的共享 HTTP 客户端连接池。
- `pkg/logger/`：Zap / OTel 结构化日志工具。
- `pkg/push/`：推送渠道 SDK 集成（Lark / Telegram / Email）。
- `pkg/mail/`：邮件发送客户端。
- `pkg/trace/`：OpenTelemetry 链路配置。
- `pkg/util/`：无副作用系统工具（Crypto / Password / UUID 等）。

### 前端目录 (`frontend/`)
- `frontend/app/`：Next.js App Router 路由与页面。
- `frontend/components/ui/`：shadcn/ui 基础通用组件。
- `frontend/components/common/`：跨页面的业务通用组件。
- `frontend/components/layout/`：Header / Sidebar / Footer 页面框架组件。
- `frontend/components/<feature>/`：特定业务域的 UI 组件（如 `auth/`、`home/`）。
- `frontend/lib/services/`：基于 `BaseService` 继承的类型化前端 API 服务。
- `frontend/contexts/` / `hooks/` / `lib/` / `types/` / `public/`：全局状态、Hook、客户端工具、TS 类型定义与静态资源。

## 后端开发规范

### API 响应规范
- **统一信封**：`{ "error_msg": "", "data": ... }`
- **成功**：HTTP 200，写出 `c.JSON(http.StatusOK, response.OK(data))` 或 `response.OKNil()`。
- **失败**：使用 `internal/common/response` 的 `Abort*` 系列函数（如 `AbortBadRequest`、`AbortUnauthorized`、`AbortNotFound`、`AbortInternal`）中断请求。
- **错误文案**：使用模块内 `errs.go` 中的 camelCase 字符串常量（如 `errBindParamsFailed`），禁止暴露底层数据库/系统错误细节给客户端。
- **Logics 分工**：`logics.go` 只接受 `context.Context`，返回 `(result, error)`，严禁依赖 `*gin.Context` 或调用 `c.JSON`/`Abort*`。
- **错误日志**：底层错误在 Handler/Logic 边界用 `pkg/logger` 打印日志，禁止使用 `_ = ...` 静默吞掉关键错误。

### 数据库操作
- 管理员代码推荐使用 `db.DB(ctx)` 保证 Trace 链路透传。
- 禁止在 Handler 写复杂 SQL；迁移文件位于 `internal/db/migrator/goose/`（禁止 GORM AutoMigrate）。
- 不创建物理外键（显式建索引）；Go 模型零值需与数据库默认值匹配。

## 前端开发规范

- 新特性开发前参考 Next.js 文档与 `frontend/app/(main)/admin/demo` 示例代码。
- **页面容器与标题栏**：
    - 页面根容器统一使用全宽 `w-full`，最外层统一用 `py-6` 或 `py-6 px-1` 对齐边距。
    - 标题容器统一 `flex items-center gap-2`（带操作按钮用 `justify-between`）。
    - 图标直接使用 Lucide 组件（`size-5 text-primary`），禁止包裹背景小卡片或装饰边框。
    - 标题文字统一使用 `<h1 className="text-2xl font-semibold tracking-tight">`。
- **组件拆分与维护**：
    - 物理路由页面 `page.tsx` 仅维护高级骨架与布局。
    - 单文件超过 600 行或含多 Tab/大复杂区块时，必须按就近原则拆分为子组件存放在路由同级的 `components/` 局部目录中（参考 `/admin/database` 的模块化拆分结构）。
- **样式与服务**：
    - 优先使用 shadcn/ui 的 `variant` 和全局 CSS 变量，不要在业务代码中硬编码颜色/背景。
    - 前端请求统一在 `frontend/lib/services/<name>/` 中继承 `BaseService` 编写并在 `index.ts` 注册。
