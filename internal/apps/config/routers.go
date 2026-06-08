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

package config

import (
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

// PublicConfigResponse 公共配置响应
type PublicConfigResponse struct {
	UploadAllowedExtensions          string `json:"upload_allowed_extensions"`           // 允许上传的图片扩展名
	SiteName                         string `json:"site_name"`                           // 站点名称
	PasswordLoginEnabled             bool   `json:"password_login_enabled"`              // 是否允许密码登录
	RegistrationEnabled              bool   `json:"registration_enabled"`                // 是否允许注册
	PasswordRegisterEnabled          bool   `json:"password_register_enabled"`           // 是否允许密码注册
	OIDCLoginEnabled                 bool   `json:"oidc_login_enabled"`                  // 是否允许 OIDC 登录
	MaxAPIKeysPerUser                int    `json:"max_api_keys_per_user"`               // 每个用户最大 API Key 数量
	CapLoginEnabled                  bool   `json:"cap_login_enabled"`                   // 是否启用人机验证
	CapAutoSolve                     bool   `json:"cap_auto_solve"`                      // 打开页面后是否自动开始计算
	EmailLoginVerificationEnabled    bool   `json:"email_login_verification_enabled"`    // 是否启用邮箱登录验证
	EmailRegisterVerificationEnabled bool   `json:"email_register_verification_enabled"` // 是否启用邮箱注册验证
}

// GetPublicConfig 获取公共配置
// @Summary 获取公共配置
// @Description 返回对前端公开的系统配置信息，如允许上传的文件类型、站点名称、是否开放注册等
// @Tags config
// @Accept json
// @Produce json
// @Success 200 {object} util.ResponseAny
// @Router /api/v1/config/public [get]
func GetPublicConfig(c *gin.Context) {
	ctx := c.Request.Context()
	var sc model.SystemConfig

	// 1. upload_allowed_extensions
	var uploadExtensions string
	if err := sc.GetByKey(ctx, model.ConfigKeyUploadAllowedExtensions); err == nil {
		uploadExtensions = sc.Value
	}

	// 2. site_name
	var siteName string
	if err := sc.GetByKey(ctx, model.ConfigKeySiteName); err == nil {
		siteName = sc.Value
	}

	// 3. registration_enabled
	var registrationEnabled bool
	if val, err := model.GetBoolByKey(ctx, model.ConfigKeyRegistrationEnabled); err == nil {
		registrationEnabled = val
	}

	// 3.1 password_login_enabled
	var passwordLoginEnabled bool
	if val, err := model.GetBoolByKey(ctx, model.ConfigKeyPasswordLoginEnabled); err == nil {
		passwordLoginEnabled = val
	}

	// 3.2 password_register_enabled
	var passwordRegisterEnabled bool
	if val, err := model.GetBoolByKey(ctx, model.ConfigKeyPasswordRegisterEnabled); err == nil {
		passwordRegisterEnabled = val
	}

	// 3.3 oidc_login_enabled
	var oidcLoginEnabled bool
	if val, err := model.GetBoolByKey(ctx, model.ConfigKeyOIDCLoginEnabled); err == nil {
		oidcLoginEnabled = val
	}

	// 3.4 cap_login_enabled
	var capLoginEnabled bool
	if val, err := model.GetBoolByKey(ctx, model.ConfigKeyCapLoginEnabled); err == nil {
		capLoginEnabled = val
	}

	// 3.5 cap_auto_solve
	capAutoSolve := true // 默认自动开始
	if val, err := model.GetBoolByKey(ctx, model.ConfigKeyCapAutoSolve); err == nil {
		capAutoSolve = val
	}

	// 4. max_api_keys_per_user
	var maxAPIKeys int
	if val, err := model.GetIntByKey(ctx, model.ConfigKeyMaxAPIKeysPerUser); err == nil {
		maxAPIKeys = val
	}

	var emailLoginVerificationEnabled bool
	if val, err := model.GetBoolByKey(ctx, model.ConfigKeyEmailLoginVerificationEnabled); err == nil {
		emailLoginVerificationEnabled = val
	}

	var emailRegisterVerificationEnabled bool
	if val, err := model.GetBoolByKey(ctx, model.ConfigKeyEmailRegisterVerificationEnabled); err == nil {
		emailRegisterVerificationEnabled = val
	}

	response := PublicConfigResponse{
		UploadAllowedExtensions:          uploadExtensions,
		SiteName:                         siteName,
		PasswordLoginEnabled:             passwordLoginEnabled,
		RegistrationEnabled:              registrationEnabled,
		PasswordRegisterEnabled:          passwordRegisterEnabled,
		OIDCLoginEnabled:                 oidcLoginEnabled,
		MaxAPIKeysPerUser:                maxAPIKeys,
		CapLoginEnabled:                  capLoginEnabled,
		CapAutoSolve:                     capAutoSolve,
		EmailLoginVerificationEnabled:    emailLoginVerificationEnabled,
		EmailRegisterVerificationEnabled: emailRegisterVerificationEnabled,
	}

	c.JSON(http.StatusOK, util.OK(response))
}
