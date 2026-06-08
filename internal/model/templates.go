/*
Copyright 2026 Arctel.net

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"text/template"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
)

type Template struct {
	ID          uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	Key         string    `json:"key" gorm:"uniqueIndex;size:80;not null"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Type        string    `json:"type" gorm:"size:20;not null;default:'email'"`
	Subject     string    `json:"subject" gorm:"size:255"`
	Content     string    `json:"content" gorm:"type:text;not null"`
	Description string    `json:"description" gorm:"size:255"`
	IsSystem    bool      `json:"is_system" gorm:"index;not null;default:false"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime;index"`
}

func (t *Template) Normalize() {
	t.Key = strings.TrimSpace(t.Key)
	t.Name = strings.TrimSpace(t.Name)
	t.Type = strings.ToLower(strings.TrimSpace(t.Type))
	t.Subject = strings.TrimSpace(t.Subject)
	t.Content = strings.TrimSpace(t.Content)
	t.Description = strings.TrimSpace(t.Description)
	if t.Type == "" {
		t.Type = "email"
	}
}

func (t *Template) Validate() error {
	t.Normalize()
	if t.Key == "" {
		return errors.New("模板标识符不能为空")
	}
	if t.Name == "" {
		return errors.New("模板名称不能为空")
	}
	if t.Content == "" {
		return errors.New("模板内容不能为空")
	}
	return nil
}

func (t *Template) Render(data any) (string, string, error) {
	// Render Subject
	var subject string
	if t.Subject != "" {
		tmplSubject, err := template.New(t.Key + "_subject").Parse(t.Subject)
		if err != nil {
			return "", "", err
		}
		var subBuf bytes.Buffer
		if err := tmplSubject.Execute(&subBuf, data); err != nil {
			return "", "", err
		}
		subject = subBuf.String()
	}

	// Render Content
	tmplContent, err := template.New(t.Key + "_content").Parse(t.Content)
	if err != nil {
		return "", "", err
	}
	var bodyBuf bytes.Buffer
	if err := tmplContent.Execute(&bodyBuf, data); err != nil {
		return "", "", err
	}

	return subject, bodyBuf.String(), nil
}

// RenderTemplate 渲染模板的高级包装。如果读取或渲染失败，将使用 fallbackSubject 和 fallbackBody 进行解析和返回。
func RenderTemplate(ctx context.Context, key string, data any, fallbackSubject, fallbackBody string) (string, string) {
	var t Template
	if err := db.DB(ctx).Where("key = ?", key).First(&t).Error; err == nil {
		subject, body, err := t.Render(data)
		if err == nil {
			return subject, body
		}
	}

	// 降级使用传入的默认模板内容渲染
	tFallback := Template{
		Key:     key + "_fallback",
		Subject: fallbackSubject,
		Content: fallbackBody,
	}
	subject, body, err := tFallback.Render(data)
	if err == nil {
		return subject, body
	}

	return fallbackSubject, fallbackBody
}
