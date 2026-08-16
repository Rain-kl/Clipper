// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package message_gateway

import (
	"context"
	"strings"
	"testing"

	appgw "github.com/Rain-kl/Wavelet/internal/apps/message_gateway"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestCreateChannel_TelegramRequiresToken(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	_, err := createChannel(context.Background(), CreateChannelRequest{Name: "tg", Type: "telegram"})
	if err == nil {
		t.Fatal("createChannel() error = nil, want token required")
	}
}

func TestCreateChannel_StoresCiphertextNotPlaintext(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	const token = "secret-token-xyz"
	dto, err := createChannel(context.Background(), CreateChannelRequest{
		Name:     "tg",
		Type:     "telegram",
		BotToken: token,
	})
	if err != nil {
		t.Fatalf("createChannel() error = %v", err)
	}
	if strings.Contains(dto.BotToken, token) {
		t.Fatalf("createChannel() dto leaked plaintext token %q", dto.BotToken)
	}
	row, err := repository.GetMessageChannel(context.Background(), dto.ID)
	if err != nil {
		t.Fatalf("GetMessageChannel() error = %v", err)
	}
	if strings.Contains(row.Credentials, token) {
		t.Fatalf("stored credentials contain plaintext token")
	}
	creds, err := appgw.DecryptCredentials(row.Credentials)
	if err != nil {
		t.Fatalf("DecryptCredentials() error = %v", err)
	}
	if creds["bot_token"] != token {
		t.Fatalf("DecryptCredentials() bot_token = %q, want %q", creds["bot_token"], token)
	}
}

func TestUpdateChannel_EmptySecretKeepsPrevious(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	created, err := createChannel(context.Background(), CreateChannelRequest{
		Name:     "tg",
		Type:     "telegram",
		BotToken: "old-token",
	})
	if err != nil {
		t.Fatalf("createChannel() error = %v", err)
	}
	name := "renamed"
	if _, err := updateChannel(context.Background(), created.ID, UpdateChannelRequest{Name: &name}); err != nil {
		t.Fatalf("updateChannel() error = %v", err)
	}
	row, err := repository.GetMessageChannel(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetMessageChannel() error = %v", err)
	}
	if row.Name != "renamed" {
		t.Fatalf("updateChannel() name = %q, want renamed", row.Name)
	}
	creds, err := appgw.DecryptCredentials(row.Credentials)
	if err != nil {
		t.Fatalf("DecryptCredentials() error = %v", err)
	}
	if creds["bot_token"] != "old-token" {
		t.Fatalf("updateChannel() bot_token = %q, want old-token", creds["bot_token"])
	}
}
