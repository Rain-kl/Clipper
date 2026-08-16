// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package logstore abstracts user access-log storage across ClickHouse, PostgreSQL and SQLite.
package logstore

import (
	"context"
	"errors"
	"time"

	analyticsmodel "github.com/Rain-kl/Wavelet/internal/model/analytics"
	analyticsrepo "github.com/Rain-kl/Wavelet/internal/repository/analytics"
)

// ErrMigrating 表示日志数据库正在迁移，当前禁止写入。
var ErrMigrating = errors.New("log database is migrating, writes are disabled")

// UserAccessLogStore 用户访问日志（w_user_access_logs）。
type UserAccessLogStore interface {
	BatchInsert(ctx context.Context, logs []analyticsmodel.UserAccessLog) error
	DeleteAll(ctx context.Context) (int64, error)
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
	Count(ctx context.Context, filter analyticsrepo.AccessLogFilter) (uint64, error)
	List(ctx context.Context, filter analyticsrepo.AccessLogFilter, page, pageSize int) ([]analyticsmodel.UserAccessLog, uint64, error)
	GetDailyTrend(ctx context.Context, days int) ([]analyticsrepo.DailyTrend, error)
	GetBrowserDistribution(ctx context.Context, startTime time.Time) ([]analyticsrepo.BrowserShare, error)
	GetTopActiveUsers(ctx context.Context, startTime time.Time, limit int) ([]analyticsrepo.TopUser, error)
	ListForMigration(ctx context.Context, afterID uint64, limit int) ([]analyticsmodel.UserAccessLog, error)
	MigrationRange(ctx context.Context) (from, to time.Time, err error)
	EnsurePartitions(ctx context.Context, from, to time.Time) error
}

// StatusStore 日志库状态。
type StatusStore interface {
	ActiveDatabase(ctx context.Context) (string, error)
}

// Store 当前生效日志库。
type Store struct {
	UserAccessLogs UserAccessLogStore
	Status         StatusStore
}
