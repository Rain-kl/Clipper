/*
Copyright 2025 linux.do

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package oauth

import (
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/linux-do/credit/internal/config"
	"github.com/linux-do/credit/internal/db"
	"github.com/linux-do/credit/internal/model"
	"github.com/linux-do/credit/internal/util"
	"github.com/shopspring/decimal"
)

// GetLoginURL godoc
// @Summary 获取登录地址
// @Description 生成 OAuth 登录 URL，前端跳转至该地址完成授权
// @Tags oauth
// @Produce json
// @Success 200 {object} util.ResponseAny
// @Router /api/v1/oauth/login [get]
func GetLoginURL(c *gin.Context) {
	ctx := c.Request.Context()

	// 生成 state
	state := uuid.NewString()
	cmd := db.Redis.Set(ctx, db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state)), state, OAuthStateCacheKeyExpiration)
	if cmd.Err() != nil {
		c.JSON(http.StatusInternalServerError, util.Err(cmd.Err().Error()))
		return
	}

	// 构造登录 URL
	var authURL string
	if config.Config.App.Env == "development" {
		authURL = fmt.Sprintf("%s/login?code=dev_mock_code&state=%s", config.Config.App.FrontendURL, state)
	} else if oidcVerifier != nil {
		// OIDC 模式：state 同时用作 nonce
		authURL = oauthConf.AuthCodeURL(state, oidc.Nonce(state))
	} else {
		// 纯 OAuth2 模式
		authURL = oauthConf.AuthCodeURL(state)
	}
	c.JSON(http.StatusOK, util.OK(authURL))
}

type CallbackRequest struct {
	State string `json:"state"`
	Code  string `json:"code"`
}

// Callback godoc
// @Summary OAuth 回调
// @Description 接收前端传回的 state 和 code，完成 OAuth/OIDC 认证并建立用户会话
// @Tags oauth
// @Accept json
// @Param request body CallbackRequest true "回调请求参数"
// @Produce json
// @Success 200 {object} util.ResponseAny
// @Router /api/v1/oauth/callback [post]
func Callback(c *gin.Context) {
	// 解析请求
	var req CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	ctx := c.Request.Context()

	// 验证 state
	cmd := db.Redis.Get(ctx, db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, req.State)))
	if cmd.Val() != req.State {
		c.JSON(http.StatusBadRequest, util.Err(InvalidState))
		return
	}
	db.Redis.Del(ctx, db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, req.State)))

	// 执行 OAuth/OIDC 认证
	user, err := doOAuth(ctx, req.Code, req.State)
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	session := sessions.Default(c)
	session.Set(UserIDKey, user.ID)
	session.Set(UserNameKey, user.Username)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	LogForAudit(ctx, user, c)

	c.JSON(http.StatusOK, util.OKNil())
}

type BasicUserInfo struct {
	ID               uint64           `json:"id"`
	Username         string           `json:"username"`
	Nickname         string           `json:"nickname"`
	TrustLevel       model.TrustLevel `json:"trust_level"`
	AvatarUrl        string           `json:"avatar_url"`
	TotalReceive     decimal.Decimal  `json:"total_receive"`
	TotalPayment     decimal.Decimal  `json:"total_payment"`
	TotalTransfer    decimal.Decimal  `json:"total_transfer"`
	TotalCommunity   decimal.Decimal  `json:"total_community"`
	CommunityBalance decimal.Decimal  `json:"community_balance"`
	AvailableBalance decimal.Decimal  `json:"available_balance"`
	PendingBalance   decimal.Decimal  `json:"pending_balance"`
	PayScore         int64            `json:"pay_score"`
	IsAdmin          bool             `json:"is_admin"`
	RemainQuota      decimal.Decimal  `json:"remain_quota"`
	PayLevel         string           `json:"pay_level"`
	DailyLimit       *int64           `json:"daily_limit"`
}

// UserInfo godoc
// @Summary 获取当前登录用户信息
// @Description 返回当前登录用户的基本信息及余额数据，需要登录
// @Tags oauth
// @Produce json
// @Success 200 {object} util.ResponseAny
// @Router /api/v1/oauth/user-info [get]
func UserInfo(c *gin.Context) {
	user, _ := util.GetFromContext[*model.User](c, UserObjKey)

	c.JSON(
		http.StatusOK,
		util.OK(BasicUserInfo{
			ID:               user.ID,
			Username:         user.Username,
			Nickname:         user.Nickname,
			TrustLevel:       user.TrustLevel,
			AvatarUrl:        user.AvatarUrl,
			TotalReceive:     user.TotalReceive,
			TotalPayment:     user.TotalPayment,
			TotalTransfer:    user.TotalTransfer,
			TotalCommunity:   user.TotalCommunity,
			CommunityBalance: user.CommunityBalance,
			AvailableBalance: user.AvailableBalance,
			PendingBalance:   user.PendingBalance,
			PayScore:         user.PayScore,
			IsAdmin:          user.IsAdmin,
			RemainQuota:      decimal.NewFromInt(-1),
			PayLevel:         "Free",
			DailyLimit:       nil,
		}),
	)
}

// Logout godoc
// @Summary 退出登录
// @Description 清除当前用户的登录会话，完成退出
// @Tags oauth
// @Produce json
// @Success 200 {object} util.ResponseAny
// @Router /api/v1/oauth/logout [get]
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Options(util.GetSessionOptions(-1))
	session.Clear()
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OKNil())
}
