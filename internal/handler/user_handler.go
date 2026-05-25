package handler

import (
	"log/slog"
	"strconv"
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

func (h *UserHandler) Register(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
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

	pkg.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"token":    token,
	})
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

	currentUserID, _ := c.Get("user_id")
	currentRole, _ := c.Get("role")
	if currentUserID.(uint) != uint(id) && currentRole.(string) != "admin" {
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

	currentUserID, _ := c.Get("user_id")
	currentRole, _ := c.Get("role")
	if currentUserID.(uint) != uint(id) && currentRole.(string) != "admin" {
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

	if updateData.Username != "" {
		user.Username = updateData.Username
	}
	if updateData.Email != "" {
		user.Email = updateData.Email
	}

	if err := h.service.UpdateUser(user); err != nil {
		slog.Error("更新用户失败", "error", err, "user_id", id)
		pkg.InternalError(c, "更新用户信息失败")
		return
	}

	pkg.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
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

	currentUserID, _ := c.Get("user_id")
	currentRole, _ := c.Get("role")
	if currentUserID.(uint) != uint(id) && currentRole.(string) != "admin" {
		pkg.Forbidden(c, "无权删除该用户")
		return
	}

	if err := h.service.DeleteUser(uint(id)); err != nil {
		slog.Error("删除用户失败", "error", err, "user_id", id)
		pkg.InternalError(c, "删除用户失败")
		return
	}

	pkg.SuccessWithMessage(c, "删除成功", nil)
}

func (h *UserHandler) GetAllUsers(c *gin.Context) {
	users, err := h.service.GetAllUsers()
	if err != nil {
		slog.Error("获取用户列表失败", "error", err)
		pkg.InternalError(c, "获取用户列表失败")
		return
	}

	result := make([]gin.H, 0, len(users))
	for _, user := range users {
		result = append(result, gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		})
	}

	pkg.Success(c, result)
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
		result = append(result, gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		})
	}

	pkg.Success(c, result)
}

func (h *UserHandler) Routes(r *gin.RouterGroup) {
	r.POST("/register", pkg.RateLimitMiddleware(5, time.Minute), h.Register)
	r.POST("/login", pkg.RateLimitMiddleware(10, time.Minute), h.Login)

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
	}
}
