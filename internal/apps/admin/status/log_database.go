// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"context"
	"errors"
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/infra/config"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/internal/repository/logstore"
	"github.com/Rain-kl/Wavelet/internal/shared/response"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	logDBNamePostgres       = "postgres"
	logDBNameSQLite         = "sqlite"
	logDBNameClickHouse     = "clickhouse"
	defaultLogRetentionDays = 30
)

// LogDatabaseStatus 日志库状态。
type LogDatabaseStatus struct {
	ActiveDatabase   string         `json:"active_database"`
	Migration        string         `json:"migration"`
	RetentionDays    map[string]int `json:"retention_days"`
	AvailableTargets []string       `json:"available_targets"`
}

// GetLogDatabaseStatus 返回当前日志库状态。
// @Summary 获取日志数据库状态
// @Description 返回当前日志主库、迁移状态、各库保留天数与合法迁移目标，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any{data=status.LogDatabaseStatus} "获取成功"
// @Failure 401 {object} response.Any "未登录"
// @Failure 403 {object} response.Any "无管理员权限"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/admin/status/log-database [get]
func GetLogDatabaseStatus(c *gin.Context) {
	ctx := c.Request.Context()
	store, err := logstore.Active(ctx)
	if err != nil {
		logger.ErrorF(ctx, "获取日志存储实例失败: %v", err)
		response.AbortInternal(c, "日志存储初始化失败")
		return
	}
	activeDB, err := store.Status.ActiveDatabase(ctx)
	if err != nil {
		logger.ErrorF(ctx, "获取日志库状态失败: %v", err)
		response.AbortInternal(c, "获取日志库状态失败")
		return
	}
	migration := "idle"
	if logstore.Migrating(ctx) {
		migration = "migrating"
	}
	c.JSON(http.StatusOK, response.OK(LogDatabaseStatus{
		ActiveDatabase: activeDB,
		Migration:      migration,
		RetentionDays: map[string]int{
			logDBNamePostgres:   retentionOr(ctx, model.ConfigKeyLogRetentionDaysPostgres),
			logDBNameSQLite:     retentionOr(ctx, model.ConfigKeyLogRetentionDaysSQLite),
			logDBNameClickHouse: retentionOr(ctx, model.ConfigKeyLogRetentionDaysClickHouse),
		},
		AvailableTargets: availableLogTargets(activeDB),
	}))
}

func retentionOr(ctx context.Context, key string) int {
	v, err := repository.GetIntByKey(ctx, key)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.ErrorF(ctx, "读取日志保留天数配置失败 key=%s: %v", key, err)
		}
		return defaultLogRetentionDays
	}
	if v < 1 {
		return defaultLogRetentionDays
	}
	return v
}

func availableLogTargets(active string) []string {
	if active == logDBNameClickHouse {
		if config.Config.Database.Enabled {
			return []string{logDBNamePostgres}
		}
		return []string{logDBNameSQLite}
	}
	if config.Config.ClickHouse.Enabled {
		return []string{logDBNameClickHouse}
	}
	return []string{}
}
