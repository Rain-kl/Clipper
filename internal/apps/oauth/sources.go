package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"strconv"

	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// AuthSourceView 登录源展示信息
type AuthSourceView struct {
	ID                     uint64 `json:"id"`
	Name                   string `json:"name"`
	Type                   string `json:"type"`
	DisplayName            string `json:"display_name"`
	IsActive               bool   `json:"is_active"`
	IconURL                string `json:"icon_url"`
	ClientSecretConfigured bool   `json:"client_secret_configured"`
}

// OAuthAuthorizeResponse 授权 URL 响应
type OAuthAuthorizeResponse struct {
	AuthorizeURL string `json:"authorize_url"`
}

// OAuthCallbackResult 回调处理结果
type OAuthCallbackResult struct {
	Status string         `json:"status"`
	User   *BasicUserInfo `json:"user,omitempty"`
}

// CallbackRequest OAuth 回调请求参数
type CallbackRequest struct {
	State string `json:"state" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

func GetUserIDFromSession(s sessions.Session) uint64 {
	userID, ok := s.Get(UserIDKey).(uint64)
	if !ok {
		return 0
	}
	return userID
}

func GetUserIDFromContext(c *gin.Context) uint64 {
	session := sessions.Default(c)
	return GetUserIDFromSession(session)
}

func resolveAuthSource(sourceName string) (*model.AuthSource, error) {
	name := strings.TrimSpace(strings.ToLower(sourceName))
	if name == "" {
		sources, err := model.GetActiveAuthSources()
		if err != nil {
			return nil, err
		}
		if len(sources) == 0 {
			return nil, errors.New("未配置可用认证源")
		}
		return &sources[0], nil
	}
	return model.GetAuthSourceByName(name)
}

func activeLoginSources() []AuthSourceView {
	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyOIDCLoginEnabled)
	if err == nil && !enabled {
		return nil
	}

	dbSources, err := model.GetActiveAuthSources()
	if err != nil {
		return nil
	}
	sources := make([]AuthSourceView, 0, len(dbSources))
	for _, source := range dbSources {
		sources = append(sources, AuthSourceView{
			ID:                     source.ID,
			Name:                   source.Name,
			Type:                   source.Type,
			DisplayName:            source.DisplayName,
			IsActive:               source.IsActive,
			IconURL:                source.IconURL,
			ClientSecretConfigured: source.ClientSecretConfigured,
		})
	}
	return sources
}

func getFrontendLoginRedirectURL(ctx context.Context) (string, error) {
	var sc model.SystemConfig
	if err := sc.GetByKey(ctx, model.ConfigKeyServerAddress); err != nil || strings.TrimSpace(sc.Value) == "" {
		return "", errors.New("服务器地址 (server_address) 未配置或配置为空，请在后台系统设置中配置后再试")
	}
	return strings.TrimRight(sc.Value, "/") + "/login", nil
}

func buildOAuthConfig(ctx context.Context, source *model.AuthSource, redirectURL string) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	if source == nil {
		return nil, nil, errors.New("认证源不能为空")
	}

	if source.OpenIDDiscoveryURL == "" {
		return nil, nil, errors.New("OIDC 认证源必须配置 Discovery URL")
	}

	// Clean the issuer URL (trim /.well-known/openid-configuration if configured by mistake)
	issuer := strings.TrimSuffix(strings.TrimSpace(source.OpenIDDiscoveryURL), "/")
	issuer = strings.TrimSuffix(issuer, "/.well-known/openid-configuration")
	issuer = strings.TrimSuffix(issuer, "/.well-known/oauth-authorization-server")

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: source.ClientID})
	scopes := strings.Fields(source.Scopes)
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	if !containsScope(scopes, oidc.ScopeOpenID) {
		scopes = append([]string{oidc.ScopeOpenID}, scopes...)
	}

	return &oauth2.Config{
		ClientID:     source.ClientID,
		ClientSecret: source.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint:     provider.Endpoint(),
	}, verifier, nil
}

func containsScope(scopes []string, scope string) bool {
	for _, item := range scopes {
		if item == scope {
			return true
		}
	}
	return false
}

func setLoginSession(c *gin.Context, user *model.User) error {
	session := sessions.Default(c)
	session.Set(UserIDKey, user.ID)
	session.Set(UserNameKey, user.Username)
	return session.Save()
}

func uniqueUsername(ctx context.Context, base string) (string, error) {
	candidate := strings.TrimSpace(base)
	if candidate == "" {
		candidate = "user"
	}
	for i := 0; i < 1000; i++ {
		var count int64
		if err := db.DB(ctx).Model(&model.User{}).Where("username = ?", candidate).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i+1)
	}
	return "", errors.New("无法生成可用用户名")
}

func buildOAuthUserInfo(ctx context.Context, source *model.AuthSource, code string, nonce string, redirectURL string) (*model.OAuthUserInfo, error) {
	authConfig, verifier, err := buildOAuthConfig(ctx, source, redirectURL)
	if err != nil {
		return nil, err
	}

	token, err := authConfig.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	userInfo := &model.OAuthUserInfo{Active: true}
	if verifier != nil {
		if rawIDToken, ok := token.Extra("id_token").(string); ok {
			idToken, verifyErr := verifier.Verify(ctx, rawIDToken)
			if verifyErr != nil {
				return nil, fmt.Errorf("%s: %w", IDTokenVerifyFailed, verifyErr)
			}
			if nonce != "" && idToken.Nonce != nonce {
				return nil, errors.New(NonceMismatch)
			}
			if claimsErr := idToken.Claims(userInfo); claimsErr != nil {
				return nil, claimsErr
			}
		}
	}

	if userInfo.Username == "" && userInfo.PreferredUsername != "" {
		userInfo.Username = userInfo.PreferredUsername
	}
	if userInfo.Username == "" && userInfo.Email != "" {
		userInfo.Username = strings.Split(userInfo.Email, "@")[0]
	}
	if userInfo.Username == "" && userInfo.Sub != "" {
		userInfo.Username = userInfo.Sub
	}
	if userInfo.Name == "" {
		userInfo.Name = userInfo.Username
	}

	return userInfo, nil
}

func normalizeOAuthUserInfo(userInfo *model.OAuthUserInfo) error {
	userInfo.Username = strings.TrimSpace(userInfo.Username)
	userInfo.PreferredUsername = strings.TrimSpace(userInfo.PreferredUsername)
	userInfo.Email = strings.TrimSpace(userInfo.Email)
	userInfo.Name = strings.TrimSpace(userInfo.Name)
	userInfo.AvatarUrl = strings.TrimSpace(userInfo.AvatarUrl)

	if userInfo.Username == "" && userInfo.PreferredUsername != "" {
		userInfo.Username = userInfo.PreferredUsername
	}
	if userInfo.Username == "" && userInfo.Email != "" {
		userInfo.Username = strings.Split(userInfo.Email, "@")[0]
	}
	if userInfo.Username == "" && userInfo.Sub != "" {
		userInfo.Username = userInfo.Sub
	}
	if userInfo.Username == "" {
		return errors.New("无法从认证源获取用户名")
	}
	if userInfo.Name == "" {
		userInfo.Name = userInfo.Username
	}
	if !userInfo.Active {
		userInfo.Active = true
	}
	return nil
}

func buildCallbackResult(user *model.User, status string) OAuthCallbackResult {
	result := OAuthCallbackResult{Status: status}
	if user != nil {
		info := BuildBasicUserInfo(user, false)
		result.User = &info
	}
	return result
}

// GetLoginSources 获取可用登录源列表
// @Summary 获取可用登录源
// @Description 返回当前系统已启用的所有 OAuth 登录源，前端展示登录按钮列表时调用
// @Tags oauth
// @Produce json
// @Success 200 {object} util.ResponseAny{data=[]oauth.AuthSourceView} "登录源列表"
// @Router /api/v1/oauth/sources [get]
func GetLoginSources(c *gin.Context) {
	c.JSON(http.StatusOK, util.OK(activeLoginSources()))
}

// GetLoginURL 获取登录授权地址
// @Summary 获取登录授权地址
// @Description 根据指定认证源生成 OAuth 授权 URL，前端跳转到该 URL 完成 OAuth 登录授权。source 参数为空时使用第一个启用的认证源。
// @Tags oauth
// @Produce json
// @Param source query string false "认证源名称，为空使用第一个启用的认证源"
// @Success 200 {object} util.ResponseAny{data=oauth.OAuthAuthorizeResponse} "授权 URL"
// @Failure 400 {object} util.ResponseAny "认证源不存在或未配置"
// @Failure 500 {object} util.ResponseAny "Redis 异常或构造 URL 失败"
// @Router /api/v1/oauth/login [get]
func GetLoginURL(c *gin.Context) {
	source, err := resolveAuthSource(c.Query("source"))
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	state := uuid.NewString()
	payloadValue, err := encodeOAuthStatePayload(oauthStatePayload{
		SourceName: source.Name,
		Purpose:    OAuthPurposeLogin,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	if err := db.Redis.Set(c.Request.Context(), db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state)), payloadValue, OAuthStateCacheKeyExpiration).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	authorizeURL, err := buildAuthorizeURL(c.Request.Context(), source, state)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OK(OAuthAuthorizeResponse{AuthorizeURL: authorizeURL}))
}

func buildAuthorizeURL(ctx context.Context, source *model.AuthSource, state string) (string, error) {
	redirectURL, err := getFrontendLoginRedirectURL(ctx)
	if err != nil {
		return "", err
	}
	authConfig, verifier, err := buildOAuthConfig(ctx, source, redirectURL)
	if err != nil {
		return "", err
	}
	if verifier != nil {
		return authConfig.AuthCodeURL(state, oidc.Nonce(state)), nil
	}
	return authConfig.AuthCodeURL(state), nil
}

// Authorize 发起指定认证源授权
// @Summary 发起指定认证源授权
// @Description 根据指定认证源名称发起 OAuth 授权，支持 purpose 参数用于区分登录和账号绑定场景。认证源必须已启用。
// @Tags oauth
// @Produce json
// @Param source path string true "认证源名称"
// @Param purpose query string false "授权目的：login（登录）或 bind（绑定账号），默认 login"
// @Success 200 {object} util.ResponseAny{data=oauth.OAuthAuthorizeResponse} "授权 URL"
// @Failure 400 {object} util.ResponseAny "认证源不存在或未启用"
// @Failure 500 {object} util.ResponseAny "Redis 异常或构造 URL 失败"
// @Router /api/v1/oauth/{source}/authorize [get]
func Authorize(c *gin.Context) {
	source, err := resolveAuthSource(c.Param("source"))
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	if !source.IsActive {
		c.JSON(http.StatusBadRequest, util.Err("认证源未启用"))
		return
	}
	purpose := strings.ToLower(strings.TrimSpace(c.Query("purpose")))
	if purpose != OAuthPurposeBind {
		purpose = OAuthPurposeLogin
	}
	state := uuid.NewString()
	payloadValue, err := encodeOAuthStatePayload(oauthStatePayload{
		SourceName: source.Name,
		Purpose:    purpose,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	if err := db.Redis.Set(c.Request.Context(), db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state)), payloadValue, OAuthStateCacheKeyExpiration).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	authorizeURL, err := buildAuthorizeURL(c.Request.Context(), source, state)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OK(OAuthAuthorizeResponse{AuthorizeURL: authorizeURL}))
}

// Callback OAuth 回调处理
// @Summary OAuth 回调处理
// @Description 接收前端传回的 state 和 code，完成 OAuth/OIDC 认证并建立会话。支持登录（login）和账号绑定（bind）两种场景。
// @Tags oauth
// @Accept json
// @Produce json
// @Param request body oauth.CallbackRequest true "回调请求参数"
// @Success 200 {object} util.ResponseAny{data=oauth.OAuthCallbackResult} "登录或绑定成功"
// @Failure 400 {object} util.ResponseAny "state 无效、参数错误或认证源错误"
// @Failure 401 {object} util.ResponseAny "绑定场景未登录"
// @Failure 500 {object} util.ResponseAny "OAuth 认证失败或内部错误"
// @Router /api/v1/oauth/callback [post]
func Callback(c *gin.Context) {
	var req CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	ctx := c.Request.Context()
	stateKey := db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, req.State))
	payloadRaw, err := db.Redis.Get(ctx, stateKey).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(InvalidState))
		return
	}
	_ = db.Redis.Del(ctx, stateKey)

	payload, err := decodeOAuthStatePayload(payloadRaw)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	source, err := resolveAuthSource(payload.SourceName)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	redirectURL, err := getFrontendLoginRedirectURL(ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	userInfo, err := buildOAuthUserInfo(ctx, source, req.Code, req.State, redirectURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	if err := normalizeOAuthUserInfo(userInfo); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	if userInfo.Sub == "" {
		userInfo.Sub = userInfo.Username
	}

	var user model.User
	if payload.Purpose == OAuthPurposeBind {
		userID := GetUserIDFromContext(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, util.Err(common.UnAuthorized))
			return
		}
		if err := db.DB(ctx).First(&user, "id = ?", userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
			return
		}
		if err := model.BindExternalAccount(&model.ExternalAccount{
			AuthSourceID:     source.ID,
			UserID:           user.ID,
			ExternalID:       userInfo.Sub,
			ExternalUsername: userInfo.Username,
			Email:            userInfo.Email,
		}); err != nil {
			c.JSON(http.StatusBadRequest, util.Err(err.Error()))
			return
		}
		user.LastLoginAt = time.Now()
		_ = db.DB(ctx).Model(&user).Update("last_login_at", user.LastLoginAt).Error
		c.JSON(http.StatusOK, util.OK(buildCallbackResult(&user, "bound")))
		return
	}

	account, err := model.FindExternalAccount(source.ID, userInfo.Sub)
	switch {
	case err == nil:
		if err := db.DB(ctx).First(&user, "id = ?", account.UserID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
			return
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		username, uniqueErr := uniqueUsername(ctx, userInfo.Username)
		if uniqueErr != nil {
			c.JSON(http.StatusInternalServerError, util.Err(uniqueErr.Error()))
			return
		}
		userInfo.Username = username
		if err := user.CreateUser(db.DB(ctx), userInfo); err != nil {
			c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
			return
		}
		if err := model.BindExternalAccount(&model.ExternalAccount{
			AuthSourceID:     source.ID,
			UserID:           user.ID,
			ExternalID:       userInfo.Sub,
			ExternalUsername: userInfo.Username,
			Email:            userInfo.Email,
		}); err != nil {
			c.JSON(http.StatusBadRequest, util.Err(err.Error()))
			return
		}
	default:
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	user.LastLoginAt = time.Now()
	_ = db.DB(ctx).Model(&user).Update("last_login_at", user.LastLoginAt).Error
	if err := setLoginSession(c, &user); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, util.OK(buildCallbackResult(&user, "logged_in")))
}

// ListExternalAccounts 获取当前用户的外部帐号绑定列表
// @Summary 获取外部帐号列表
// @Description 返回当前登录用户已绑定的所有外部 OAuth 帐号信息，需要登录
// @Tags oauth
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=[]model.ExternalAccountView} "外部帐号列表"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/oauth/external-accounts [get]
func ListExternalAccounts(c *gin.Context) {
	userID := GetUserIDFromContext(c)
	accounts, err := model.ListExternalAccountsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OK(accounts))
}

// DeleteExternalAccount 解除外部帐号绑定
// @Summary 解除外部帐号绑定
// @Description 解除当前登录用户与指定外部帐号的绑定关系，需要登录
// @Tags oauth
// @Produce json
// @Security SessionCookie
// @Param id path uint64 true "外部帐号绑定记录 ID"
// @Success 200 {object} util.ResponseAny{data=string} "解除绑定成功"
// @Failure 400 {object} util.ResponseAny "ID 无效或解除失败"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Router /api/v1/oauth/external-accounts/{id}/delete [post]
func DeleteExternalAccount(c *gin.Context) {
	userID := GetUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, util.Err(common.UnAuthorized))
		return
	}
	rawID := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, util.Err("绑定记录 ID 无效"))
		return
	}
	if err := model.DeleteExternalAccountForUser(id, userID); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OKNil())
}
