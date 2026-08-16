// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package message_gateway provides user bind/unbind APIs and credential helpers.
package message_gateway

import (
	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes mounts login-required bind/unbind APIs.
func RegisterUserRoutes(apiV1Router *gin.RouterGroup) {
	g := apiV1Router.Group("/message-gateway")
	g.Use(oauth.LoginRequired())
	{
		g.GET("/channels", ListChannels)
		g.GET("/bindings", ListBindings)
		g.POST("/bindings", BindBinding)
		g.DELETE("/bindings/:id", UnbindBinding)
	}
}
