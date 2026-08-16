// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package item

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/shared/response"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/gin-gonic/gin"
)

type createItemRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	UploadIDs []string `json:"upload_ids"`
}

type listItemsRequest struct {
	Page            int    `form:"page"`
	PageSize        int    `form:"page_size"`
	Q               string `form:"q"`
	Lifecycle       string `form:"lifecycle"`
	Importance      string `form:"importance"`
	ContentType     string `form:"content_type"`
	IncludeArchived bool   `form:"include_archived"`
	IncludeTrash    bool   `form:"include_trash"`
}

type patchItemRequest struct {
	Title      *string `json:"title"`
	Body       *string `json:"body"`
	Lifecycle  *string `json:"lifecycle"`
	Importance *string `json:"importance"`
}

type timelineRequest struct {
	Days           int    `form:"days"`
	ExpandArchived bool   `form:"expand_archived"`
	Day            string `form:"day"`
}

// itemHandlers groups HTTP handlers so names can match exported logics without collision.
type itemHandlers struct{}

func abortItemError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	switch msg {
	case errBindParamsFailed, errEmptyContent, errInvalidTransition, errInvalidUpload:
		response.AbortBadRequest(c, msg)
	case errItemNotFound:
		response.AbortNotFound(c, msg)
	case errInternal:
		response.AbortInternal(c, msg)
	default:
		logger.ErrorF(c.Request.Context(), "item: unexpected handler error: %v", err)
		response.AbortInternal(c, errInternal)
	}
}

func parseUint64ID(s string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(s), 10, 64)
}

func parseUploadIDs(ids []string) ([]uint64, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make([]uint64, 0, len(ids))
	for _, s := range ids {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := parseUint64ID(s)
		if err != nil {
			return nil, err
		}
		if id == 0 {
			return nil, errors.New(errBindParamsFailed)
		}
		out = append(out, id)
	}
	return out, nil
}

// CreateItem 创建捕获条目
// @Summary 创建捕获条目
// @Description 创建一条 pending 捕获条目，可附带文本与已有上传附件 ID
// @Tags item
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body createItemRequest true "创建请求"
// @Success 200 {object} response.Any{data=ItemDTO} "创建成功"
// @Failure 400 {object} response.Any "参数错误或内容为空"
// @Failure 401 {object} response.Any "未登录"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/items [post]
func (itemHandlers) CreateItem(c *gin.Context) {
	var req createItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, errBindParamsFailed)
		return
	}
	uploadIDs, err := parseUploadIDs(req.UploadIDs)
	if err != nil {
		response.AbortBadRequest(c, errBindParamsFailed)
		return
	}
	userID := oauth.GetUserIDFromContext(c)
	dto, err := CreateItem(c.Request.Context(), userID, CreateItemInput{
		Title:     req.Title,
		Body:      req.Body,
		UploadIDs: uploadIDs,
	})
	if err != nil {
		abortItemError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK(dto))
}

// ListItems 分页列表
// @Summary 分页列表条目
// @Description 按生命周期、重要度、关键词等过滤当前用户的捕获条目
// @Tags item
// @Produce json
// @Security SessionCookie
// @Param page query int false "页码（默认 1）"
// @Param page_size query int false "每页数量（默认 20，最大 100）"
// @Param q query string false "标题/正文关键词"
// @Param lifecycle query string false "生命周期过滤 pending|active|archived|trash"
// @Param importance query string false "重要度过滤 none|fragment|note|vault"
// @Param content_type query string false "内容类型 text|image|file"
// @Param include_archived query bool false "未指定 lifecycle 时是否包含已归档"
// @Param include_trash query bool false "未指定 lifecycle 时是否包含回收站"
// @Success 200 {object} response.Any{data=ListItemsResult} "查询成功"
// @Failure 400 {object} response.Any "参数错误"
// @Failure 401 {object} response.Any "未登录"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/items [get]
func (itemHandlers) ListItems(c *gin.Context) {
	var req listItemsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.AbortBadRequest(c, errBindParamsFailed)
		return
	}
	userID := oauth.GetUserIDFromContext(c)
	result, err := ListItems(c.Request.Context(), userID, ListItemsQuery{
		Page:            req.Page,
		PageSize:        req.PageSize,
		Q:               req.Q,
		Lifecycle:       req.Lifecycle,
		Importance:      req.Importance,
		ContentType:     req.ContentType,
		IncludeArchived: req.IncludeArchived,
		IncludeTrash:    req.IncludeTrash,
	})
	if err != nil {
		abortItemError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK(result))
}

