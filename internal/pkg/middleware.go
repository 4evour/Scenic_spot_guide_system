package pkg

import (
	"log/slog"
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

	// 后台清理过期条目，避免在请求热路径中遍历全量 map
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, item := range entries {
				if now.After(item.resetAt) {
					delete(entries, ip)
				}
			}
			mu.Unlock()
		}
	}()

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
		if applyDevAdminBypass(c) {
			c.Next()
			return
		}

		var token string
		// 优先从 Authorization header 获取
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}
		// 回退到 HttpOnly Cookie
		if token == "" {
			token, _ = c.Cookie("auth_token")
		}

		if token == "" {
			c.AbortWithStatusJSON(401, Response{
				Code:    401,
				Message: "未登录，请先登录",
			})
			return
		}

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

func WSTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string
		// 优先从 Sec-WebSocket-Protocol 子协议提取
		protocols := c.GetHeader("Sec-WebSocket-Protocol")
		for _, p := range strings.Split(protocols, ",") {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "auth.token.") {
				token = strings.TrimPrefix(p, "auth.token.")
				break
			}
		}
		// 回退到 query 参数（兼容旧客户端）
		if token == "" {
			token = c.Query("token")
			if token != "" {
				slog.Warn("WebSocket token 通过 URL query 传递，建议迁移至子协议")
			}
		}
		if token == "" {
			c.AbortWithStatusJSON(401, Response{
				Code:    401,
				Message: "missing token",
			})
			return
		}
		claims, err := ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, Response{
				Code:    401,
				Message: "invalid token",
			})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}
