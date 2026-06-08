---
name: "new-async-task"
description: "项目专用：指导如何在本项目中新增异步任务（TaskHandler），包括常量定义、处理器实现、注册、Cron 调度、AppendLog 日志规范等完整步骤。"
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
- `internal/task/constants.go` — 任务类型常量、`TaskMeta`、`DispatchableTasks`
- `internal/task/executor.go` — 注册、下发、执行、日志追加
- `internal/task/worker/worker.go` — Worker 启动、处理器注册、路由
- `internal/task/scheduler/scheduler.go` — Cron 定时调度
- `internal/model/task_execution.go` — `TaskExecution` GORM 模型

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

```go
var DispatchableTasks = []TaskMeta{
    {
        Type:         TaskTypeCleanupUploads,       // 管理员 API 用的标识
        AsynqTask:    CleanupUnusedUploadsTask,     // Asynq 路由用的标识
        Name:         "清理未使用上传",               // 管理后台显示名
        Description:  "清理超过1小时未使用的上传文件",    // 管理后台显示描述
        SupportsTime: false,                        // 是否支持时间范围参数（暂未使用）
        MaxRetry:     3,                            // Asynq 最大自动重试次数
        Queue:        QueueDefault,                 // 投递到的队列名
        Retryable:    true,                         // 管理端是否允许手动重试
    },
}
```

### 第 2 步：实现 TaskHandler

在 `internal/apps/<module>/` 下创建 `tasks.go`，定义结构体并实现 `TaskHandler` 接口：

```go
package upload

import (
    "context"
    "fmt"
    "time"

    "github.com/linux-do/credit/internal/db"
    "github.com/linux-do/credit/internal/model"
    "github.com/linux-do/credit/internal/task"
)

// MyTaskHandler 你的任务处理器
type CleanupUnusedUploadsHandler struct{}

// Execute 实现 TaskHandler 接口
//   - ctx: 已注入 OTel Span 和 taskID，传给 task.AppendLog 使用
//   - payload: 下发时传入的原始字节（可为 nil）
func (h *CleanupUnusedUploadsHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
    task.AppendLog(ctx, "开始扫描未使用上传文件，阈值: %s", time.Now().Add(-1*time.Hour).Format(time.RFC3339))

    // ... 业务逻辑 ...
    // 如需处理批量数据，使用游标分页模式（参见 upload/tasks.go）

    msg := fmt.Sprintf("共处理 %d 个文件，成功删除 %d 个", totalProcessed, totalDeleted)
    task.AppendLog(ctx, "%s", msg)
    return &task.TaskResult{Message: msg}, nil
}
```

**返回值约定**：
- 成功：返回 `&task.TaskResult{Message: "摘要", Detail: "可选详细JSON"}`。`ProcessTask` 会将 `Message + "\n" + Detail` 存入 `TaskExecution.Result`。
- 失败：返回 `nil, fmt.Errorf("...")`。`ProcessTask` 会将错误信息存入 `TaskExecution.ErrorMessage`，状态标记为 `failed`，并将 error 返回给 Asynq 触发自动重试。

### 第 3 步：在 worker.go 注册

打开 `internal/task/worker/worker.go`，做两件事：

**1. 在 `init()` 中注册处理器**：

```go
func init() {
    task.RegisterHandler(task.CleanupUnusedUploadsTask, &upload.CleanupUnusedUploadsHandler{})
    // 添加你的：
    task.RegisterHandler(task.YourNewTask, &yourmodule.YourHandler{})
}
```

**2. 在 `StartWorker()` 的 mux 中添加路由**：

```go
mux := asynq.NewServeMux()
mux.Use(taskLoggingMiddleware)
mux.HandleFunc(task.CleanupUnusedUploadsTask, task.ProcessTask)
// 添加你的：
mux.HandleFunc(task.YourNewTask, task.ProcessTask)
```

所有路由都指向同一个 `task.ProcessTask`，它内部根据 Asynq task type 查找对应的 handler 分发执行。

### 第 4 步（可选）：添加 Cron 定时调度

如果任务需要定时执行，编辑 `internal/task/scheduler/scheduler.go`，在 `StartScheduler()` 中注册：

```go
if _, err = scheduler.Register(
    config.Config.Scheduler.CleanupUnusedUploadsTaskCron, // cron 表达式
    asynq.NewTask(task.CleanupUnusedUploadsTask, nil),
    asynq.Unique(23*time.Hour), // 防止重复执行
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

## 重试机制

项目中存在两层重试机制，互不干扰：

**Asynq 自动重试**：`ProcessTask` 返回 error 时，Asynq 根据入队时设置的 `MaxRetry` 自动重试。但 `TaskExecution` 记录在第一次失败时已标记为 `failed`，后续重试会覆盖同一条记录。

**管理端手动重试**：通过 `POST /api/v1/admin/tasks/executions/{id}/retry` 触发。会创建一个全新的 `TaskExecution` 记录（新 TaskID `retry_{count}_{originalTaskID}`），`RetryCount` 递增。校验条件：状态必须为 `failed`、`Retryable` 为 `true`、`RetryCount < MaxRetry`。

## 参考：现有任务处理器

完整的任务处理器示例参见 `internal/apps/upload/tasks.go` 中的 `CleanupUnusedUploadsHandler`，它展示了：
- 游标分页批量处理
- 每批次的 `AppendLog` 记录
- 事务内操作（DB 状态更新 + 外部调用）
- 单条失败时记录日志并 continue 而非整体终止
- 最终汇总结果通过 `TaskResult.Message` 返回

