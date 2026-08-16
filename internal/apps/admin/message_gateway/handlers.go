// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package message_gateway

import (
	"net/http"
	"strconv"

	"github.com/Rain-kl/Wavelet/internal/shared/response"
	"github.com/gin-gonic/gin"
)

// ListChannelDefinitions returns form schemas for supported channel types.
// @Summary List message gateway channel definitions
// @Description Returns form field definitions for Telegram and QQ channels
// @Tags admin-message-gateway
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any{data=[]Definition}
// @Router /api/v1/admin/message-gateway/channels/definitions [get]
func ListChannelDefinitions(c *gin.Context) {
	c.JSON(http.StatusOK, response.OK(channelDefinitions()))
}

// ListChannels lists configured messaging channels with secrets masked.
// @Summary List message gateway channels
// @Description Returns all messaging channels; secrets are masked
// @Tags admin-message-gateway
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any{data=[]ChannelDTO}
// @Router /api/v1/admin/message-gateway/channels [get]
func ListChannels(c *gin.Context) {
	rows, err := listChannels(c.Request.Context())
	if err != nil {
		response.AbortInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.OK(rows))
}

// CreateChannel creates a messaging channel.
// @Summary Create message gateway channel
// @Description Creates a Telegram or QQ channel with encrypted credentials
// @Tags admin-message-gateway
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body CreateChannelRequest true "create body"
// @Success 200 {object} response.Any{data=ChannelDTO}
// @Failure 400 {object} response.Any
// @Router /api/v1/admin/message-gateway/channels [post]
func CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, err.Error())
		return
	}
	dto, err := createChannel(c.Request.Context(), req)
	if err != nil {
		response.AbortBadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.OK(dto))
}

// UpdateChannel patches a messaging channel. Empty secrets keep the previous values.
// @Summary Update message gateway channel
// @Description Updates a channel; empty secrets keep the current ciphertext
// @Tags admin-message-gateway
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path int true "channel id"
// @Param request body UpdateChannelRequest true "update body"
// @Success 200 {object} response.Any{data=ChannelDTO}
// @Failure 400 {object} response.Any
// @Failure 404 {object} response.Any
// @Router /api/v1/admin/message-gateway/channels/{id} [patch]
func UpdateChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.AbortBadRequest(c, "invalid channel id")
		return
	}
	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, err.Error())
		return
	}
	dto, err := updateChannel(c.Request.Context(), id, req)
	if err != nil {
		if err.Error() == errChannelNotFound {
			response.AbortNotFound(c, err.Error())
			return
		}
		response.AbortBadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.OK(dto))
}

// DeleteChannel removes a channel and its bindings/pairing codes.
// @Summary Delete message gateway channel
// @Description Deletes a channel and cascaded bindings and pairing codes
// @Tags admin-message-gateway
// @Produce json
// @Security SessionCookie
// @Param id path int true "channel id"
// @Success 200 {object} response.Any
// @Failure 404 {object} response.Any
// @Router /api/v1/admin/message-gateway/channels/{id} [delete]
func DeleteChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.AbortBadRequest(c, "invalid channel id")
		return
	}
	if err := deleteChannel(c.Request.Context(), id); err != nil {
		if err.Error() == errChannelNotFound {
			response.AbortNotFound(c, err.Error())
			return
		}
		response.AbortInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.OKNil())
}

// TestChannel probes stored credentials (Telegram getMe or QQ token).
// @Summary Test message gateway channel
// @Description Probes stored credentials without returning secrets
// @Tags admin-message-gateway
// @Produce json
// @Security SessionCookie
// @Param id path int true "channel id"
// @Success 200 {object} response.Any
// @Failure 400 {object} response.Any
// @Failure 404 {object} response.Any
// @Router /api/v1/admin/message-gateway/channels/{id}/test [post]
func TestChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.AbortBadRequest(c, "invalid channel id")
		return
	}
	if err := probeChannel(c.Request.Context(), id); err != nil {
		if err.Error() == errChannelNotFound {
			response.AbortNotFound(c, err.Error())
			return
		}
		response.AbortBadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.OKNil())
}
