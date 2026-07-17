package pkg

import (
	"context"
	"crypto/sha256"
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

// csrfExemptPaths 列出无需 CSRF 校验的认证类端点（用户尚无 CSRF cookie）。
// 使用完整的注册路径(api/v1 前缀),配合 isCSRFExempt 的精确匹配,
// 避免 /api/v1/admin/x/login 之类同后缀路径被误豁免。
var csrfExemptPaths = []string{
	"/api/v1/auth/guest-login",
	"/api/v1/login",
	"/api/v1/register",
}

func isCSRFExempt(path string) bool {
	for _, p := range csrfExemptPaths {
		// 精确匹配,避免 /api/v1/admin/evil/login 之类同后缀路径被误豁免。
		if path == p {
			return true
		}
	}
	return false
}

func csrfProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		// 认证类端点（登录/注册/游客登录）豁免 CSRF
		if isCSRFExempt(c.Request.URL.Path) {
			c.Next()
			return
		}
		// Double-submit cookie:严格校验 X-CSRF-Token header 必须与 csrf_token cookie 一致。
		// 此前此处有一个 "带 X-Requested-With: XMLHttpRequest 头即放行" 的分支,该头完全由
		// 客户端可控,攻击者跨站携带受害者 cookie 发起请求时只需加这个头即可绕过 CSRF,
		// 使双重提交形同虚设。现已移除——配合 cookie 的 SameSite=Strict 形成纵深防御。
		cookieToken, _ := c.Cookie("csrf_token")
		if strings.TrimSpace(cookieToken) == "" {
			c.AbortWithStatusJSON(403, Response{Code: 403, Message: "missing csrf token"})
			return
		}
		// Double-submit cookie: header must match cookie value
		headerToken := c.GetHeader("X-CSRF-Token")
		if headerToken == "" || headerToken != cookieToken {
			c.AbortWithStatusJSON(403, Response{Code: 403, Message: "invalid csrf token"})
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

// APIKeyMiddleware verifies requests carry a valid API key via X-API-Key header.
// Intended for internal service-to-service endpoints (e.g. OpenAI proxy).
func APIKeyMiddleware() gin.HandlerFunc {
	expectedKey := os.Getenv("SCENIC_GUIDE_API_KEY")
	return func(c *gin.Context) {
		if expectedKey == "" {
			c.AbortWithStatusJSON(403, Response{Code: 403, Message: "API key not configured on server"})
			return
		}
		provided := c.GetHeader("X-API-Key")
		if provided == "" {
			authHeader := c.GetHeader("Authorization")
			if parts := strings.SplitN(authHeader, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				provided = strings.TrimSpace(parts[1])
			}
		}
		if provided != expectedKey {
			c.AbortWithStatusJSON(401, Response{Code: 401, Message: "invalid or missing API key"})
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
				Message: T(c, "err_unauthorized"),
			})
			return
		}

		claims, err := ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, Response{
				Code:    401,
				Message: T(c, "err_token_expired"),
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
				Message: T(c, "err_forbidden"),
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
		// 回退到 HttpOnly Cookie。此前还有一个 c.Query("token") 回退分支,
		// 它会把 JWT 暴露在 URL(访问日志/代理日志可见),且当前前端已改用 cookie,
		// 故移除该回退,避免凭据经 URL 泄露。
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

// OptionalAuthMiddleware 尝试解析 JWT，成功则注入用户信息，失败则继续处理（不中断请求）
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if applyDevAdminBypass(c) {
			c.Next()
			return
		}

		var token string
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}
		if token == "" {
			token, _ = c.Cookie("auth_token")
		}
		if token == "" {
			c.Next()
			return
		}

		claims, err := ParseToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// GuestCreatorFunc 游客创建函数类型（避免 pkg 和 service 循环依赖）
type GuestCreatorFunc func(fingerprint string) (userID uint, username, displayName, role, token string, err error)

// EnsureGuestMiddleware 确保每个请求都有用户身份
type EnsureGuestMiddleware struct {
	CreateGuest GuestCreatorFunc
}

func NewEnsureGuestMiddleware(createGuest GuestCreatorFunc) *EnsureGuestMiddleware {
	return &EnsureGuestMiddleware{CreateGuest: createGuest}
}

func (m *EnsureGuestMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if uid, exists := c.Get("user_id"); exists {
			if id, ok := uid.(uint); ok && id > 0 {
				c.Next()
				return
			}
		}

		fp := c.ClientIP() + "|" + c.GetHeader("User-Agent")
		fingerprint := sha256Hex(fp)

		userID, username, displayName, role, token, err := m.CreateGuest(fingerprint)
		if err != nil {
			slog.Warn("自动创建游客失败", "error", err)
			c.Next()
			return
		}

		secureCookie := strings.EqualFold(os.Getenv("SCENIC_GUIDE_COOKIE_SECURE"), "true") || strings.EqualFold(os.Getenv("GIN_MODE"), "release")
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("auth_token", token, 24*3600, "/", "", secureCookie, true)
		SetCSRFCookie(c)

		c.Set("user_id", userID)
		c.Set("username", username)
		c.Set("display_name", displayName)
		c.Set("role", role)

		slog.Debug("自动创建游客账号", "user_id", userID, "username", username)
		c.Next()
	}
}

func sha256Hex(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
