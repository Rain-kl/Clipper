// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package item

import (
	"context"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestArchivePendingHandler_Execute(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	created, err := CreateItem(ctx, 1001, CreateItemInput{Body: "handler archive me"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	old := time.Now().Add(-5 * 24 * time.Hour)
	if err := dbConn.Model(&model.Item{}).Where("id = ?", created.ID).Update("created_at", old).Error; err != nil {
		t.Fatalf("backdate created_at: %v", err)
	}

	handler := &ArchivePendingHandler{}
	result, err := handler.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == nil || result.Message == "" {
		t.Fatalf("result = %+v, want non-empty message", result)
	}

	got, err := GetItem(ctx, 1001, created.ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if got.Lifecycle != model.ItemLifecycleArchived || got.Importance != model.ItemImportanceFragment {
		t.Fatalf("state = %s/%s, want archived/fragment", got.Lifecycle, got.Importance)
	}
	if got.ArchivedAt == nil {
		t.Fatal("ArchivedAt = nil, want set")
	}
}

func TestPurgeTrashHandler_Execute(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	created, err := CreateItem(ctx, 1001, CreateItemInput{Body: "handler purge me"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if err := DeleteItem(ctx, 1001, created.ID, false); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}

	old := time.Now().Add(-40 * 24 * time.Hour)
	if err := dbConn.Model(&model.Item{}).Where("id = ?", created.ID).Update("trashed_at", old).Error; err != nil {
		t.Fatalf("backdate trashed_at: %v", err)
	}

	handler := &PurgeTrashHandler{}
	result, err := handler.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == nil || result.Message == "" {
		t.Fatalf("result = %+v, want non-empty message", result)
	}

	var count int64
	if err := dbConn.Model(&model.Item{}).Where("id = ?", created.ID).Count(&count).Error; err != nil {
		t.Fatalf("count item: %v", err)
	}
	if count != 0 {
		t.Fatalf("item still exists after purge, count=%d", count)
	}
}
