// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package message_gateway provides admin HTTP APIs for messaging channels.
package message_gateway

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts admin message-gateway APIs under /admin.
func RegisterRoutes(adminRouter *gin.RouterGroup) {
	g := adminRouter.Group("/message-gateway")
	{
		g.GET("/channels/definitions", ListChannelDefinitions)
		g.GET("/channels", ListChannels)
		g.POST("/channels", CreateChannel)
		g.PATCH("/channels/:id", UpdateChannel)
		g.DELETE("/channels/:id", DeleteChannel)
		g.POST("/channels/:id/test", TestChannel)
	}
}
