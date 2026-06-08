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

package migrator

import (
	"context"
	"log"
	"time"

	"github.com/Rain-kl/Wavelet/internal/model"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/db/idgen"
)

// dbType 返回当前数据库类型名称（用于日志输出）
func dbType() string {
	if !config.Config.Database.Enabled {
		return "SQLite"
	}
	return "PostgreSQL"
}

func Migrate() {
	if err := db.DB(context.Background()).AutoMigrate(
		&model.User{},
		&model.AuthSource{},
		&model.ExternalAccount{},
		&model.SystemConfig{},
		&model.Upload{},
		&model.AccessToken{},
		&model.TaskExecution{},
		&model.Template{},
	); err != nil {
		log.Fatalf("[%s] auto migrate failed: %v\n", dbType(), err)
	}
	log.Printf("[%s] auto migrate success\n", dbType())

	// 初始化系统配置数据
	initSystemConfigs()
	// 初始化默认管理员用户
	initDefaultAdmin()
	// 初始化系统内置模板
	initTemplates()
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
			log.Printf("[%s] failed to create system config key %s: %v\n", dbType(), key, err)
		} else {
			log.Printf("[%s] initialized system config key %s\n", dbType(), key)
		}
	}
}

// initSystemConfigs 初始化系统配置数据
func initSystemConfigs() {
	tx := db.DB(context.Background())

	var count int64
	if err := tx.Model(&model.SystemConfig{}).Count(&count).Error; err != nil {
		log.Printf("[%s] failed to check system_config table: %v\n", dbType(), err)
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
		ensureConfigKeyExists(model.ConfigKeyServerAddress, "", "system", "服务器地址（用于跨域源控制，不设定则允许任意源）")
		ensureConfigKeyExists(model.ConfigKeySMTPHost, "", "system", "SMTP 服务器地址（例如 smtp.example.com）")
		ensureConfigKeyExists(model.ConfigKeySMTPPort, "587", "system", "SMTP 端口（例如 587 或 465）")
		ensureConfigKeyExists(model.ConfigKeySMTPUsername, "", "system", "SMTP 账户（如 sender@example.com）")
		ensureConfigKeyExists(model.ConfigKeySMTPPassword, "", "system", "SMTP 访问凭证（授权码/密码）")
		ensureConfigKeyExists(model.ConfigKeyEmailLoginVerificationEnabled, "false", "system", "是否开启邮箱登录验证（true/false）")
		ensureConfigKeyExists(model.ConfigKeyEmailRegisterVerificationEnabled, "false", "system", "是否开启邮箱注册验证（true/false）")
		ensureConfigKeyExists(model.ConfigKeyMenuDisplayConfig, "{}", "system", "目录显示配置（JSON 字符串，格式为 {url: enabled}）")
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
			Key:         model.ConfigKeyServerAddress,
			Value:       "",
			Type:        "system",
			Description: "服务器地址（用于跨域源控制，不设定则允许任意源）",
		},
		{
			Key:         model.ConfigKeySMTPHost,
			Value:       "",
			Type:        "system",
			Description: "SMTP 服务器地址（例如 smtp.example.com）",
		},
		{
			Key:         model.ConfigKeySMTPPort,
			Value:       "587",
			Type:        "system",
			Description: "SMTP 端口（例如 587 或 465）",
		},
		{
			Key:         model.ConfigKeySMTPUsername,
			Value:       "",
			Type:        "system",
			Description: "SMTP 账户（如 sender@example.com）",
		},
		{
			Key:         model.ConfigKeySMTPPassword,
			Value:       "",
			Type:        "system",
			Description: "SMTP 访问凭证（授权码/密码）",
		},
		{
			Key:         model.ConfigKeyUploadAllowedExtensions,
			Value:       "jpg,png,webp",
			Type:        "system",
			Description: "允许上传的图片扩展名（逗号分隔）",
		},
		{
			Key:         model.ConfigKeySiteName,
			Value:       "Wavelet",
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
		{
			Key:         model.ConfigKeyEmailLoginVerificationEnabled,
			Value:       "false",
			Type:        "system",
			Description: "是否开启邮箱登录验证（true/false）",
		},
		{
			Key:         model.ConfigKeyEmailRegisterVerificationEnabled,
			Value:       "false",
			Type:        "system",
			Description: "是否开启邮箱注册验证（true/false）",
		},
		{
			Key:         model.ConfigKeyMenuDisplayConfig,
			Value:       "{}",
			Type:        "system",
			Description: "目录显示配置（JSON 字符串，格式为 {url: enabled}）",
		},
	}

	if err := tx.Create(&defaultConfigs).Error; err != nil {
		log.Printf("[%s] failed to create default system configs: %v\n", dbType(), err)
	} else {
		log.Printf("[%s] initialized %d default system configs\n", dbType(), len(defaultConfigs))
	}
}

// initDefaultAdmin 初始化默认管理员用户
func initDefaultAdmin() {
	tx := db.DB(context.Background())

	var count int64
	if err := tx.Model(&model.User{}).Where("username = ?", "admin").Count(&count).Error; err != nil {
		log.Printf("[%s] failed to check default admin user: %v\n", dbType(), err)
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
		log.Printf("[%s] failed to create default admin user: %v\n", dbType(), err)
	} else {
		log.Printf("[%s] default admin user created successfully (username: admin, password: 12345678)\n", dbType())
	}
}

// initTemplates 初始化系统内置模板
func initTemplates() {
	tx := db.DB(context.Background())
	var count int64
	if err := tx.Model(&model.Template{}).Count(&count).Error; err != nil {
		log.Printf("[%s] failed to check templates table: %v\n", dbType(), err)
		return
	}

	defaultTemplates := []model.Template{
		{
			Key:         "login_email",
			Name:        "登录验证码邮件",
			Type:        "email",
			Subject:     "Wavelet 登录验证码",
			Content:     "<h3>Wavelet 登录验证</h3><p>您的登录验证码为：<strong>{{.Code}}</strong>，5分钟内有效，请勿将验证码泄露给他人。</p>",
			Description: "用户密码登录时发送的验证码邮件模板，支持变量：{{.Code}}",
			IsSystem:    true,
		},
		{
			Key:         "register_email",
			Name:        "注册验证码邮件",
			Type:        "email",
			Subject:     "Wavelet 注册验证码",
			Content:     "<h3>Wavelet 注册验证</h3><p>您的注册验证码为：<strong>{{.Code}}</strong>，5分钟内有效，请勿泄露给他人。</p>",
			Description: "用户注册时发送的验证码邮件模板，支持变量：{{.Code}}",
			IsSystem:    true,
		},
	}

	if count > 0 {
		// 确保系统预置模板存在
		for _, dt := range defaultTemplates {
			var t model.Template
			if err := tx.Where("key = ?", dt.Key).First(&t).Error; err != nil {
				if err := tx.Create(&dt).Error; err != nil {
					log.Printf("[%s] failed to create template key %s: %v\n", dbType(), dt.Key, err)
				} else {
					log.Printf("[%s] initialized template key %s\n", dbType(), dt.Key)
				}
			}
		}
		return
	}

	if err := tx.Create(&defaultTemplates).Error; err != nil {
		log.Printf("[%s] failed to create default templates: %v\n", dbType(), err)
	} else {
		log.Printf("[%s] initialized %d default templates\n", dbType(), len(defaultTemplates))
	}
}
