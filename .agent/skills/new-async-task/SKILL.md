---
name: "new-async-task"
description: "项目专用：指导如何在 wavelet 项目中新增异步任务（TaskHandler）。当用户提到新增任务、Asynq、后台任务、定时任务、任务日志、任务重试、TaskMeta、TaskParam 或 PayloadValidator 时必须使用本技能，覆盖常量定义、参数校验、处理器实现、统一注册、Worker 路由、Cron 调度、AppendLog 日志和测试验证。"
---

# 新增异步任务开发指南

本项目中异步任务基于 Asynq（Redis 任务队列）构建。开发者只需编写业务逻辑，框架负责执行记录、状态流转、日志、重试等全部外围工作。

通用 Wavelet 目录、Handler、配置、数据库和前端规范见仓库根目录 `Agents.md`。本技能只覆盖异步任务相关规则。

## 架构概览

```
DispatchTask (创建记录 + 入队)
        │
        ▼
   Asynq Redis Queue
        │
        ▼
ProcessTask (状态 pending→running，调用 handler)
        │
        ▼
TaskHandler.Execute (← 你写这一步)
        │
   成功 → succeeded + 存储结果
   失败 → failed + 记录错误 + Asynq 可能自动重试
```

**关键文件**：
- `internal/task/handler.go` — `TaskHandler` 接口、`TaskResult` 类型、`PayloadValidator` 可选接口
- `internal/task/constants.go` — 任务类型常量、`TaskMeta`、`TaskParam`、`DispatchableTasks`
- `internal/task/executor.go` — 注册、下发、执行、日志追加、参数校验分发
- `internal/task/handlers/register.go` — 内置任务处理器统一注册点，Admin API 和 Worker 都依赖它
- `internal/task/worker/worker.go` — Worker 启动、Asynq mux 路由、队列配置
- `internal/task/scheduler/scheduler.go` — Cron 定时调度
- `internal/model/task_execution.go` — `TaskExecution` GORM 模型
- `internal/apps/admin/task/routers.go` — 管理 API（下发、查询、重试）

**队列优先级**：`webhook` > `whitelist_only` > `default`。

**Admin 任务管理 API**：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/tasks/types` | 获取可调度任务类型列表 |
| POST | `/api/v1/admin/tasks/dispatch` | 手动下发任务 |
| GET | `/api/v1/admin/tasks/executions` | 分页查询任务执行记录（支持 status / task_type 筛选） |
| GET | `/api/v1/admin/tasks/executions/:id` | 查询单条任务执行详情（含完整 Log） |
| POST | `/api/v1/admin/tasks/executions/:id/retry` | 重试失败任务（校验 Retryable && RetryCount < MaxRetry） |

## 新增任务的完整步骤

### 第 1 步：在 constants.go 添加常量和 TaskMeta

打开 `internal/task/constants.go`，添加三样东西：

**1. Asynq 任务类型常量**（作为 Redis 消息类型标识，格式 `{module}:{action}`）：

```go
const CleanupUnusedUploadsTask = "upload:cleanup_unused"
```

**2. 管理员下发用的任务类型常量**（用于 Admin API 调用）：

```go
const TaskTypeCleanupUploads = "cleanup_unused_uploads"
```

**3. 在 `DispatchableTasks` 切片中添加 TaskMeta**：

不带参数的任务：
```go
{
    Type:         TaskTypeCleanupUploads,
    AsynqTask:    CleanupUnusedUploadsTask,
    Name:         "清理未使用上传",
    Description:  "清理超过1小时未使用的上传文件",
    SupportsTime: false,
    MaxRetry:     3,
    Queue:        QueueDefault,
    Retryable:    true,
}
```

带参数的任务（`Params` 定义前端表单字段）：
```go
{
    Type:         TaskTypeSendEmail,
    AsynqTask:    SendEmailTask,
    Name:         "发送邮件",
    Description:  "异步发送系统邮件",
    SupportsTime: false,
    MaxRetry:     3,
    Queue:        QueueDefault,
    Retryable:    true,
    Params: []TaskParam{
        {
            Name:        "to",
            Label:       "接收邮箱 (To)",
            Type:        "string",
            Required:    true,
            Placeholder: "receiver@example.com",
            Description: "接收邮件的目标邮箱地址",
        },
        {
            Name:        "subject",
            Label:       "邮件主题 (Subject)",
            Type:        "string",
            Required:    true,
            Placeholder: "请输入邮件主题",
            Description: "发送邮件的主题标题",
        },
        {
            Name:        "body",
            Label:       "邮件内容 (Body)",
            Type:        "text",
            Required:    true,
            Placeholder: "请输入邮件内容（支持 HTML 格式）",
            Description: "发送邮件的内容主体",
        },
    },
}
```

**`TaskParam` 字段说明**：

| 字段 | 作用 |
|------|------|
| `Name` | 参数键名，与 Handler 中 Payload 结构体的 JSON tag 一致 |
| `Label` | 前端表单显示的标签 |
| `Type` | 前端控件类型：`string`（单行输入）、`text`（多行文本）、`number`（数字） |
| `Required` | 前端是否标为必填（仅前端提示，不做服务端校验） |
| `Placeholder` | 前端输入框占位文字 |
| `Description` | 前端显示的参数说明 |

**注意**：`TaskParam` 纯粹是前端表单元数据。服务端参数校验通过 Handler 实现 `PayloadValidator` 接口完成（见第 2 步）。

### 第 2 步：实现 TaskHandler（+ 可选 PayloadValidator）

在 `internal/apps/<module>/` 下创建 `tasks.go`。

#### 无参数的任务（参考 `internal/apps/upload/tasks.go`）

只需实现 `TaskHandler` 接口：

```go
type CleanupUnusedUploadsHandler struct{}

func (h *CleanupUnusedUploadsHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
    task.AppendLog(ctx, "开始扫描...")
    // ... 直接执行业务逻辑，忽略 payload ...
    return &task.TaskResult{Message: "完成"}, nil
}
```

#### 带参数的任务（参考 `internal/apps/user/tasks.go`）

需要定义 Payload 结构体，并实现 `PayloadValidator` 接口进行参数校验：

```go
// 定义载荷结构体（字段与 TaskParam.Name 对应）
type SendEmailPayload struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

type SendEmailHandler struct{}

// ValidatePayload 实现 task.PayloadValidator 接口
// 框架在 Admin 下发时自动调用，校验失败返回 error，前端收到 400 + 错误信息
func (h *SendEmailHandler) ValidatePayload(payload []byte) ([]byte, error) {
    if len(payload) == 0 {
        return nil, errors.New("任务参数不能为空")
    }

    var req SendEmailPayload
    if err := json.Unmarshal(payload, &req); err != nil {
        return nil, fmt.Errorf("无效的 JSON 格式: %w", err)
    }

    // 标准化：Trim 空白
    req.To = strings.TrimSpace(req.To)
    req.Subject = strings.TrimSpace(req.Subject)
    req.Body = strings.TrimSpace(req.Body)

    // 校验必填字段
    if req.To == "" || req.Subject == "" || req.Body == "" {
        return nil, errors.New("to、subject、body 不能为空")
    }

    // 返回标准化后的 bytes，会作为最终 payload 存入 DB 和入队
    return json.Marshal(req)
}

// Execute 实现 task.TaskHandler 接口
func (h *SendEmailHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
    var req SendEmailPayload
    if err := json.Unmarshal(payload, &req); err != nil {
        return nil, fmt.Errorf("解析参数失败: %w", err)
    }

    task.AppendLog(ctx, "开始发送邮件到: %s, 主题: %s", req.To, req.Subject)
    // ... 业务逻辑 ...

    msg := fmt.Sprintf("邮件成功发送至: %s", req.To)
    task.AppendLog(ctx, "%s", msg)
    return &task.TaskResult{Message: msg}, nil
}
```

**`PayloadValidator` 接口说明**：

```go
type PayloadValidator interface {
    ValidatePayload(payload []byte) ([]byte, error)
}
```

这是一个**可选接口**。框架在 Admin 下发任务时通过 `task.ValidateAndNormalizePayload` 自动检测：

- Handler 实现了 `PayloadValidator` → 调用 `ValidatePayload`，校验通过返回标准化后的 bytes，失败返回 error（前端收到 400）
- Handler 没实现 → 直接透传原始 payload，不做任何校验

**不需要修改 Admin dispatch handler**。`routers.go` 中的 `DispatchTask` 函数已经是通用的，会自动调用对应 Handler 的 `ValidatePayload`。

**`Execute` 中仍然需要 `json.Unmarshal`**，因为任务可能通过 Scheduler 或重试触发，不经过 Admin 校验路径。但不需要重复必填校验。

**返回值约定**：
- 成功：返回 `&task.TaskResult{Message: "摘要", Detail: "可选详细JSON"}`。`ProcessTask` 会将 `Message + "\n" + Detail` 存入 `TaskExecution.Result`。
- 失败：返回 `nil, fmt.Errorf("...")`。`ProcessTask` 会将错误信息存入 `TaskExecution.ErrorMessage`，状态标记为 `failed`，并将 error 返回给 Asynq 触发自动重试。

### 第 3 步：统一注册 Handler，并在 worker.go 添加路由

新增任务必须同时满足两条链路：

- Admin API 下发任务时能找到 Handler，从而调用 `PayloadValidator` 做服务端参数校验。
- Worker 消费 Asynq 消息时能路由到 `task.ProcessTask`，再由框架分发到 Handler。

#### 3.1 在 `internal/task/handlers/register.go` 注册处理器

打开 `internal/task/handlers/register.go`，导入你的业务模块并在 `Register()` 中注册：

```go
package handlers