// GetTimeline 回顾时间线
// @Summary 回顾时间线
// @Description 按 UTC 自然日分组返回非回收站条目；可展开归档项
// @Tags item
// @Produce json
// @Security SessionCookie
// @Param days query int false "回溯天数（默认 90）"
// @Param expand_archived query bool false "是否展开归档条目"
// @Param day query string false "仅展开指定日期 YYYY-MM-DD 的归档"
// @Success 200 {object} response.Any{data=TimelineResult} "查询成功"
// @Failure 401 {object} response.Any "未登录"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/items/timeline [get]
func (itemHandlers) GetTimeline(c *gin.Context) {
	var req timelineRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.AbortBadRequest(c, errBindParamsFailed)
		return
	}
	userID := oauth.GetUserIDFromContext(c)
	result, err := Timeline(c.Request.Context(), userID, TimelineQuery(req))
	if err != nil {
		abortItemError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK(result))
}

// GetStats 导航角标统计
// @Summary 条目统计
// @Description 返回各生命周期与 vault 角标数量
// @Tags item
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any{data=ItemStats} "查询成功"
// @Failure 401 {object} response.Any "未登录"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/items/stats [get]
func (itemHandlers) GetStats(c *gin.Context) {
	userID := oauth.GetUserIDFromContext(c)
	result, err := Stats(c.Request.Context(), userID)
	if err != nil {
		abortItemError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK(result))
}

// GetItem 获取单条
// @Summary 获取条目详情
// @Description 返回当前用户拥有的单条捕获及其附件
// @Tags item
// @Produce json
// @Security SessionCookie
// @Param id path string true "条目 ID"
// @Success 200 {object} response.Any{data=ItemDTO} "查询成功"
// @Failure 400 {object} response.Any "参数错误"
// @Failure 401 {object} response.Any "未登录"
// @Failure 404 {object} response.Any "记录不存在"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/items/{id} [get]
func (itemHandlers) GetItem(c *gin.Context) {
	id, err := parseUint64ID(c.Param("id"))
	if err != nil || id == 0 {
		response.AbortBadRequest(c, errBindParamsFailed)
		return
	}
	userID := oauth.GetUserIDFromContext(c)
	dto, err := GetItem(c.Request.Context(), userID, id)
	if err != nil {
		abortItemError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK(dto))
}

// PatchItem 部分更新
// @Summary 更新条目
// @Description 更新标题/正文，或执行生命周期与重要度状态迁移
// @Tags item
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "条目 ID"
// @Param request body patchItemRequest true "更新请求"
// @Success 200 {object} response.Any{data=ItemDTO} "更新成功"
// @Failure 400 {object} response.Any "参数错误或非法状态变更"
// @Failure 401 {object} response.Any "未登录"
// @Failure 404 {object} response.Any "记录不存在"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/items/{id} [patch]
func (itemHandlers) PatchItem(c *gin.Context) {
	id, err := parseUint64ID(c.Param("id"))
	if err != nil || id == 0 {
		response.AbortBadRequest(c, errBindParamsFailed)
		return
	}
	var req patchItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, errBindParamsFailed)
		return
	}
	input := PatchItemInput{
		Title: req.Title,
		Body:  req.Body,
	}
	if req.Lifecycle != nil {
		lc := model.ItemLifecycle(strings.TrimSpace(*req.Lifecycle))
		input.Lifecycle = &lc
	}
	if req.Importance != nil {
		im := model.ItemImportance(strings.TrimSpace(*req.Importance))
		input.Importance = &im
	}
	userID := oauth.GetUserIDFromContext(c)
	dto, err := PatchItem(c.Request.Context(), userID, id, input)
	if err != nil {
		abortItemError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK(dto))
}

// DeleteItem 软删除或硬删除
// @Summary 删除条目
// @Description 默认移入回收站；force=1 时硬删除并清理附件
// @Tags item
// @Produce json
// @Security SessionCookie
// @Param id path string true "条目 ID"
// @Param force query string false "为 1 时硬删除"
// @Success 200 {object} response.Any "删除成功"
// @Failure 400 {object} response.Any "参数错误或非法状态变更"
// @Failure 401 {object} response.Any "未登录"
// @Failure 404 {object} response.Any "记录不存在"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/items/{id} [delete]
func (itemHandlers) DeleteItem(c *gin.Context) {
	id, err := parseUint64ID(c.Param("id"))
	if err != nil || id == 0 {
		response.AbortBadRequest(c, errBindParamsFailed)
		return
	}
	force := c.Query("force") == "1"
	userID := oauth.GetUserIDFromContext(c)
	if err := DeleteItem(c.Request.Context(), userID, id, force); err != nil {
		abortItemError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKNil())
}
