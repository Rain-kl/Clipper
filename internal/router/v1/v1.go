// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package v1 contains router registrations for API V1
package v1

import (
	"github.com/gin-gonic/gin"
)

// RegisterV1Routes registers all routes under API V1.
func RegisterV1Routes(apiV1Router *gin.RouterGroup, apiGroup *gin.RouterGroup) {
	// 1. Public (captcha, health, config)
	RegisterPublicRoutes(apiV1Router, apiGroup)

	// 2. OAuth, User, Upload
	RegisterUserRoutes(apiV1Router)

	// 3. Admin
	RegisterAdminRoutes(apiV1Router)

	// 4. Register custom business routes
	RegisterCustomRoutes(apiV1Router)
}
