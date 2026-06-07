/*
Copyright 2025-2026 linux.do

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

package worker

import (
	"time"

	"github.com/hibiken/asynq"
	"github.com/linux-do/credit/internal/apps/upload"
	"github.com/linux-do/credit/internal/config"
	"github.com/linux-do/credit/internal/task"
)

func init() {
	// 注册所有任务处理器
	task.RegisterHandler(task.CleanupUnusedUploadsTask, &upload.CleanupUnusedUploadsHandler{})
}

// StartWorker 启动任务处理服务器
func StartWorker() error {
	asynqServer := asynq.NewServer(
		task.RedisOpt,
		asynq.Config{
			Concurrency:     config.Config.Worker.Concurrency,
			ShutdownTimeout: 3 * time.Minute,
			Queues:          buildQueuesFromConfig(),
			StrictPriority:  config.Config.Worker.StrictPriority,
		},
	)

	// 注册 Asynq 任务路由
	mux := asynq.NewServeMux()
	mux.Use(taskLoggingMiddleware)

	// 统一使用 task.ProcessTask 处理所有任务类型
	// 框架内部自动分发到对应的 TaskHandler 实现
	mux.HandleFunc(task.CleanupUnusedUploadsTask, task.ProcessTask)

	// 启动服务器
	return asynqServer.Run(mux)
}

// buildQueuesFromConfig 从配置构建队列映射
func buildQueuesFromConfig() map[string]int {
	queues := make(map[string]int)

	// 从配置读取队列
	if len(config.Config.Worker.Queues) > 0 {
		for _, q := range config.Config.Worker.Queues {
			if q.Name != "" && q.Priority > 0 {
				queues[q.Name] = q.Priority
			}
		}
	}

	// 如果配置为空，使用默认队列
	if len(queues) == 0 {
		queues = map[string]int{
			task.QueueDefault: 1,
		}
	}

	return queues
}
