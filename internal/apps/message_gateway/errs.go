// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package message_gateway

import "errors"

var (
	errCodeInvalid          = errors.New("invalid or expired pairing code")
	errChannelMismatch      = errors.New("pairing code does not match channel")
	errPlatformAlreadyBound = errors.New("this platform account is already bound")
	errBindingNotFound      = errors.New("binding not found")
	errBindingForbidden     = errors.New("cannot unbind another user's binding")
	errChannelIDRequired    = errors.New("channel_id is required")
	errChannelDisabled      = errors.New("channel is not enabled")
)
