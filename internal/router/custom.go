// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package router

import (
	"github.com/Rain-kl/Wavelet/internal/apps/custom"
	"github.com/gin-gonic/gin"
)

// registerCustomRoutes registers custom business routes to keep router.go clean and stable.
func registerCustomRoutes(apiV1Router *gin.RouterGroup) {
	customRouter := apiV1Router.Group("/custom")
	{
		customRouter.GET("/hello", custom.Hello)
	}
}
