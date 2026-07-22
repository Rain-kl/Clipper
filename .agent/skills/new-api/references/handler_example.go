// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package references

import (
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/common/response"
	"github.com/gin-gonic/gin"
)

// createChannelRequest 客户端请求体 DTO
type createChannelRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// createChannelResponse API 响应体 DTO
type createChannelResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// CreateChannel 示例：产品域 Handler（应放在 internal/apps/channel/routers.go）
// @Summary 创建频道
// @Description 示例：语义路径下的业务接口，而非 /api/v1/custom/...
// @Tags channel
// @Accept json
// @Produce json
// @Param request body createChannelRequest true "业务请求参数"
// @Success 200 {object} response.Any{data=createChannelResponse} "操作成功"
// @Failure 400 {object} response.Any "参数错误"
// @Failure 401 {object} response.Any "未登录"
// @Router /api/v1/channels [post]
func CreateChannel(c *gin.Context) {
	var req createChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, "参数校验失败")
		return
	}

	// 通常结合 oauth.LoginRequired()；此处仅演示从上下文取用户
	userID := int64(9527)

	result, err := CreateChannelLogic(c.Request.Context(), userID, req.Name)
	if err != nil {
		response.AbortBadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, response.OK(createChannelResponse{
		ID:   result.ID,
		Name: result.Name,
	}))
}
