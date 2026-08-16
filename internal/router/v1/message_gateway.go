// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	appgw "github.com/Rain-kl/Wavelet/internal/apps/message_gateway"
	"github.com/gin-gonic/gin"
)

// RegisterMessageGatewayUserRoutes mounts user bind/unbind APIs.
func RegisterMessageGatewayUserRoutes(apiV1Router *gin.RouterGroup) {
	appgw.RegisterUserRoutes(apiV1Router)
}
