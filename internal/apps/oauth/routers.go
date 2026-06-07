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
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/linux-do/credit/internal/model"
	"github.com/linux-do/credit/internal/util"
	"github.com/shopspring/decimal"
)

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

func BuildBasicUserInfo(user *model.User) BasicUserInfo {
	return BasicUserInfo{
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
	}
}

// UserInfo 获取当前登录用户信息
// @Summary 获取当前登录用户信息
// @Description 返回当前登录用户的基本信息及余额数据，需要登录。包括用户 ID、用户名、信任等级、各类余额信息等。
// @Tags oauth
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=oauth.BasicUserInfo} "用户信息"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Router /api/v1/oauth/user-info [get]
func UserInfo(c *gin.Context) {
	user, _ := util.GetFromContext[*model.User](c, UserObjKey)

	c.JSON(
		http.StatusOK,
		util.OK(BuildBasicUserInfo(user)),
	)
}

// GetLoginURL 获取登录地址
// @Summary 获取登录地址
// @Description 生成 OAuth 登录 URL，前端跳转至该地址完成授权。返回的 URL 中包含 state 参数用于 CSRF 防护。
// @Tags oauth
// @Produce json
// @Success 200 {object} util.ResponseAny{data=string} "OAuth 登录 URL"
// @Failure 500 {object} util.ResponseAny "Redis 异常或内部错误"
// @Router /api/v1/oauth/login [get]

// Logout 退出登录
// @Summary 退出登录
// @Description 清除当前用户的登录会话，完成退出。清除 Cookie 中的 Session 数据。
// @Tags oauth
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=string} "退出成功"
// @Failure 500 {object} util.ResponseAny "Session 清除失败"
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
