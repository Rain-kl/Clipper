// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package v1 contains router registrations for API V1
package v1

import (
	capApp "github.com/Rain-kl/Wavelet/internal/apps/cap"
	publicconfig "github.com/Rain-kl/Wavelet/internal/apps/config"
	"github.com/Rain-kl/Wavelet/internal/apps/health"
	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes registers all public routes (captcha, health, config).
func RegisterPublicRoutes(apiV1Router *gin.RouterGroup, apiGroup *gin.RouterGroup) {
	// CAPTCHA
	registerCaptchaRoutes(apiGroup)

	// Health
	apiGroup.GET("/health", health.Health)

	// Config (public)
	registerConfigRoutes(apiV1Router)
}

func registerCaptchaRoutes(apiGroup *gin.RouterGroup) {
	capGroup := apiGroup.Group("/cap")
	{
		capGroup.POST("/challenge", capApp.Challenge)
		capGroup.POST("/redeem", capApp.Redeem)
	}
}

func registerConfigRoutes(apiV1Router *gin.RouterGroup) {
	configRouter := apiV1Router.Group("/config")
	{
		configRouter.GET("/public", publicconfig.GetPublicConfig)
	}
}
