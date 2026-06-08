/*
Copyright 2025 linux.do

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

package oauth

import (
	"context"
	"log"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/linux-do/credit/internal/config"
	"golang.org/x/oauth2"
)

var (
	oauthConf    *oauth2.Config
	oidcVerifier *oidc.IDTokenVerifier
)

func init() {
	cfg := config.Config.OAuth2

	if cfg.Issuer != "" {
		ctx := context.Background()
		provider, err := oidc.NewProvider(ctx, cfg.Issuer)
		if err != nil {
			log.Printf("[OAuth] 初始化 OIDC Provider 失败: %v，将仅使用 OAuth2", err)
		} else {
			oidcVerifier = provider.Verifier(&oidc.Config{
				ClientID: cfg.ClientID,
			})
			log.Printf("[OAuth] OIDC Provider 初始化成功: %s", cfg.Issuer)
		}
	}

	// 初始化 OAuth2 配置
	scopes := []string{"profile", "email"}
	if oidcVerifier != nil {
		// 启用 OIDC 时添加 openid scope
		scopes = append([]string{oidc.ScopeOpenID}, scopes...)
	}

	oauthConf = &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   cfg.AuthorizationEndpoint,
			TokenURL:  cfg.TokenEndpoint,
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
	}
}
