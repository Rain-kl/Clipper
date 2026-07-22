// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package references

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rain-kl/Wavelet/pkg/logger"
	"go.uber.org/zap"
)

// channelCreated 示例 logics 返回值（真实代码可用 model 或专用 DTO）
type channelCreated struct {
	ID   int64
	Name string
}

// CreateChannelLogic 示例：模块内闭环业务（放在 apps/channel/logics.go）
// 接收 context.Context，不依赖 gin.Context，便于单测与 Worker 复用。
func CreateChannelLogic(ctx context.Context, userID int64, name string) (*channelCreated, error) {
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}

	logger.Info(ctx, "creating channel",
		zap.Int64("user_id", userID),
		zap.String("name", name),
	)

	// 轻量级本地逻辑；复杂持久化可进 model/repository
	return &channelCreated{
		ID:   1,
		Name: fmt.Sprintf("%s (by %d)", name, userID),
	}, nil
}
