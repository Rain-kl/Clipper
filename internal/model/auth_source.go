package model

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/linux-do/credit/internal/db"
	"gorm.io/gorm"
)

const (
	AuthSourceTypeOIDC = "oidc"
)

var authSourceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)

type AuthSource struct {
	ID                     uint64    `json:"id" gorm:"primaryKey"`
	Name                   string    `json:"name" gorm:"uniqueIndex;size:80;not null"`
	Type                   string    `json:"type" gorm:"size:20;not null"`
	DisplayName            string    `json:"display_name" gorm:"size:100"`
	IsActive               bool      `json:"is_active" gorm:"index;not null;default:false"`
	ClientID               string    `json:"client_id" gorm:"size:255"`
	ClientSecret           string    `json:"-" gorm:"size:1024"`
	OpenIDDiscoveryURL     string    `json:"openid_discovery_url" gorm:"column:openid_discovery_url;size:1024"`
	Scopes                 string    `json:"scopes" gorm:"size:255"`
	IconURL                string    `json:"icon_url" gorm:"size:1024"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	ClientSecretConfigured bool      `json:"client_secret_configured" gorm:"-"`
}

type ExternalAccount struct {
	ID               uint64    `json:"id" gorm:"primaryKey"`
	AuthSourceID     uint64    `json:"auth_source_id" gorm:"uniqueIndex:idx_external_accounts_source_external,priority:1;index"`
	UserID           uint64    `json:"user_id" gorm:"index;not null"`
	ExternalID       string    `json:"external_id" gorm:"uniqueIndex:idx_external_accounts_source_external,priority:2;size:255;not null"`
	ExternalUsername string    `json:"external_username" gorm:"size:255"`
	Email            string    `json:"email" gorm:"size:255"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ExternalAccountView struct {
	ID               uint64    `json:"id"`
	AuthSourceID     uint64    `json:"auth_source_id"`
	AuthSourceName   string    `json:"auth_source_name"`
	AuthSourceType   string    `json:"auth_source_type"`
	AuthSourceLabel  string    `json:"auth_source_label"`
	ExternalUsername string    `json:"external_username"`
	Email            string    `json:"email"`
	CreatedAt        time.Time `json:"created_at"`
}

func (source *AuthSource) Normalize() {
	source.Type = strings.ToLower(strings.TrimSpace(source.Type))
	source.Name = strings.TrimSpace(source.Name)
	source.DisplayName = strings.TrimSpace(source.DisplayName)
	source.ClientID = strings.TrimSpace(source.ClientID)
	source.ClientSecret = strings.TrimSpace(source.ClientSecret)
	source.OpenIDDiscoveryURL = strings.TrimSpace(source.OpenIDDiscoveryURL)
	source.Scopes = strings.TrimSpace(source.Scopes)
	source.IconURL = strings.TrimSpace(source.IconURL)
	if source.DisplayName == "" {
		source.DisplayName = source.Name
	}
	if source.Type == AuthSourceTypeOIDC && source.Scopes == "" {
		source.Scopes = "openid profile email"
	}
}

func (source *AuthSource) Validate() error {
	source.Normalize()
	if source.Name == "" {
		return errors.New("认证源名称不能为空")
	}
	if !authSourceNamePattern.MatchString(source.Name) {
		return errors.New("认证源名称只能包含字母、数字、短横线或下划线，且必须以字母或数字开头")
	}
	if source.Type != AuthSourceTypeOIDC {
		return errors.New("认证源类型仅支持 oidc")
	}
	if source.OpenIDDiscoveryURL == "" {
		return errors.New("OIDC 认证源必须配置 Discovery URL")
	}
	if source.IsActive && (source.ClientID == "" || source.ClientSecret == "") {
		return errors.New("启用认证源前必须配置 Client ID 和 Client Secret")
	}
	return nil
}

func (source *AuthSource) Sanitize() {
	source.ClientSecretConfigured = source.ClientSecret != ""
	source.ClientSecret = ""
}

func GetAuthSources() ([]AuthSource, error) {
	var sources []AuthSource
	if err := db.DB(context.Background()).Order("id asc").Find(&sources).Error; err != nil {
		return nil, err
	}
	for i := range sources {
		sources[i].Sanitize()
	}
	return sources, nil
}

func GetActiveAuthSources() ([]AuthSource, error) {
	var sources []AuthSource
	if err := db.DB(context.Background()).Where("is_active = ?", true).Order("id asc").Find(&sources).Error; err != nil {
		return nil, err
	}
	for i := range sources {
		sources[i].Sanitize()
	}
	return sources, nil
}

func GetAuthSourceByID(id uint64) (*AuthSource, error) {
	if id == 0 {
		return nil, errors.New("认证源 ID 不能为空")
	}
	var source AuthSource
	if err := db.DB(context.Background()).First(&source, "id = ?", id).Error; err != nil {
		return nil, err
	}
	source.ClientSecretConfigured = source.ClientSecret != ""
	return &source, nil
}

func GetAuthSourceByName(name string) (*AuthSource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("认证源名称不能为空")
	}
	var source AuthSource
	if err := db.DB(context.Background()).First(&source, "name = ?", name).Error; err != nil {
		return nil, err
	}
	source.ClientSecretConfigured = source.ClientSecret != ""
	return &source, nil
}

