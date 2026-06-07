# 项目开发规范

> 本文档面向 AI 代理（Agent）和开发者，描述项目的目录结构、模块职责与开发规范。

---

## 一、技术栈

### 后端

| 技术 | 用途 |
|------|------|
| Go (1.25+) | 主语言 |
| Gin | HTTP 框架 |
| GORM | ORM，主库 PostgreSQL，可选 ClickHouse |
| Redis | 缓存 / Session / 队列 |
| Asynq | 异步任务队列（基于 Redis） |
| Cobra + Viper | CLI 入口 + 配置加载 |
| Swaggo | Swagger 文档生成 |
| OpenTelemetry | 链路追踪 |
| Zap | 结构化日志 |
| AWS SDK v2 | S3 兼容文件存储 |
| Snowflake | 分布式 ID 生成 |

### 前端

| 技术 | 用途 |
|------|------|
| Next.js (App Router) | 前端框架 |
| TypeScript | 主语言 |
| Tailwind CSS | 样式 |
| pnpm | 包管理 |
| shadcn/ui | 组件库 |

---

## 二、顶层目录结构

```
Refreshing/                        # 项目根目录（模块名: github.com/linux-do/credit）
├── main.go                        # 程序入口，调用 internal/cmd
├── go.mod / go.sum                # Go 模块依赖
├── config.yaml                    # 运行时配置（不提交到 Git）
├── config.example.yaml            # 配置模板（需提交）
├── Makefile                       # 常用命令（swagger/tidy/license）
├── Dockerfile                     # 后端容器镜像构建
├── .editorconfig                  # 编辑器格式规范
├── .gitignore
├── docs/                          # Swagger 自动生成文档（不要手动编辑）
├── frontend/                      # Next.js 前端项目
├── internal/                      # 后端核心代码（Go private，不对外暴露）
├── scripts/                       # CI/本地工具脚本
└── support-files/                 # 辅助文件（如 nginx 配置等）
```

---

## 三、后端 `internal/` 目录结构

