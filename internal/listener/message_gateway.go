// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package listener

import (
	"context"

	"github.com/Rain-kl/Wavelet/pkg/message_gateway"
)

// EventMessageGatewayInbound is the domain event name for authorized inbound messages.
const EventMessageGatewayInbound = "message_gateway.inbound"

// MessageGatewayInbound is emitted when a bound user sends a private message.
type MessageGatewayInbound struct {
	Msg message_gateway.InboundMessage
}

// MessageGatewayInboundHandler handles inbound messaging events.
type MessageGatewayInboundHandler func(ctx context.Context, event MessageGatewayInbound)

var messageGatewayInboundHandlers []MessageGatewayInboundHandler

// OnMessageGatewayInbound registers a handler. Call from bootstrap only.
func OnMessageGatewayInbound(handler MessageGatewayInboundHandler) {
	messageGatewayInboundHandlers = append(messageGatewayInboundHandlers, handler)
}

// EmitMessageGatewayInbound dispatches a bound inbound message.
func EmitMessageGatewayInbound(ctx context.Context, msg message_gateway.InboundMessage) {
	if msg.BindingUserID == nil {
		return
	}
	event := MessageGatewayInbound{Msg: msg}
	for _, handler := range messageGatewayInboundHandlers {
		handler(ctx, event)
	}
}
