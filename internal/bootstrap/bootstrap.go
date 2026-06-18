// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package bootstrap wires cross-module integrations at the application composition root.
// All registrations use sync.Once so entry points can call them safely without import-order side effects.
package bootstrap

import (
	"sync"

	admin_push "github.com/Rain-kl/Wavelet/internal/apps/admin/push"
	"github.com/Rain-kl/Wavelet/internal/apps/admin/push/custom_events"
	taskhandlers "github.com/Rain-kl/Wavelet/internal/task/handlers"
)

var (
	registerTasksOnce            sync.Once
	registerPushDomainEventsOnce sync.Once
	registerTaskListenersOnce    sync.Once
)

// RegisterTasks registers all built-in task handlers and metadata.
func RegisterTasks() {
	registerTasksOnce.Do(func() {
		taskhandlers.Register()
	})
}

// RegisterPushDomainEvents wires push notification handlers for domain events.
func RegisterPushDomainEvents() {
	registerPushDomainEventsOnce.Do(func() {
		custom_events.Register()
	})
}

// RegisterTaskListeners wires operational listeners to task framework hooks.
func RegisterTaskListeners() {
	registerTaskListenersOnce.Do(func() {
		admin_push.RegisterTaskListeners()
	})
}

// RegisterAPI wires integrations required by the HTTP API process.
func RegisterAPI() {
	RegisterTasks()
	RegisterPushDomainEvents()
}

// RegisterWorker wires integrations required by the task worker process.
func RegisterWorker() {
	RegisterTasks()
	RegisterTaskListeners()
}

// RegisterScheduler wires integrations required by the task scheduler process.
func RegisterScheduler() {
	RegisterTasks()
}

// RegisterAll wires integrations for fused mode (API + Worker + Scheduler).
func RegisterAll() {
	RegisterTasks()
	RegisterPushDomainEvents()
	RegisterTaskListeners()
}