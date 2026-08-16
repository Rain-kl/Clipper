// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package item

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/upload"
	"github.com/Rain-kl/Wavelet/internal/infra/persistence"
	"github.com/Rain-kl/Wavelet/internal/infra/persistence/idgen"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"gorm.io/gorm"
)

const (
	defaultPageSize           = 20
	maxPageSize               = 100
	defaultPendingArchiveDays = 3
	defaultTrashPurgeDays     = 30
	defaultTimelineWindowDays = 90
)

// CreateItemInput is the payload for creating a capture item.
type CreateItemInput struct {
	Title     string
	Body      string
	UploadIDs []uint64
}

// PatchItemInput is a partial update for title/body and lifecycle transitions.
type PatchItemInput struct {
	Title      *string
	Body       *string
	Lifecycle  *model.ItemLifecycle
	Importance *model.ItemImportance
}

// ListItemsQuery filters and paginates item lists.
type ListItemsQuery struct {
	Page            int
	PageSize        int
	Q               string
	Lifecycle       string
	Importance      string
	ContentType     string
	IncludeArchived bool
	IncludeTrash    bool
	CreatedFrom     *time.Time
	CreatedTo       *time.Time
}

// ListItemsResult is a paginated item list.
type ListItemsResult struct {
	Total   int64     `json:"total"`
	Results []ItemDTO `json:"results"`
}

// TimelineQuery loads review timeline data.
type TimelineQuery struct {
	Days           int
	ExpandArchived bool
	Day            string // optional YYYY-MM-DD for expand
}

// TimelineDay is one calendar day bucket (UTC date string).
type TimelineDay struct {
	Date          string    `json:"date"`
	Items         []ItemDTO `json:"items"`
	ArchivedCount int64     `json:"archived_count"`
	ArchivedItems []ItemDTO `json:"archived_items,omitempty"`
}

// TimelineResult is the review timeline response.
type TimelineResult struct {
	Days []TimelineDay `json:"days"`
}

// ItemStats holds badge counts for navigation.
//
//nolint:revive // keep Item prefix for API/Swagger clarity
type ItemStats struct {
	Pending  int64 `json:"pending"`
	Active   int64 `json:"active"`
	Archived int64 `json:"archived"`
	Trash    int64 `json:"trash"`
	Vault    int64 `json:"vault"`
}

// AttachmentDTO is attachment metadata for API responses.
type AttachmentDTO struct {
	ID       uint64 `json:"id,string"`
	UploadID uint64 `json:"upload_id,string"`
	Sort     int    `json:"sort"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

// ItemDTO is the item view returned by logics.
//
//nolint:revive // keep Item prefix for API/Swagger clarity
type ItemDTO struct {
	ID          uint64                `json:"id,string"`
	UserID      uint64                `json:"user_id,string"`
	ContentType model.ItemContentType `json:"content_type"`
	Title       string                `json:"title"`
	Body        string                `json:"body"`
	Lifecycle   model.ItemLifecycle   `json:"lifecycle"`
	Importance  model.ItemImportance  `json:"importance"`
	Source      string                `json:"source"`
	ArchivedAt  *time.Time            `json:"archived_at,omitempty"`
	TrashedAt   *time.Time            `json:"trashed_at,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	Attachments []AttachmentDTO       `json:"attachments,omitempty"`
}

// CreateItem creates a pending item owned by userID.
func CreateItem(ctx context.Context, userID uint64, input CreateItemInput) (*ItemDTO, error) {
	body := strings.TrimSpace(input.Body)
	title := strings.TrimSpace(input.Title)
	uploadIDs := uniqueUint64(input.UploadIDs)

	if body == "" && len(uploadIDs) == 0 {
		return nil, errors.New(errEmptyContent)
	}

	uploads := make([]model.Upload, 0, len(uploadIDs))
	for _, uid := range uploadIDs {
		u, err := repository.GetActiveUploadByID(ctx, uid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New(errInvalidUpload)
			}
			logger.ErrorF(ctx, "item: load upload failed: upload_id=%d err=%v", uid, err)
			return nil, errors.New(errInternal)
		}
		if u.UserID != userID {
			return nil, errors.New(errInvalidUpload)
		}
		uploads = append(uploads, u)
	}

	contentType := detectContentType(body, uploads)
	now := time.Now()
	item := model.Item{
		ID:          idgen.NextUint64ID(),
		UserID:      userID,
		ContentType: contentType,
		Title:       title,
		Body:        body,
		Lifecycle:   model.ItemLifecyclePending,
		Importance:  model.ItemImportanceNone,
		Source:      model.ItemSourceWeb,
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
		return nil
	}); err != nil {
		logger.ErrorF(ctx, "item: create failed: user_id=%d err=%v", userID, err)
		return nil, errors.New(errInternal)
	}

	return loadItemDTO(ctx, userID, item.ID)
}

