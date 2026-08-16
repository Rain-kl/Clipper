// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

// ItemIngestKey makes one platform message idempotent.
type ItemIngestKey struct {
	ChannelID uint64    `json:"channel_id,string" gorm:"primaryKey"`
	MessageID string    `json:"message_id" gorm:"primaryKey;size:128"`
	UserID    uint64    `json:"user_id,string" gorm:"not null"`
	ItemID    uint64    `json:"item_id,string" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns c_item_ingest_keys.
func (ItemIngestKey) TableName() string { return "c_item_ingest_keys" }
