// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package v1 contains router registrations for API V1
package v1

import (
	"github.com/Rain-kl/Wavelet/internal/apps/custom"
	"github.com/gin-gonic/gin"
)

// RegisterCustomRoutes is a scaffold SAMPLE only (demo: GET /api/v1/custom/hello).
// Real product APIs belong in apps/<domain>/ with semantic paths and a dedicated
// Register*Routes file (e.g. channel.go), not piled into this package. See skill new-api.
func RegisterCustomRoutes(apiV1Router *gin.RouterGroup) {
	customRouter := apiV1Router.Group("/custom")
	{
		customRouter.GET("/hello", custom.Hello)
	}
}
