// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"github.com/Rain-kl/Wavelet/internal/apps/item"
	"github.com/gin-gonic/gin"
)

// RegisterItemRoutes registers Clipper item capture APIs under /api/v1/items.
func RegisterItemRoutes(apiV1Router *gin.RouterGroup) {
	item.RegisterRoutes(apiV1Router)
}
