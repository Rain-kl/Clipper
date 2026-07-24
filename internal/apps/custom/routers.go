// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package custom is a scaffold SAMPLE, not a product business home.
// Real domains live in internal/apps/<domain>/ (sibling of oauth, user, upload).
package custom

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Rain-kl/Wavelet/internal/shared/response"
)

// Hello is a sample handler only — do not grow real product logic in this package.
// @Summary Sample Hello API
// @Description Scaffold demo API; product APIs use semantic paths under apps/<domain>
// @Tags custom
// @Produce json
// @Success 200 {object} response.Any{data=string} "成功"
// @Router /api/v1/custom/hello [get]
func Hello(c *gin.Context) {
	c.JSON(http.StatusOK, response.OK("Hello from custom business module!"))
}
