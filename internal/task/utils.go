/*
Copyright 2025 linux.do

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package task

import (
	"github.com/hibiken/asynq"
	"github.com/linux-do/credit/internal/config"
)

// RedisOpt asynq Redis 连接配置（兼容 Standalone/Sentinel/Cluster）
var RedisOpt asynq.RedisConnOpt

func init() {
	RedisOpt = NewRedisConnOpt()
}

// NewRedisConnOpt 根据配置返回对应的 asynq Redis 连接选项
func NewRedisConnOpt() asynq.RedisConnOpt {
	cfg := config.Config.Redis
	addrs := cfg.Addrs

	if cfg.ClusterMode {
		return asynq.RedisClusterClientOpt{
			Addrs:    addrs,
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	if cfg.MasterName != "" {
		return asynq.RedisFailoverClientOpt{
			MasterName:    cfg.MasterName,
			SentinelAddrs: addrs,
			Username:      cfg.Username,
			Password:      cfg.Password,
			DB:            cfg.DB,
		}
	}

	addr := "localhost:6379"
	if len(addrs) > 0 {
		addr = addrs[0]
	}
	return asynq.RedisClientOpt{
		Addr:     addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	}
}

// PrefixedQueue 返回带前缀的队列名，用于 Cluster 模式隔离
func PrefixedQueue(queue string) string {
	prefix := config.Config.Redis.KeyPrefix
	if prefix == "" {
		return queue
	}
	return prefix + queue
}
