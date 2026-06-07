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

	"github.com/gin-gonic/gin"
	"github.com/linux-do/credit/internal/model"
	"github.com/linux-do/credit/internal/util"
)

// PublicConfigResponse 公共配置响应
type PublicConfigResponse struct {
	UploadAllowedExtensions string `json:"upload_allowed_extensions"` // 允许上传的图片扩展名
	SiteName                string `json:"site_name"`                 // 站点名称
	RegistrationEnabled     bool   `json:"registration_enabled"`      // 是否允许注册
	MaxAPIKeysPerUser       int    `json:"max_api_keys_per_user"`     // 每个用户最大 API Key 数量
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

	// 4. max_api_keys_per_user
	var maxAPIKeys int
	if val, err := model.GetIntByKey(ctx, model.ConfigKeyMaxAPIKeysPerUser); err == nil {
		maxAPIKeys = val
	}

	response := PublicConfigResponse{
		UploadAllowedExtensions: uploadExtensions,
		SiteName:                siteName,
		RegistrationEnabled:     registrationEnabled,
		MaxAPIKeysPerUser:       maxAPIKeys,
	}

	c.JSON(http.StatusOK, util.OK(response))
}