// GetItem returns an item owned by userID, or not-found style error.
func GetItem(ctx context.Context, userID, id uint64) (*ItemDTO, error) {
	return loadItemDTO(ctx, userID, id)
}

// ListItems returns a filtered paginated list for userID.
func ListItems(ctx context.Context, userID uint64, query ListItemsQuery) (*ListItemsResult, error) {
	page, pageSize := normalizePage(query.Page, query.PageSize)
	q := db.DB(ctx).Model(&model.Item{}).Where("user_id = ?", userID)

	if query.Lifecycle != "" {
		q = q.Where("lifecycle = ?", query.Lifecycle)
	} else {
		lifecycles := []model.ItemLifecycle{model.ItemLifecyclePending, model.ItemLifecycleActive}
		if query.IncludeArchived {
			lifecycles = append(lifecycles, model.ItemLifecycleArchived)
		}
		if query.IncludeTrash {
			lifecycles = append(lifecycles, model.ItemLifecycleTrash)
		}
		q = q.Where("lifecycle IN ?", lifecycles)
	}
	if query.Importance != "" {
		q = q.Where("importance = ?", query.Importance)
	}
	if query.ContentType != "" {
		q = q.Where("content_type = ?", query.ContentType)
	}
	if kw := strings.TrimSpace(query.Q); kw != "" {
		like := "%" + strings.ToLower(kw) + "%"
		q = q.Where("LOWER(body) LIKE ? OR LOWER(title) LIKE ?", like, like)
	}
	if query.CreatedFrom != nil {
		q = q.Where("created_at >= ?", *query.CreatedFrom)
	}
	if query.CreatedTo != nil {
		q = q.Where("created_at <= ?", *query.CreatedTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		logger.ErrorF(ctx, "item: list count failed: user_id=%d err=%v", userID, err)
		return nil, errors.New(errInternal)
	}

	var rows []model.Item
	offset := (page - 1) * pageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		logger.ErrorF(ctx, "item: list failed: user_id=%d err=%v", userID, err)
		return nil, errors.New(errInternal)
	}

	results := make([]ItemDTO, 0, len(rows))
	for i := range rows {
		dto, err := toItemDTO(ctx, &rows[i], false)
		if err != nil {
			return nil, err
		}
		results = append(results, *dto)
	}
	return &ListItemsResult{Total: total, Results: results}, nil
}

