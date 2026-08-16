// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package item

import (
	"context"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/infra/persistence"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestCreateItem_TextOnly(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	userID := uint64(1001)
	item, err := CreateItem(ctx, userID, CreateItemInput{Body: "hello"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if item.Lifecycle != model.ItemLifecyclePending || item.Importance != model.ItemImportanceNone {
		t.Fatalf("state = %s/%s", item.Lifecycle, item.Importance)
	}
	if item.ContentType != model.ItemContentTypeText || item.Body != "hello" {
		t.Fatalf("content mismatch: %+v", item)
	}
	if item.Source != model.ItemSourceWeb {
		t.Fatalf("Source = %s, want %s", item.Source, model.ItemSourceWeb)
	}
	if item.ID == 0 {
		t.Fatal("CreateItem() ID = 0, want non-zero")
	}
}

func TestCreateItem_EmptyRejected(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	_, err := CreateItem(ctx, 1001, CreateItemInput{})
	if err == nil {
		t.Fatal("CreateItem() error = nil, want empty content error")
	}
	if err.Error() != errEmptyContent {
		t.Fatalf("CreateItem() error = %v, want %s", err, errEmptyContent)
	}
}

func TestCreateItem_CrossUserUploadRejected(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	otherUpload := model.Upload{
		ID:        88001,
		UserID:    9999,
		FileName:  "secret.png",
		FilePath:  "uploads/secret.png",
		FileSize:  10,
		MimeType:  "image/png",
		Extension: "png",
		Type:      "attachment",
		Status:    model.UploadStatusPending,
	}
	if err := dbConn.Create(&otherUpload).Error; err != nil {
		t.Fatalf("seed upload: %v", err)
	}

	_, err := CreateItem(ctx, 1001, CreateItemInput{UploadIDs: []uint64{otherUpload.ID}})
	if err == nil {
		t.Fatal("CreateItem() error = nil, want invalid upload error")
	}
	if err.Error() != errInvalidUpload {
		t.Fatalf("CreateItem() error = %v, want %s", err, errInvalidUpload)
	}
}

func TestClassifyPendingToNote(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	created, err := CreateItem(ctx, 1001, CreateItemInput{Body: "classify me"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	lifecycle := model.ItemLifecycleActive
	importance := model.ItemImportanceNote
	updated, err := PatchItem(ctx, 1001, created.ID, PatchItemInput{
		Lifecycle:  &lifecycle,
		Importance: &importance,
	})
	if err != nil {
		t.Fatalf("PatchItem: %v", err)
	}
	if updated.Lifecycle != model.ItemLifecycleActive || updated.Importance != model.ItemImportanceNote {
		t.Fatalf("PatchItem() state = %s/%s, want active/note", updated.Lifecycle, updated.Importance)
	}
}

func TestTrashAndRestore(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	created, err := CreateItem(ctx, 1001, CreateItemInput{Body: "trash me"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	if err := DeleteItem(ctx, 1001, created.ID, false); err != nil {
		t.Fatalf("DeleteItem soft: %v", err)
	}
	trashed, err := GetItem(ctx, 1001, created.ID)
	if err != nil {
		t.Fatalf("GetItem after trash: %v", err)
	}
	if trashed.Lifecycle != model.ItemLifecycleTrash {
		t.Fatalf("Lifecycle = %s, want trash", trashed.Lifecycle)
	}
	if trashed.TrashedAt == nil {
		t.Fatal("TrashedAt = nil, want set")
	}

	lifecycle := model.ItemLifecycleActive
	restored, err := PatchItem(ctx, 1001, created.ID, PatchItemInput{Lifecycle: &lifecycle})
	if err != nil {
		t.Fatalf("PatchItem restore: %v", err)
	}
	if restored.Lifecycle != model.ItemLifecyclePending {
		t.Fatalf("Lifecycle after restore = %s, want pending (importance none)", restored.Lifecycle)
	}
	if restored.TrashedAt != nil {
		t.Fatalf("TrashedAt after restore = %v, want nil", restored.TrashedAt)
	}
}

func TestListDefaultExcludesArchivedAndTrash(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	userID := uint64(1001)

	pending, err := CreateItem(ctx, userID, CreateItemInput{Body: "pending"})
	if err != nil {
		t.Fatalf("CreateItem pending: %v", err)
	}
	activeBody, err := CreateItem(ctx, userID, CreateItemInput{Body: "to active"})
	if err != nil {
		t.Fatalf("CreateItem active: %v", err)
	}
	lc := model.ItemLifecycleActive
	imp := model.ItemImportanceNote
	if _, err := PatchItem(ctx, userID, activeBody.ID, PatchItemInput{Lifecycle: &lc, Importance: &imp}); err != nil {
		t.Fatalf("PatchItem active: %v", err)
	}

	trashItem, err := CreateItem(ctx, userID, CreateItemInput{Body: "to trash"})
	if err != nil {
		t.Fatalf("CreateItem trash: %v", err)
	}
	if err := DeleteItem(ctx, userID, trashItem.ID, false); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}

	archivedSrc, err := CreateItem(ctx, userID, CreateItemInput{Body: "to archive"})
	if err != nil {
		t.Fatalf("CreateItem archive: %v", err)
	}
	// classify then archive
	if _, err := PatchItem(ctx, userID, archivedSrc.ID, PatchItemInput{Lifecycle: &lc, Importance: &imp}); err != nil {
		t.Fatalf("PatchItem classify: %v", err)
	}
	arch := model.ItemLifecycleArchived
	if _, err := PatchItem(ctx, userID, archivedSrc.ID, PatchItemInput{Lifecycle: &arch}); err != nil {
		t.Fatalf("PatchItem archive: %v", err)
	}

	result, err := ListItems(ctx, userID, ListItemsQuery{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("ListItems Total = %d, want 2", result.Total)
	}
	ids := map[uint64]bool{}
	for _, it := range result.Results {
		ids[it.ID] = true
		if it.Lifecycle == model.ItemLifecycleArchived || it.Lifecycle == model.ItemLifecycleTrash {
			t.Fatalf("ListItems included lifecycle %s", it.Lifecycle)
		}
	}
	if !ids[pending.ID] || !ids[activeBody.ID] {
		t.Fatalf("ListItems missing pending/active ids: %v", ids)
	}
}

func TestGetItem_OtherUserNotFound(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	created, err := CreateItem(ctx, 1001, CreateItemInput{Body: "private"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	_, err = GetItem(ctx, 2002, created.ID)
	if err == nil {
		t.Fatal("GetItem() error = nil, want not found")
	}
	if err.Error() != errItemNotFound {
		t.Fatalf("GetItem() error = %v, want %s", err, errItemNotFound)
	}
}

func TestArchivePendingJob(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	created, err := CreateItem(ctx, 1001, CreateItemInput{Body: "old pending"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	old := time.Now().Add(-5 * 24 * time.Hour)
	if err := dbConn.Model(&model.Item{}).Where("id = ?", created.ID).Update("created_at", old).Error; err != nil {
		t.Fatalf("backdate created_at: %v", err)
	}

	n, err := ArchivePendingItems(ctx, 3*24*time.Hour)
	if err != nil {
		t.Fatalf("ArchivePendingItems: %v", err)
	}
	if n != 1 {
		t.Fatalf("ArchivePendingItems() = %d, want 1", n)
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

func TestPurgeTrashJob(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	created, err := CreateItem(ctx, 1001, CreateItemInput{Body: "purge me"})
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

	n, err := PurgeTrashItems(ctx, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("PurgeTrashItems: %v", err)
	}
	if n != 1 {
		t.Fatalf("PurgeTrashItems() = %d, want 1", n)
	}

	var count int64
	if err := db.DB(ctx).Model(&model.Item{}).Where("id = ?", created.ID).Count(&count).Error; err != nil {
		t.Fatalf("count item: %v", err)
	}
	if count != 0 {
		t.Fatalf("item still exists after purge, count=%d", count)
	}
}

func TestCreateItem_WhitespaceBodyRejected(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	_, err := CreateItem(ctx, 1001, CreateItemInput{Body: "   \n\t  "})
	if err == nil {
		t.Fatal("CreateItem() error = nil, want empty content")
	}
	if err.Error() != errEmptyContent {
		t.Fatalf("CreateItem() error = %v, want %s", err, errEmptyContent)
	}
}

func TestStats_VaultExcludesArchived(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	userID := uint64(1001)

	activeVault, err := CreateItem(ctx, userID, CreateItemInput{Body: "active vault"})
	if err != nil {
		t.Fatalf("CreateItem active vault: %v", err)
	}
	lcActive := model.ItemLifecycleActive
	impVault := model.ItemImportanceVault
	if _, err := PatchItem(ctx, userID, activeVault.ID, PatchItemInput{
		Lifecycle:  &lcActive,
		Importance: &impVault,
	}); err != nil {
		t.Fatalf("PatchItem active vault: %v", err)
	}

	archivedVault, err := CreateItem(ctx, userID, CreateItemInput{Body: "archived vault"})
	if err != nil {
		t.Fatalf("CreateItem archived vault: %v", err)
	}
	if _, err := PatchItem(ctx, userID, archivedVault.ID, PatchItemInput{
		Lifecycle:  &lcActive,
		Importance: &impVault,
	}); err != nil {
		t.Fatalf("PatchItem classify vault: %v", err)
	}
	lcArchived := model.ItemLifecycleArchived
	if _, err := PatchItem(ctx, userID, archivedVault.ID, PatchItemInput{Lifecycle: &lcArchived}); err != nil {
		t.Fatalf("PatchItem archive: %v", err)
	}

	stats, err := Stats(ctx, userID)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Vault != 1 {
		t.Fatalf("Stats.Vault = %d, want 1 (active only; archived vault excluded)", stats.Vault)
	}
	if stats.Archived != 1 {
		t.Fatalf("Stats.Archived = %d, want 1", stats.Archived)
	}
	if stats.Active != 1 {
		t.Fatalf("Stats.Active = %d, want 1", stats.Active)
	}
}
