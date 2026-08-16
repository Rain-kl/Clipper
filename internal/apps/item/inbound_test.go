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

func TestIngestInbound_NilBindingRejected(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	msg := tgMsg(1001, "m1", "x")
	msg.BindingUserID = nil
	if err := IngestInbound(context.Background(), msg); err == nil {
		t.Fatal("expected error")
	}
}
