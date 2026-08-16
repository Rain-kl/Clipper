// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package item

import (
	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers authenticated item REST routes under the given API group.
func RegisterRoutes(r *gin.RouterGroup) {
	var h itemHandlers
	g := r.Group("/items")
	g.Use(oauth.LoginRequired())
	{
		g.POST("", h.CreateItem)
		g.GET("", h.ListItems)
		// static paths before /:id
		g.GET("/timeline", h.GetTimeline)
		g.GET("/stats", h.GetStats)
		g.GET("/:id", h.GetItem)
		g.PATCH("/:id", h.PatchItem)
		g.DELETE("/:id", h.DeleteItem)
	}
}