// PatchItem updates fields and enforces lifecycle transitions.
func PatchItem(ctx context.Context, userID, id uint64, input PatchItemInput) (*ItemDTO, error) {
	item, err := loadOwnedItem(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if input.Title != nil {
		updates["title"] = strings.TrimSpace(*input.Title)
	}
	if input.Body != nil {
		updates["body"] = strings.TrimSpace(*input.Body)
	}

	if input.Lifecycle != nil || input.Importance != nil {
		newLifecycle := item.Lifecycle
		newImportance := item.Importance
		if input.Lifecycle != nil {
			newLifecycle = *input.Lifecycle
		}
		if input.Importance != nil {
			newImportance = *input.Importance
		}

		applied, err := applyTransition(item, newLifecycle, newImportance)
		if err != nil {
			return nil, err
		}
		updates["lifecycle"] = applied.Lifecycle
		updates["importance"] = applied.Importance
		updates["archived_at"] = applied.ArchivedAt
		updates["trashed_at"] = applied.TrashedAt
	}

	if len(updates) == 0 {
		return loadItemDTO(ctx, userID, id)
	}

	if err := db.DB(ctx).Model(&model.Item{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error; err != nil {
		logger.ErrorF(ctx, "item: patch failed: item_id=%d err=%v", id, err)
		return nil, errors.New(errInternal)
	}
	return loadItemDTO(ctx, userID, id)
}

// DeleteItem soft-trashes (force=false) or hard-deletes (force=true) an item.
func DeleteItem(ctx context.Context, userID, id uint64, force bool) error {
	if force {
		item, err := loadOwnedItem(ctx, userID, id)
		if err != nil {
			return err
		}
		return hardDeleteItem(ctx, item)
	}
	lifecycle := model.ItemLifecycleTrash
	_, err := PatchItem(ctx, userID, id, PatchItemInput{Lifecycle: &lifecycle})
	return err
}

// Timeline returns non-trash items grouped by UTC calendar day.
func Timeline(ctx context.Context, userID uint64, query TimelineQuery) (*TimelineResult, error) {
	days := query.Days
	if days <= 0 {
		days = defaultTimelineWindowDays
	}
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	var rows []model.Item
	if err := db.DB(ctx).Where("user_id = ? AND lifecycle != ? AND created_at >= ?",
		userID, model.ItemLifecycleTrash, since).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		logger.ErrorF(ctx, "item: timeline failed: user_id=%d err=%v", userID, err)
		return nil, errors.New(errInternal)
	}

	type dayBucket struct {
		items         []ItemDTO
		archived      []ItemDTO
		archivedCount int64
	}
	order := make([]string, 0)
	buckets := map[string]*dayBucket{}

	for i := range rows {
		dto, err := toItemDTO(ctx, &rows[i], true)
		if err != nil {
			return nil, err
		}
		date := rows[i].CreatedAt.UTC().Format("2006-01-02")
		b, ok := buckets[date]
		if !ok {
			b = &dayBucket{}
			buckets[date] = b
			order = append(order, date)
		}
		switch rows[i].Lifecycle {
		case model.ItemLifecycleArchived:
			b.archivedCount++
			if query.ExpandArchived && (query.Day == "" || query.Day == date) {
				b.archived = append(b.archived, *dto)
			}
		default:
			b.items = append(b.items, *dto)
		}
	}

	result := &TimelineResult{Days: make([]TimelineDay, 0, len(order))}
	for _, date := range order {
		b := buckets[date]
		day := TimelineDay{
			Date:          date,
			Items:         b.items,
			ArchivedCount: b.archivedCount,
		}
		if query.ExpandArchived {
			day.ArchivedItems = b.archived
		}
		result.Days = append(result.Days, day)
	}
	return result, nil
}

// Stats returns lifecycle/importance counts for userID.
func Stats(ctx context.Context, userID uint64) (*ItemStats, error) {
	stats := &ItemStats{}
	type row struct {
		Lifecycle  model.ItemLifecycle
		Importance model.ItemImportance
		Cnt        int64
	}
	var rows []row
	if err := db.DB(ctx).Model(&model.Item{}).
		Select("lifecycle, importance, COUNT(*) as cnt").
		Where("user_id = ?", userID).
		Group("lifecycle, importance").
		Scan(&rows).Error; err != nil {
		logger.ErrorF(ctx, "item: stats failed: user_id=%d err=%v", userID, err)
		return nil, errors.New(errInternal)
	}
	for _, r := range rows {
		switch r.Lifecycle {
		case model.ItemLifecyclePending:
			stats.Pending += r.Cnt
		case model.ItemLifecycleActive:
			stats.Active += r.Cnt
		case model.ItemLifecycleArchived:
			stats.Archived += r.Cnt
		case model.ItemLifecycleTrash:
			stats.Trash += r.Cnt
		}
		if r.Importance == model.ItemImportanceVault &&
			(r.Lifecycle == model.ItemLifecyclePending || r.Lifecycle == model.ItemLifecycleActive) {
			stats.Vault += r.Cnt
		}
	}
	return stats, nil
}

// ArchivePendingItems archives pending items older than olderThan.
func ArchivePendingItems(ctx context.Context, olderThan time.Duration) (int, error) {
	if olderThan <= 0 {
		days, err := repository.GetIntByKey(ctx, model.ConfigKeyItemPendingArchiveAfterDays)
		if err != nil || days <= 0 {
			days = defaultPendingArchiveDays
		}
		olderThan = time.Duration(days) * 24 * time.Hour
	}
	cutoff := time.Now().Add(-olderThan)
	now := time.Now()
	res := db.DB(ctx).Model(&model.Item{}).
		Where("lifecycle = ? AND created_at < ?", model.ItemLifecyclePending, cutoff).
		Updates(map[string]any{
			"lifecycle":   model.ItemLifecycleArchived,
			"importance":  model.ItemImportanceFragment,
			"archived_at": now,
		})
	if res.Error != nil {
		logger.ErrorF(ctx, "item: archive pending failed: %v", res.Error)
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

// PurgeTrashItems hard-deletes trash items older than olderThan.
func PurgeTrashItems(ctx context.Context, olderThan time.Duration) (int, error) {
	if olderThan <= 0 {
		days, err := repository.GetIntByKey(ctx, model.ConfigKeyItemTrashPurgeAfterDays)
		if err != nil || days <= 0 {
			days = defaultTrashPurgeDays
		}
		olderThan = time.Duration(days) * 24 * time.Hour
	}
	cutoff := time.Now().Add(-olderThan)
	var items []model.Item
	if err := db.DB(ctx).Where("lifecycle = ? AND trashed_at IS NOT NULL AND trashed_at < ?",
		model.ItemLifecycleTrash, cutoff).Find(&items).Error; err != nil {
		logger.ErrorF(ctx, "item: list trash for purge failed: %v", err)
		return 0, err
	}
	n := 0
	for i := range items {
		if err := hardDeleteItem(ctx, &items[i]); err != nil {
			logger.ErrorF(ctx, "item: purge hard delete failed: item_id=%d err=%v", items[i].ID, err)
			continue
		}
		n++
	}
	return n, nil
}

func hardDeleteItem(ctx context.Context, item *model.Item) error {
	var atts []model.ItemAttachment
	if err := db.DB(ctx).Where("item_id = ?", item.ID).Find(&atts).Error; err != nil {
		return err
	}
	for _, att := range atts {
		if _, err := upload.RemoveOwned(ctx, item.UserID, att.UploadID); err != nil {
			// best-effort: upload may already be gone; still remove attachment/item rows
			logger.WarnF(ctx, "item: RemoveOwned during hard delete: item_id=%d upload_id=%d err=%v", item.ID, att.UploadID, err)
		}
	}
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("item_id = ?", item.ID).Delete(&model.ItemAttachment{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", item.ID, item.UserID).Delete(&model.Item{}).Error
	})
}

func applyTransition(item *model.Item, toLifecycle model.ItemLifecycle, toImportance model.ItemImportance) (*model.Item, error) {
	out := *item
	switch item.Lifecycle {
	case model.ItemLifecyclePending:
		return transitionFromPending(&out, toLifecycle, toImportance)
	case model.ItemLifecycleActive:
		return transitionFromActive(&out, toLifecycle, toImportance)
	case model.ItemLifecycleTrash:
		return transitionFromTrash(&out, toLifecycle)
	case model.ItemLifecycleArchived:
		return transitionFromArchived(&out, toLifecycle)
	default:
		return nil, errors.New(errInvalidTransition)
	}
}

func isClassifiedImportance(v model.ItemImportance) bool {
	return v == model.ItemImportanceFragment ||
		v == model.ItemImportanceNote ||
		v == model.ItemImportanceVault
}

func markTrash(out *model.Item) *model.Item {
	now := time.Now()
	out.Lifecycle = model.ItemLifecycleTrash
	out.TrashedAt = &now
	out.ArchivedAt = nil
	return out
}

func transitionFromPending(out *model.Item, toLifecycle model.ItemLifecycle, toImportance model.ItemImportance) (*model.Item, error) {
	switch toLifecycle {
	case model.ItemLifecycleActive:
		if !isClassifiedImportance(toImportance) {
			return nil, errors.New(errInvalidTransition)
		}
		out.Lifecycle = model.ItemLifecycleActive
		out.Importance = toImportance
		return out, nil
	case model.ItemLifecycleTrash:
		return markTrash(out), nil
	case model.ItemLifecyclePending:
		if toImportance != model.ItemImportanceNone && toImportance != out.Importance {
			return nil, errors.New(errInvalidTransition)
		}
		return out, nil
	default:
		return nil, errors.New(errInvalidTransition)
	}
}

func transitionFromActive(out *model.Item, toLifecycle model.ItemLifecycle, toImportance model.ItemImportance) (*model.Item, error) {
	switch toLifecycle {
	case model.ItemLifecycleActive:
		if !isClassifiedImportance(toImportance) {
			return nil, errors.New(errInvalidTransition)
		}
		out.Importance = toImportance
		return out, nil
	case model.ItemLifecycleTrash:
		return markTrash(out), nil
	case model.ItemLifecycleArchived:
		now := time.Now()
		out.Lifecycle = model.ItemLifecycleArchived
		out.ArchivedAt = &now
		return out, nil
	default:
		return nil, errors.New(errInvalidTransition)
	}
}

func transitionFromTrash(out *model.Item, toLifecycle model.ItemLifecycle) (*model.Item, error) {
	switch toLifecycle {
	case model.ItemLifecycleActive, model.ItemLifecyclePending:
		if out.Importance == model.ItemImportanceNone {
			out.Lifecycle = model.ItemLifecyclePending
		} else {
			out.Lifecycle = model.ItemLifecycleActive
		}
		out.TrashedAt = nil
		return out, nil
	case model.ItemLifecycleTrash:
		return out, nil
	default:
		return nil, errors.New(errInvalidTransition)
	}
}

func transitionFromArchived(out *model.Item, toLifecycle model.ItemLifecycle) (*model.Item, error) {
	switch toLifecycle {
	case model.ItemLifecycleActive:
		out.Lifecycle = model.ItemLifecycleActive
		out.ArchivedAt = nil
		return out, nil
	case model.ItemLifecycleArchived:
		return out, nil
	case model.ItemLifecycleTrash:
		return markTrash(out), nil
	default:
		return nil, errors.New(errInvalidTransition)
	}
}

func loadOwnedItem(ctx context.Context, userID, id uint64) (*model.Item, error) {
	var item model.Item
	err := db.DB(ctx).Where("id = ? AND user_id = ?", id, userID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(errItemNotFound)
		}
		logger.ErrorF(ctx, "item: load failed: item_id=%d err=%v", id, err)
		return nil, errors.New(errInternal)
	}
	return &item, nil
}

func loadItemDTO(ctx context.Context, userID, id uint64) (*ItemDTO, error) {
	item, err := loadOwnedItem(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return toItemDTO(ctx, item, true)
}

func toItemDTO(ctx context.Context, item *model.Item, withAttachments bool) (*ItemDTO, error) {
	dto := &ItemDTO{
		ID:          item.ID,
		UserID:      item.UserID,
		ContentType: item.ContentType,
		Title:       item.Title,
		Body:        item.Body,
		Lifecycle:   item.Lifecycle,
		Importance:  item.Importance,
		Source:      item.Source,
		ArchivedAt:  item.ArchivedAt,
		TrashedAt:   item.TrashedAt,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
	if !withAttachments {
		return dto, nil
	}
	var atts []model.ItemAttachment
	if err := db.DB(ctx).Where("item_id = ?", item.ID).Order("sort ASC").Find(&atts).Error; err != nil {
		logger.ErrorF(ctx, "item: load attachments failed: item_id=%d err=%v", item.ID, err)
		return nil, errors.New(errInternal)
	}
	if len(atts) == 0 {
		dto.Attachments = []AttachmentDTO{}
		return dto, nil
	}
	dto.Attachments = make([]AttachmentDTO, 0, len(atts))
	for _, a := range atts {
		ad := AttachmentDTO{ID: a.ID, UploadID: a.UploadID, Sort: a.Sort}
		if u, err := repository.GetActiveUploadByID(ctx, a.UploadID); err == nil {
			ad.FileName = u.FileName
			ad.MimeType = u.MimeType
			ad.FileSize = u.FileSize
		}
		dto.Attachments = append(dto.Attachments, ad)
	}
	return dto, nil
}

func detectContentType(body string, uploads []model.Upload) model.ItemContentType {
	if body != "" {
		return model.ItemContentTypeText
	}
	if len(uploads) == 0 {
		return model.ItemContentTypeText
	}
	if isImageUpload(uploads[0]) {
		return model.ItemContentTypeImage
	}
	return model.ItemContentTypeFile
}

func isImageUpload(u model.Upload) bool {
	mime := strings.ToLower(u.MimeType)
	if strings.HasPrefix(mime, "image/") {
		return true
	}
	ext := strings.ToLower(u.Extension)
	switch ext {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp", "svg", "heic", "heif", "avif":
		return true
	default:
		return false
	}
}

func uniqueUint64(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}
