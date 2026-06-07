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

	"github.com/linux-do/credit/internal/apps/admin"
	admin_auth_source "github.com/linux-do/credit/internal/apps/admin/auth_source"
	admin_task "github.com/linux-do/credit/internal/apps/admin/task"
	admin_user "github.com/linux-do/credit/internal/apps/admin/user"
	publicconfig "github.com/linux-do/credit/internal/apps/config"
	"github.com/linux-do/credit/internal/apps/health"
	"github.com/linux-do/credit/internal/apps/upload"
	"github.com/linux-do/credit/internal/apps/user"
	"github.com/linux-do/credit/internal/util"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	_ "github.com/linux-do/credit/docs"
	"github.com/linux-do/credit/internal/apps/admin/system_config"
	"github.com/linux-do/credit/internal/apps/oauth"
	"github.com/linux-do/credit/internal/config"
	"github.com/linux-do/credit/internal/otel_trace"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func Serve() {
	// 运行模式
	if config.Config.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化路由
	r := gin.New()
	r.Use(gin.Recovery())

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

	sessionStore.Options(util.GetSessionOptions(config.Config.App.SessionAge))

	r.Use(sessions.Sessions(config.Config.App.SessionCookieName, sessionStore))

	// 补充中间件
	r.Use(otelgin.Middleware(config.Config.App.AppName), loggerMiddleware())

	// Serve files by ID
	r.GET("/f/:id", upload.ServeFileByID)

	apiGroup := r.Group(config.Config.App.APIPrefix)
	{
		if !config.Config.App.IsProduction() {
			// Swagger
			apiGroup.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}

		// API V1
		apiV1Router := apiGroup.Group("/v1")
		{
			// Health
			apiV1Router.GET("/health", health.Health)

			// OAuth
			apiV1Router.GET("/oauth/sources", oauth.GetLoginSources)
			apiV1Router.GET("/oauth/login", oauth.GetLoginURL)
			apiV1Router.GET("/oauth/:source/authorize", oauth.Authorize)
			apiV1Router.GET("/oauth/logout", oauth.Logout)
			apiV1Router.POST("/oauth/callback", oauth.Callback)
			apiV1Router.GET("/oauth/user-info", oauth.LoginRequired(), oauth.UserInfo)
			apiV1Router.GET("/oauth/external-accounts", oauth.LoginRequired(), oauth.ListExternalAccounts)
			apiV1Router.POST("/oauth/external-accounts/:id/delete", oauth.LoginRequired(), oauth.DeleteExternalAccount)

			// User
			userRouter := apiV1Router.Group("/user")
			{
				userRouter.POST("/login", user.Login)
				userRouter.POST("/register", user.Register)
				userRouter.GET("/logout", user.Logout)
				userRouter.GET("/self", oauth.LoginRequired(), oauth.UserInfo)

				// Access Token
				tokenRouter := userRouter.Group("/access-tokens")
				tokenRouter.Use(oauth.LoginRequired())
				{
					tokenRouter.GET("", user.ListAccessTokens)
					tokenRouter.POST("", user.CreateAccessToken)
					tokenRouter.DELETE("/:id", user.DeleteAccessToken)
					tokenRouter.POST("/:id/rotate", user.RotateAccessToken)
				}
			}

			// Upload
			uploadRouter := apiV1Router.Group("/upload")
			uploadRouter.Use(oauth.LoginRequired())
			{
				uploadRouter.POST("", upload.UploadFile)
				uploadRouter.GET("/my", upload.ListMyFiles)
				uploadRouter.DELETE("/:id", upload.DeleteFile)
				uploadRouter.GET("/download/:id", upload.DownloadFile)
				uploadRouter.POST("/download/batch", upload.BatchDownloadFiles)
			}

			// Config (public)
			configRouter := apiV1Router.Group("/config")
			{
				configRouter.GET("/public", publicconfig.GetPublicConfig)
			}

			// Admin
			adminRouter := apiV1Router.Group("/admin")
			adminRouter.Use(oauth.LoginRequired(), admin.LoginAdminRequired())
			{
				// Task dispatch
				adminRouter.GET("/tasks/types", admin_task.ListTaskTypes)
				adminRouter.POST("/tasks/dispatch", admin_task.DispatchTask)

				// Users
				adminRouter.GET("/users", admin_user.ListUsers)
				adminRouter.PUT("/users/:id/status", admin_user.UpdateUserStatus)

				// System Config
				adminRouter.POST("/system-configs", system_config.CreateSystemConfig)
				adminRouter.GET("/system-configs", system_config.ListSystemConfigs)

				systemConfigRouter := adminRouter.Group("/system-configs/:key")
				{
					systemConfigRouter.GET("", system_config.GetSystemConfig)
					systemConfigRouter.PUT("", system_config.UpdateSystemConfig)
					systemConfigRouter.DELETE("", system_config.DeleteSystemConfig)
				}

				// Auth Sources
				adminRouter.GET("/auth-sources", admin_auth_source.ListAuthSources)
				adminRouter.POST("/auth-sources", admin_auth_source.CreateAuthSource)
				adminRouter.PUT("/auth-sources/:id", admin_auth_source.UpdateAuthSource)
				adminRouter.PUT("/auth-sources/:id/toggle", admin_auth_source.ToggleAuthSource)
				adminRouter.DELETE("/auth-sources/:id", admin_auth_source.DeleteAuthSource)
			}
		}
	}

	srv := &http.Server{
		Addr:    config.Config.App.Addr,
		Handler: r,
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
	defer cancel()

	otel_trace.Shutdown(shutdownCtx)

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[API] server forced to shutdown: %v\n", err)
	}

	log.Println("[API] server exited")
}
