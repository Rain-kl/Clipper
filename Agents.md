# 项目开发规范

> 本文档面向 AI 代理（Agent）与开发者，描述项目的目录结构、模块职责、代码规范与开发流程。

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

## 二、项目目录结构

### 2.1 顶层目录结构

以下是项目的顶层目录结构及其职责：

```
wavelet/                           # 项目根目录（模块名: github.com/Rain-kl/Wavelet）
├── main.go                        # 程序入口，调用 internal/cmd
├── go.mod / go.sum                # Go 模块依赖
├── config.yaml                    # 运行时配置（不提交到 Git）
├── config.example.yaml            # 配置模板（需提交）
├── DEPLOYMENT_zh.md               # 部署说明文档（中文版）
├── Makefile                       # 常用命令（swagger/tidy/license/code-check）
├── docker/                        # Docker 镜像构建文件（集成/前端/后端）
│   ├── Dockerfile                 # 标准集成镜像（前端静态导出嵌入后端）
│   ├── Dockerfile.frontend        # 仅前端镜像（Next.js）
│   └── Dockerfile.backend         # 仅后端镜像（Go API/Worker/Scheduler）
├── docker-compose.yml             # 本地依赖服务（PostgreSQL / Redis / ClickHouse）
├── .editorconfig                  # 编辑器格式规范
├── .gitignore
├── docs/                          # Swagger 自动生成文档（不要手动编辑）
├── frontend/                      # Next.js 前端项目
├── internal/                      # 后端核心代码（Go private，不对外暴露）
├── scripts/                       # CI/本地工具脚本
└── support-files/                 # 辅助文件（如 nginx 配置等）
```

### 2.2 后端 `internal/` 目录结构

以下是 `internal/` 目录的结构及其职责：

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
│   ├── uploads.go                 # Upload 实体（上传文件记录）
│   └── task_execution.go          # TaskExecution 实体（异步任务执行记录 + CRUD）
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
│   ├── constants.go               # 任务类型名称常量（TaskType）、队列名、TaskMeta（含 Retryable）
│   ├── handler.go                 # TaskHandler 接口定义 + TaskResult 结构体
│   ├── executor.go                # 核心运行机制：RegisterHandler / DispatchTask / ProcessTask / RetryTask / AppendLog
│   ├── utils.go                   # 任务工具函数（RedisOpt、AsynqClient）
│   ├── scheduler/                 # Asynq 定时任务调度器（Cron 注册）
│   └── worker/                    # Asynq Worker 服务端（任务处理器注册）
│       ├── worker.go              # StartWorker 入口，注册 Handler
│       └── middlewares.go         # Worker 中间件
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

### 2.3 `apps/` 业务模块文件规范

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

#### `admin/` 子模块结构示例：

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

### 2.4 前端 `frontend/` 目录结构

以下是前端 `frontend/` 目录的结构：

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
│   ├── common/                    # 通用业务组件（跨页面复用），详见下方说明
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

### 2.5 前端 `components/common/` 通用业务组件详解

`common/` 目录存放跨页面复用的业务组件，按功能域分为五个子目录：