import (
    "github.com/Rain-kl/Wavelet/internal/apps/upload"
    "github.com/Rain-kl/Wavelet/internal/apps/user"
    "github.com/Rain-kl/Wavelet/internal/task"
)

func Register() {
    task.RegisterHandler(task.CleanupUnusedUploadsTask, &upload.CleanupUnusedUploadsHandler{})
    task.RegisterHandler(task.SendEmailTask, &user.SendEmailHandler{})
}
```

`Register()` 会被 Admin task API 和 Worker 共同调用。不要只在 `worker` 包里注册，否则 Admin dispatch 时 `ValidateAndNormalizePayload` 找不到 Handler，带参数任务会绕过 `PayloadValidator`。

#### 3.2 在 `internal/task/worker/worker.go` 添加 Asynq 路由

在 `StartWorker()` 的 mux 中添加任务类型路由：

```go
mux := asynq.NewServeMux()
mux.Use(taskLoggingMiddleware)
mux.HandleFunc(task.CleanupUnusedUploadsTask, task.ProcessTask)
mux.HandleFunc(task.SendEmailTask, task.ProcessTask)
```

所有路由都指向同一个 `task.ProcessTask`，它内部根据 Asynq task type 查找对应的 handler 分发执行。

`worker.go` 的 `init()` 应保持调用统一注册函数：

```go
func init() {
    taskhandlers.Register()
}
```

### 第 4 步（可选）：添加 Cron 定时调度

如果任务需要定时执行，编辑 `internal/task/scheduler/scheduler.go`，在 `StartScheduler()` 中注册：

```go
if _, err = scheduler.Register(
    config.Config.Scheduler.CleanupUnusedUploadsTaskCron,
    asynq.NewTask(task.CleanupUnusedUploadsTask, nil),
    asynq.Unique(23*time.Hour),
    asynq.MaxRetry(3),
); err != nil {
    return
}
```

然后在 `internal/config/model.go` 的 `schedulerConfig` 中添加字段：

```go
type schedulerConfig struct {
    CleanupUnusedUploadsTaskCron string `mapstructure:"cleanup_unused_uploads_task_cron"`
    YourNewTaskCron              string `mapstructure:"your_new_task_cron"`
}
```

在 `config.example.yaml` 的 `scheduler` 段添加：

```yaml
scheduler:
  cleanup_unused_uploads_task_cron: "0 */2 * * *"
  your_new_task_cron: "0 */6 * * *"
```

**注意**：通过 Scheduler 定时触发的任务不会创建 `TaskExecution` 记录（直接入队 Asynq）。`ProcessTask` 执行时若在数据库中找不到对应记录，会打印警告日志但仍会正常执行业务逻辑。如果需要状态追踪，应通过 Admin API 手动下发。

## 参数校验的完整链路

```
前端表单（根据 TaskMeta.Params 动态渲染）
  → POST /api/v1/admin/tasks/dispatch { task_type, payload: JSON字符串 }
  → task.ValidateAndNormalizePayload(asynqTaskType, payload)
      → Handler 实现了 PayloadValidator？
          → 是：调用 ValidatePayload()，校验 + 标准化
          → 否：直接透传
  → task.DispatchTask(validatedPayload) 存入 DB + 入队 Asynq
  → ProcessTask → handler.Execute(payload) → handler 内 json.Unmarshal 使用参数
