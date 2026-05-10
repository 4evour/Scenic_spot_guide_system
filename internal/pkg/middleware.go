package pkg

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	var mu sync.Mutex
	entries := make(map[string]rateLimitEntry)

	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()

		mu.Lock()
		entry := entries[key]
		if entry.resetAt.IsZero() || now.After(entry.resetAt) {
			entry = rateLimitEntry{resetAt: now.Add(window)}
		}
		entry.count++
		entries[key] = entry

		for ip, item := range entries {
			if now.After(item.resetAt) {
				delete(entries, ip)
			}
		}

		allowed := entry.count <= limit
		resetAt := entry.resetAt
		mu.Unlock()

		if !allowed {
			c.AbortWithStatusJSON(429, Response{
				Code:    429,
				Message: "too many requests",
				Data: gin.H{
					"retry_after_seconds": int(time.Until(resetAt).Seconds()),
				},
			})
			return
		}

		c.Next()
	}
}

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