```
components/common/
├── admin/                          # 管理员后台组件
│   ├── tasks.tsx                   # TaskManager — 异步任务调度管理页面，展示所有可用任务类型，
│   │                               #   支持通过弹窗配置参数后立即下发任务到后台队列执行
│   ├── task-executions.tsx         # TaskExecutionsManager — 任务日志页面，展示异步任务执行记录，
│   │                               #   支持状态/类型筛选、分页、详情抽屉查看完整日志与失败任务重试
│   ├── system.tsx                  # SystemConfigs — 系统 KV 配置管理页面，以表格展示系统/业务两类
│   │                               #   配置项，支持在线编辑（布尔类型自动渲染为 Switch）并保存/删除
│   └── users.tsx                   # UsersManager — 用户管理页面，提供分页、搜索、筛选的用户列表表格，
│                                   #   支持在侧边抽屉查看用户详情，以及启用/禁用（封禁/解封）切换
│
├── docs/                           # 文档页面组件，包括法律文档（隐私政策/服务条款）和接口文档
│
├── general/                        # 通用框架组件
│   ├── manage-pannel.tsx           # ManagePage（泛型）— 通用管理页面框架，封装"列表 + 详情面板"布局，
│   │                               #   包含数据加载/错误/空状态处理、表格渲染、选中/悬停交互、
│   │                               #   编辑/保存/删除逻辑；ManageDetailPanel 为带保存按钮的详情面板；
│   │                               #   ManageTable 为配置驱动型表格组件
│   └── password-dialog.tsx         # PasswordDialog — 密码确认弹窗，用于敏感操作前的二次身份验证，
│                                   #   包含 6 位 OTP 输入框，支持 Enter 快捷确认，带加载状态显示
│
├── home/                           # 首页组件
│   └── home-main.tsx               # HomeMain — 系统首页主内容，展示当前用户的快捷导航卡片
│                                   #   （个人资料、开发接口文档、使用文档），管理员额外显示后台管理入口
│
└── settings/                       # 设置页面组件
    ├── access-token.tsx            # AccessTokenMain — 个人访问令牌管理页面，展示用户 API 密钥列表，
    │                               #   支持创建（仅展示一次明文）、轮换、撤销/删除令牌
    ├── appearance.tsx              # AppearanceMain — 外观设置页面，分为主题模式选择
    │                               #   （明亮/黑暗/自动）和界面配色方案（可视化色卡网格切换）
    ├── auth-source-modal.tsx       # AuthSourceModal — OIDC 认证源新增/编辑弹窗，包含标识符、
    │                               #   Client ID/Secret、Discovery URL、Scopes、图标等表单字段
    ├── notifications.tsx           # NotificationsMain — 通知设置页面，控制顶部导航栏
    │                               #   是否显示通知铃铛图标，通过 Context 持久化偏好
    ├── profile.tsx                 # ProfileMain — 个人资料页面，展示用户基本信息，提供第三方
    │                               #   账号绑定管理（查看已绑定 OIDC 账号、解除绑定、绑定新认证源）
    ├── system-settings.tsx         # SystemSettingsMain — 系统设置主页面（管理员专属），包含系统安全与登录控制、
                                    #   认证源管理、人机验证配置、邮件服务 (SMTP) 设置以及菜单显示控制
```

---

## 三、后端开发规范

### 3.1 命名规范

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

### 3.2 HTTP Handler 规范

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
- 通过 `response.RespondSuccess / RespondFailure` 均可（两套工具共存）

### 3.3 中间件使用规范

| 中间件 | 位置 | 作用 |
|--------|------|------|
| `gin.Recovery()` | 全局 | Panic 恢复 |
| `otelgin.Middleware()` | 全局 | OTel 链路追踪 |
| `loggerMiddleware()` | 全局 | 请求日志 |
| `sessions.Sessions()` | 全局 | Session 注入 |
| `oauth.LoginRequired()` | 路由组 | 登录校验 |
| `admin.LoginAdminRequired()` | Admin 路由组 | 管理员校验 |

### 3.4 配置访问规范

- 所有配置通过 `config.Config.<Section>.<Field>` 访问（全局单例）。
- 不允许在业务代码中使用 `os.Getenv()` 读取配置，统一通过 Viper 加载。
- 新增配置项：先在 `config.example.yaml` 添加注释模板，再在 `internal/config/model.go` 添加结构体字段。

### 3.5 数据库访问规范

- 直接使用 GORM：`model.DB.Where(...).Find(&result)`（适合简单查询）。
- 通过 `db.DB(ctx)` 获取带链路追踪的 DB 实例（Admin 模块推荐）。
- 禁止在 Handler 层直接写复杂 SQL，应封装到 `model/` 层方法或 `service/` 层。
- 数据库迁移使用 `db/migrator/` 中的 AutoMigrate，不允许手动执行 DDL。

