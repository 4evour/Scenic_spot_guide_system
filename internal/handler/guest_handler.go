package handler

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

// GuestHandler 游客认证相关接口
type GuestHandler struct {
	guestService     *service.GuestService
	tokenExpireHours int
}

func NewGuestHandler(guestService *service.GuestService, tokenExpireHours int) *GuestHandler {
	if tokenExpireHours <= 0 {
		tokenExpireHours = 24
	}
	return &GuestHandler{guestService: guestService, tokenExpireHours: tokenExpireHours}
}

// GuestLogin 游客自动登录
// POST /api/v1/auth/guest-login
func (h *GuestHandler) GuestLogin(c *gin.Context) {
	var req struct {
		DeviceFingerprint string `json:"device_fingerprint"`
	}
	_ = c.ShouldBindJSON(&req)

	fingerprint := req.DeviceFingerprint
	if fingerprint == "" {
		// 基于 IP + User-Agent 生成指纹
		fp := c.ClientIP() + "|" + c.GetHeader("User-Agent")
		hash := sha256.Sum256([]byte(fp))
		fingerprint = fmt.Sprintf("%x", hash)
	}

	user, token, err := h.guestService.CreateGuestAccount(fingerprint)
	if err != nil {
		slog.Error("游客登录失败", "error", err)
		pkg.InternalError(c, "游客登录失败")
		return
	}

	// 设置 Cookie
	secureCookie := os.Getenv("SCENIC_GUIDE_COOKIE_SECURE") == "true" || os.Getenv("GIN_MODE") == "release"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("auth_token", token, h.tokenExpireHours*3600, "/", "", secureCookie, true)
	pkg.SetCSRFCookie(c)

	pkg.Success(c, gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"role":         user.Role,
	})
}

// UpgradeGuest 游客升级为正式账号
// POST /api/v1/auth/upgrade-guest
func (h *GuestHandler) UpgradeGuest(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		pkg.Unauthorized(c, "未登录")
		return
	}
	userID, _ := userIDVal.(uint)

	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	if role != "guest" {
		pkg.BadRequest(c, "只有游客账号可以升级")
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "请提供用户名和密码")
		return
	}

	// 用户名格式校验
	if !validateUsername(req.Username) {
		pkg.BadRequest(c, "用户名只能包含字母、数字和下划线，长度 3-32")
		return
	}
	if req.Email != "" && !validateEmail(req.Email) {
		pkg.BadRequest(c, "邮箱格式不正确")
		return
	}

	user, err := h.guestService.UpgradeGuest(userID, req.Username, req.Password, req.Email)
	if err != nil {
		slog.Warn("游客升级失败", "error", err, "user_id", userID)
		pkg.BadRequest(c, err.Error())
		return
	}

	// 重新签发 token
	token, err := pkg.GenerateToken(user.ID, user.Username, user.Role, user.TokenVersion, h.tokenExpireHours)
	if err != nil {
		pkg.InternalError(c, "签发令牌失败")
		return
	}

	// 更新 Cookie
	secureCookie := os.Getenv("SCENIC_GUIDE_COOKIE_SECURE") == "true" || os.Getenv("GIN_MODE") == "release"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("auth_token", token, h.tokenExpireHours*3600, "/", "", secureCookie, true)
	pkg.SetCSRFCookie(c)

	slog.Info("游客升级为正式账号", "user_id", userID, "username", req.Username)
	pkg.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

// CreateGuestFunc 返回供中间件使用的游客创建函数
func (h *GuestHandler) CreateGuestFunc() pkg.GuestCreatorFunc {
	return func(fingerprint string) (uint, string, string, string, string, error) {
		user, token, err := h.guestService.CreateGuestAccount(fingerprint)
		if err != nil {
			return 0, "", "", "", "", err
		}
		return user.ID, user.Username, user.DisplayName, user.Role, token, nil
	}
}

// Routes 注册游客相关路由
func (h *GuestHandler) Routes(r *gin.RouterGroup) {
	r.POST("/auth/guest-login", pkg.RateLimitMiddleware(10, time.Minute), h.GuestLogin)

	// 游客升级需要认证
	auth := r.Group("")
	auth.Use(pkg.AuthMiddleware())
	{
		auth.POST("/auth/upgrade-guest", h.UpgradeGuest)
	}
}
