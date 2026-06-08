/*
Copyright 2025 linux.do
Modified by Arctel.net, 2026

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

package model

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/util"
	"gorm.io/gorm"
)

// OAuthUserInfo 用户信息结构（同时支持 OIDC ID Token claims 和 UserEndpoint 响应）
type OAuthUserInfo struct {
	Id                uint64 `json:"id"`
	Sub               string `json:"sub"`
	Username          string `json:"username"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	Active            bool   `json:"active"`
	AvatarUrl         string `json:"avatar_url"`
}

// GetID 获取用户 ID
func (u *OAuthUserInfo) GetID() uint64 {
	if u.Id != 0 {
		return u.Id
	}
	// 从 sub 解析（OIDC 格式）
	if u.Sub != "" {
		if id, err := strconv.ParseUint(u.Sub, 10, 64); err == nil {
			return id
		}
	}
	return 0
}

type User struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	Username    string    `json:"username" gorm:"size:64;uniqueIndex"`
	Password    string    `json:"password,omitempty" gorm:"size:255"`
	Nickname    string    `json:"nickname" gorm:"size:255"`
	Email       string    `json:"email" gorm:"size:255;index"`
	AvatarUrl   string    `json:"avatar_url" gorm:"size:255"`
	IsActive    bool      `json:"is_active" gorm:"default:true;index"`
	IsAdmin     bool      `json:"is_admin" gorm:"default:false"`
	LastLoginAt time.Time `json:"last_login_at" gorm:"index"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime;index"`
}

func (u *User) SetPassword(password string) error {
	u.Password = password
	return nil
}

func (u *User) SetEncryptedPassword(password string) error {
	if password == "" {
		u.Password = ""
		return nil
	}
	hashed, err := util.HashPassword(password)
	if err != nil {
		return err
	}
	u.Password = hashed
	return nil
}

func (u *User) CheckPassword(password string) bool {
	if u.Password == "" || password == "" {
		return false
	}
	isBcrypt := strings.HasPrefix(u.Password, "$2a$") || strings.HasPrefix(u.Password, "$2b$") || strings.HasPrefix(u.Password, "$2y$")
	if isBcrypt {
		return util.CheckPasswordHash(u.Password, password)
	}
	return u.Password == password
}

func (u *User) GetByID(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(u).Error; err != nil {
		return err
	}
	return nil
}

// UpdateFromOAuthInfo 根据 OAuth 信息更新用户数据
func (u *User) UpdateFromOAuthInfo(oauthInfo *OAuthUserInfo) {
	u.Username = oauthInfo.Username
	u.Nickname = oauthInfo.Name
	u.Email = oauthInfo.Email
	u.AvatarUrl = oauthInfo.AvatarUrl
	u.IsActive = oauthInfo.Active
	u.LastLoginAt = time.Now()
}

// CheckActive 检查用户账户是否激活,未激活则返回错误
func (u *User) CheckActive() error {
	if !u.IsActive {
		return errors.New(common.BannedAccount)
	}
	return nil
}

// CreateUser 创建新用户
func (u *User) CreateUser(tx *gorm.DB, oauthInfo *OAuthUserInfo) error {
	now := time.Now()
	newUser := User{
		ID:          oauthInfo.GetID(),
		Username:    oauthInfo.Username,
		Nickname:    oauthInfo.Name,
		Email:       oauthInfo.Email,
		AvatarUrl:   oauthInfo.AvatarUrl,
		IsActive:    oauthInfo.Active,
		LastLoginAt: now,
		IsAdmin:     false,
	}
	if err := tx.Create(&newUser).Error; err != nil {
		return err
	}

	*u = newUser
	return nil
}
