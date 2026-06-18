package handler

import (
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

type UserHandler struct {
	service          service.UserService
	tokenExpireHours int
}

func NewUserHandler(service service.UserService, tokenExpireHours int) *UserHandler {
	if tokenExpireHours <= 0 {
		tokenExpireHours = 4
	}
	return &UserHandler{service: service, tokenExpireHours: tokenExpireHours}
}

func validateUsername(username string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]{3,32}$`, username)
	return matched
}

func validateEmail(email string) bool {
	return email == "" || (strings.Contains(email, "@") && len(email) <= 254)
}

func validateRole(role string) bool {
	return role == "admin" || role == "visitor" || role == "guest"
}

func userPayload(user *model.User) gin.H {
	return gin.H{
		"id":                  user.ID,
		"username":            user.Username,
		"email":               user.Email,
		"role":                user.Role,
		"preferred_avatar_id": service.NormalizeDigitalHumanAvatarID(user.PreferredAvatarID),
		"created_at":          user.CreatedAt,
		"updated_at":          user.UpdatedAt,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]{3,32}$`, user.Username); !matched {
		pkg.BadRequest(c, pkg.T(c, "msg_username_invalid"))
		return
	}
	if user.Email != "" {
		if !strings.Contains(user.Email, "@") || len(user.Email) > 254 {
			pkg.BadRequest(c, pkg.T(c, "msg_email_invalid"))
			return
		}
	}

	user.Role = "visitor"

	if err := h.service.Register(&user); err != nil {
		slog.Warn(pkg.T(c, "msg_register_failed"), "error", err, "username", user.Username)
		pkg.BadRequest(c, pkg.T(c, "msg_register_failed"))
		return
	}

	pkg.SuccessWithMessage(c, pkg.T(c, "msg_register_success"), nil)
}

func (h *UserHandler) Login(c *gin.Context) {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	user, err := h.service.Login(loginData.Username, loginData.Password)
	if err != nil {
		pkg.Unauthorized(c, pkg.T(c, "msg_login_failed"))
		return
	}

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Role, h.tokenExpireHours)
	if err != nil {
		pkg.InternalError(c, pkg.T(c, "msg_token_failed"))
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	secureCookie := os.Getenv("SCENIC_GUIDE_COOKIE_SECURE") == "true" || os.Getenv("GIN_MODE") == "release"
	c.SetCookie("auth_token", token, h.tokenExpireHours*3600, "/", "", secureCookie, true)

	pkg.SetCSRFCookie(c)
	pkg.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

func (h *UserHandler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	secureCookie := os.Getenv("SCENIC_GUIDE_COOKIE_SECURE") == "true" || os.Getenv("GIN_MODE") == "release"
	c.SetCookie("auth_token", "", -1, "/", "", secureCookie, true)
	pkg.SuccessWithMessage(c, pkg.T(c, "msg_logout_success"), nil)
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	pkg.SetCSRFCookie(c)

	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	id, _ := userID.(uint)
	preferredAvatarID := service.DefaultDigitalHumanAvatarID
	if id != 0 {
		if avatarID, err := h.service.GetAvatarPreference(id); err == nil {
			preferredAvatarID = avatarID
		}
	}

	pkg.Success(c, gin.H{
		"id":                  userID,
		"username":            username,
		"role":                role,
		"preferred_avatar_id": preferredAvatarID,
	})
}

func (h *UserHandler) GetAvatarPreference(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		pkg.Unauthorized(c, pkg.T(c, "err_unauthorized"))
		return
	}
	userID, _ := userIDVal.(uint)
	avatarID, err := h.service.GetAvatarPreference(userID)
	if err != nil {
		pkg.NotFound(c, pkg.T(c, "msg_user_not_found"))
		return
	}
	pkg.Success(c, gin.H{"avatar_id": avatarID})
}

