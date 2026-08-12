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

// ChannelService 示例有状态 Service（放在 internal/apps/channel/service.go）
// 需要注入 DB/客户端时使用；简单逻辑优先 logics.go 纯函数。
type ChannelService struct {
	// 例如：repo ChannelRepository
}

// NewChannelService 构造函数
func NewChannelService() *ChannelService {
	return &ChannelService{}
}

// Create 核心业务：首位参数必须是 context.Context；禁止依赖 Gin。
func (s *ChannelService) Create(ctx context.Context, userID int64, name string) (int64, error) {
	if name == "" {
		return 0, errors.New("name cannot be empty")
	}

	logger.Info(ctx, "channel service create",
		zap.Int64("user_id", userID),
		zap.String("name", name),
	)

	// DB 事务、远程调用等
	_ = fmt.Sprintf("user=%d name=%s", userID, name)
	return 1, nil
}
