---
name: "new-async-task"
description: "项目专用：指导如何在本项目中新增异步任务（TaskHandler），包括常量定义、参数传递（TaskParam）、处理器实现、Admin 校验、注册、Cron 调度、AppendLog 日志规范等完整步骤。"
---

# 新增异步任务开发指南

本项目中异步任务基于 Asynq（Redis 任务队列）构建。开发者只需编写业务逻辑，框架负责执行记录、状态流转、日志、重试等全部外围工作。

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
- `internal/task/handler.go` — `TaskHandler` 接口、`TaskResult` 类型
- `internal/task/constants.go` — 任务类型常量、`TaskMeta`、`TaskParam`、`DispatchableTasks`
- `internal/task/executor.go` — 注册、下发、执行、日志追加
- `internal/task/worker/worker.go` — Worker 启动、处理器注册、路由
- `internal/task/scheduler/scheduler.go` — Cron 定时调度
- `internal/model/task_execution.go` — `TaskExecution` GORM 模型
- `internal/apps/admin/task/routers.go` — 管理 API（下发、查询、重试）

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

**注意**：`TaskParam` 纯粹是前端表单元数据。服务端不基于它做参数校验——校验逻辑在 Admin dispatch handler 和 Handler Execute 中各写一份。

### 第 2 步：实现 TaskHandler

在 `internal/apps/<module>/` 下创建 `tasks.go`。

**无参数的任务**（参考 `internal/apps/upload/tasks.go`）：

```go
type CleanupUnusedUploadsHandler struct{}

func (h *CleanupUnusedUploadsHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
    task.AppendLog(ctx, "开始扫描...")
    // ... 直接执行业务逻辑，忽略 payload ...
    return &task.TaskResult{Message: "完成"}, nil
}
```

**带参数的任务**（参考 `internal/apps/user/tasks.go`）：

需要定义自己的 Payload 结构体，在 `Execute` 中反序列化：

```go
// 定义载荷结构体（字段与 TaskParam.Name 对应）
type SendEmailPayload struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

type SendEmailHandler struct{}

func (h *SendEmailHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
    // 第一步：反序列化参数
    var req SendEmailPayload
    if err := json.Unmarshal(payload, &req); err != nil {
        task.AppendLog(ctx, "解析参数失败: %v", err)
        return nil, fmt.Errorf("解析参数失败: %w", err)
    }

    task.AppendLog(ctx, "开始发送邮件到: %s, 主题: %s", req.To, req.Subject)

    // ... 业务逻辑 ...

    msg := fmt.Sprintf("邮件成功发送至: %s", req.To)
    task.AppendLog(ctx, "%s", msg)
    return &task.TaskResult{Message: msg}, nil
}
```

**返回值约定**：
- 成功：返回 `&task.TaskResult{Message: "摘要", Detail: "可选详细JSON"}`。`ProcessTask` 会将 `Message + "\n" + Detail` 存入 `TaskExecution.Result`。
- 失败：返回 `nil, fmt.Errorf("...")`。`ProcessTask` 会将错误信息存入 `TaskExecution.ErrorMessage`，状态标记为 `failed`，并将 error 返回给 Asynq 触发自动重试。

### 第 3 步：在 Admin dispatch handler 中添加参数校验

打开 `internal/apps/admin/task/routers.go`，在 `DispatchTask` 函数中为新任务类型添加校验逻辑。

下发请求结构体中，前端传入的参数通过 `Payload` 字段（JSON 字符串）传递：

```go
type DispatchTaskRequest struct {
    TaskType  string     `json:"task_type" binding:"required"`
    StartTime *time.Time `json:"start_time"`
    EndTime   *time.Time `json:"end_time"`
    UserID    *uint64    `json:"user_id"`
    Payload   string     `json:"payload"`   // ← 前端传的参数 JSON 字符串
}
```

在 `DispatchTask` 函数中按 task type 分支校验：

```go
var payloadBytes []byte
if req.TaskType == task.TaskTypeSendEmail {
    // 校验 payload 非空
    if strings.TrimSpace(req.Payload) == "" {
        c.JSON(http.StatusBadRequest, util.Err("任务参数 Payload 不能为空"))
        return
    }
    // 解析 JSON 并校验必填字段
    var mailPayload struct {
        To      string `json:"to"`
        Subject string `json:"subject"`
        Body    string `json:"body"`
    }
    if err := json.Unmarshal([]byte(req.Payload), &mailPayload); err != nil {
        c.JSON(http.StatusBadRequest, util.Err("无效的 JSON 格式: "+err.Error()))
        return
    }
    // Trim + 必填校验
    mailPayload.To = strings.TrimSpace(mailPayload.To)
    mailPayload.Subject = strings.TrimSpace(mailPayload.Subject)
    mailPayload.Body = strings.TrimSpace(mailPayload.Body)
    if mailPayload.To == "" || mailPayload.Subject == "" || mailPayload.Body == "" {
        c.JSON(http.StatusBadRequest, util.Err("to、subject、body 不能为空"))
        return
    }
    // 序列化回 bytes 传给 DispatchTask
    payloadBytes, _ = json.Marshal(mailPayload)
} else {
    // 无参数任务：直接透传 payload
    if req.Payload != "" {
        payloadBytes = []byte(req.Payload)
    }
}

taskID, err := task.DispatchTask(c.Request.Context(), req.TaskType, payloadBytes, "manual")
```

**参数传递的完整链路**：

```
前端表单（根据 TaskMeta.Params 动态渲染）
  → POST /api/v1/admin/tasks/dispatch { task_type, payload: JSON字符串 }
  → Admin DispatchTask handler 校验 + json.Marshal
  → task.DispatchTask(payloadBytes) 存入 DB + 入队 Asynq
  → ProcessTask → handler.Execute(payload) → handler 内 json.Unmarshal
```

### 第 4 步：在 worker.go 注册

打开 `internal/task/worker/worker.go`，做两件事：

**1. 在 `init()` 中注册处理器**：

```go
func init() {
    task.RegisterHandler(task.CleanupUnusedUploadsTask, &upload.CleanupUnusedUploadsHandler{})
    task.RegisterHandler(task.SendEmailTask, &user.SendEmailHandler{})
}
```

**2. 在 `StartWorker()` 的 mux 中添加路由**：

```go
mux := asynq.NewServeMux()
mux.Use(taskLoggingMiddleware)
mux.HandleFunc(task.CleanupUnusedUploadsTask, task.ProcessTask)
mux.HandleFunc(task.SendEmailTask, task.ProcessTask)
```

所有路由都指向同一个 `task.ProcessTask`，它内部根据 Asynq task type 查找对应的 handler 分发执行。

### 第 5 步（可选）：添加 Cron 定时调度

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

开发者不需要关心以下内容，全部由 `ProcessTask` 框架层透明处理：

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
- `internal/apps/user/tasks.go` — `SendEmailHandler`：带参数任务，展示 Payload 结构体定义、json.Unmarshal 反序列化、参数日志记录。

