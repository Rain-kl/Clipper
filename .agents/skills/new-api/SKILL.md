---
name: "new-api"
description: "Wavelet 项目专用：当新增或修改业务 API、Handler、服务层逻辑、路由注册时必须使用。本技能指导 apps 业务包划分、路由注册、Handler/logics 分层、Swagger 与质量门禁；纠正把一切塞进 custom.go / apps/custom 或产品伞包的错误写法。"
---

# 新增业务 API 开发与路由注册规范

本技能是 Wavelet 接口开发与路由注册的唯一指导规范。在开发任何新接口前，请按本指南做架构决策与路由注册。

---

## 先搞清：脚手架 vs 产品化

Wavelet 是**通用全栈脚手架**。仓库里的 `custom` 相关代码是**示例/占位**，不是产品业务的标准落点。

| 层级 | 含义 | 典型包 |
| :--- | :--- | :--- |
| **平台能力** | 脚手架自带、与具体产品无关 | `oauth`、`user`、`admin/*`、`upload`、`cap`、`config`、`health`、`risk_control` |
| **产品业务** | 基于脚手架做具体产品时新增的域 | 直接落在 `internal/apps/<domain>/`，与平台包**平级** |

**一旦用脚手架开发具体产品，整个仓库就是该产品**——例如要做「消息平台」，业务模块应是 `apps/channel`、`apps/conversation`、`apps/delivery` 等，而不是先建 `apps/message` 伞包再往里塞子模块。

---

## 反模式（AI 最常踩的坑）

### 1. 把所有业务路由塞进 `custom.go` / 路径前缀 `/custom`

仓库中的：

- `internal/router/v1/custom.go`
- `internal/router/root/custom.go`
- `internal/apps/custom/`

是**演示如何挂一条示例接口**（`GET /api/v1/custom/hello`），**不是**「所有自定义业务必须写在这里」的规定。

| 错误 | 正确 |
| :--- | :--- |
| 新功能一律改 `v1/custom.go`，路径全是 `/api/v1/custom/...` | 按域新建 `apps/<domain>/`，路由用语义化路径（如 `/api/v1/channels`），在 `router/v1/` 下用**独立注册文件**挂载 |
| 把 `custom` 包当成业务垃圾桶 | 保留或删除示例均可；真正业务用独立包名 |

### 2. 产品伞包 + 深层子包

| 错误 | 正确 |
| :--- | :--- |
| `apps/message/channel`、`apps/message/inbox`、`apps/message/delivery`（先套一层产品名） | `apps/channel`、`apps/inbox`、`apps/delivery`（域模块与 `oauth`/`user` 平级） |
| `apps/myapp/...` 再嵌套所有业务 | 仓库即产品，**不要**再包一层产品根 |

**判定**：模块名应对齐**业务能力/限界上下文**（channel、order、invoice），而不是对齐产品营销名（message-platform、myapp）。

### 3. 其它仍须遵守的防线

- 不要在 `internal/router/router.go` 里直接挂业务 Handler（只做高层委派）。
- 不要破坏平台模块既有语义去硬塞无关业务（例如把消息逻辑塞进 `apps/user`）。
- 错误响应使用 `response.Abort*`，禁止 `c.JSON(..., response.Err(...))`（见 `AGENTS.md`）。

---

## 路由注册模型

### 谁可以改

| 文件 | 角色 | 产品化时 |
| :--- | :--- | :--- |
| `internal/router/router.go` | 引擎、中间件、委派入口 | 一般不改；特殊全局中间件才动 |
| `internal/router/v1/v1.go` | V1 分发：调用各 `Register*Routes` | **允许**：增加对新业务注册函数的一行调用 |
| `internal/router/v1/user.go` / `admin.go` | 平台用户端 / 管理端路由 | **优先不改**；仅当扩展平台能力（OAuth、上传、用户资料）时修改 |
| `internal/router/v1/<domain>.go`（新建） | 产品业务路由注册 | **推荐落点** |
| `internal/router/v1/custom.go` | **示例** | 可删可留；**不要**把真实业务堆在这里 |
| `internal/router/root/default.go` / `frontend.go` | 文件服务、health、前端静态 | 平台级，勿塞产品 API |
| `internal/router/root/custom.go` | 根路径**示例**占位 | 仅当确需根路径回调/短链时，用**语义路径**注册，或新建 `root/<domain>.go` 并由 `root.go` 调用 |

### 路径归属（产品 API 用语义路径）

| 目标路径特征 | 注册位置 | 说明 |
| :--- | :--- | :--- |
| `/api/v1/<domain>/...`（如 `/api/v1/channels`） | `v1/<domain>.go` 的 `Register<Domain>Routes`，在 `v1.go` 调用 | **产品业务默认做法** |
| `/api/v1/admin/<domain>/...` | 管理端：可在 `admin.go` 增加小组，或 `v1/admin_<domain>.go` 再由 `RegisterAdminRoutes`/ `v1.go` 组装 | 需 `admin.LoginAdminRequired()` |
| `/api/v1/user/...`、`/oauth/...`、`/upload/...` 等 | `user.go` 等平台文件 | 平台能力，勿把无关产品塞进来 |
| 根路径特殊接口（Webhook、短链） | `root` 下独立注册函数 | **不要**默认塞进 `custom` 前缀 |
| `GET /f/:id`、`/api/health`、`robots.txt` | `root/default.go` | 平台，勿改用途 |

