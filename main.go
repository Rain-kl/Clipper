// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package main 是 Clipper 程序入口
package main

import "github.com/Rain-kl/Wavelet/internal/cmd"

// @title Clipper API
// @version 1.0.0
// @description Clipper 后端 API：捕获条目、用户认证、系统配置、任务调度与文件能力。
// @contact.name Clipper
// @contact.url https://github.com/Rain-kl/Clipper
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /
// @securityDefinitions.apikey SessionCookie
// @in cookie
// @name session
func main() {
	cmd.Execute()
}
