// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package item

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/infra/objectstore"
	"github.com/Rain-kl/Wavelet/internal/infra/persistence"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/Rain-kl/Wavelet/pkg/message_gateway"
)

func uidPtr(id uint64) *uint64 { return &id }

func tgMsg(userID uint64, mid, text string) message_gateway.InboundMessage {
	return message_gateway.InboundMessage{
		ChannelID:      10,
		ChannelType:    model.ItemSourceTelegram,
		PlatformUserID: "tg-1",
		MessageID:      mid,
		Text:           text,
		BindingUserID:  uidPtr(userID),
	}
}

func TestIngestInbound_CreatesTelegramText(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	if err := IngestInbound(ctx, tgMsg(1001, "m1", "hello")); err != nil {
		t.Fatalf("IngestInbound() error = %v", err)
	}
	list, err := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if err != nil || list.Total != 1 {
		t.Fatalf("ListItems() total = %d err = %v, want 1", list.Total, err)
	}
	got := list.Results[0]
	if got.Source != model.ItemSourceTelegram || got.Body != "hello" || got.Title != "" {
		t.Fatalf("item = %+v", got)
	}
}

func TestIngestInbound_MergesWithin60s(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	if err := IngestInbound(ctx, tgMsg(1001, "m1", "a")); err != nil {
		t.Fatal(err)
	}
	if err := IngestInbound(ctx, tgMsg(1001, "m2", "b")); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 1 || list.Results[0].Body != "a\nb" {
		t.Fatalf("merge = %+v", list)
	}
}

func TestIngestInbound_NewClipAfter60s(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	if err := IngestInbound(ctx, tgMsg(1001, "m1", "a")); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if err := db.DB(ctx).Model(&model.Item{}).Where("id = ?", list.Results[0].ID).
		Update("updated_at", time.Now().Add(-61*time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if err := IngestInbound(ctx, tgMsg(1001, "m2", "b")); err != nil {
		t.Fatal(err)
	}
	list, _ = ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 2 {
		t.Fatalf("total = %d, want 2", list.Total)
	}
}

func TestIngestInbound_SkipsNonPending(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	_ = IngestInbound(ctx, tgMsg(1001, "m1", "a"))
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	active := model.ItemLifecycleActive
	imp := model.ItemImportanceFragment
	if _, err := PatchItem(ctx, 1001, list.Results[0].ID, PatchItemInput{Lifecycle: &active, Importance: &imp}); err != nil {
		t.Fatal(err)
	}
	_ = IngestInbound(ctx, tgMsg(1001, "m2", "b"))
	list, _ = ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 2 {
		t.Fatalf("total = %d, want 2", list.Total)
	}
}

func TestIngestInbound_DoesNotMergeQQWithTelegram(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	_ = IngestInbound(ctx, tgMsg(1001, "m1", "a"))
	qq := tgMsg(1001, "m2", "b")
	qq.ChannelType = model.ItemSourceQQ
	_ = IngestInbound(ctx, qq)
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 2 {
		t.Fatalf("total = %d, want 2", list.Total)
	}
}

func TestIngestInbound_WebCreateDoesNotMerge(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	if _, err := CreateItem(ctx, 1001, CreateItemInput{Body: "web"}); err != nil {
		t.Fatal(err)
	}
	if err := IngestInbound(ctx, tgMsg(1001, "m1", "bot")); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 2 {
		t.Fatalf("total = %d, want 2", list.Total)
	}
}

func TestIngestInbound_EmptyRejected(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	err := IngestInbound(context.Background(), tgMsg(1001, "m1", "  "))
	if err == nil || err.Error() != errEmptyContent {
		t.Fatalf("error = %v, want %s", err, errEmptyContent)
	}
}

func TestIngestInbound_DedupSameMessageID(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	msg := tgMsg(1001, "same", "hello")
	if err := IngestInbound(ctx, msg); err != nil {
		t.Fatal(err)
	}
	if err := IngestInbound(ctx, msg); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 1 || list.Results[0].Body != "hello" {
		t.Fatalf("dedup = %+v", list)
	}
}

func TestIngestInbound_ImageCaption(t *testing.T) {
	restore, disable := setupInboundMockStorage(t)
	defer restore()
	defer disable()
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(path, []byte("jpeg-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	msg := tgMsg(1001, "img1", "caption")
	msg.Attachments = []message_gateway.Attachment{{Path: path, FileName: "photo.jpg", MIME: "image/jpeg"}}
	if err := IngestInbound(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(context.Background(), 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 1 {
		t.Fatalf("ListItems() total = %d, want 1", list.Total)
	}
	got, err := GetItem(context.Background(), 1001, list.Results[0].ID)
	if err != nil {
		t.Fatalf("GetItem() error = %v", err)
	}
	if len(got.Attachments) != 1 {
		t.Fatalf("GetItem() attachments = %+v, want 1", got)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("temp file must be removed")
	}
}

func setupInboundMockStorage(t *testing.T) (restore func(), disable func()) {
	t.Helper()
	mockFiles := make(map[string][]byte)
	restore = objectstore.MockStorage(
		func(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
			data, err := io.ReadAll(body)
			if err != nil {
				return err
			}
			mockFiles[key] = data
			return nil
		},
		func(ctx context.Context, key string) (*objectstore.Object, error) {
			data, ok := mockFiles[key]
			if !ok {
				return nil, os.ErrNotExist
			}
			return &objectstore.Object{
				Body:          io.NopCloser(bytes.NewReader(data)),
				ContentLength: int64(len(data)),
				ContentType:   "application/octet-stream",
			}, nil
		},
		func(ctx context.Context, key string) error {
			delete(mockFiles, key)
			return nil
		},
	)
	objectstore.IsEnabledFunc = func() bool { return true }
	objectstore.ResetCache()
	disable = func() {
		objectstore.IsEnabledFunc = func() bool { return false }
		objectstore.ResetCache()
	}
	return restore, disable
}

func TestIngestInbound_NilBindingRejected(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	msg := tgMsg(1001, "m1", "x")
	msg.BindingUserID = nil
	if err := IngestInbound(context.Background(), msg); err == nil {
		t.Fatal("expected error")
	}
}