func CreateAuthSource(source *AuthSource) error {
	if err := source.Validate(); err != nil {
		return err
	}
	return db.DB(context.Background()).Create(source).Error
}

func UpdateAuthSource(source *AuthSource, keepSecret bool) error {
	if source.ID == 0 {
		return errors.New("认证源 ID 不能为空")
	}
	var current AuthSource
	if err := db.DB(context.Background()).First(&current, "id = ?", source.ID).Error; err != nil {
		return err
	}
	if keepSecret {
		source.ClientSecret = current.ClientSecret
	}
	if err := source.Validate(); err != nil {
		return err
	}
	return db.DB(context.Background()).Model(&current).Updates(map[string]any{
		"name":                 source.Name,
		"type":                 source.Type,
		"display_name":         source.DisplayName,
		"is_active":            source.IsActive,
		"client_id":            source.ClientID,
		"client_secret":        source.ClientSecret,
		"openid_discovery_url": source.OpenIDDiscoveryURL,
		"scopes":               source.Scopes,
		"icon_url":             source.IconURL,
	}).Error
}

func ToggleAuthSource(id uint64, isActive bool) error {
	source, err := GetAuthSourceByID(id)
	if err != nil {
		return err
	}
	source.IsActive = isActive
	if err := source.Validate(); err != nil {
		return err
	}
	return db.DB(context.Background()).Model(&AuthSource{}).Where("id = ?", id).Update("is_active", isActive).Error
}

func DeleteAuthSource(id uint64) error {
	if id == 0 {
		return errors.New("认证源 ID 不能为空")
	}
	return db.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("auth_source_id = ?", id).Delete(&ExternalAccount{}).Error; err != nil {
			return err
		}
		return tx.Delete(&AuthSource{}, "id = ?", id).Error
	})
}

func FindExternalAccount(sourceID uint64, externalID string) (*ExternalAccount, error) {
	var account ExternalAccount
	if err := db.DB(context.Background()).Where("auth_source_id = ? AND external_id = ?", sourceID, externalID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func BindExternalAccount(account *ExternalAccount) error {
	if account.UserID == 0 || strings.TrimSpace(account.ExternalID) == "" {
		return errors.New("外部账号绑定信息不完整")
	}
	account.ExternalID = strings.TrimSpace(account.ExternalID)
	account.ExternalUsername = strings.TrimSpace(account.ExternalUsername)
	account.Email = strings.TrimSpace(account.Email)

	return db.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		var current ExternalAccount
		err := tx.Where("auth_source_id = ? AND external_id = ?", account.AuthSourceID, account.ExternalID).First(&current).Error
		if err == nil {
			if current.UserID != account.UserID {
				return errors.New("该外部账号已绑定到其他用户")
			}
			return tx.Model(&current).Updates(map[string]any{
				"external_username": account.ExternalUsername,
				"email":             account.Email,
			}).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(account).Error
	})
}

func ListExternalAccountsByUserID(userID uint64) ([]ExternalAccountView, error) {
	if userID == 0 {
		return nil, errors.New("用户 ID 不能为空")
	}
	var accounts []ExternalAccount
	if err := db.DB(context.Background()).Where("user_id = ?", userID).Order("id asc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	views := make([]ExternalAccountView, 0, len(accounts))
	for _, account := range accounts {
		var name, sourceType, label string
		if account.AuthSourceID == 0 {
			name = "default"
			sourceType = "oidc"
			label = "历史认证源"
		} else {
			source, err := GetAuthSourceByID(account.AuthSourceID)
			if err != nil {
				continue
			}
			name = source.Name
			sourceType = source.Type
			label = source.DisplayName
			if label == "" {
				label = source.Name
			}
		}
		views = append(views, ExternalAccountView{
			ID:               account.ID,
			AuthSourceID:     account.AuthSourceID,
			AuthSourceName:   name,
			AuthSourceType:   sourceType,
			AuthSourceLabel:  label,
			ExternalUsername: account.ExternalUsername,
			Email:            account.Email,
			CreatedAt:        account.CreatedAt,
		})
	}
	return views, nil
}

func DeleteExternalAccountForUser(id uint64, userID uint64) error {
	if id == 0 || userID == 0 {
		return errors.New("绑定记录 ID 不能为空")
	}
	return db.DB(context.Background()).Where("id = ? AND user_id = ?", id, userID).Delete(&ExternalAccount{}).Error
}
