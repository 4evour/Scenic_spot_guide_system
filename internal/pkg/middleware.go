package pkg

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, Response{
				Code:    401,
				Message: "未登录，请先登录",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(401, Response{
				Code:    401,
				Message: "token格式错误",
			})
			return
		}

		token := parts[1]
		claims, err := ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, Response{
				Code:    401,
				Message: "token无效或已过期",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.AbortWithStatusJSON(403, Response{
				Code:    403,
				Message: "权限不足，需要管理员权限",
			})
			return
		}
		c.Next()
	}
}