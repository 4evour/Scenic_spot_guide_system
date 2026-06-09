package pkg

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	mw, _ := newRateLimitMiddlewareWithStopper(limit, window)
	return mw
}

func newRateLimitMiddlewareWithStopper(limit int, window time.Duration) (gin.HandlerFunc, func()) {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	var mu sync.Mutex
	entries := make(map[string]rateLimitEntry)
	stop := make(chan struct{})

	// 后台清理过期条目，避免在请求热路径中遍历全量 map
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				now := time.Now()
				for ip, item := range entries {
					if now.After(item.resetAt) {
						delete(entries, ip)
					}
				}
				mu.Unlock()
			case <-stop:
				return
			}
		}
	}()

	handler := func(c *gin.Context) {
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
	return handler, func() { close(stop) }
}

var rateLimitCleanups []func()

func StopRateLimiters() {
	for _, stop := range rateLimitCleanups {
		stop()
	}
	rateLimitCleanups = nil
}

// RedisRateLimitMiddleware uses Redis INCR + EXPIRE for distributed rate limiting.
// Falls back to the in-memory RateLimitMiddleware when Redis is unavailable.
func RedisRateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	client := GetRedis()
	if client == nil {
		// Redis not configured, fall back to in-memory limiter with stopper
		handler, stopper := newRateLimitMiddlewareWithStopper(limit, window)
		rateLimitCleanups = append(rateLimitCleanups, stopper)
		return handler
	}

	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	return func(c *gin.Context) {
		ctx := context.Background()
		key := fmt.Sprintf("ratelimit:%s", c.ClientIP())

		pipe := client.TxPipeline()
		incr := pipe.Incr(ctx, key)
		pipe.ExpireNX(ctx, key, window)
		if _, err := pipe.Exec(ctx); err != nil {
			slog.Error("Redis rate limit 执行失败，降级到内存限流", "error", err)
			fallback, stopper := newRateLimitMiddlewareWithStopper(limit, window)
			rateLimitCleanups = append(rateLimitCleanups, stopper)
			fallback(c)
			return
		}

		count := incr.Val()
		if count == 1 {
			// First request in window; TTL was set by Expire above.
		}

		if count > int64(limit) {
			ttl, err := client.TTL(ctx, key).Result()
			retryAfter := int64(window.Seconds())
			if err == nil && ttl > 0 {
				retryAfter = int64(ttl.Seconds())
			}

			c.AbortWithStatusJSON(429, Response{
				Code:    429,
				Message: "too many requests",
				Data: gin.H{
					"retry_after_seconds": retryAfter,
				},
			})
			return
		}

		remaining := int64(limit) - count
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Next()
	}
}

func csrfProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		if strings.EqualFold(c.GetHeader("X-Requested-With"), "XMLHttpRequest") {
			c.Next()
			return
		}
		token, _ := c.Cookie("csrf_token")
		if strings.TrimSpace(token) == "" {
			c.AbortWithStatusJSON(403, Response{Code: 403, Message: "missing csrf token"})
			return
		}
		c.Next()
	}
}

func CSRFProtection() gin.HandlerFunc {
	return csrfProtection()
}

func SetCSRFCookie(c *gin.Context) {
	token, err := GenerateToken(0, "csrf", "csrf", 12)
	if err != nil {
		return
	}
	secure := strings.EqualFold(os.Getenv("SCENIC_GUIDE_COOKIE_SECURE"), "true") || strings.EqualFold(os.Getenv("GIN_MODE"), "release")
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("csrf_token", token, 12*3600, "/", "", secure, false)
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
			token, _ = c.Cookie("auth_token")
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