### 3.6 异步任务规范

**定义任务**：
1. 在 `internal/task/constants.go` 中定义任务类型常量。
2. 实现 Handler 函数（放在对应 `apps/` 模块的 `tasks.go` 文件中）。
3. 在 `internal/task/worker/worker.go` 中注册 Handler：`mux.HandleFunc(task.XxxTask, handler)`。
4. 调度：在 `internal/task/scheduler/` 中按 Cron 表达式调度，或通过 Admin API 手动触发。

**队列优先级**（从高到低）：`webhook` > `whitelist_only` > `default`

---

## 四、前端开发规范

### 4.1 组件样式规范

**基础组件必须遵循系统的色彩主题系统。** 所有基于 shadcn/ui 的基础组件（Button、Dialog、Input 等）应使用组件内置 of `variant` 属性来控制样式，禁止通过 `className` 手写颜色或背景等样式。

**错误示例（禁止）**：
```tsx
// ❌ 禁止通过 className 手写颜色、背景、阴影等样式
<Button
  type="button"
  size="sm"
  className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-md shadow-indigo-600/10 transition-colors"
>
  <Plus className="mr-1.5 size-3.5" />
  新增认证源
</Button>
```

**正确示例**：
```tsx
// ✅ 使用 variant 属性，让组件遵循系统主题
<Button
  type="button"
  size="sm"
  variant="secondary"
>
  <Plus className="mr-1.5 size-3.5" />
  新增认证源
</Button>
```

> **原则**：组件的视觉表现由 shadcn/ui 的 variant 系统和全局 CSS 变量统一控制，保持应用内所有页面风格一致。如现有 variant 无法满足需求，应扩展 shadcn/ui 组件的 variant 定义，而非在业务代码中硬编码颜色值。

### 4.2 页面宽度自适应规范

**开发或更新前端页面时，页面主容器必须支持全宽（Full Width）自适应。** 页面组件的根容器禁止硬编码固定最大宽度（如 `max-w-6xl`、`max-w-4xl` 等），而应统一使用 `w-full`。

由于系统主布局（`MainLayout`）已包含全局 "切换全宽" 状态与按钮，页面主容器不设最大宽度即可让页面宽度完美跟随全局状态。默认情况下由主布局约束在正常宽度限制内，开启全宽后能自动拉伸至 `100%`。

**错误示例（禁止限制宽度）**：
```tsx
// ❌ 禁止在页面外层组件硬编码 max-w 限制
export function FeatureMain() {
  return (
    <div className="py-6 space-y-6 max-w-6xl mx-auto">
      {/* 页面内容 */}
    </div>
  )
}
```

**正确示例（推荐自适应全宽）**：
```tsx
// ✅ 容器使用 w-full，使其自适应外层 layout 的宽度调整
export function FeatureMain() {
  return (
    <div className="py-6 space-y-6 w-full">
      {/* 页面内容 */}
    </div>
  )
}
```

### 4.3 组件规范

- 组件应按功能分类。
- 公共业务组件放在 `components/common` 目录。
- ShadcnUI 基础组件放在 `components/ui` 目录。
- 自定义图标应放置在 `/components/icons/` 目录下以命名导出形式管理。常规图标统一使用 Lucide 库。

### 4.4 服务层架构与接口服务新建

服务层架构是前端与 API 交互的统一入口，基于以下原则：
1. **关注点分离** - 每个服务类只负责一个业务领域。
2. **统一入口** - 统一通过 `services` 对象对外导出所有服务。
3. **类型安全** - 所有请求参数和返回响应均有明确的 TypeScript 类型定义。

#### 如何新建接口服务：

1. **创建目录结构**:
   ```
   /services/新服务名/
     - types.ts       // 类型定义
     - 服务名.service.ts  // 服务实现
     - index.ts       // 导出服务
   ```

