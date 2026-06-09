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

package user

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

type registerRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Nickname    string `json:"nickname"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Code        string `json:"code"`
}

type sendEmailCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Scene string `json:"scene" binding:"required"`
}

func isEmailLoginVerificationEnabled() bool {
	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyEmailLoginVerificationEnabled)
	if err != nil {
		return false
	}
	return enabled
}

func isEmailRegisterVerificationEnabled() bool {
	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyEmailRegisterVerificationEnabled)
	if err != nil {
		return false
	}
	return enabled
}

func isSMTPConfigured(ctx context.Context) bool {
	var sc model.SystemConfig
	var host, port, username string

	if err := sc.GetByKey(ctx, model.ConfigKeySMTPHost); err == nil {
		host = sc.Value
	}
	if err := sc.GetByKey(ctx, model.ConfigKeySMTPPort); err == nil {
		port = sc.Value
	}
	if err := sc.GetByKey(ctx, model.ConfigKeySMTPUsername); err == nil {
		username = sc.Value
	}

	return host != "" && port != "" && username != ""
}

func generateVerificationCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(verificationCodeRange))
	return fmt.Sprintf("%06d", n.Int64()+verificationCodeOffset)
}

func getEmailCodeKey(scene, email string) string {
	return fmt.Sprintf("email_code:%s:%s", scene, email)
}

func getEmailCooldownKey(email string) string {
	return fmt.Sprintf("email_code:cooldown:%s", email)
}

func sendEmailVerificationCode(ctx context.Context, email, scene, templateName string) error {
	// 校验 SMTP 配置是否完整
	if !isSMTPConfigured(ctx) {
		return errors.New(errSMTPConfigIncomplete)
	}

	code := generateVerificationCode()
	codeKey := getEmailCodeKey(scene, email)
	cooldownKey := getEmailCooldownKey(email)

	// 使用模板管理获取并渲染邮件标题和正文。模板缺失或渲染失败时不发送验证码。
	emailSubject, emailBody, err := model.RenderTemplate(
		ctx,
		templateName,
		map[string]any{"Code": code},
	)
	if err != nil {
		return fmt.Errorf(errRenderEmailTemplateFailed, err)
	}

	// 存验证码，5分钟有效
	if err := db.SetJSON(ctx, codeKey, code, emailCodeExpiry); err != nil {
		return errors.New(errGenerateEmailCodeFailed)
	}
	// 存冷却，60秒有效
	_ = db.SetJSON(ctx, cooldownKey, "1", emailCodeCooldown)

	// 构建异步邮件发送任务
	payload := SendEmailPayload{
		To:      email,
		Subject: emailSubject,
		Body:    emailBody,
	}
	payloadBytes, _ := json.Marshal(payload)
	_, err = task.DispatchTask(ctx, task.TaskTypeSendEmail, payloadBytes, "system")
	if err != nil {
		return errors.New(errDispatchEmailTaskFailed)
	}
	return nil
}

func verifyEmailCode(ctx context.Context, email, scene, code string) bool {
	codeKey := getEmailCodeKey(scene, email)
	var storedCode string
	if err := db.GetJSON(ctx, codeKey, &storedCode); err != nil {
		return false
	}
	if storedCode != code {
		return false
	}
	// 验证成功，删除验证码
	_ = db.Redis.Del(ctx, db.PrefixedKey(codeKey)).Err()
	return true
}

func isPasswordLoginEnabled() bool {
	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyPasswordLoginEnabled)
	if err != nil {
		return true
	}
	return enabled
}

func isPasswordRegisterEnabled() bool {
	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyPasswordRegisterEnabled)
	if err != nil {
		return true
	}
	return enabled
}

func isRegistrationEnabled() bool {
	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyRegistrationEnabled)
	if err != nil {
		return true
	}
	return enabled
}

func setLoginSession(c *gin.Context, user *model.User) error {
	session := sessions.Default(c)
	session.Set(oauth.UserIDKey, user.ID)
	session.Set(oauth.UserNameKey, user.Username)
	if err := session.Save(); err != nil {
		return err
	}
	return nil
}

// handleLoginEmailVerification 处理登录时的邮箱验证码校验流程
func handleLoginEmailVerification(ctx context.Context, c *gin.Context, req *loginRequest, user *model.User) error {
	if user.Email == "" {
		c.JSON(http.StatusOK, util.Err(errLoginEmailMissing))
		return errors.New("handled")
	}

	if req.Code == "" {
		// 校验 Redis 发送冷却时间
		cooldownKey := getEmailCooldownKey(user.Email)
		var temp string
		err := db.GetJSON(ctx, cooldownKey, &temp)
		if err != nil {
			// 没有冷却，触发验证码发送
			if err := sendEmailVerificationCode(ctx, user.Email, "login", "login_email"); err != nil {
				c.JSON(http.StatusOK, util.Err(err.Error()))
				return errors.New("handled")
			}
		}

		// 脱敏邮箱并返回错误，提示前端需要输入验证码
		maskedEmail := util.MaskEmail(user.Email)
		c.JSON(http.StatusOK, util.Err(errNeedEmailCodePrefix+maskedEmail))
		return errors.New("handled")
	}

	// 校验验证码
	if !verifyEmailCode(ctx, user.Email, "login", req.Code) {
		c.JSON(http.StatusOK, util.Err(errEmailCodeInvalidOrExpired))
		return errors.New("handled")
	}
	return nil
}

// Login 用户密码登录
// @Summary 用户密码登录
// @Description 使用用户名和密码登录，登录成功后建立 Session。若管理员已关闭密码登录功能则返回错误。
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.loginRequest true "登录请求参数"
// @Success 200 {object} util.ResponseAny{data=oauth.BasicUserInfo} "登录成功，返回用户信息"
// @Failure 400 {object} util.ResponseAny "用户名或密码错误、帐号已禁用等"
// @Failure 500 {object} util.ResponseAny "服务内部错误"
// @Router /api/v1/user/login [post]
func Login(c *gin.Context) {
	if !isPasswordLoginEnabled() {
		c.JSON(http.StatusOK, util.Err(errPasswordLoginDisabled))
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusOK, util.Err(errInvalidParams))
		return
	}

	var user model.User
	ctx := c.Request.Context()
	if err := db.DB(ctx).Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errUsernameOrPasswordWrong))
		return
	}
	if !user.IsActive {
		c.JSON(http.StatusOK, util.Err(common.BannedAccount))
		return
	}

	// 判定是否是明文密码存储
	isPlaintext := !user.IsPasswordEncrypted()

	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusOK, util.Err(errUsernameOrPasswordWrong))
		return
	}

	if isEmailLoginVerificationEnabled() {
		if emailErr := handleLoginEmailVerification(ctx, c, &req, &user); emailErr != nil {
			return
		}
	}

	session := sessions.Default(c)
	needChangePassword := false

	// 如果是以明文密码登录，在数据库中置换为加密密码
	if isPlaintext {
		if err := user.SetEncryptedPassword(req.Password); err == nil {
			if err := db.DB(ctx).Model(&user).Update("password", user.Password).Error; err != nil {
				c.JSON(http.StatusOK, util.Err(errPasswordUpgradeFailed))
				return
			}
			needChangePassword = true
			session.Set("need_change_password", true)
			_ = session.Save()
		}
	}

	user.LastLoginAt = time.Now()
	if err := db.DB(ctx).Model(&user).Update("last_login_at", user.LastLoginAt).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}
	if err := setLoginSession(c, &user); err != nil {
		c.JSON(http.StatusOK, util.Err(errSaveSessionFailed))
		return
	}

	// 检查是否有未完成 of OAuth/OIDC 绑定
	completePendingOAuthBinding(session, &user)

	c.JSON(http.StatusOK, util.OK(oauth.BuildBasicUserInfo(&user, needChangePassword)))
}

// Register 用户注册
// @Summary 用户注册
// @Description 使用用户名和密码注册新账号，注册成功后自动登录并建立 Session。密码长度不能少于 8 位。
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.registerRequest true "注册请求参数"
// @Success 200 {object} util.ResponseAny{data=oauth.BasicUserInfo} "注册并登录成功，返回用户信息"
// @Failure 400 {object} util.ResponseAny "参数错误、用户名已存在或注册已关闭"
// @Failure 500 {object} util.ResponseAny "服务内部错误"
// @Router /api/v1/user/register [post]
func Register(c *gin.Context) {
	if !isRegistrationEnabled() || !isPasswordRegisterEnabled() {
		c.JSON(http.StatusOK, util.Err(errRegistrationDisabled))
		return
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Email = strings.TrimSpace(req.Email)
	req.Code = strings.TrimSpace(req.Code)

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusOK, util.Err(errInvalidParams))
		return
	}
	if len(req.Password) < minPasswordLength {
		c.JSON(http.StatusOK, util.Err(errPasswordTooShort))
		return
	}

	ctx := c.Request.Context()

	// 邮箱注册验证校验
	if err := validateRegisterEmailVerification(ctx, &req); err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	user := model.User{
		Username:    req.Username,
		Nickname:    req.Nickname,
		Email:       req.Email,
		AvatarURL:   "",
		IsActive:    true,
		IsAdmin:     false,
		LastLoginAt: time.Now(),
	}
	if user.Nickname == "" {
		user.Nickname = req.DisplayName
	}
	if user.Nickname == "" {
		user.Nickname = req.Username
	}
	if err := user.SetPassword(req.Password); err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	if err := user.RegisterUser(ctx, db.DB(ctx)); err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	if err := setLoginSession(c, &user); err != nil {
		c.JSON(http.StatusOK, util.Err(errSaveSessionFailed))
		return
	}

	c.JSON(http.StatusOK, util.OK(oauth.BuildBasicUserInfo(&user, false)))
}

// Logout 用户退出登录
// @Summary 用户退出登录
// @Description 清除用户登录 Session，完成退出
// @Tags user
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=string} "退出成功"
// @Failure 500 {object} util.ResponseAny "Session 清除失败"
// @Router /api/v1/user/logout [get]
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Options(util.GetSessionOptions(-1))
	session.Clear()
	if err := session.Save(); err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OK(""))
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePassword 修改用户密码
// @Summary 修改用户密码
// @Description 修改当前登录用户的密码。修改成功后，如果是首次明文登录的升级提示，则清除修改密码的提示状态。
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.changePasswordRequest true "修改密码请求参数"
// @Success 200 {object} util.ResponseAny{data=string} "修改密码成功"
// @Failure 400 {object} util.ResponseAny "原密码错误或新密码不符合要求"
// @Failure 401 {object} util.ResponseAny "请先登录"
// @Router /api/v1/user/change-password [post]
func ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	req.OldPassword = strings.TrimSpace(req.OldPassword)
	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if req.OldPassword == "" || req.NewPassword == "" {
		c.JSON(http.StatusOK, util.Err(errInvalidParams))
		return
	}
	if len(req.NewPassword) < minPasswordLength {
		c.JSON(http.StatusOK, util.Err(errNewPasswordTooShort))
		return
	}

	userObj, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	if userObj == nil {
		c.JSON(http.StatusUnauthorized, util.Err(errLoginRequired))
		return
	}

	ctx := c.Request.Context()
	var dbUser model.User
	if err := db.DB(ctx).Where("id = ?", userObj.ID).First(&dbUser).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errUserNotFound))
		return
	}

	// 校验旧密码
	if !dbUser.CheckPassword(req.OldPassword) {
		c.JSON(http.StatusOK, util.Err(errOldPasswordIncorrect))
		return
	}

	// 加密并更新为新密码
	if err := dbUser.SetEncryptedPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusOK, util.Err(errPasswordEncryptFailed))
		return
	}

	if err := db.DB(ctx).Model(&dbUser).Update("password", dbUser.Password).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	// 清除 Session 中修改密码提示状态
	session := sessions.Default(c)
	session.Delete("need_change_password")
	_ = session.Save()

	c.JSON(http.StatusOK, util.OK("密码修改成功"))
}

// SendEmailCode 发送邮箱验证码
// @Summary 发送邮箱验证码
// @Description 向指定邮箱发送验证码（用于注册场景）
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.sendEmailCodeRequest true "发送验证码请求参数"
// @Success 200 {object} util.ResponseAny "发送成功"
// @Failure 400 {object} util.ResponseAny "参数错误"
// @Router /api/v1/user/send-email-code [post]
func SendEmailCode(c *gin.Context) {
	var req sendEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		c.JSON(http.StatusOK, util.Err(errEmailRequired))
		return
	}

	if req.Scene != "register" {
		c.JSON(http.StatusOK, util.Err(errUnsupportedEmailScene))
		return
	}

	ctx := c.Request.Context()

	// 1. 检查邮箱是否已被注册
	var count int64
	if err := db.DB(ctx).Model(&model.User{}).Where("email = ?", req.Email).Count(&count).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}
	if count > 0 {
		c.JSON(http.StatusOK, util.Err(errEmailAlreadyRegistered))
		return
	}

	// 2. 校验 Redis 发送冷却时间
	cooldownKey := getEmailCooldownKey(req.Email)
	var temp string
	err := db.GetJSON(ctx, cooldownKey, &temp)
	if err == nil {
		c.JSON(http.StatusOK, util.Err(errEmailCodeCooldown))
		return
	}

	// 3. 发送验证码
	if err := sendEmailVerificationCode(ctx, req.Email, "register", "register_email"); err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, util.OKNil())
}

type updateProfileRequest struct {
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	Phone     string `json:"phone"`
	Gender    string `json:"gender"`
	Website   string `json:"website"`
	Location  string `json:"location"`
}

// UpdateProfile 修改当前登录用户的个人资料
// @Summary 修改当前登录用户的个人资料
// @Description 修改当前登录用户的昵称、邮箱、头像、简介、电话、性别、个人网站和所在地。
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.updateProfileRequest true "更新请求参数"
// @Success 200 {object} util.ResponseAny{data=oauth.BasicUserInfo} "修改成功，返回更新后的用户信息"
// @Failure 400 {object} util.ResponseAny "邮箱已被占用或参数错误"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Router /api/v1/user/profile [put]
func UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	userObj, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	if userObj == nil {
		c.JSON(http.StatusUnauthorized, util.Err(errLoginRequired))
		return
	}

	ctx := c.Request.Context()
	var dbUser model.User
	if err := db.DB(ctx).Where("id = ?", userObj.ID).First(&dbUser).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errUserNotFound))
		return
	}

	// 校验邮箱格式与唯一性
	req.Email = strings.TrimSpace(req.Email)
	if req.Email != "" && req.Email != dbUser.Email {
		if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
			c.JSON(http.StatusOK, util.Err(errEmailFormatInvalid))
			return
		}

		var count int64
		if err := db.DB(ctx).Model(&model.User{}).Where("email = ? AND id != ?", req.Email, dbUser.ID).Count(&count).Error; err != nil {
			c.JSON(http.StatusOK, util.Err(err.Error()))
			return
		}
		if count > 0 {
			c.JSON(http.StatusOK, util.Err(errEmailAlreadyBound))
			return
		}
	}

	// 更新字段
	dbUser.Nickname = strings.TrimSpace(req.Nickname)
	if dbUser.Nickname == "" {
		dbUser.Nickname = dbUser.Username
	}
	dbUser.Email = req.Email
	dbUser.AvatarURL = req.AvatarURL
	dbUser.Bio = req.Bio
	dbUser.Phone = strings.TrimSpace(req.Phone)
	dbUser.Gender = strings.TrimSpace(req.Gender)
	dbUser.Website = strings.TrimSpace(req.Website)
	dbUser.Location = strings.TrimSpace(req.Location)

	if err := db.DB(ctx).Save(&dbUser).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	session := sessions.Default(c)
	needChange := session.Get("need_change_password") == true

	c.JSON(http.StatusOK, util.OK(oauth.BuildBasicUserInfo(&dbUser, needChange)))
}

// validateRegisterEmailVerification 校验注册时的邮箱验证码
func validateRegisterEmailVerification(ctx context.Context, req *registerRequest) error {
	if !isEmailRegisterVerificationEnabled() {
		return nil
	}
	if req.Email == "" || req.Code == "" {
		return errors.New(errEmailOrCodeRequired)
	}
	if !verifyEmailCode(ctx, req.Email, "register", req.Code) {
		return errors.New(errEmailCodeInvalidOrExpired)
	}
	return nil
}

// completePendingOAuthBinding 完成登录后的 OAuth 待绑定绑定流程
func completePendingOAuthBinding(session sessions.Session, user *model.User) {
	pendingSourceID := session.Get(oauth.PendingOAuthSourceIDKey)
	pendingExternalID := session.Get(oauth.PendingOAuthExternalIDKey)
	pendingExternalUsername := session.Get(oauth.PendingOAuthExternalUsernameKey)
	pendingEmail := session.Get(oauth.PendingOAuthEmailKey)

	if pendingSourceID == nil || pendingExternalID == nil {
		return
	}

	var sourceID uint64
	switch v := pendingSourceID.(type) {
	case uint64:
		sourceID = v
	case int:
		sourceID = uint64(v)
	case float64:
		sourceID = uint64(v)
	}
	externalID, _ := pendingExternalID.(string)
	externalUsername, _ := pendingExternalUsername.(string)
	email, _ := pendingEmail.(string)

	if sourceID != 0 && externalID != "" {
		_ = model.BindExternalAccount(&model.ExternalAccount{
			AuthSourceID:     sourceID,
			UserID:           user.ID,
			ExternalID:       externalID,
			ExternalUsername: externalUsername,
			Email:            email,
		})
	}
	// 清除 pending 信息
	session.Delete(oauth.PendingOAuthSourceIDKey)
	session.Delete(oauth.PendingOAuthExternalIDKey)
	session.Delete(oauth.PendingOAuthExternalUsernameKey)
	session.Delete(oauth.PendingOAuthEmailKey)
	_ = session.Save()
}
