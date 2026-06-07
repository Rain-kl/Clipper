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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linux-do/credit/internal/common"
	"github.com/linux-do/credit/internal/db"
	"github.com/linux-do/credit/internal/model"
	"github.com/linux-do/credit/internal/otel_trace"
	"github.com/linux-do/credit/internal/util"
)

type loginRequiredAuditLog struct {
	UserID     uint64 `json:"user_id"`
	Username   string `json:"username"`
	ClientIP   string `json:"client_ip"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	RequestURI string `json:"request_uri"`
	UserAgent  string `json:"user_agent"`
	Referer    string `json:"referer"`
}

func LoginRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// init trace
		ctx, span := otel_trace.Start(c.Request.Context(), "LoginRequired")
		defer span.End()

		// check token in headers
		tokenStr := c.GetHeader("X-Access-Token")
		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenStr = authHeader[7:]
			}
		}

		var user model.User
		var authenticated bool

		if tokenStr != "" {
			tokenHash := model.HashToken(tokenStr)
			var tokenRecord model.AccessToken
			if err := db.DB(ctx).Where("token_hash = ?", tokenHash).First(&tokenRecord).Error; err == nil {
				if err := db.DB(ctx).Where("id = ? AND is_active = ?", tokenRecord.UserID, true).First(&user).Error; err == nil {
					authenticated = true
					// update token last used time
					now := time.Now()
					db.DB(ctx).Model(&tokenRecord).Update("last_used_at", &now)
				}
			}
		}

		if !authenticated {
			// load user from session
			userId := GetUserIDFromContext(c)
			if userId <= 0 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error_msg": common.UnAuthorized, "data": nil})
				return
			}

			// load user from db to make sure is active
			tx := db.DB(ctx).Where("id = ? AND is_active = ?", userId, true).First(&user)
			if tx.Error != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error_msg": common.UnAuthorized, "data": nil})
				return
			}
		}

		// log
		LogForAudit(ctx, &user, c)

		// set user info
		util.SetToContext(c, UserObjKey, &user)

		if risk, ok := checkOpenAPIUserRisk(ctx, user.ID); ok {
			if blocked := applyOpenAPIUserRisk(c, risk); blocked {
				return
			}
		}

		// next
		c.Next()
	}
}