2. **实现服务类**:
   ```typescript
   // 新服务名/服务名.service.ts
   import {BaseService} from '../core/base.service';

   export class 新服务类 extends BaseService {
     protected static readonly basePath = '/api/v1/路径';

     static async 方法名(参数): Promise<返回类型> {
       return this.get<返回类型>('/endpoint');
     }
   }
   ```

3. **在 `services/index.ts` 注册**:
   ```typescript
   import {新服务类} from './新服务名';

   const services = {
     auth: AuthService,
     新服务名: 新服务类
   };
   ```

4. **使用方法**:
   ```typescript
   import services from '@/lib/services';

   // 调用服务方法
   const 结果 = await services.新服务名.方法名(参数);
   ```

---

## 五、代码质量与审查规范

### 5.1 统一代码检查与提交

> [!IMPORTANT]
> **代码开发完成后，提交前必须在项目根目录运行 `make code-check` 进行本地代码质量与风格检查，确保前端 ESLint 没有报错和警告（`--max-warnings 0`）。**

```bash
# 执行根目录的检查指令
make code-check
```

### 5.2 后端规范检查

**基础检查**：
- 后端 Go 代码需要通过 CodeQL 扫描。较复杂的逻辑建议结合 Copilot 检查。

**API 文档**：
- 所有 HTTP 接口都必须编写完整的 Swagger 注释。提交前需运行 `make swagger` 自动生成与更新接口文档。

**统一响应格式**：
```json
// 响应数据最外层固定包含 error_msg 与 data
{
    "error_msg": "",
    "data": null
}

// 示例：单条实体数据
{
    "error_msg": "",
    "data": {}
}

// 示例：分页数据格式
{
    "error_msg": "",
    "data": {
        "total": 0,
        "results": []
    }
}
```

### 5.3 数据库设计规范

- **外键约束**：禁止在数据库物理层面使用外键（FK），但需要在对应关联字段上显式建立索引。
- **默认值一致性**：数据库表字段的默认值必须与 Go model struct 的默认零值一致（如 `nil`, `0`, `false`, `""`），防止由于插入时漏填导致字段产生数据库异常默认值。

### 5.4 前端类型安全规范

- **禁止使用 `any`**：`any` 类型会绕过 TypeScript 编译期的类型检查系统，掩盖潜在的运行时错误，因此全面禁止使用。
- **合理使用 `unknown`**：`unknown` 是类型安全的 `any`，接收到此类值后必须先进行类型断言（Type Assertion）或类型收窄（Type Narrowing）后方可使用。
- **合理使用 `never`**：`never` 类型表示永远不会发生的值类型，必须谨慎使用，并在其使用处编写清晰的注释。
- **静态分析检查**：前端代码必须通过 ESLint 检查和 CodeQL 静态漏洞扫描。

---

## 六、包依赖与安全禁止项

### 6.1 严格禁止事项

| 禁止行为 | 说明 |
|----------|------|
| **禁止删除 `node_modules` 目录** | `node_modules` 为前端依赖安装目录，删除会导致项目无法运行。如需重新安装依赖，使用 `pnpm install` 覆盖更新即可，严禁执行 `rm -rf node_modules`。 |
| **`internal/util/` 下禁止引用框架包** | `util/` 及其子包（如 `util/cap`）定位为**纯工具层**，不得 `import` 任何 HTTP / ORM / 框架包，包括但不限于 `github.com/gin-gonic/gin`、`gorm.io/gorm`、`github.com/gin-contrib/sessions`。违反此约束会导致工具层与框架产生耦合，无法独立测试。详见下文的建议方案。 |

### 6.2 `util/` 包依赖约束与建议方案

#### 约束范围：

`internal/util/` 及其全部子包（如 `util/cap`、`util/crypto` 等）只允许引用：
- Go 标准库（`context`、`crypto`、`encoding`、`net/http` 原生包等）
- 项目内同级别的纯工具包（`internal/config`、`internal/db`、`internal/model` 等无框架依赖的包）
- 与框架无关的第三方库（如 `github.com/redis/go-redis`、`github.com/shopspring/decimal` 等）

