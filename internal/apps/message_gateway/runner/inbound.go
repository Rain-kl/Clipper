// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/pkg/message_gateway"
	"gorm.io/gorm"
)

const pairingTTL = 15 * time.Minute

type inboundDeps struct {
	LookupBinding func(ctx context.Context, channelID uint64, platformUserID string) (*model.MessageBinding, error)
	UpsertCode    func(ctx context.Context, channelID uint64, platformUserID, code string, expiresAt time.Time) (*model.MessagePairingCode, error)
	GenerateCode  func() (string, error)
	Emit          func(context.Context, message_gateway.InboundMessage) error
	Send          func(context.Context, message_gateway.Recipient, message_gateway.OutboundMessage) error
}

// Handle pairs unbound senders or emits bound inbound messages.
func (d inboundDeps) Handle(ctx context.Context, msg message_gateway.InboundMessage) error {
	binding, err := d.LookupBinding(ctx, msg.ChannelID, msg.PlatformUserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	to := message_gateway.Recipient{ChatID: msg.ChatID, PlatformUserID: msg.PlatformUserID}
	if binding == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		gen := d.GenerateCode
		if gen == nil {
			gen = message_gateway.GenerateCode
		}
		code, err := gen()
		if err != nil {
			return err
		}
		row, err := d.UpsertCode(ctx, msg.ChannelID, msg.PlatformUserID, code, time.Now().Add(pairingTTL))
		if err != nil {
			return err
		}
		display := message_gateway.FormatCode(row.Code)
		return d.Send(ctx, to, message_gateway.OutboundMessage{
			Text:      fmt.Sprintf("Your pairing code is %s. Open Settings → Profile → Bind a bot and enter this code. It expires in 15 minutes.", display),
			ReplyToID: msg.MessageID,
		})
	}

	uid := binding.UserID
	msg.BindingUserID = &uid
	if err := d.Emit(ctx, msg); err != nil {
		_ = d.Send(ctx, to, message_gateway.OutboundMessage{
			Text:      "could not save your message",
			ReplyToID: msg.MessageID,
		})
		return err
	}
	return d.Send(ctx, to, message_gateway.OutboundMessage{
		Text:      "received",
		ReplyToID: msg.MessageID,
	})
}
