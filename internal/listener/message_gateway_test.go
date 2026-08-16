// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package listener

import (
	"context"
	"testing"

	"github.com/Rain-kl/Wavelet/pkg/message_gateway"
)

func TestEmitMessageGatewayInbound_SkipsUnbound(t *testing.T) {
	called := 0
	OnMessageGatewayInbound(func(ctx context.Context, ev MessageGatewayInbound) { called++ })
	EmitMessageGatewayInbound(context.Background(), message_gateway.InboundMessage{Text: "x"})
	if called != 0 {
		t.Fatal("unbound must not emit")
	}
	uid := uint64(9)
	EmitMessageGatewayInbound(context.Background(), message_gateway.InboundMessage{BindingUserID: &uid, Text: "x"})
	if called != 1 {
		t.Fatalf("called=%d", called)
	}
}
