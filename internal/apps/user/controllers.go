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

func generateVerificationCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
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
		c.JSON(http.StatusOK, util.Err("管理员关闭了密码登录"))
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusOK, util.Err("无效的参数"))
		return
	}

	var user model.User
	ctx := c.Request.Context()
	if err := db.DB(ctx).Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, util.Err("用户名或密码错误"))
		return
	}
	if !user.IsActive {
		c.JSON(http.StatusOK, util.Err(common.BannedAccount))
		return
	}

	// 判定是否是明文密码存储
	isPlaintext := !(strings.HasPrefix(user.Password, "$2a$") || strings.HasPrefix(user.Password, "$2b$") || strings.HasPrefix(user.Password, "$2y$"))

	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusOK, util.Err("用户名或密码错误"))
		return
	}

	if isEmailLoginVerificationEnabled() {
		if user.Email == "" {
			c.JSON(http.StatusOK, util.Err("该账号未绑定邮箱，请联系管理员绑定邮箱后再登录"))
			return
		}

		if req.Code == "" {
			// 校验 Redis 发送冷却时间
			cooldownKey := fmt.Sprintf("email_code:cooldown:%s", user.Email)
			var temp string
			err := db.GetJSON(ctx, cooldownKey, &temp)
			if err != nil {
				// 没有冷却，触发验证码发送
				code := generateVerificationCode()
				codeKey := fmt.Sprintf("email_code:login:%s", user.Email)
				// 存验证码，5分钟有效
				if err := db.SetJSON(ctx, codeKey, code, 5*time.Minute); err != nil {
					c.JSON(http.StatusOK, util.Err("生成验证码失败，请重试"))
					return
				}
				// 存冷却，60秒有效
				_ = db.SetJSON(ctx, cooldownKey, "1", 60*time.Second)

				// 使用模板管理获取并渲染邮件标题和正文
				emailSubject, emailBody := model.RenderTemplate(
					ctx,
					"login_email",
					map[string]any{"Code": code},
					"Wavelet 登录验证码",
					fmt.Sprintf("<h3>Wavelet 登录验证</h3><p>您的登录验证码为：<strong>%s</strong>，5分钟内有效，请勿将验证码泄露给他人。</p>", code),
				)

				// 构建异步邮件发送任务
				payload := SendEmailPayload{
					To:      user.Email,
					Subject: emailSubject,
					Body:    emailBody,
				}
				payloadBytes, _ := json.Marshal(payload)
				_, err = task.DispatchTask(ctx, task.TaskTypeSendEmail, payloadBytes, "system")
				if err != nil {
					c.JSON(http.StatusOK, util.Err("投递验证邮件发送任务失败，请重试"))
					return
				}
			}

			// 脱敏邮箱并返回错误，提示前端需要输入验证码
			maskedEmail := util.MaskEmail(user.Email)
			c.JSON(http.StatusOK, util.Err("need_email_code:"+maskedEmail))
			return
		}

		// 校验验证码
		codeKey := fmt.Sprintf("email_code:login:%s", user.Email)
		var storedCode string
		if err := db.GetJSON(ctx, codeKey, &storedCode); err != nil {
			c.JSON(http.StatusOK, util.Err("验证码错误或已过期"))
			return
		}
		if storedCode != req.Code {
			c.JSON(http.StatusOK, util.Err("验证码错误或已过期"))
			return
		}

		// 验证成功，删除验证码
		_ = db.Redis.Del(ctx, db.PrefixedKey(codeKey)).Err()
	}

	session := sessions.Default(c)
	needChangePassword := false

	// 如果是以明文密码登录，在数据库中置换为加密密码
	if isPlaintext {
		if err := user.SetEncryptedPassword(req.Password); err == nil {
			if err := db.DB(ctx).Model(&user).Update("password", user.Password).Error; err != nil {
				c.JSON(http.StatusOK, util.Err("升级密码安全算法失败，请重试"))
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
		c.JSON(http.StatusOK, util.Err("无法保存会话信息，请重试"))
		return
	}

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
		c.JSON(http.StatusOK, util.Err("管理员关闭了注册"))
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
		c.JSON(http.StatusOK, util.Err("无效的参数"))
		return
	}
	if len(req.Password) < 8 {
		c.JSON(http.StatusOK, util.Err("密码长度不能少于 8 位"))
		return
	}

	ctx := c.Request.Context()

	// 邮箱注册验证校验
	if isEmailRegisterVerificationEnabled() {
		if req.Email == "" || req.Code == "" {
			c.JSON(http.StatusOK, util.Err("邮箱或验证码未填写"))
			return
		}

		codeKey := fmt.Sprintf("email_code:register:%s", req.Email)
		var storedCode string
		if err := db.GetJSON(ctx, codeKey, &storedCode); err != nil {
			c.JSON(http.StatusOK, util.Err("验证码错误或已过期"))
			return
		}
		if storedCode != req.Code {
			c.JSON(http.StatusOK, util.Err("验证码错误或已过期"))
			return
		}

		// 验证通过，删除 Redis 中的验证码
		_ = db.Redis.Del(ctx, db.PrefixedKey(codeKey)).Err()
	}

	var count int64
	if err := db.DB(ctx).Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}
	if count > 0 {
		c.JSON(http.StatusOK, util.Err("用户名已存在"))
		return
	}

	// 校验邮箱是否已被其他用户使用
	if req.Email != "" {
		var emailCount int64
		if err := db.DB(ctx).Model(&model.User{}).Where("email = ?", req.Email).Count(&emailCount).Error; err != nil {
			c.JSON(http.StatusOK, util.Err(err.Error()))
			return
		}
		if emailCount > 0 {
			c.JSON(http.StatusOK, util.Err("该邮箱已被其他账号绑定"))
			return
		}
	}

	user := model.User{
		Username:    req.Username,
		Nickname:    req.Nickname,
		Email:       req.Email,
		AvatarUrl:   "",
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

	if err := db.DB(ctx).Create(&user).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	if err := setLoginSession(c, &user); err != nil {
		c.JSON(http.StatusOK, util.Err("无法保存会话信息，请重试"))
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
		c.JSON(http.StatusOK, util.Err("无效的参数"))
		return
	}
	if len(req.NewPassword) < 8 {
		c.JSON(http.StatusOK, util.Err("新密码长度不能少于 8 位"))
		return
	}

	userObj, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	if userObj == nil {
		c.JSON(http.StatusUnauthorized, util.Err("请先登录"))
		return
	}

	ctx := c.Request.Context()
	var dbUser model.User
	if err := db.DB(ctx).Where("id = ?", userObj.ID).First(&dbUser).Error; err != nil {
		c.JSON(http.StatusOK, util.Err("未找到该用户"))
		return
	}

	// 校验旧密码
	if !dbUser.CheckPassword(req.OldPassword) {
		c.JSON(http.StatusOK, util.Err("原密码不正确"))
		return
	}

	// 加密并更新为新密码
	if err := dbUser.SetEncryptedPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusOK, util.Err("密码加密失败，请重试"))
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
		c.JSON(http.StatusOK, util.Err("邮箱地址不能为空"))
		return
	}

	if req.Scene != "register" {
		c.JSON(http.StatusOK, util.Err("不支持的验证场景"))
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
		c.JSON(http.StatusOK, util.Err("该邮箱已被注册"))
		return
	}

	// 2. 校验 Redis 发送冷却时间
	cooldownKey := fmt.Sprintf("email_code:cooldown:%s", req.Email)
	var temp string
	err := db.GetJSON(ctx, cooldownKey, &temp)
	if err == nil {
		c.JSON(http.StatusOK, util.Err("验证码发送频繁，请稍后再试"))
		return
	}

	// 3. 生成并缓存验证码
	code := generateVerificationCode()
	codeKey := fmt.Sprintf("email_code:register:%s", req.Email)
	if err := db.SetJSON(ctx, codeKey, code, 5*time.Minute); err != nil {
		c.JSON(http.StatusOK, util.Err("生成验证码失败，请重试"))
		return
	}
	_ = db.SetJSON(ctx, cooldownKey, "1", 60*time.Second)

	// 使用模板管理获取并渲染邮件标题和正文
	emailSubject, emailBody := model.RenderTemplate(
		ctx,
		"register_email",
		map[string]any{"Code": code},
		"Wavelet 注册验证码",
		fmt.Sprintf("<h3>Wavelet 注册验证</h3><p>您的注册验证码为：<strong>%s</strong>，5分钟内有效，请勿泄露给他人。</p>", code),
	)

	// 4. 投递异步邮件发送任务
	payload := SendEmailPayload{
		To:      req.Email,
		Subject: emailSubject,
		Body:    emailBody,
	}
	payloadBytes, _ := json.Marshal(payload)
	_, err = task.DispatchTask(ctx, task.TaskTypeSendEmail, payloadBytes, "system")
	if err != nil {
		c.JSON(http.StatusOK, util.Err("投递验证邮件发送任务失败，请重试"))
		return
	}

	c.JSON(http.StatusOK, util.OKNil())
}
