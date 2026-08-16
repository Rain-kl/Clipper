// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

const (
	// MessageChannelTypeTelegram is a Telegram bot channel.
	MessageChannelTypeTelegram = "telegram"
	// MessageChannelTypeQQ is an official QQ bot channel.
	MessageChannelTypeQQ = "qq"
	// MessageOwnerScopeSystem is an instance-level shared bot.
	MessageOwnerScopeSystem = "system"
)

// MessageChannel is an admin-configured messaging adapter.
type MessageChannel struct {
	ID          uint64    `json:"id,string" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Type        string    `json:"type" gorm:"size:32;not null;index"`
	OwnerScope  string    `json:"owner_scope" gorm:"size:16;not null;default:system"`
	OwnerID     *uint64   `json:"owner_id,string"`
	Enabled     bool      `json:"enabled" gorm:"not null;default:true"`
	Credentials string    `json:"-" gorm:"type:text"`
	Extra       string    `json:"extra" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns w_message_channels.
func (MessageChannel) TableName() string { return "w_message_channels" }

// MessageBinding maps a platform user to a Wavelet user on one channel.
type MessageBinding struct {
	ID             uint64    `json:"id,string" gorm:"primaryKey"`
	UserID         uint64    `json:"user_id,string" gorm:"not null;index"`
	ChannelID      uint64    `json:"channel_id,string" gorm:"not null;index"`
	PlatformUserID string    `json:"platform_user_id" gorm:"size:128;not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns w_message_bindings.
func (MessageBinding) TableName() string { return "w_message_bindings" }

// MessagePairingCode is a one-time bind code.
type MessagePairingCode struct {
	Code           string    `json:"code" gorm:"primaryKey;size:16"`
	ChannelID      uint64    `json:"channel_id,string" gorm:"not null"`
	PlatformUserID string    `json:"platform_user_id" gorm:"size:128;not null"`
	ExpiresAt      time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns w_message_pairing_codes.
func (MessagePairingCode) TableName() string { return "w_message_pairing_codes" }