func (h *UserHandler) UpdateAvatarPreference(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		pkg.Unauthorized(c, pkg.T(c, "err_unauthorized"))
		return
	}
	userID, _ := userIDVal.(uint)
	var req struct {
		AvatarID string `json:"avatar_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	if err := h.service.UpdateAvatarPreference(userID, req.AvatarID); err != nil {
		pkg.BadRequest(c, err.Error())
		return
	}
	pkg.Success(c, gin.H{"avatar_id": service.NormalizeDigitalHumanAvatarID(req.AvatarID)})
}

func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, pkg.T(c, "msg_invalid_id"))
		return
	}

	uidVal, uidOK := c.Get("user_id")
	roleVal, roleOK := c.Get("role")
	uid, _ := uidVal.(uint)
	role, _ := roleVal.(string)
	if !uidOK || !roleOK {
		pkg.Unauthorized(c, pkg.T(c, "err_unauthorized"))
		return
	}
	if uid != uint(id) && role != "admin" {
		pkg.Forbidden(c, pkg.T(c, "msg_no_permission_user"))
		return
	}

	user, err := h.service.GetUserByID(uint(id))
	if err != nil {
		pkg.NotFound(c, pkg.T(c, "msg_user_not_found"))
		return
	}

	pkg.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, pkg.T(c, "msg_invalid_id"))
		return
	}

	uidVal, uidOK := c.Get("user_id")
	roleVal, roleOK := c.Get("role")
	uid, _ := uidVal.(uint)
	role, _ := roleVal.(string)
	if !uidOK || !roleOK {
		pkg.Unauthorized(c, pkg.T(c, "err_unauthorized"))
		return
	}
	if uid != uint(id) && role != "admin" {
		pkg.Forbidden(c, pkg.T(c, "msg_no_permission_modify"))
		return
	}

	var updateData struct {
		Username        string `json:"username"`
		Email           string `json:"email"`
		CurrentPassword string `json:"current_password"`
	}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	user, err := h.service.GetUserByID(uint(id))
	if err != nil {
		pkg.NotFound(c, pkg.T(c, "msg_user_not_found"))
		return
	}

	newUsername := user.Username
	newEmail := user.Email
	if updateData.Username != "" {
		if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]{3,32}$`, updateData.Username); !matched {
			pkg.BadRequest(c, pkg.T(c, "msg_username_invalid"))
			return
		}
		newUsername = updateData.Username
	}
	if updateData.Email != "" && updateData.Email != user.Email {
		if updateData.CurrentPassword == "" {
			pkg.BadRequest(c, pkg.T(c, "msg_verify_password"))
			return
		}
		if _, err := h.service.Login(user.Username, updateData.CurrentPassword); err != nil {
			pkg.Unauthorized(c, pkg.T(c, "msg_wrong_password"))
			return
		}
		newEmail = updateData.Email
	}

	if err := h.service.UpdateProfile(uint(id), newUsername, newEmail); err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, pkg.T(c, "msg_user_not_found"))
			return
		}
		slog.Error("更新用户失败", "error", err, "user_id", id)
		pkg.InternalError(c, pkg.T(c, "msg_update_failed"))
		return
	}

	pkg.Success(c, gin.H{
		"id":       user.ID,
		"username": newUsername,
		"email":    newEmail,
		"role":     user.Role,
	})
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, pkg.T(c, "msg_invalid_id"))
		return
	}

	uidVal, uidOK := c.Get("user_id")
	roleVal, roleOK := c.Get("role")
	uid, _ := uidVal.(uint)
	role, _ := roleVal.(string)
	if !uidOK || !roleOK {
		pkg.Unauthorized(c, pkg.T(c, "err_unauthorized"))
		return
	}
	if uid != uint(id) && role != "admin" {
		pkg.Forbidden(c, pkg.T(c, "msg_no_permission_delete"))
		return
	}

	if err := h.service.DeleteUser(uint(id)); err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, pkg.T(c, "msg_user_not_found"))
			return
		}
		slog.Error(pkg.T(c, "msg_delete_failed"), "error", err, "user_id", id)
		pkg.InternalError(c, pkg.T(c, "msg_delete_failed"))
		return
	}

	pkg.SuccessWithMessage(c, pkg.T(c, "msg_delete_success"), nil)
}

