package user

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/linux-do/credit/internal/apps/oauth"
	"github.com/linux-do/credit/internal/common"
	"github.com/linux-do/credit/internal/common/bind"
	"github.com/linux-do/credit/internal/db"
	"github.com/linux-do/credit/internal/model"
	"github.com/linux-do/credit/internal/util"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Nickname    string `json:"nickname"`
	DisplayName string `json:"display_name"`
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
	if !bind.JSON(c, &req) {
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
	if !bind.JSON(c, &req) {
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusOK, util.Err("无效的参数"))
		return
	}
	if len(req.Password) < 8 {
		c.JSON(http.StatusOK, util.Err("密码长度不能少于 8 位"))
		return
	}

	ctx := c.Request.Context()
	var count int64
	if err := db.DB(ctx).Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}
	if count > 0 {
		c.JSON(http.StatusOK, util.Err("用户名已存在"))
		return
	}

	user := model.User{
		Username:    req.Username,
		Nickname:    req.Nickname,
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
	if !bind.JSON(c, &req) {
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
