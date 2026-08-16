// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/pkg/message_gateway"
	"gorm.io/gorm"
)

func TestHandle_UnboundMintsCodeAndDoesNotEmit(t *testing.T) {
	var sent []message_gateway.OutboundMessage
	var emitted int
	var upserted string
	d := inboundDeps{
		LookupBinding: func(ctx context.Context, channelID uint64, platformUserID string) (*model.MessageBinding, error) {
			return nil, gorm.ErrRecordNotFound
		},
		GenerateCode: func() (string, error) { return "ABCD2345", nil },
		UpsertCode: func(ctx context.Context, channelID uint64, platformUserID, code string, expiresAt time.Time) (*model.MessagePairingCode, error) {
			upserted = code
			return &model.MessagePairingCode{Code: code, ChannelID: channelID, PlatformUserID: platformUserID, ExpiresAt: expiresAt}, nil
		},
		Emit: func(ctx context.Context, msg message_gateway.InboundMessage) error {
			emitted++
			return nil
		},
		Send: func(ctx context.Context, to message_gateway.Recipient, msg message_gateway.OutboundMessage) error {
			sent = append(sent, msg)
			return nil
		},
	}
	err := d.Handle(context.Background(), message_gateway.InboundMessage{
		ChannelID: 1, PlatformUserID: "u1", ChatID: "u1", Text: "hi",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if emitted != 0 {
		t.Fatalf("Handle() emitted = %d, want 0", emitted)
	}
	if upserted != "ABCD2345" {
		t.Fatalf("UpsertCode() code = %q, want %q", upserted, "ABCD2345")
	}
	if len(sent) != 1 {
		t.Fatalf("Send() calls = %d, want 1", len(sent))
	}
	if !strings.Contains(sent[0].Text, "ABCD-2345") {
		t.Fatalf("Handle() send text = %q, want pairing code ABCD-2345", sent[0].Text)
	}
}

func TestHandle_UnboundReusesExistingCode(t *testing.T) {
	var sent string
	d := inboundDeps{
		LookupBinding: func(ctx context.Context, channelID uint64, platformUserID string) (*model.MessageBinding, error) {
			return nil, gorm.ErrRecordNotFound
		},
		GenerateCode: func() (string, error) { return "NEWCODE1", nil },
		UpsertCode: func(ctx context.Context, channelID uint64, platformUserID, code string, expiresAt time.Time) (*model.MessagePairingCode, error) {
			return &model.MessagePairingCode{Code: "OLDCODE2"}, nil
		},
		Emit: func(ctx context.Context, msg message_gateway.InboundMessage) error {
			t.Fatal("must not emit")
			return nil
		},
		Send: func(ctx context.Context, to message_gateway.Recipient, msg message_gateway.OutboundMessage) error {
			sent = msg.Text
			return nil
		},
	}
	if err := d.Handle(context.Background(), message_gateway.InboundMessage{ChannelID: 1, PlatformUserID: "u1"}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !strings.Contains(sent, "OLDC-ODE2") {
		t.Fatalf("Handle() send text = %q, want reused code OLDC-ODE2", sent)
	}
}

func TestHandle_BoundEmitsAndAcks(t *testing.T) {
	var got message_gateway.InboundMessage
	var sent string
	d := inboundDeps{
		LookupBinding: func(ctx context.Context, channelID uint64, platformUserID string) (*model.MessageBinding, error) {
			return &model.MessageBinding{UserID: 9, ChannelID: 1, PlatformUserID: "u1"}, nil
		},
		Emit: func(ctx context.Context, msg message_gateway.InboundMessage) error {
			got = msg
			return nil
		},
		Send: func(ctx context.Context, to message_gateway.Recipient, msg message_gateway.OutboundMessage) error {
			sent = msg.Text
			return nil
		},
	}
	err := d.Handle(context.Background(), message_gateway.InboundMessage{
		ChannelID: 1, PlatformUserID: "u1", Text: "hello",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got.Text != "hello" || got.BindingUserID == nil || *got.BindingUserID != 9 {
		t.Fatalf("Handle() emit = %+v, want text=hello user=9", got)
	}
	if sent != "received" {
		t.Fatalf("Handle() ack = %q, want %q", sent, "received")
	}
}

func TestHandle_BoundEmitError(t *testing.T) {
	var sent string
	d := inboundDeps{
		LookupBinding: func(ctx context.Context, channelID uint64, platformUserID string) (*model.MessageBinding, error) {
			return &model.MessageBinding{UserID: 9, ChannelID: 1, PlatformUserID: "u1"}, nil
		},
		Emit: func(ctx context.Context, msg message_gateway.InboundMessage) error {
			return errors.New("listener failed")
		},
		Send: func(ctx context.Context, to message_gateway.Recipient, msg message_gateway.OutboundMessage) error {
			sent = msg.Text
			return nil
		},
	}
	err := d.Handle(context.Background(), message_gateway.InboundMessage{ChannelID: 1, PlatformUserID: "u1", Text: "x"})
	if err == nil {
		t.Fatal("Handle() error = nil, want listener error")
	}
	if sent != "could not save your message" {
		t.Fatalf("Handle() send = %q, want could not save your message", sent)
	}
}