```
internal/
├── cmd/                           # CLI 命令入口（Cobra）
│   ├── root.go                    # 根命令，加载配置、初始化依赖
│   ├── api.go                     # 启动 HTTP API 服务器子命令
│   ├── scheduler.go               # 启动定时任务调度器子命令
│   └── worker.go                  # 启动 Asynq Worker 子命令
│
├── config/                        # 配置加载与结构定义
│   ├── model.go                   # 所有配置结构体（AppConfig / DB / Redis 等）
│   └── config.go                  # Viper 加载逻辑，暴露全局 config.Config
│
├── router/                        # HTTP 路由注册（唯一路由注册点）
│   ├── router.go                  # 路由总入口，注册所有分组路由、中间件、启动 HTTP Server
│   └── middlewares.go             # 全局中间件（如请求日志）
│
├── apps/                          # 业务功能模块（按功能域划分）
│   ├── oauth/                     # OAuth / OIDC 登录、会话、用户信息
│   ├── user/                      # 用户密码登录、注册、登出
│   ├── upload/                    # 文件上传、文件服务、清理任务
│   ├── health/                    # 健康检查端点
│   ├── config/                    # 公开配置接口（前端读取）
│   └── admin/                     # 管理后台功能（需 Admin 权限）
│       ├── middlewares.go          # Admin 鉴权中间件
│       ├── errs.go                 # Admin 错误常量
│       ├── auth_source/           # 认证源管理（CRUD）
│       ├── system_config/         # 系统配置管理（CRUD）
│       ├── task/                  # 任务手动调度接口
│       └── user/                  # 用户管理（列表、状态）
│
├── model/                         # 数据模型（GORM 实体 + 业务方法）
│   ├── users.go                   # User 实体、OAuthUserInfo、查询/更新方法
│   ├── auth_source.go             # AuthSource 实体（OAuth 接入源）
│   ├── system_configs.go          # SystemConfig 实体（KV 系统配置）
│   └── uploads.go                 # Upload 实体（上传文件记录）
│
├── db/                            # 数据库连接与基础设施
│   ├── postgres.go                # PostgreSQL 初始化、读写分离、GORM 配置
│   ├── redis.go                   # Redis 初始化（单机/哨兵/集群）
│   ├── clickhouse.go              # ClickHouse 初始化（可选）
│   ├── postgres_logger.go         # 自定义 GORM 日志（对接 Zap）
│   ├── idgen/                     # Snowflake 分布式 ID 生成器
│   └── migrator/                  # 数据库迁移（AutoMigrate）
│
├── storage/                       # 文件存储抽象层
│   ├── s3.go                      # S3 兼容存储（上传/下载/URL 生成）
│   ├── cache.go                   # 本地磁盘缓存（S3 内容缓存）
│   └── errs.go                    # 存储层错误常量
│
├── task/                          # 异步任务定义与调度
│   ├── constants.go               # 任务类型名称常量（TaskType）、队列名
│   ├── utils.go                   # 任务工具函数（RedisOpt 等）
│   ├── scheduler/                 # Asynq 定时任务调度器（Cron 注册）
│   └── worker/                    # Asynq Worker 服务端（任务处理器注册）
│       ├── worker.go              # StartWorker 入口，注册 Handler
│       └── middlewares.go         # Worker 中间件（日志等）
│
├── service/                       # 复杂业务逻辑服务层（当前占位，待填充）
│
├── common/                        # 跨模块共享代码
│   ├── constants.go               # 全局常量（错误消息字符串等）
│   ├── errs.go                    # 通用错误定义
│   ├── bind/                      # 请求参数绑定封装（统一处理错误响应）
│   └── response/                  # 统一 HTTP 响应格式封装
│
├── util/                          # 无业务依赖的纯工具函数
│   ├── crypto.go                  # 加密/签名工具
│   ├── password.go                # 密码 Hash（bcrypt）
│   ├── http_clients.go            # HTTP 客户端封装
│   ├── context.go                 # Context 存取工具
│   ├── response.go                # ResponseAny 等响应结构体
│   ├── session.go                 # Session 选项构建
│   ├── uuid.go                    # UUID / 唯一 ID 生成
│   ├── strings.go                 # 字符串工具
│   ├── validate.go                # 参数校验工具
│   └── custom_types.go            # 自定义类型
│
├── logger/                        # 日志封装（基于 Zap + OTel）
│   ├── logger.go                  # 全局 Logger 初始化
│   └── utils.go                   # InfoF / WarnF / ErrorF 快捷函数
│
├── listener/                      # 事件监听器（Webhook / 消息消费）
│
└── otel_trace/                    # OpenTelemetry 链路追踪封装
    └── ...                        # Span 创建、Exporter 配置
```

---

## 四、`apps/` 模块内部文件规范

每个业务模块（`apps/<module>/`）内部按照以下约定组织文件：

| 文件名 | 职责 |
|--------|------|
| `routers.go` | **HTTP Handler 函数**（业务逻辑入口，对应 Controller 层）|
| `controllers.go` | 可选，当 Handler 较多时拆分（同 `routers.go` 职责）|
| `middlewares.go` | 本模块专属中间件（如 `LoginRequired`、`LoginAdminRequired`）|
| `errs.go` | 本模块专属错误消息字符串常量（`const`）|
| `constants.go` | 本模块专属业务常量（非错误）|

> **规则**：  
> - 路由 **不在** 模块内部注册，统一在 `internal/router/router.go` 中注册。  
> - `errs.go` 只定义字符串常量，不定义 `error` 类型值，错误通过 `response.RespondFailure(c, errMsg)` 输出。

### `admin/` 子模块结构示例

