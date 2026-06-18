// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"testing"

	admin_push "github.com/Rain-kl/Wavelet/internal/apps/admin/push"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestInitSyncsPushEventsOnce(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	if err := dbConn.AutoMigrate(&model.PushEvent{}); err != nil {
		t.Fatalf("auto migrate push events failed: %v", err)
	}

	ctx := context.Background()
	Init(ctx, Options{})
	Init(ctx, Options{API: true})

	var count int64
	if err := dbConn.Model(&model.PushEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count push events failed: %v", err)
	}
	if count != int64(len(admin_push.BuiltInEvents)) {
		t.Fatalf("push event count = %d, want %d", count, len(admin_push.BuiltInEvents))
	}
}