func (h *UserHandler) AdminCreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	if !validateUsername(req.Username) {
		pkg.BadRequest(c, pkg.T(c, "msg_username_invalid"))
		return
	}
	if !validateEmail(req.Email) {
		pkg.BadRequest(c, pkg.T(c, "msg_email_invalid"))
		return
	}
	if req.Role == "" {
		req.Role = "visitor"
	}
	if !validateRole(req.Role) {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	user := &model.User{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Role:     req.Role,
	}
	if err := h.service.CreateUser(user); err != nil {
		slog.Warn("admin create user failed", "error", err, "username", req.Username)
		pkg.BadRequest(c, pkg.T(c, "msg_create_failed"))
		return
	}

	pkg.Success(c, userPayload(user))
}

func (h *UserHandler) AdminUpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		pkg.BadRequest(c, pkg.T(c, "msg_invalid_id"))
		return
	}

	var req struct {
		Username *string `json:"username"`
		Password *string `json:"password"`
		Email    *string `json:"email"`
		Role     *string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	if req.Username != nil && !validateUsername(*req.Username) {
		pkg.BadRequest(c, pkg.T(c, "msg_username_invalid"))
		return
	}
	if req.Email != nil && !validateEmail(*req.Email) {
		pkg.BadRequest(c, pkg.T(c, "msg_email_invalid"))
		return
	}
	if req.Role != nil && !validateRole(*req.Role) {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	user, err := h.service.UpdateAdminUser(uint(id), req.Username, req.Email, req.Role, req.Password)
	if err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, pkg.T(c, "msg_user_not_found"))
			return
		}
		slog.Error("admin update user failed", "error", err, "user_id", id)
		pkg.BadRequest(c, pkg.T(c, "msg_update_failed"))
		return
	}

	pkg.Success(c, userPayload(user))
}

func (h *UserHandler) GetAllUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.service.GetAllUsersPaginated(page, pageSize)
	if err != nil {
		slog.Error(pkg.T(c, "msg_get_users_failed"), "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_get_users_failed"))
		return
	}

	result := make([]gin.H, 0, len(users))
	for _, user := range users {
		u := user
		result = append(result, userPayload(&u))
	}

	pkg.Success(c, gin.H{
		"list":      result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *UserHandler) GetUsersByRole(c *gin.Context) {
	role := c.Query("role")
	if role == "" {
		pkg.BadRequest(c, pkg.T(c, "msg_role_required"))
		return
	}

	users, err := h.service.GetUsersByRole(role)
	if err != nil {
		slog.Error("按角色获取用户列表失败", "error", err, "role", role)
		pkg.InternalError(c, pkg.T(c, "msg_get_users_failed"))
		return
	}

	result := make([]gin.H, 0, len(users))
	for _, user := range users {
		u := user
		result = append(result, userPayload(&u))
	}

	pkg.Success(c, result)
}

func (h *UserHandler) Routes(r *gin.RouterGroup) {
	r.POST("/register", pkg.RateLimitMiddleware(5, time.Minute), h.Register)
	r.POST("/login", pkg.RateLimitMiddleware(10, time.Minute), h.Login)
	r.POST("/logout", h.Logout)

	auth := r.Group("")
	auth.Use(pkg.AuthMiddleware())
	{
		auth.GET("/user/me", h.GetCurrentUser)
		auth.GET("/user/avatar-preference", h.GetAvatarPreference)
		auth.PUT("/user/avatar-preference", h.UpdateAvatarPreference)
		auth.GET("/users/:id", h.GetUser)
		auth.PUT("/users/:id", h.UpdateUser)
		auth.DELETE("/users/:id", h.DeleteUser)
	}

	admin := r.Group("/admin")
	admin.Use(pkg.AuthMiddleware(), pkg.AdminMiddleware())
	{
		admin.GET("/users", h.GetAllUsers)
		admin.GET("/users/role", h.GetUsersByRole)
		admin.POST("/users", h.AdminCreateUser)
		admin.PUT("/users/:id", h.AdminUpdateUser)
		admin.DELETE("/users/:id", h.DeleteUser)
	}
}
