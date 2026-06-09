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
	return role == "admin" || role == "visitor"
}

func userPayload(user *model.User) gin.H {
	return gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]{3,32}$`, user.Username); !matched {
		pkg.BadRequest(c, "用户名须为3-32位字母、数字或下划线")
		return
	}
	if user.Email != "" {
		if !strings.Contains(user.Email, "@") || len(user.Email) > 254 {
			pkg.BadRequest(c, "邮箱格式不正确")
			return
		}
	}

	user.Role = "visitor"

	if err := h.service.Register(&user); err != nil {
		slog.Warn("注册失败", "error", err, "username", user.Username)
		pkg.BadRequest(c, "注册失败")
		return
	}

	pkg.SuccessWithMessage(c, "注册成功", nil)
}

func (h *UserHandler) Login(c *gin.Context) {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	user, err := h.service.Login(loginData.Username, loginData.Password)
	if err != nil {
		pkg.Unauthorized(c, "用户名或密码错误")
		return
	}

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Role, h.tokenExpireHours)
	if err != nil {
		pkg.InternalError(c, "生成token失败")
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
	pkg.SuccessWithMessage(c, "已退出登录", nil)
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	pkg.Success(c, gin.H{
		"id":       userID,
		"username": username,
		"role":     role,
	})
}

func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	uidVal, uidOK := c.Get("user_id")
	roleVal, roleOK := c.Get("role")
	uid, _ := uidVal.(uint)
	role, _ := roleVal.(string)
	if !uidOK || !roleOK {
		pkg.Unauthorized(c, "未登录")
		return
	}
	if uid != uint(id) && role != "admin" {
		pkg.Forbidden(c, "无权访问该用户信息")
		return
	}

	user, err := h.service.GetUserByID(uint(id))
	if err != nil {
		pkg.NotFound(c, "用户不存在")
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
		pkg.BadRequest(c, "无效的ID")
		return
	}

	uidVal, uidOK := c.Get("user_id")
	roleVal, roleOK := c.Get("role")
	uid, _ := uidVal.(uint)
	role, _ := roleVal.(string)
	if !uidOK || !roleOK {
		pkg.Unauthorized(c, "未登录")
		return
	}
	if uid != uint(id) && role != "admin" {
		pkg.Forbidden(c, "无权修改该用户信息")
		return
	}

	var updateData struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	user, err := h.service.GetUserByID(uint(id))
	if err != nil {
		pkg.NotFound(c, "用户不存在")
		return
	}

	newUsername := user.Username
	newEmail := user.Email
	if updateData.Username != "" {
		if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]{3,32}$`, updateData.Username); !matched {
			pkg.BadRequest(c, "用户名须为3-32位字母、数字或下划线")
			return
		}
		newUsername = updateData.Username
	}
	if updateData.Email != "" {
		newEmail = updateData.Email
	}

	if err := h.service.UpdateProfile(uint(id), newUsername, newEmail); err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, "用户不存在")
			return
		}
		slog.Error("更新用户失败", "error", err, "user_id", id)
		pkg.InternalError(c, "更新用户信息失败")
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
		pkg.BadRequest(c, "无效的ID")
		return
	}

	uidVal, uidOK := c.Get("user_id")
	roleVal, roleOK := c.Get("role")
	uid, _ := uidVal.(uint)
	role, _ := roleVal.(string)
	if !uidOK || !roleOK {
		pkg.Unauthorized(c, "未登录")
		return
	}
	if uid != uint(id) && role != "admin" {
		pkg.Forbidden(c, "无权删除该用户")
		return
	}

	if err := h.service.DeleteUser(uint(id)); err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, "user not found")
			return
		}
		slog.Error("删除用户失败", "error", err, "user_id", id)
		pkg.InternalError(c, "删除用户失败")
		return
	}

	pkg.SuccessWithMessage(c, "删除成功", nil)
}

func (h *UserHandler) AdminCreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "invalid request")
		return
	}
	if !validateUsername(req.Username) {
		pkg.BadRequest(c, "invalid username")
		return
	}
	if !validateEmail(req.Email) {
		pkg.BadRequest(c, "invalid email")
		return
	}
	if req.Role == "" {
		req.Role = "visitor"
	}
	if !validateRole(req.Role) {
		pkg.BadRequest(c, "invalid role")
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
		pkg.BadRequest(c, "create user failed")
		return
	}

	pkg.Success(c, userPayload(user))
}

func (h *UserHandler) AdminUpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		pkg.BadRequest(c, "invalid id")
		return
	}

	var req struct {
		Username *string `json:"username"`
		Password *string `json:"password"`
		Email    *string `json:"email"`
		Role     *string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "invalid request")
		return
	}
	if req.Username != nil && !validateUsername(*req.Username) {
		pkg.BadRequest(c, "invalid username")
		return
	}
	if req.Email != nil && !validateEmail(*req.Email) {
		pkg.BadRequest(c, "invalid email")
		return
	}
	if req.Role != nil && !validateRole(*req.Role) {
		pkg.BadRequest(c, "invalid role")
		return
	}

	user, err := h.service.UpdateAdminUser(uint(id), req.Username, req.Email, req.Role, req.Password)
	if err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, "user not found")
			return
		}
		slog.Error("admin update user failed", "error", err, "user_id", id)
		pkg.BadRequest(c, "update user failed")
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
		slog.Error("获取用户列表失败", "error", err)
		pkg.InternalError(c, "获取用户列表失败")
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
		pkg.BadRequest(c, "角色参数不能为空")
		return
	}

	users, err := h.service.GetUsersByRole(role)
	if err != nil {
		slog.Error("按角色获取用户列表失败", "error", err, "role", role)
		pkg.InternalError(c, "获取用户列表失败")
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
