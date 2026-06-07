/*
Copyright 2026 linux.do

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

package testhelper

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/hibiken/asynq"
	"github.com/linux-do/credit/internal/db"
	"github.com/linux-do/credit/internal/model"
	"github.com/linux-do/credit/internal/task"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SetupTestEnvironment initializes an in-memory SQLite DB, seeds default configurations,
// starts miniredis, and overrides the global db/Redis clients. It returns a cleanup function.
func SetupTestEnvironment(t *testing.T) (*gorm.DB, *miniredis.Miniredis, func()) {
	// Initialize GORM in-memory SQLite
	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite db: %v", err)
	}

	// AutoMigrate all tables
	err = sqliteDB.AutoMigrate(
		&model.User{},
		&model.AuthSource{},
		&model.ExternalAccount{},
		&model.SystemConfig{},
		&model.Upload{},
		&model.TaskExecution{},
	)
	if err != nil {
		t.Fatalf("failed to auto migrate tables: %v", err)
	}

	// Set global db
	db.SetDB(sqliteDB)

	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	// Hook up Redis Client to miniredis
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	db.Redis = redisClient

	// Hook up AsynqClient to miniredis
	task.AsynqClient = asynq.NewClient(asynq.RedisClientOpt{
		Addr: mr.Addr(),
	})

	// Seed default configurations
	seedDefaultConfigs(t, sqliteDB)

	// Cleanup function
	cleanup := func() {
		redisClient.Close()
		mr.Close()
		// Reset database and Redis references
		db.SetDB(nil)
		db.Redis = nil
		task.AsynqClient = nil
	}

	return sqliteDB, mr, cleanup
}

func seedDefaultConfigs(t *testing.T, tx *gorm.DB) {
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
		t.Fatalf("failed to seed default system configs: %v", err)
	}

	// Also seed these in miniredis context if required, but they are stored in postgres first.
	// We'll write configs to miniredis in actual handlers.
	for _, config := range defaultConfigs {
		_ = db.HSetJSON(context.Background(), model.SystemConfigRedisHashKey, config.Key, &config)
	}
}
