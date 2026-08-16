// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // task handlers share a thin retention-days → domain-call shell by design
package item

import (
	"context"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/infra/task"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
)

const (
	// ArchivePendingTask is the Asynq task type for archiving stale pending items.
	ArchivePendingTask = "item:archive_pending"
	// TaskTypeArchivePending is the admin/schedule task_type (must match migration seed).
	TaskTypeArchivePending = "item_archive_pending"
)

// ArchivePendingMeta describes the pending-item archive task.
var ArchivePendingMeta = task.TaskMeta{
	Type:         TaskTypeArchivePending,
	AsynqTask:    ArchivePendingTask,
	Name:         "Item 未处理归档",
	Description:  "将超时未处理的 Item 归档为片段",
	SupportsTime: false,
	MaxRetry:     task.DefaultMaxRetry,
	Queue:        task.QueueDefault,
	Retryable:    true,
}

// ArchivePendingHandler archives pending items older than the configured retention.
type ArchivePendingHandler struct{}

// Execute loads retention days and archives overdue pending items.
func (h *ArchivePendingHandler) Execute(ctx context.Context, _ []byte) (*task.TaskResult, error) {
	days, err := repository.GetIntByKey(ctx, model.ConfigKeyItemPendingArchiveAfterDays)
	if err != nil || days <= 0 {
		days = defaultPendingArchiveDays
	}

	task.AppendLog(ctx, "开始归档超时 pending Item，阈值天数: %d", days)

	n, err := ArchivePendingItems(ctx, time.Duration(days)*24*time.Hour)
	if err != nil {
		task.AppendLog(ctx, "归档 pending Item 失败: %v", err)
		return nil, err
	}

	msg := fmt.Sprintf("Item 未处理归档完成，共归档 %d 条", n)
	task.AppendLog(ctx, "%s", msg)
	return &task.TaskResult{Message: msg}, nil
}

var _ task.TaskHandler = (*ArchivePendingHandler)(nil)