```
apps/admin/
├── middlewares.go          # LoginAdminRequired 中间件
├── errs.go                 # admin 级别错误常量
├── auth_source/            # 认证源 CRUD
│   └── routers.go
├── system_config/          # 系统 KV 配置 CRUD
│   └── routers.go
├── task/                   # 任务调度接口
│   └── routers.go
├── user/                   # 用户管理
│   ├── routers.go
│   └── errs.go
└── user_pay_config/        # 用户支付配置
    └── routers.go
```

---

## 五、前端 `frontend/` 目录结构

```
frontend/
├── app/                           # Next.js App Router 页面目录
│   ├── layout.tsx                 # 根布局（全局 Provider、字体、meta）
│   ├── globals.css                # 全局样式
│   ├── page.tsx                   # 首页重定向
│   ├── (auth)/                    # 认证相关页面组（登录/注册/OAuth 回调）
│   ├── (main)/                    # 主应用页面组（用户界面）
│   └── (docs)/                    # 文档类页面组
│
├── components/                    # 可复用 React 组件
│   ├── ui/                        # shadcn/ui 基础组件（Button/Input/Dialog 等）
│   ├── common/                    # 通用业务组件（跨页面复用）
│   ├── layout/                    # 布局组件（Header / Sidebar / Footer）
│   ├── auth/                      # 认证相关组件
│   ├── home/                      # 首页专属组件
│   ├── animate-ui/                # 动画 UI 组件
│   └── providers/                 # Context Provider 组件
│
├── contexts/                      # React Context（全局状态）
├── hooks/                         # 自定义 React Hooks
├── lib/                           # 前端工具函数、API 客户端封装
├── types/                         # TypeScript 类型定义
├── public/                        # 静态资源
├── proxy.ts                       # 开发环境代理配置
├── next.config.ts                 # Next.js 配置
├── package.json
├── tsconfig.json
├── .env                           # 环境变量（不提交）
└── .env.example                   # 环境变量模板（需提交）
```

---

## 六、开发规范

### 6.1 命名规范

| 对象 | 规范 | 示例 |
|------|------|------|
| Go 包名 | 小写，下划线分词（单词） | `auth_source`、`system_config` |
| Go 文件名 | 小写，下划线分词 | `routers.go`、`postgres_logger.go` |
| Go 导出函数 | PascalCase | `ListUsers`、`StartWorker` |
| Go 未导出函数 | camelCase | `buildQueuesFromConfig` |
| Go 结构体请求/响应 | camelCase + 后缀 | `listUsersRequest`、`listUsersResponse` |
| 错误常量 | camelCase 字符串 `const` | `const userNotFound = "用户不存在"` |
| 任务类型常量 | 全大写蛇形 | `CleanupUnusedUploadsTask` |
| 配置 Key | 全小写蛇形（YAML） | `session_cookie_name`、`max_idle_conn` |

### 6.2 HTTP Handler 规范

```go
// Handler 函数命名：动词 + 名词（PascalCase）
func ListUsers(c *gin.Context) {
    // 1. 参数绑定（使用 ShouldBindQuery / ShouldBindJSON）
    var req listUsersRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(http.StatusBadRequest, util.Err(err.Error()))
        return
    }

    // 2. 业务逻辑

    // 3. 统一响应
    c.JSON(http.StatusOK, util.OK(data))
}
```

**响应格式约定**：
- 成功：`util.OK(data)` 或 `util.OKNil()`  
- 失败：`util.Err(msg)` + 对应 HTTP 状态码  
- 通过 `response.RespondSuccess / RespondFailure` 也可（两套工具共存）

### 6.3 Swagger 注释规范

所有对外 Handler 必须添加 Swaggo 注释：

```go
// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 分页返回用户列表，支持按用户 ID 和用户名筛选，需要管理员权限
// @Tags admin
// @Produce json
// @Param request query listUsersRequest true "查询参数"
// @Success 200 {object} util.ResponseAny
// @Router /api/v1/admin/users [get]
func ListUsers(c *gin.Context) { ... }
```

生成文档：`make swagger`（执行 `scripts/swagger.sh`）