```

关键点：Admin dispatch handler (`routers.go`) 是**通用的**，不需要按任务类型写 if 分支。新增带参数的任务时，只需在 Handler 上实现 `ValidatePayload` 方法即可。

注意：`ValidateAndNormalizePayload` 依赖 Handler 已注册。新增任务后必须更新 `internal/task/handlers/register.go`，否则 Admin API 不会执行该任务的 `PayloadValidator`。

## AppendLog 使用指南

`task.AppendLog(ctx, format string, args ...interface{})` 在 `TaskHandler.Execute` 内部调用，将日志追加到 `TaskExecution.Log` 字段。管理端 API 可查看完整执行日志。

**日志格式**：每行自动添加 `[HH:MM:SS]` 时间前缀，使用 SQL `COALESCE(log, '') || ?` 拼接到 log 字段末尾。

**何时记录日志**：

| 时机 | 示例 |
|------|------|
| 任务开始时 | `task.AppendLog(ctx, "开始扫描，阈值: %s", threshold)` |
| 参数解析成功后 | `task.AppendLog(ctx, "开始发送邮件到: %s, 主题: %s", req.To, req.Subject)` |
| 批量处理的每一批次 | `task.AppendLog(ctx, "本批次处理 %d 条记录", len(batch))` |
| 关键中间状态 | `task.AppendLog(ctx, "已删除对象 %s", filePath)` |
| 遇到可继续的错误时 | `task.AppendLog(ctx, "清理文件失败 [ID:%d]: %v", id, err)` |
| 任务完成时 | `task.AppendLog(ctx, "共处理 %d 个，成功 %d 个", total, success)` |

**降级行为**：如果 `ctx` 中没有 `taskID`（比如 Scheduler 触发的任务没有数据库记录），`AppendLog` 会降级为 `logger.InfoF` 输出到应用日志，不会报错。

**不要过度记录**：避免在循环中对每条记录都追加日志（尤其是处理量很大的场景），可以在批次级别记录摘要。每个 `AppendLog` 都会执行一次数据库 UPDATE。

## 框架自动处理的事项

开发者不需要关心以下内容，全部由框架透明处理：

- **参数校验分发**：`ValidateAndNormalizePayload` 自动检测 Handler 是否实现 `PayloadValidator`，实现了就调用，否则透传
- **记录创建**：`DispatchTask` 自动创建 `TaskExecution` 记录，生成唯一 `TaskID`（格式 `{triggeredBy}_{taskType}_{snowflakeID}`）
- **状态流转**：`pending` → `running` → `succeeded`/`failed`
- **耗时统计**：自动记录 `StartedAt`、`FinishedAt`、`Duration`（毫秒）
- **错误记录**：失败时自动存储 `ErrorMessage`
- **结果存储**：成功时自动存储 `TaskResult.Message` 和 `TaskResult.Detail`
- **OTel 追踪**：自动创建 Span，记录任务类型、Payload 大小、TaskID
- **重试计数**：手动重试时自动递增 `RetryCount`，校验 `RetryCount < MaxRetry`
- **队列路由**：根据 `TaskMeta.Queue` 投递到对应优先级队列
- **前端表单渲染**：`ListTaskTypes` API 返回 `DispatchableTasks`（含 `Params`），前端据此动态渲染参数表单

## 重试机制

项目中存在两层重试机制，互不干扰：

**Asynq 自动重试**：`ProcessTask` 返回 error 时，Asynq 根据入队时设置的 `MaxRetry` 自动重试。但 `TaskExecution` 记录在第一次失败时已标记为 `failed`，后续重试会覆盖同一条记录。

**管理端手动重试**：通过 `POST /api/v1/admin/tasks/executions/{id}/retry` 触发。会创建一个全新的 `TaskExecution` 记录（新 TaskID `retry_{count}_{originalTaskID}`），`RetryCount` 递增，`Payload` 原样复制。校验条件：状态必须为 `failed`、`Retryable` 为 `true`、`RetryCount < MaxRetry`。

## 参考：现有任务处理器

- `internal/apps/upload/tasks.go` — `CleanupUnusedUploadsHandler`：无参数任务，展示游标分页批量处理、每批次 AppendLog、事务内操作、单条失败 continue 不终止。
- `internal/apps/user/tasks.go` — `SendEmailHandler`：带参数任务，展示 Payload 结构体定义、`PayloadValidator` 实现（`ValidatePayload` 用于校验+标准化）、`Execute` 中 json.Unmarshal 使用参数。

## 测试与验证

新增或调整异步任务后，至少做以下验证：

1. 单测 Handler 本身：无参数任务验证业务结果；带参数任务额外覆盖 `ValidatePayload` 的成功、非法 JSON、缺失必填字段。
2. 单测 Admin dispatch：`POST /api/v1/admin/tasks/dispatch` 对合法 payload 返回 200，对非法 payload 返回 400，并包含清晰错误信息。
3. 单测 Retry：可重试失败任务应创建新的 `TaskExecution`，达到 `MaxRetry`、非 failed 状态、`Retryable=false` 都应拒绝。
4. 跑目标包测试：

```bash
go test ./internal/task ./internal/apps/admin/task ./internal/apps/<module>
```

5. 最后跑全量测试：

```bash
go test ./...
```

测试环境中如需下发/重试任务，使用 `miniredis` 初始化 `task.AsynqClient`；不要把 `internal/task` 依赖塞回通用 `internal/testhelper`，否则容易让 `model`/`task` 测试产生 import cycle。
