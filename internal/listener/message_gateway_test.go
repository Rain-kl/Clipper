// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package listener

import (
	"context"
	"errors"
	"testing"

	"github.com/Rain-kl/Wavelet/pkg/message_gateway"
)

func TestEmitMessageGatewayInbound_SkipsUnbound(t *testing.T) {
	messageGatewayInboundHandlers = nil
	called := 0
	OnMessageGatewayInbound(func(ctx context.Context, ev MessageGatewayInbound) error {
		called++
		return nil
	})
	if err := EmitMessageGatewayInbound(context.Background(), message_gateway.InboundMessage{Text: "x"}); err != nil {
		t.Fatalf("EmitMessageGatewayInbound() error = %v", err)
	}
	if called != 0 {
		t.Fatal("unbound must not emit")
	}
	uid := uint64(9)
	if err := EmitMessageGatewayInbound(context.Background(), message_gateway.InboundMessage{BindingUserID: &uid, Text: "x"}); err != nil {
		t.Fatalf("EmitMessageGatewayInbound() error = %v", err)
	}
	if called != 1 {
		t.Fatalf("called=%d", called)
	}
}

func TestEmitMessageGatewayInbound_ReturnsHandlerError(t *testing.T) {
	messageGatewayInboundHandlers = nil
	want := errors.New("save failed")
	OnMessageGatewayInbound(func(ctx context.Context, ev MessageGatewayInbound) error {
		return want
	})
	uid := uint64(1)
	err := EmitMessageGatewayInbound(context.Background(), message_gateway.InboundMessage{BindingUserID: &uid, Text: "x"})
	if !errors.Is(err, want) {
		t.Fatalf("Emit() error = %v, want %v", err, want)
	}
}