### 6.4 错误处理规范

- **模块内错误消息**：定义在本模块 `errs.go` 中，使用 `const` 字符串。
- **跨模块错误消息**：定义在 `internal/common/errs.go` 或 `common/constants.go`。
- **数据库错误**：直接 `err.Error()` 返回给响应（开发阶段），生产环境应屏蔽详情。
- **gorm.ErrRecordNotFound**：显式判断，返回 404。

### 6.5 中间件使用规范

| 中间件 | 位置 | 作用 |
|--------|------|------|
| `gin.Recovery()` | 全局 | Panic 恢复 |
| `otelgin.Middleware()` | 全局 | OTel 链路追踪 |
| `loggerMiddleware()` | 全局 | 请求日志 |
| `sessions.Sessions()` | 全局 | Session 注入 |
| `oauth.LoginRequired()` | 路由组 | 登录校验 |
| `admin.LoginAdminRequired()` | Admin 路由组 | 管理员校验 |

### 6.6 配置访问规范

- 所有配置通过 `config.Config.<Section>.<Field>` 访问（全局单例）。
- 不允许在业务代码中使用 `os.Getenv()` 读取配置，统一通过 Viper 加载。
- 新增配置项：先在 `config.example.yaml` 添加注释模板，再在 `internal/config/model.go` 添加结构体字段。

### 6.7 数据库访问规范

- 直接使用 GORM：`model.DB.Where(...).Find(&result)`（适合简单查询）。
- 通过 `db.DB(ctx)` 获取带链路追踪的 DB 实例（Admin 模块推荐）。
- 禁止在 Handler 层直接写复杂 SQL，应封装到 `model/` 层方法或 `service/` 层。
- 数据库迁移使用 `db/migrator/` 中的 AutoMigrate，不允许手动执行 DDL。

### 6.8 异步任务规范

**定义任务**：
1. 在 `internal/task/constants.go` 中定义任务类型常量。
2. 实现 Handler 函数（放在对应 `apps/` 模块的 `tasks.go` 文件中）。
3. 在 `internal/task/worker/worker.go` 中注册 Handler：`mux.HandleFunc(task.XxxTask, handler)`。
4. 调度：在 `internal/task/scheduler/` 中按 Cron 表达式调度，或通过 Admin API 手动触发。

**队列优先级**（从高到低）：`webhook` > `whitelist_only` > `default`

---

## 七、新增功能开发流程

以新增 **管理员功能模块** 为例：

```
1. 在 internal/model/ 中定义/扩展数据模型
2. 在 db/migrator/ 中注册 AutoMigrate
3. 在 internal/apps/admin/<module>/ 中创建：
   - routers.go    （Handler 实现 + Swagger 注释）
   - errs.go       （错误常量，按需）
4. 在 internal/router/router.go 中注册路由
5. 执行 make swagger 更新文档
```

以新增 **异步任务** 为例：

```
1. 在 internal/task/constants.go 定义任务类型常量
2. 在对应 apps/<module>/tasks.go 实现 Handle 函数
3. 在 internal/task/worker/worker.go 注册 Handler
4. 在 internal/task/scheduler/ 添加 Cron 调度（或 Admin API 手动触发）
5. 在 config.example.yaml 的 scheduler 段添加 Cron 配置项
6. 在 internal/config/model.go 添加配置字段
```

---

## 八、关键依赖版本

| 依赖 | 版本 |
|------|------|
| Go | 1.25+ |
| Gin | v1.11.0 |
| GORM | v1.31.1 |
| go-redis | v9.16.0 |
| Asynq | v0.25.1 |
| Cobra | v1.10.1 |
| Viper | v1.21.0 |
| Zap | v1.27.0 |
| Snowflake | v0.3.0 |
| OpenTelemetry | v1.36.0 |
| Next.js | (见 frontend/package.json) |
| pnpm | (见 frontend/pnpm-workspace.yaml) |
