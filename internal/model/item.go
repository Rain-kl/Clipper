// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

// ItemContentType is the capture media kind for an item.
type ItemContentType string

// Item content type values.
const (
	ItemContentTypeText  ItemContentType = "text"
	ItemContentTypeImage ItemContentType = "image"
	ItemContentTypeFile  ItemContentType = "file"
)

// ItemLifecycle is the retention pipeline stage for an item.
type ItemLifecycle string

// Item lifecycle values.
const (
	ItemLifecyclePending  ItemLifecycle = "pending"
	ItemLifecycleActive   ItemLifecycle = "active"
	ItemLifecycleArchived ItemLifecycle = "archived"
	ItemLifecycleTrash    ItemLifecycle = "trash"
)

// ItemImportance is the classification tier after review.
type ItemImportance string

// Item importance values.
const (
	ItemImportanceNone     ItemImportance = "none"
	ItemImportanceFragment ItemImportance = "fragment"
	ItemImportanceNote     ItemImportance = "note"
	ItemImportanceVault    ItemImportance = "vault"
)

// ItemSourceWeb is the default capture source for web UI.
const ItemSourceWeb = "web"

// ItemSourceTelegram is a clip captured from a bound Telegram DM.
const ItemSourceTelegram = "telegram"

// ItemSourceQQ is a clip captured from a bound QQ C2C message.
const ItemSourceQQ = "qq"

// Item is one capture unit owned by a user.
type Item struct {
	ID          uint64          `json:"id,string" gorm:"primaryKey"`
	UserID      uint64          `json:"user_id,string" gorm:"index;not null"`
	ContentType ItemContentType `json:"content_type" gorm:"size:16;not null;index"`
	Title       string          `json:"title" gorm:"size:255"`
	Body        string          `json:"body" gorm:"type:text"`
	Lifecycle   ItemLifecycle   `json:"lifecycle" gorm:"size:16;not null;index"`
	Importance  ItemImportance  `json:"importance" gorm:"size:16;not null;index"`
	Source      string          `json:"source" gorm:"size:32;not null;default:web"`
	ArchivedAt  *time.Time      `json:"archived_at"`
	TrashedAt   *time.Time      `json:"trashed_at"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the items table name.
func (Item) TableName() string { return "c_items" }
