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

package migrator

import (
	"context"
	"log"
	"time"

	"github.com/linux-do/credit/internal/model"

	"github.com/linux-do/credit/internal/config"
	"github.com/linux-do/credit/internal/db"
	"github.com/linux-do/credit/internal/db/idgen"
)

func Migrate() {
	if !config.Config.Database.Enabled {
		return
	}

	if err := db.DB(context.Background()).AutoMigrate(
		&model.User{},
		&model.AuthSource{},
		&model.ExternalAccount{},
		&model.SystemConfig{},
		&model.Upload{},
		&model.AccessToken{},
		&model.TaskExecution{},
	); err != nil {
		log.Fatalf("[PostgreSQL] auto migrate failed: %v\n", err)
	}
	log.Printf("[PostgreSQL] auto migrate success\n")

	// 初始化系统配置数据
	initSystemConfigs()
	// 初始化默认管理员用户
	initDefaultAdmin()
}

// ensureConfigKeyExists ensures a system config key exists in the database
func ensureConfigKeyExists(key, value, configType, description string) {
	tx := db.DB(context.Background())
	var cfg model.SystemConfig
	if err := tx.Where("key = ?", key).First(&cfg).Error; err != nil {
		newConfig := model.SystemConfig{
			Key:         key,
			Value:       value,
			Type:        configType,
			Description: description,
		}
		if err := tx.Create(&newConfig).Error; err != nil {
			log.Printf("[PostgreSQL] failed to create system config key %s: %v\n", key, err)
		} else {
			log.Printf("[PostgreSQL] initialized system config key %s\n", key)
		}
	}
}

// initSystemConfigs 初始化系统配置数据
func initSystemConfigs() {
	tx := db.DB(context.Background())

	var count int64
	if err := tx.Model(&model.SystemConfig{}).Count(&count).Error; err != nil {
		log.Printf("[PostgreSQL] failed to check system_config table: %v\n", err)
		return
	}

	if count > 0 {
		ensureConfigKeyExists(model.ConfigKeyCapLoginEnabled, "false", "system", "是否启用登录人机验证（true/false）")
		ensureConfigKeyExists(model.ConfigKeyCapAutoSolve, "true", "system", "打开页面后是否自动开始计算，关闭则需用户手动点击触发")
		ensureConfigKeyExists(model.ConfigKeyCapChallengeCount, "1", "system", "客户端需求解的 PoW 难题总数，默认 1，推荐 1～5")
		ensureConfigKeyExists(model.ConfigKeyCapChallengeSize, "32", "system", "人机验证盐值长度")
		ensureConfigKeyExists(model.ConfigKeyCapChallengeDifficulty, "4", "system", "人机验证 PoW 难度（目标前缀长度）")
		ensureConfigKeyExists(model.ConfigKeyCapChallengeTTL, "600", "system", "人机验证难题有效时间（秒）")
		ensureConfigKeyExists(model.ConfigKeyCapTokenTTL, "1200", "system", "人机验证兑换凭证有效时间（秒）")
		return
	}

	defaultConfigs := []model.SystemConfig{
		{
			Key:         model.ConfigKeyCapLoginEnabled,
			Value:       "false",
			Type:        "system",
			Description: "是否启用登录人机验证（true/false）",
		},
		{
			Key:         model.ConfigKeyCapAutoSolve,
			Value:       "true",
			Type:        "system",
			Description: "打开页面后是否自动开始计算，关闭则需用户手动点击触发",
		},
		{
			Key:         model.ConfigKeyCapChallengeCount,
			Value:       "1",
			Type:        "system",
			Description: "客户端需求解的 PoW 难题总数，默认 1，推荐 1～5",
		},
		{
			Key:         model.ConfigKeyCapChallengeSize,
			Value:       "32",
			Type:        "system",
			Description: "人机验证盐值长度",
		},
		{
			Key:         model.ConfigKeyCapChallengeDifficulty,
			Value:       "4",
			Type:        "system",
			Description: "人机验证 PoW 难度（目标前缀长度）",
		},
		{
			Key:         model.ConfigKeyCapChallengeTTL,
			Value:       "600",
			Type:        "system",
			Description: "人机验证难题有效时间（秒）",
		},
		{
			Key:         model.ConfigKeyCapTokenTTL,
			Value:       "1200",
			Type:        "system",
			Description: "人机验证兑换凭证有效时间（秒）",
		},
		{
			Key:         model.ConfigKeyUploadAllowedExtensions,
			Value:       "jpg,png,webp",
			Type:        "system",
			Description: "允许上传的图片扩展名（逗号分隔）",
		},
		{
			Key:         model.ConfigKeySiteName,
			Value:       "Antigravity Project",
			Type:        "system",
			Description: "系统平台的展示名称",
		},
		{
			Key:         model.ConfigKeyPasswordLoginEnabled,
			Value:       "true",
			Type:        "system",
			Description: "是否允许使用账号密码登录",
		},
		{
			Key:         model.ConfigKeyRegistrationEnabled,
			Value:       "true",
			Type:        "system",
			Description: "控制普通用户是否可以自主注册（true/false）",
		},
		{
			Key:         model.ConfigKeyPasswordRegisterEnabled,
			Value:       "true",
			Type:        "system",
			Description: "是否允许通过密码创建本地账号",
		},
		{
			Key:         model.ConfigKeyOIDCLoginEnabled,
			Value:       "true",
			Type:        "system",
			Description: "是否允许使用第三方 OIDC 认证源登录",
		},
		{
			Key:         model.ConfigKeyMaxAPIKeysPerUser,
			Value:       "5",
			Type:        "business",
			Description: "限制每个普通用户可以创建的 API Key 最大数量",
		},
	}

	if err := tx.Create(&defaultConfigs).Error; err != nil {
		log.Printf("[PostgreSQL] failed to create default system configs: %v\n", err)
	} else {
		log.Printf("[PostgreSQL] initialized %d default system configs\n", len(defaultConfigs))
	}
}

// initDefaultAdmin 初始化默认管理员用户
func initDefaultAdmin() {
	tx := db.DB(context.Background())

	var count int64
	if err := tx.Model(&model.User{}).Where("username = ?", "admin").Count(&count).Error; err != nil {
		log.Printf("[PostgreSQL] failed to check default admin user: %v\n", err)
		return
	}

	if count > 0 {
		return
	}

	adminUser := model.User{
		ID:          idgen.NextUint64ID(),
		Username:    "admin",
		Password:    "12345678", // 密码使用明文存储
		Nickname:    "Administrator",
		AvatarUrl:   "",
		IsActive:    true,
		IsAdmin:     true,
		LastLoginAt: time.Now(),
	}

	if err := tx.Create(&adminUser).Error; err != nil {
		log.Printf("[PostgreSQL] failed to create default admin user: %v\n", err)
	} else {
		log.Printf("[PostgreSQL] default admin user created successfully (username: admin, password: 12345678)\n")
	}
}