**严禁引用**：`github.com/gin-gonic/gin`、`gorm.io/gorm`、`github.com/gin-contrib/sessions` 及任何 HTTP 框架 / Web 中间件相关包。

#### 常见误区与建议方案：

| 误区 | 建议方案 |
|------|----------|
| 在 `util/` 中写 `gin.HandlerFunc` 形式的中间件 | 将中间件移至对应的 `apps/<module>/middleware.go`，通过**函数参数**接收 `util/` 层的核心对象（如 `*cap.Manager`） |
| 在 `util/` 中通过 `*gin.Context` 写响应 | 只在 `util/` 中计算/校验逻辑并返回 `(result, error)`，由 `apps/` 层的 Handler 负责调用 `c.AbortWithStatusJSON` 写响应 |
| 在 `util/` 中使用 `gorm.DB` 直接查询 | 将数据库查询封装在 `internal/model/` 层方法中，`util/` 只接收已查出的数据结构 |

#### 正确示例：

```go
// ✅ internal/util/cap/manager.go — 纯逻辑，无框架依赖
func (m *Manager) VerifyToken(ctx context.Context, token, scope string) (bool, error) {
    // 只依赖 context、标准库、redis client
    ...
}

// ✅ internal/apps/cap/middleware.go — 框架胶水层，持有 gin 依赖
func VerifyMiddleware(mgr *caputil.Manager, scope string, enabledFunc func() bool) gin.HandlerFunc {
    return func(c *gin.Context) {
        valid, err := mgr.VerifyToken(c.Request.Context(), token, scope) // 调用纯逻辑
        if err != nil || !valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, util.Err("验证码校验失败"))
            return
        }
        c.Next()
    }
}
```

#### 错误示例（禁止）：

```go
// ❌ internal/util/cap/middleware.go — util/ 层不应出现 gin
import "github.com/gin-gonic/gin"

func (m *Manager) VerifyMiddleware(...) gin.HandlerFunc { ... }
```

---

## 七、新增功能开发流程

### 7.1 新增模块开发流程

新增 **异步任务**：使用项目专属 SKILL: `new-async-task` 进行开发。

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

### 7.2 Handler 文件拆分规则

逻辑简单的 CRUD 可以全部放在 `routers.go` 中。但当文件代码行数增长时，必须按以下规则拆分：

| 条件 | 拆分方式 |
|------|----------|
| 文件超过 **600 行** | 必须拆分 |
| 包含复杂业务逻辑（如外部调用、多步校验、事务处理） | 将业务逻辑拆到 `logic.go` 或 `logics.go` |
| 同一模块有多个独立功能域 | 按功能域拆分多个文件，如 `user_routers.go`、`role_routers.go` |

#### 拆分后的模块文件结构示例：

```
apps/admin/<module>/
├── routers.go          # 路由注册入口 + 简单 Handler（参数绑定 → 调用逻辑 → 响应）
├── logics.go           # 复杂业务逻辑（外部调用、事务、多步处理）
├── errs.go             # 错误常量
└── constants.go        # 业务常量（按需）
```

#### 职责边界：
- `routers.go` 只做三件事：参数绑定、调用 logic 函数、返回响应。不包含任何业务判断逻辑。
- `logics.go` 负责所有业务逻辑，接收已校验的参数，返回处理结果和错误。函数以 `PascalCase` 导出，供 `routers.go` 调用。

---

## 八、后端任务管理 API 接口

任务管理 API 路由（Admin）：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/tasks/types` | 获取可调度任务类型列表 |
| POST | `/api/v1/admin/tasks/dispatch` | 手动下发任务 |
| GET | `/api/v1/admin/tasks/executions` | 分页查询任务执行记录（支持 status / task_type 筛选） |
| GET | `/api/v1/admin/tasks/executions/:id` | 查询单条任务执行详情（含完整 Log） |
| POST | `/api/v1/admin/tasks/executions/:id/retry` | 重试失败任务（校验 Retryable && RetryCount < MaxRetry） |