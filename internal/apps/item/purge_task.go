// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // task handlers share a thin retention-days → domain-call shell by design
package item

import (
	"context"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/internal/infra/task"
)

const (
	// PurgeTrashTask is the Asynq task type for hard-deleting old trash items.
	PurgeTrashTask = "item:purge_trash"
	// TaskTypePurgeTrash is the admin/schedule task_type (must match migration seed).
	TaskTypePurgeTrash = "item_purge_trash"
)

// PurgeTrashMeta describes the trash purge task.
var PurgeTrashMeta = task.TaskMeta{
	Type:         TaskTypePurgeTrash,
	AsynqTask:    PurgeTrashTask,
	Name:         "Item 垃圾箱清理",
	Description:  "硬删除超时停留在垃圾箱的 Item 及其附件",
	SupportsTime: false,
	MaxRetry:     task.DefaultMaxRetry,
	Queue:        task.QueueDefault,
	Retryable:    true,
}

// PurgeTrashHandler hard-deletes trash items older than the configured retention.
type PurgeTrashHandler struct{}

// Execute loads retention days and purges overdue trash items.
func (h *PurgeTrashHandler) Execute(ctx context.Context, _ []byte) (*task.TaskResult, error) {
	days, err := repository.GetIntByKey(ctx, model.ConfigKeyItemTrashPurgeAfterDays)
	if err != nil || days <= 0 {
		days = defaultTrashPurgeDays
	}

	task.AppendLog(ctx, "开始清理超时 trash Item，阈值天数: %d", days)

	n, err := PurgeTrashItems(ctx, time.Duration(days)*24*time.Hour)
	if err != nil {
		task.AppendLog(ctx, "清理 trash Item 失败: %v", err)
		return nil, err
	}

	msg := fmt.Sprintf("Item 垃圾箱清理完成，共删除 %d 条", n)
	task.AppendLog(ctx, "%s", msg)
	return &task.TaskResult{Message: msg}, nil
}

var _ task.TaskHandler = (*PurgeTrashHandler)(nil)
