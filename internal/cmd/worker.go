// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"log"

	gwrunner "github.com/Rain-kl/Wavelet/internal/apps/message_gateway/runner"
	"github.com/Rain-kl/Wavelet/internal/infra/task/worker"
	"github.com/Rain-kl/Wavelet/internal/platform/bootstrap"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "wavelet Worker",
	Run: func(_ *cobra.Command, _ []string) {
		runBootstrap(bootstrap.Options{})
		printStartupBanner(startupState{mode: "Worker", relationalDB: latestMigrationState.relationalDB, clickHouseDB: latestMigrationState.clickHouseDB})
		go func() {
			if err := gwrunner.Start(context.Background()); err != nil {
				log.Printf("[Worker] message gateway stopped: %v", err)
			}
		}()
		log.Println("[Worker] 启动任务处理服务")
		if err := worker.StartWorker(); err != nil {
			log.Fatalf("[工作器] 启动失败: %v", err)
		}
	},
}
