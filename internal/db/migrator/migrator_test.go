/*
Copyright 2026 Arctel.net

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
	"testing"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMigrateInitializesSQLiteDatabase(t *testing.T) {
	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("gorm.Open(sqlite) error = %v", err)
	}

	previousDBEnabled := config.Config.Database.Enabled
	config.Config.Database.Enabled = false
	db.SetDB(sqliteDB)
	t.Cleanup(func() {
		config.Config.Database.Enabled = previousDBEnabled
		db.SetDB(nil)
	})

	Migrate()

	var systemConfigCount int64
	if err := sqliteDB.Table("system_configs").Count(&systemConfigCount).Error; err != nil {
		t.Fatalf("Migrate() count system_configs error = %v", err)
	}
	if systemConfigCount != 23 {
		t.Errorf("Migrate() system_configs count = %d, want %d", systemConfigCount, 23)
	}

	var adminCount int64
	if err := sqliteDB.Table("users").Where("username = ?", "admin").Count(&adminCount).Error; err != nil {
		t.Fatalf("Migrate() count admin user error = %v", err)
	}
	if adminCount != 1 {
		t.Errorf("Migrate() admin user count = %d, want %d", adminCount, 1)
	}

	var templateCount int64
	if err := sqliteDB.Table("templates").Count(&templateCount).Error; err != nil {
		t.Fatalf("Migrate() count templates error = %v", err)
	}
	if templateCount != 2 {
		t.Errorf("Migrate() templates count = %d, want %d", templateCount, 2)
	}
}
