// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	adminmsg "github.com/Rain-kl/Wavelet/internal/apps/admin/message_gateway"
	"github.com/gin-gonic/gin"
)

func registerAdminMessageGatewayRoutes(adminRouter *gin.RouterGroup) {
	adminmsg.RegisterRoutes(adminRouter)
}
