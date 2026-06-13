// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package handlers 注册异步任务处理器
package handlers

import (
	"github.com/Rain-kl/Wavelet/internal/apps/upload"
	"github.com/Rain-kl/Wavelet/internal/apps/user"
	"github.com/Rain-kl/Wavelet/internal/task"
)

// Register registers all built-in task handlers and their metadata.
func Register() {
	task.RegisterHandler(upload.StorageMigrationTask, &upload.MigrationHandler{})
	task.RegisterTaskMeta(upload.StorageMigrationMeta)

	// upload
	task.RegisterHandler(upload.CleanupUnusedUploadsTask, &upload.CleanupUnusedUploadsHandler{})
	task.RegisterTaskMeta(upload.CleanupUnusedUploadsMeta)
	task.RegisterHandler(upload.WarmImageCacheTask, &upload.WarmImageCacheHandler{})
	task.RegisterTaskMeta(upload.WarmImageCacheMeta)

	// user
	task.RegisterHandler(user.SendEmailTask, &user.SendEmailHandler{})
	task.RegisterTaskMeta(user.SendEmailMeta)
}
