/*
 * MIT License
 *
 * Copyright (c) 2025 linux.do
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package listener

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/linux-do/credit/internal/config"
	"github.com/linux-do/credit/internal/db"
	"github.com/linux-do/credit/internal/logger"
	"github.com/linux-do/credit/internal/model"
	"github.com/redis/go-redis/v9"
)

// orderExpireKeyPrefix 订单过期 Key 前缀
const orderExpireKeyPrefix = "payment:order:expire:"

// StartExpireListener 启动过期监听器
func StartExpireListener(ctx context.Context) error {
	if db.Redis == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	// 初始化时先处理已过期的订单
	model.ExpirePendingOrders(ctx)

	cfg := config.Config.Redis

	// Cluster 模式：订阅所有分片节点
	if cfg.ClusterMode {
		clusterClient, ok := db.Redis.(*redis.ClusterClient)
		if !ok {
			return fmt.Errorf("redis client is not a ClusterClient")
		}
		return startClusterExpireListener(ctx, clusterClient)
	}

	// Standalone/Sentinel 模式
	return startStandaloneExpireListener(ctx, cfg.DB)
}

// startStandaloneExpireListener Standalone/Sentinel 模式的过期监听
func startStandaloneExpireListener(ctx context.Context, dbIndex int) error {
	// 确保 Redis 开启 keyspace notifications
	configResult := db.Redis.ConfigSet(ctx, "notify-keyspace-events", "Ex")
	if configResult.Err() != nil {
		log.Printf("[Expire Listener] 警告: 设置 keyspace notifications 失败: %v", configResult.Err())
		log.Printf("[Expire Listener] 请手动执行: CONFIG SET notify-keyspace-events Ex")
		return configResult.Err()
	}

	// 订阅过期事件频道
	expiredChannel := fmt.Sprintf("__keyevent@%d__:expired", dbIndex)
	pubSub := db.Redis.Subscribe(ctx, expiredChannel)

	go subscribeExpireEvents(ctx, pubSub, expiredChannel)

	return nil
}

// startClusterExpireListener Cluster 模式的过期监听
// 需要订阅所有分片节点的过期事件
func startClusterExpireListener(ctx context.Context, clusterClient *redis.ClusterClient) error {
	expiredChannel := "__keyevent@0__:expired"

	err := clusterClient.ForEachShard(ctx, func(ctx context.Context, shard *redis.Client) error {
		// 为每个分片设置 keyspace notifications
		if err := shard.ConfigSet(ctx, "notify-keyspace-events", "Ex").Err(); err != nil {
			log.Printf("[Expire Listener] 警告: 分片 %s 设置 keyspace notifications 失败: %v",
				shard.Options().Addr, err)
			// 继续处理其他分片
		}

		// 订阅该分片的过期事件
		pubSub := shard.Subscribe(ctx, expiredChannel)
		go subscribeExpireEvents(ctx, pubSub, fmt.Sprintf("%s (shard: %s)", expiredChannel, shard.Options().Addr))

		return nil
	})

	return err
}

// subscribeExpireEvents 订阅并处理过期事件
func subscribeExpireEvents(ctx context.Context, pubSub *redis.PubSub, channelDesc string) {
	defer pubSub.Close()
	log.Printf("[Expire Listener] 过期监听器已启动，监听频道: %s", channelDesc)

	for {
		msg, err := pubSub.ReceiveMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Printf("[Expire Listener] 监听器已停止: %s", channelDesc)
				return
			}
			logger.ErrorF(ctx, "接收 Redis 过期事件失败: %v", err)
			continue
		}

		// 处理过期事件
		handleExpiredKey(ctx, msg.Payload)
	}
}

// handleExpiredKey 处理过期的 Redis key
func handleExpiredKey(ctx context.Context, expiredKey string) {
	fullPrefix := db.PrefixedKey(orderExpireKeyPrefix)

	// 只处理订单过期相关的 key
	if !strings.HasPrefix(expiredKey, fullPrefix) {
		return
	}

	orderIDStr := strings.TrimPrefix(expiredKey, fullPrefix)
	orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
	if err != nil {
		logger.ErrorF(ctx, "解析订单ID失败: key=%s, error=%v", expiredKey, err)
		return
	}

	// 更新订单状态为过期
	result := db.DB(ctx).Model(&model.Order{}).
		Where("id = ? AND status = ?", orderID, model.OrderStatusPending).
		Update("status", model.OrderStatusExpired)

	if result.Error != nil {
		logger.ErrorF(ctx, "更新订单状态为过期失败: order_id=%d, error=%v", orderID, result.Error)
	} else if result.RowsAffected > 0 {
		logger.InfoF(ctx, "订单已过期: order_id=%d", orderID)
	}
}
