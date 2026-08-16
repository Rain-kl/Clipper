// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package message_gateway

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"gorm.io/gorm"
)

func seedChannel(t *testing.T, ctx context.Context) *model.MessageChannel {
	t.Helper()
	ch := &model.MessageChannel{
		Name:       "tg",
		Type:       model.MessageChannelTypeTelegram,
		OwnerScope: model.MessageOwnerScopeSystem,
		Enabled:    true,
	}
	if err := repository.CreateMessageChannel(ctx, ch); err != nil {
		t.Fatalf("CreateMessageChannel() error = %v", err)
	}
	return ch
}

func TestBind_ExpiredCode(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	ch := seedChannel(t, ctx)
	if _, err := repository.UpsertPairingCode(ctx, ch.ID, "u1", "ABCD2345", time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("UpsertPairingCode() error = %v", err)
	}
	_, err := bindChannel(ctx, 1, BindRequest{ChannelID: fmt.Sprint(ch.ID), Code: "ABCD-2345"})
	if err == nil {
		t.Fatal("bindChannel() error = nil, want expired code")
	}
}

func TestBind_HappyPathDeletesCode(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	ch := seedChannel(t, ctx)
	if _, err := repository.UpsertPairingCode(ctx, ch.ID, "plat-1", "ABCD2345", time.Now().Add(15*time.Minute)); err != nil {
		t.Fatalf("UpsertPairingCode() error = %v", err)
	}
	dto, err := bindChannel(ctx, 42, BindRequest{ChannelID: fmt.Sprint(ch.ID), Code: "abcd-2345"})
	if err != nil {
		t.Fatalf("bindChannel() error = %v", err)
	}
	if dto.PlatformUserID != "plat-1" || dto.ChannelID != ch.ID || dto.UserID != 42 {
		t.Fatalf("bindChannel() dto = %+v", dto)
	}
	_, err = repository.GetPairingCode(ctx, "ABCD2345")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GetPairingCode() error = %v, want not found", err)
	}
}

func TestBind_ConflictOtherUser(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	ch := seedChannel(t, ctx)
	if err := repository.CreateMessageBinding(ctx, &model.MessageBinding{
		UserID: 7, ChannelID: ch.ID, PlatformUserID: "plat-1",
	}); err != nil {
		t.Fatalf("CreateMessageBinding() error = %v", err)
	}
	if _, err := repository.UpsertPairingCode(ctx, ch.ID, "plat-1", "ABCD2345", time.Now().Add(15*time.Minute)); err != nil {
		t.Fatalf("UpsertPairingCode() error = %v", err)
	}
	_, err := bindChannel(ctx, 42, BindRequest{ChannelID: fmt.Sprint(ch.ID), Code: "ABCD2345"})
	if !errors.Is(err, errPlatformAlreadyBound) {
		t.Fatalf("bindChannel() error = %v, want %v", err, errPlatformAlreadyBound)
	}
}

func TestUnbind_OnlyOwnBinding(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	ch := seedChannel(t, ctx)
	b := &model.MessageBinding{UserID: 7, ChannelID: ch.ID, PlatformUserID: "plat-1"}
	if err := repository.CreateMessageBinding(ctx, b); err != nil {
		t.Fatalf("CreateMessageBinding() error = %v", err)
	}
	if err := unbindChannel(ctx, 42, b.ID); err == nil {
		t.Fatal("unbindChannel() error = nil, want forbidden")
	}
	if err := unbindChannel(ctx, 7, b.ID); err != nil {
		t.Fatalf("unbindChannel() own binding error = %v", err)
	}
	_, err := repository.GetMessageBinding(ctx, b.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GetMessageBinding() error = %v, want deleted", err)
	}
}
