// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package v1 contains router registrations for API V1
package v1

import (
	"github.com/Rain-kl/Wavelet/internal/apps/admin"
	admin_auth_source "github.com/Rain-kl/Wavelet/internal/apps/admin/auth_source"
	admin_cache "github.com/Rain-kl/Wavelet/internal/apps/admin/cache"
	admin_db_manage "github.com/Rain-kl/Wavelet/internal/apps/admin/db_manage"
	admin_logs "github.com/Rain-kl/Wavelet/internal/apps/admin/logs"
	admin_push "github.com/Rain-kl/Wavelet/internal/apps/admin/push"
	admin_status "github.com/Rain-kl/Wavelet/internal/apps/admin/status"
	admin_task "github.com/Rain-kl/Wavelet/internal/apps/admin/task"
	admin_template "github.com/Rain-kl/Wavelet/internal/apps/admin/template"
	admin_updater "github.com/Rain-kl/Wavelet/internal/apps/admin/updater"
	admin_user "github.com/Rain-kl/Wavelet/internal/apps/admin/user"
	"github.com/Rain-kl/Wavelet/internal/apps/admin/system_config"
	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/apps/upload"
	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes registers all admin-related routes.
func RegisterAdminRoutes(apiV1Router *gin.RouterGroup) {
	adminRouter := apiV1Router.Group("/admin")
	adminRouter.Use(oauth.LoginRequired(), admin.LoginAdminRequired())
	{
		// System status
		adminRouter.GET("/status", admin_status.GetSystemStatus)

		// Database info & export
		adminRouter.GET("/db-info", admin_status.GetDatabaseInfo)
		adminRouter.GET("/db-export", admin_status.ExportDatabase)

		// Database management
		adminRouter.GET("/db-manage/overview", admin_db_manage.GetDBOverview)
		adminRouter.GET("/db-manage/tables", admin_db_manage.ListDBTables)
		adminRouter.GET("/db-manage/table-data", admin_db_manage.GetDBTableData)
		adminRouter.POST("/db-manage/query", admin_db_manage.ExecuteSQL)

		// Cache management
		adminRouter.GET("/cache/status", admin_cache.GetCacheStatus)
		adminRouter.POST("/cache/config", admin_cache.UpdateCacheConfig)
		adminRouter.POST("/cache/clear", admin_cache.ClearCache)

		// Application update
		adminRouter.GET("/update", admin_updater.GetUpdateStatus)
		adminRouter.POST("/update/apply", admin_updater.ApplyUpdate)

		// System logs
		adminRouter.GET("/logs", admin_logs.GetLogs)
		adminRouter.GET("/logs/access", admin_logs.GetAccessLogs)
		adminRouter.GET("/logs/analytics", admin_logs.GetLogsAnalytics)
		adminRouter.GET("/logs/ws", admin_logs.HandleLogWebSocket)

		// Task dispatch
		registerAdminTaskRoutes(adminRouter)

		// Users
		adminRouter.GET("/users", admin_user.ListUsers)
		adminRouter.POST("/users", admin_user.CreateUser)
		adminRouter.GET("/users/:id", admin_user.GetUser)
		adminRouter.PUT("/users/:id/status", admin_user.UpdateUserStatus)
		adminRouter.DELETE("/users/:id", admin_user.DeleteUser)

		// Uploads
		registerAdminUploadRoutes(adminRouter)

		// System Config
		adminRouter.POST("/system-configs", system_config.CreateSystemConfig)
		adminRouter.GET("/system-configs", system_config.ListSystemConfigs)
		adminRouter.POST("/system-configs/smtp/test", system_config.TestSMTP)

		systemConfigRouter := adminRouter.Group("/system-configs/:key")
		{
			systemConfigRouter.GET("", system_config.GetSystemConfig)
			systemConfigRouter.PUT("", system_config.UpdateSystemConfig)
		}

		// Templates
		adminRouter.GET("/templates", admin_template.ListTemplates)
		adminRouter.POST("/templates", admin_template.CreateTemplate)

		templateRouter := adminRouter.Group("/templates/:key")
		{
			templateRouter.GET("", admin_template.GetTemplate)
			templateRouter.PUT("", admin_template.UpdateTemplate)
			templateRouter.DELETE("", admin_template.DeleteTemplate)
		}

		// Auth Sources
		adminRouter.GET("/auth-sources", admin_auth_source.ListAuthSources)
		adminRouter.POST("/auth-sources", admin_auth_source.CreateAuthSource)
		adminRouter.PUT("/auth-sources/:id", admin_auth_source.UpdateAuthSource)
		adminRouter.PUT("/auth-sources/:id/toggle", admin_auth_source.ToggleAuthSource)
		adminRouter.DELETE("/auth-sources/:id", admin_auth_source.DeleteAuthSource)

		// Push Notifications
		adminRouter.GET("/push/events", admin_push.ListEvents)
		adminRouter.GET("/push/events/builtin", admin_push.ListBuiltInEvents)
		adminRouter.POST("/push/events", admin_push.CreateEvent)
		adminRouter.PUT("/push/events/:id", admin_push.UpdateEvent)
		adminRouter.DELETE("/push/events/:id", admin_push.DeleteEvent)
		adminRouter.POST("/push/events/:id/toggle", admin_push.ToggleEvent)
		adminRouter.GET("/push/histories", admin_push.ListHistories)
		adminRouter.POST("/push/test", admin_push.TestPush)

		// Message Channels CRUD
		adminRouter.GET("/push/channels/definitions", admin_push.ListChannelDefinitions)
		adminRouter.GET("/push/channels", admin_push.ListChannels)
		adminRouter.POST("/push/channels", admin_push.CreateChannel)
		adminRouter.PUT("/push/channels/:id", admin_push.UpdateChannel)
		adminRouter.DELETE("/push/channels/:id", admin_push.DeleteChannel)
		adminRouter.POST("/push/channels/test", admin_push.TestChannel)
	}
}

func registerAdminTaskRoutes(adminRouter *gin.RouterGroup) {
	// Task dispatch
	adminRouter.GET("/tasks/types", admin_task.ListTaskTypes)
	adminRouter.POST("/tasks/dispatch", admin_task.DispatchTask)

	// Task executions
	adminRouter.GET("/tasks/executions", admin_task.ListTaskExecutions)
	adminRouter.GET("/tasks/executions/:id", admin_task.GetTaskExecution)
	adminRouter.POST("/tasks/executions/:id/retry", admin_task.RetryTask)

	// Task schedules
	adminRouter.GET("/tasks/schedules", admin_task.ListSchedules)
	adminRouter.POST("/tasks/schedules", admin_task.CreateSchedule)
	adminRouter.PUT("/tasks/schedules/:id", admin_task.UpdateSchedule)
	adminRouter.DELETE("/tasks/schedules/:id", admin_task.DeleteSchedule)
}

func registerAdminUploadRoutes(adminRouter *gin.RouterGroup) {
	adminUploadsRouter := adminRouter.Group("/uploads")
	{
		adminUploadsRouter.GET("", upload.ListFiles)
		adminUploadsRouter.GET("/stats", upload.GetFileStats)
		adminUploadsRouter.DELETE("/:id", upload.DeleteFile)
		adminUploadsRouter.GET("/download/:id", upload.DownloadFile)
		adminUploadsRouter.POST("/download/batch", upload.BatchDownloadFiles)
		adminUploadsRouter.GET("/types", upload.GetDistinctUploadTypes)
	}
}
