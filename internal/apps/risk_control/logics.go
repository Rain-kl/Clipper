// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package risk_control

import (
	"context"
	"time"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/model/analytics"
	analyticsrepo "github.com/Rain-kl/Wavelet/internal/repository/analytics"
	"github.com/Rain-kl/Wavelet/pkg/logger"
)

var logChan chan *analytics.UserAccessLog

const (
	defaultQueueSize = 10000
	maxBatchSize     = 1000
	flushInterval    = 1 * time.Second
)

// InitLogWriter 初始化日志写入通道和后台写入协程
func InitLogWriter(ctx context.Context) {
	if !config.Config.ClickHouse.Enabled {
		return
	}

	logChan = make(chan *analytics.UserAccessLog, defaultQueueSize)
	go startBatchWorker(context.WithoutCancel(ctx))
}

// IsBufferFull 检查当前本地缓冲队列是否已满
// 如果没有启用 ClickHouse，默认返回 false，不触发限流
func IsBufferFull() bool {
	if !config.Config.ClickHouse.Enabled || logChan == nil {
		return false
	}
	return len(logChan) >= cap(logChan)
}

// QueueAccessLog 异步非阻塞地将日志推入缓冲队列
func QueueAccessLog(logItem *analytics.UserAccessLog) {
	if !config.Config.ClickHouse.Enabled || logChan == nil {
		return
	}

	select {
	case logChan <- logItem:
	default:
		logger.WarnF(context.Background(), "[RiskControl] Log queue full, dropping log item for path: %s", logItem.Path)
	}
}

func startBatchWorker(ctx context.Context) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	var batch []*analytics.UserAccessLog

	flush := func() {
		if len(batch) == 0 {
			return
		}

		items := make([]analytics.UserAccessLog, len(batch))
		for i, item := range batch {
			items[i] = *item
		}
		if err := analyticsrepo.BatchInsert(ctx, items); err != nil {
			logger.ErrorF(ctx, "[RiskControl] Send ClickHouse batch failed: %v", err)
		}
		batch = nil
	}

	for {
		select {
		case item, ok := <-logChan:
			if !ok {
				flush()
				return
			}
			batch = append(batch, item)
			if len(batch) >= maxBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}