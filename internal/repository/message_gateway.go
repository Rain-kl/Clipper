// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package repository

import (
	"context"
	"errors"
	"time"

	db "github.com/Rain-kl/Wavelet/internal/infra/persistence"
	"github.com/Rain-kl/Wavelet/internal/infra/persistence/idgen"
	"github.com/Rain-kl/Wavelet/internal/model"
	"gorm.io/gorm"
)

// CreateMessageChannel inserts a channel row.
func CreateMessageChannel(ctx context.Context, ch *model.MessageChannel) error {
	if ch.ID == 0 {
		ch.ID = idgen.NextUint64ID()
	}
	return db.DB(ctx).Create(ch).Error
}

// UpdateMessageChannel saves a channel row.
func UpdateMessageChannel(ctx context.Context, ch *model.MessageChannel) error {
	return db.DB(ctx).Save(ch).Error
}

// GetMessageChannel loads a channel by id.
func GetMessageChannel(ctx context.Context, id uint64) (*model.MessageChannel, error) {
	var ch model.MessageChannel
	if err := db.DB(ctx).Where("id = ?", id).First(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

// ListMessageChannels returns all channels newest first.
func ListMessageChannels(ctx context.Context) ([]model.MessageChannel, error) {
	var rows []model.MessageChannel
	if err := db.DB(ctx).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// DeleteMessageChannel removes pairings, bindings, then the channel.
func DeleteMessageChannel(ctx context.Context, id uint64) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_id = ?", id).Delete(&model.MessagePairingCode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("channel_id = ?", id).Delete(&model.MessageBinding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.MessageChannel{}, id).Error
	})
}

// CreateMessageBinding inserts a binding.
func CreateMessageBinding(ctx context.Context, b *model.MessageBinding) error {
	if b.ID == 0 {
		b.ID = idgen.NextUint64ID()
	}
	return db.DB(ctx).Create(b).Error
}

// GetBindingByChannelPlatform finds a binding for a platform user on a channel.
func GetBindingByChannelPlatform(ctx context.Context, channelID uint64, platformUserID string) (*model.MessageBinding, error) {
	var b model.MessageBinding
	err := db.DB(ctx).Where("channel_id = ? AND platform_user_id = ?", channelID, platformUserID).First(&b).Error
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// ListBindingsByUser lists bindings for a Wavelet user.
func ListBindingsByUser(ctx context.Context, userID uint64) ([]model.MessageBinding, error) {
	var rows []model.MessageBinding
	if err := db.DB(ctx).Where("user_id = ?", userID).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetMessageBinding loads a binding by id.
func GetMessageBinding(ctx context.Context, id uint64) (*model.MessageBinding, error) {
	var b model.MessageBinding
	if err := db.DB(ctx).Where("id = ?", id).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// DeleteMessageBinding deletes a binding by id.
func DeleteMessageBinding(ctx context.Context, id uint64) error {
	return db.DB(ctx).Delete(&model.MessageBinding{}, id).Error
}

// UpsertPairingCode reuses an unexpired code for the same channel+platform user.
func UpsertPairingCode(ctx context.Context, channelID uint64, platformUserID, code string, expiresAt time.Time) (*model.MessagePairingCode, error) {
	var existing model.MessagePairingCode
	err := db.DB(ctx).
		Where("channel_id = ? AND platform_user_id = ? AND expires_at > ?", channelID, platformUserID, time.Now()).
		First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	row := &model.MessagePairingCode{
		Code:           code,
		ChannelID:      channelID,
		PlatformUserID: platformUserID,
		ExpiresAt:      expiresAt,
	}
	if err := db.DB(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// GetPairingCode loads a pairing code by normalized code string.
func GetPairingCode(ctx context.Context, code string) (*model.MessagePairingCode, error) {
	var row model.MessagePairingCode
	if err := db.DB(ctx).Where("code = ?", code).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// DeletePairingCode removes a pairing code.
func DeletePairingCode(ctx context.Context, code string) error {
	return db.DB(ctx).Where("code = ?", code).Delete(&model.MessagePairingCode{}).Error
}

// DeleteExpiredPairingCodes removes expired pairing rows.
func DeleteExpiredPairingCodes(ctx context.Context) error {
	return db.DB(ctx).Where("expires_at <= ?", time.Now()).Delete(&model.MessagePairingCode{}).Error
}

// ListEnabledMessageChannels returns enabled channels.
func ListEnabledMessageChannels(ctx context.Context) ([]model.MessageChannel, error) {
	var rows []model.MessageChannel
	if err := db.DB(ctx).Where("enabled = ?", true).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
