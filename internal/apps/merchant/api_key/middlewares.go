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

package api_key

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/linux-do/credit/internal/apps/merchant"
	"github.com/linux-do/credit/internal/apps/oauth"
	"github.com/linux-do/credit/internal/db"
	"github.com/linux-do/credit/internal/model"
	"github.com/linux-do/credit/internal/util"
)

func RequireAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)

		var apiKey model.MerchantAPIKey
		if err := db.DB(c.Request.Context()).
			Where("id = ? AND user_id = ?", c.Param("id"), user.ID).
			First(&apiKey).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, util.Err(APIKeyNotFound))
			return
		}

		util.SetToContext(c, merchant.APIKeyObjKey, &apiKey)

		c.Next()
	}
}
