// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package item

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/upload"
	"github.com/Rain-kl/Wavelet/internal/infra/persistence"
	"github.com/Rain-kl/Wavelet/internal/infra/persistence/idgen"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/Rain-kl/Wavelet/pkg/message_gateway"
	"gorm.io/gorm"
)

const inboundMergeWindow = 60 * time.Second

// IngestInbound turns a bound gateway message into a clip (create or 60s merge).
func IngestInbound(ctx context.Context, msg message_gateway.InboundMessage) error {
	if msg.BindingUserID == nil {
		return errors.New(errEmptyContent)
	}
	userID := *msg.BindingUserID
	source := strings.TrimSpace(msg.ChannelType)
	if source != model.ItemSourceTelegram && source != model.ItemSourceQQ {
		return errors.New(errEmptyContent)
	}

	uploadIDs, err := ingestInboundAttachments(ctx, userID, msg.Attachments)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" && len(uploadIDs) == 0 {
		return errors.New(errEmptyContent)
	}

	if msg.MessageID != "" {
		var existing model.ItemIngestKey
		err := db.DB(ctx).Where("channel_id = ? AND message_id = ?", msg.ChannelID, msg.MessageID).First(&existing).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.ErrorF(ctx, "item inbound dedup lookup: %v", err)
			return errors.New(errInternal)
		}
	}

	var last model.Item
	err = db.DB(ctx).Where("user_id = ? AND source = ? AND lifecycle = ?", userID, source, model.ItemLifecyclePending).
		Order("updated_at DESC").First(&last).Error
	merge := err == nil && time.Since(last.UpdatedAt) < inboundMergeWindow
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.ErrorF(ctx, "item inbound last clip: %v", err)
		return errors.New(errInternal)
	}

	if merge {
		return appendInbound(ctx, &last, userID, text, uploadIDs, msg)
	}
	return createInboundItem(ctx, userID, source, text, uploadIDs, msg)
}

func ingestInboundAttachments(ctx context.Context, userID uint64, atts []message_gateway.Attachment) ([]uint64, error) {
	var ids []uint64
	for _, att := range atts {
		path := att.Path
		if att.Error != "" || path == "" {
			logger.WarnF(ctx, "item inbound skip attachment: %s", att.Error)
			continue
		}
		data, err := os.ReadFile(path)
		_ = os.Remove(path)
		if err != nil {
			logger.WarnF(ctx, "item inbound read attachment: %v", err)
			continue
		}
		sum := sha256.Sum256(data)
		res, err := upload.Ingest(ctx, upload.IngestRequest{
			UserID:             userID,
			Reader:             bytes.NewReader(data),
			Size:               int64(len(data)),
			FileName:           att.FileName,
			MimeType:           att.MIME,
			Hash:               hex.EncodeToString(sum[:]),
			Type:               "clip",
			Policy:             upload.PolicyResolveExisting,
			SkipExtensionCheck: true,
		})
		if err != nil {
			logger.WarnF(ctx, "item inbound ingest: %v", err)
			continue
		}
		ids = append(ids, res.Upload.ID)
	}
	return ids, nil
}

func appendInbound(ctx context.Context, last *model.Item, userID uint64, text string, uploadIDs []uint64, msg message_gateway.InboundMessage) error {
	uploads, err := loadOwnedUploads(ctx, userID, uploadIDs)
	if err != nil {
		return err
	}
	var existingAtts []model.ItemAttachment
	if err := db.DB(ctx).Where("item_id = ?", last.ID).Order("sort ASC").Find(&existingAtts).Error; err != nil {
		return errors.New(errInternal)
	}
	existingUploads := make([]model.Upload, 0, len(existingAtts))
	for _, att := range existingAtts {
		u, err := repository.GetActiveUploadByID(ctx, att.UploadID)
		if err == nil {
			existingUploads = append(existingUploads, u)
		}
	}
	body := last.Body
	if text != "" {
		if body != "" {
			body = body + "\n" + text
		} else {
			body = text
		}
	}
	allUploads := append(existingUploads, uploads...)
	last.Body = body
	last.ContentType = detectContentType(body, allUploads)
	now := time.Now()
	nextSort := 0
	if len(existingAtts) > 0 {
		nextSort = existingAtts[len(existingAtts)-1].Sort + 1
	}

	if err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(last).Error; err != nil {
			return err
		}
		for i, u := range uploads {
			att := model.ItemAttachment{
				ID:        idgen.NextUint64ID(),
				ItemID:    last.ID,
				UploadID:  u.ID,
				Sort:      nextSort + i,
				CreatedAt: now,
			}
			if err := tx.Create(&att).Error; err != nil {
				return err
			}
			if u.Status == model.UploadStatusPending {
				if err := tx.Model(&model.Upload{}).Where("id = ?", u.ID).
					Update("status", model.UploadStatusUsed).Error; err != nil {
					return err
				}
			}
		}
		return insertIngestKey(tx, msg, userID, last.ID)
	}); err != nil {
		logger.ErrorF(ctx, "item inbound merge failed: user_id=%d err=%v", userID, err)
		return errors.New(errInternal)
	}
	return nil
}

func createInboundItem(ctx context.Context, userID uint64, source, text string, uploadIDs []uint64, msg message_gateway.InboundMessage) error {
	uploads, err := loadOwnedUploads(ctx, userID, uploadIDs)
	if err != nil {
		return err
	}
	now := time.Now()
	item := model.Item{
		ID:          idgen.NextUint64ID(),
		UserID:      userID,
		ContentType: detectContentType(text, uploads),
		Title:       "",
		Body:        text,
		Lifecycle:   model.ItemLifecyclePending,
		Importance:  model.ItemImportanceNone,
		Source:      source,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		for i, u := range uploads {
			att := model.ItemAttachment{
				ID:        idgen.NextUint64ID(),
				ItemID:    item.ID,
				UploadID:  u.ID,
				Sort:      i,
				CreatedAt: now,
			}
			if err := tx.Create(&att).Error; err != nil {
				return err
			}
			if u.Status == model.UploadStatusPending {
				if err := tx.Model(&model.Upload{}).Where("id = ?", u.ID).
					Update("status", model.UploadStatusUsed).Error; err != nil {
					return err
				}
			}
		}
		return insertIngestKey(tx, msg, userID, item.ID)
	}); err != nil {
		logger.ErrorF(ctx, "item inbound create failed: user_id=%d err=%v", userID, err)
		return errors.New(errInternal)
	}
	return nil
}

func insertIngestKey(tx *gorm.DB, msg message_gateway.InboundMessage, userID, itemID uint64) error {
	if msg.MessageID == "" {
		return nil
	}
	return tx.Create(&model.ItemIngestKey{
		ChannelID: msg.ChannelID,
		MessageID: msg.MessageID,
		UserID:    userID,
		ItemID:    itemID,
	}).Error
}

func loadOwnedUploads(ctx context.Context, userID uint64, uploadIDs []uint64) ([]model.Upload, error) {
	ids := uniqueUint64(uploadIDs)
	uploads := make([]model.Upload, 0, len(ids))
	for _, uid := range ids {
		u, err := repository.GetActiveUploadByID(ctx, uid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New(errInvalidUpload)
			}
			logger.ErrorF(ctx, "item inbound load upload: upload_id=%d err=%v", uid, err)
			return nil, errors.New(errInternal)
		}
		if u.UserID != userID {
			return nil, errors.New(errInvalidUpload)
		}
		uploads = append(uploads, u)
	}
	return uploads, nil
}