`custom.go` 里现有的 `/api/v1/custom/...` **仅作脚手架演示**，不代表业务必须挂在 `/custom` 下。

---

## 推荐目录结构（产品业务）

以「频道 / channel」域为例（消息平台中的一个限界上下文）：

```text
internal/
├── router/
│   └── v1/
│       ├── v1.go              # [修改] 调用 RegisterChannelRoutes
│       └── channel.go         # [新建] 只负责挂载 channel 路由
└── apps/
    └── channel/               # 与 oauth、user、upload 平级
        ├── routers.go         # HTTP Handlers（绑定、鉴权上下文、响应）
        ├── logics.go          # 纯业务：context.Context，无 gin
        ├── errs.go            # 模块错误文案常量（可选）
        └── ...                # 需要时再加 service.go、tasks.go 等
```

**不要**建成：

```text
internal/apps/message/          # ❌ 产品伞包
    channel/
    inbox/
internal/apps/custom/           # ❌ 示例包当业务垃圾桶
    channel_handler.go
```

模块内若复杂度高，可在**该域包内**分子目录（如 `apps/channel/handler`），但仍是一个域包，不是「产品名/子域」两层品牌结构。

---

## 路由注册示例

### `internal/router/v1/channel.go`（产品业务）

```go
package v1

import (
	"github.com/Rain-kl/Wavelet/internal/apps/channel"
	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/gin-gonic/gin"
)

// RegisterChannelRoutes mounts channel domain APIs under /api/v1.
func RegisterChannelRoutes(apiV1Router *gin.RouterGroup) {
	r := apiV1Router.Group("/channels")
	r.Use(oauth.LoginRequired())
	{
		r.GET("", channel.ListChannels)
		r.POST("", channel.CreateChannel)
		r.GET("/:id", channel.GetChannel)
	}
}
```

### `internal/router/v1/v1.go`（增加一行委派）

```go
func RegisterV1Routes(apiV1Router *gin.RouterGroup, apiGroup *gin.RouterGroup) {
	RegisterUserRoutes(apiV1Router, apiGroup)
	RegisterAdminRoutes(apiV1Router)
	RegisterChannelRoutes(apiV1Router) // 产品域
	RegisterCustomRoutes(apiV1Router)  // 可选：仅保留脚手架示例
}
```

### 根路径 Webhook（确有需要时）

在 `root` 用语义路径，例如 `POST /webhooks/stripe`，注册函数可放在 `root/webhooks.go` 或扩展现有 root 注册；**不要**为了「只能写 custom」而使用无意义的 `/custom` 前缀。

---

## 核心开发步骤

### 步骤 1：划定域包名

- 用**业务能力**命名：`channel`、`order`、`invoice`。
- 与现有 `apps/` 下平台包平级；禁止产品伞包。

### 步骤 2：库表与 model

若涉及新表/字段：按 [database-migration](../database-migration/SKILL.md) 在 goose 迁移与 `internal/model/` 中定义。

### 步骤 3：`logics.go` / `service.go`

放在 `internal/apps/<domain>/`：

- **优先**纯函数 `logics.go`：`context.Context` 入参，无 `*gin.Context`。
- 有状态依赖时用 `service.go` 构造注入。
- 跨模块副作用（推送、任务）经 `internal/listener` + `bootstrap`，禁止业务直接 import push（见 `push-notification`）。

### 步骤 4：Handler（`routers.go`）

- `ShouldBindJSON` / `ShouldBindQuery`。
- 成功：`c.JSON(http.StatusOK, response.OK(data))` 或 `response.OKNil()`。
- 失败：`response.AbortBadRequest` / `AbortUnauthorized` / `AbortNotFound` / `AbortInternal` 等，**禁止** `response.Err` 直接 `c.JSON`。
- 完整 Swagger 注释；`@Router` 使用真实语义路径。

参考：`references/handler_example.go`、`logics_example.go`、`service_example.go`（示例域名，非强制包名 `custom`）。

### 步骤 5：注册路由

新建 `internal/router/v1/<domain>.go`，在 `v1.go` 调用；管理端按需挂到 admin 组。

---

## 与平台路由的边界

- **扩展平台能力**（用户资料字段、上传策略、OAuth 源）：改对应平台 `apps/*` 与 `user.go`/`admin.go`。
- **新产品功能**：新建 `apps/<domain>` + `router/v1/<domain>.go`，**不要**塞进 `custom` 或某个无关平台包。
- 管理端产品配置页 API：路径宜为 `/api/v1/admin/<domain>/...`，中间件与现有 admin 组一致。

---

## 质量验证门禁

1. `make license`（新 Go 文件许可头）
2. `make swagger`（Handler/Swagger 有变时）
3. `make format` 与 `make code-check`
4. `go test` 覆盖相关包

---

## 自检清单

- [ ] 未把真实业务堆进 `apps/custom` 或 `v1/custom.go`
- [ ] 未创建 `apps/<产品名>/` 伞包再塞子域
- [ ] 业务包与 `oauth`/`user`/`upload` 平级，路径语义化（非强制 `/custom`）
- [ ] 路由在 `router/v1/<domain>.go`（或 admin 对应处）注册，并由 `v1.go` 委派
- [ ] Handler 用 `response.Abort*` / `response.OK`，logics 不依赖 gin
- [ ] 需要时已跑 swagger / code-check
