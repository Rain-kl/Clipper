// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package router

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	admin_push "github.com/Rain-kl/Wavelet/internal/apps/admin/push"
	publicconfig "github.com/Rain-kl/Wavelet/internal/apps/config"
	"github.com/Rain-kl/Wavelet/internal/apps/risk_control"
	"github.com/Rain-kl/Wavelet/internal/apps/upload"
	v1 "github.com/Rain-kl/Wavelet/internal/router/v1"

	// Swagger 文档生成
	_ "github.com/Rain-kl/Wavelet/docs"
	_ "github.com/Rain-kl/Wavelet/internal/apps/admin/push/custom_events"
	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/config"
	otel_trace "github.com/Rain-kl/Wavelet/pkg/trace"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Serve 启动 HTTP API 服务
func Serve() {
	// 运行模式
	if config.Config.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化 ClickHouse 异步日志写入器
	risk_control.InitLogWriter()

	// 运行内置事件同步
	if err := admin_push.SyncEvents(context.Background()); err != nil {
		log.Printf("[API] sync push events failed: %v\n", err)
	}

	// 初始化路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	cfg := config.Config.Redis
	addrs := cfg.Addrs
	sessionAddr := "localhost:6379"
	if len(addrs) > 0 {
		sessionAddr = addrs[0]
	}

	sessionStore, err := redis.NewStoreWithDB(
		cfg.MinIdleConn,
		"tcp",
		sessionAddr,
		cfg.Username,
		cfg.Password,
		strconv.Itoa(cfg.DB),
		[]byte(config.Config.App.SessionSecret),
	)
	if err != nil {
		log.Fatalf("[API] init session store failed: %v\n", err)
	}

	// 设置 Session Redis Key 前缀
	if cfg.KeyPrefix != "" {
		if err := redis.SetKeyPrefix(sessionStore, cfg.KeyPrefix+"session:"); err != nil {
			log.Printf("[API] set session key prefix failed: %v\n", err)
		}
	}

	sessionStore.Options(oauth.GetSessionOptions(config.Config.App.SessionAge))

	r.Use(sessions.Sessions(config.Config.App.SessionCookieName, sessionStore))

	// 补充中间件
	r.Use(otelgin.Middleware(config.Config.App.AppName), loggerMiddleware(), risk_control.RiskControlMiddleware())

	registerRoutes(r)

	srv := &http.Server{
		Addr:              config.Config.App.Addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("[API] server starting on %s\n", config.Config.App.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[API] server failed: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Config.App.GracefulShutdownTimeout)*time.Second)

	otel_trace.Shutdown(shutdownCtx)

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[API] server forced to shutdown: %v\n", err)
		cancel()
		os.Exit(1)
	}
	cancel()

	log.Println("[API] server exited")
}

func registerRoutes(r *gin.Engine) {
	// Serve files by ID
	r.GET("/f/:id", upload.ServeFileByID)

	// Dynamic robots.txt serving
	r.GET("/robots.txt", publicconfig.GetRobotsTXT)

	apiGroup := r.Group(config.Config.App.APIPrefix)
	{
		if !config.Config.App.IsProduction() {
			// Swagger
			apiGroup.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}

		// API V1
		apiV1Router := apiGroup.Group("/v1")
		{
			// Public (captcha, health, config)
			v1.RegisterPublicRoutes(apiV1Router, apiGroup)

			// OAuth, User, Upload
			v1.RegisterUserRoutes(apiV1Router)

			// Admin
			v1.RegisterAdminRoutes(apiV1Router)

			// Register custom business routes
			v1.RegisterCustomRoutes(apiV1Router)
		}
	}

	// 注册前端静态路由（当启用 embed_frontend 编译标签时）
	registerFrontend(r)
}

var registerFrontend = func(_ *gin.Engine) {
	// No-op when not embedding frontend
}
