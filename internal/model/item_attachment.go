// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

// ItemAttachment links an item to an upload record.
type ItemAttachment struct {
	ID        uint64    `json:"id,string" gorm:"primaryKey"`
	ItemID    uint64    `json:"item_id,string" gorm:"index;not null"`
	UploadID  uint64    `json:"upload_id,string" gorm:"index;not null"`
	Sort      int       `json:"sort" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns the item attachments table name.
func (ItemAttachment) TableName() string { return "c_item_attachments" }
