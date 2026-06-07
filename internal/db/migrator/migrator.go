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

	"github.com/linux-do/credit/internal/model"

	"github.com/linux-do/credit/internal/config"
	"github.com/linux-do/credit/internal/db"
)

func Migrate() {
	if !config.Config.Database.Enabled {
		return
	}

	if err := db.DB(context.Background()).AutoMigrate(
		&model.User{},
		&model.SystemConfig{},
		&model.Upload{},
	); err != nil {
		log.Fatalf("[PostgreSQL] auto migrate failed: %v\n", err)
	}
	log.Printf("[PostgreSQL] auto migrate success\n")

	// 初始化系统配置数据
	initSystemConfigs()
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
		return
	}

	defaultConfigs := []model.SystemConfig{
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
			Key:         model.ConfigKeyRegistrationEnabled,
			Value:       "true",
			Type:        "system",
			Description: "控制普通用户是否可以自主注册（true/false）",
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
