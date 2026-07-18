package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

func SetupRoutes(r *gin.Engine, handlers *Handlers) {
	r.Use(bodySizeMiddleware())
	r.Use(corsMiddleware(handlers.AllowedOrigins))
	r.Use(securityHeaders())
	r.Use(pkg.MetricsMiddleware())

	r.GET("/metrics", pkg.AuthMiddleware(), pkg.AdminMiddleware(), pkg.PrometheusHandler())

	r.Static("/static", "./static")
	r.Any("/vtuber-ws/*path", pkg.WSTokenAuth(), pkg.WSProxyHandler("http://127.0.0.1:12393"))

	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	vueApp := func(c *gin.Context) {
		c.File("./static/vue-app/index.html")
	}
	r.GET("/app", vueApp)
	r.GET("/app/*path", vueApp)
	r.GET("/scan", vueApp) // 二维码扫码落地页
	r.GET("/dashboard", vueApp)
	r.GET("/admin", vueApp)
	r.GET("/digital-human", vueApp)
	r.GET("/map", vueApp)

	api := r.Group("/api/v1")
	api.Use(pkg.LanguageMiddleware())
	api.Use(pkg.CSRFProtection())

	handlers.ScenicSpot.Routes(api)
	handlers.GuideContent.Routes(api)
	handlers.TourRoute.Routes(api)
	if handlers.VisitorExperience != nil {
		handlers.VisitorExperience.Routes(api)
	}
	handlers.VisitorQuery.Routes(api)
	handlers.User.Routes(api)
	handlers.TTS.Routes(api)
	handlers.DigitalHuman.Routes(api)
	handlers.Admin.Routes(api)
	handlers.ScenicProfile.Routes(api)

	// 游客认证路由
	if handlers.Guest != nil {
		handlers.Guest.Routes(api)
	}

	// 会话管理路由
	if handlers.Session != nil {
		handlers.Session.Routes(api)
	}

	// 二维码扫码导览路由
	if handlers.QR != nil {
		handlers.QR.Routes(api)
	}

	// AI Chat 路由（带可选认证 + 自动游客登录）
	if handlers.AI != nil {
		api.GET("/ai/model-health", handlers.AI.ModelHealth)
		chatGroup := api.Group("")
		chatGroup.Use(pkg.OptionalAuthMiddleware())
		if handlers.Guest != nil {
			ensureGuest := pkg.NewEnsureGuestMiddleware(handlers.Guest.CreateGuestFunc())
			chatGroup.Use(ensureGuest.Handle())
		}
		chatGroup.POST("/ai/chat", pkg.RateLimitMiddleware(30, time.Minute), handlers.AI.Chat)
		chatGroup.POST("/ai/multimodal/chat", pkg.RateLimitMiddleware(10, time.Minute), handlers.AI.MultimodalChat)
		chatGroup.POST("/ai/feedback", handlers.AI.Feedback)
		handlers.AI.KnowledgeRoutes(api)
	}

	// 轻量级行为追踪接口（页面访问、用户操作）
	allowedActions := map[string]bool{
		"visit": true, "click": true, "search": true,
		"chat": true, "feedback": true, "voice": true,
	}
	api.POST("/track", getRateLimitMiddleware(30, time.Minute), func(c *gin.Context) {
		var req struct {
			Page    string `json:"page"`
			Action  string `json:"action"`
			Details string `json:"details"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(200, gin.H{"code": 0})
			return
		}
		if req.Page != "" && !isAllowedTrackingPage(req.Page) {
			pkg.BadRequest(c, "invalid page")
			return
		}
		if req.Action != "" && !allowedActions[req.Action] {
			pkg.BadRequest(c, "invalid action")
			return
		}
		// user_id 来自认证会话，不从客户端接受
		var userID uint
		if uid, exists := c.Get("user_id"); exists {
			userID, _ = uid.(uint)
		}
		if s := pkg.GetStatsService(); s != nil {
			if statsSvc, ok := s.(*service.StatsService); ok {
				source := "page_visit"
				if req.Action != "" {
					source = "user_action"
				}
				statsSvc.RecordInteraction(service.InteractionRecord{
					UserID:   userID,
					Query:    req.Page,
					Response: req.Details,
					Category: req.Action,
					Source:   source,
				})
			}
		}
		c.JSON(200, gin.H{"code": 0})
	})

	// OpenAI-compatible endpoint for Open-LLM-VTuber.
	r.POST("/v1/chat/completions", pkg.APIKeyMiddleware(), getRateLimitMiddleware(30, time.Minute), handlers.OpenAIProxy.ChatCompletions)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "scenic guide service is running",
		})
	})
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	// Build lookup set
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With, X-CSRF-Token")
			c.Header("Access-Control-Max-Age", "86400")
		} else {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// maxBodySize is the default maximum request body size (12 MB).
const maxBodySize = 12 << 20

const maxMultimodalBodySize = 64 << 20

// bodySizeMiddleware wraps the request body with a size limit.
func bodySizeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := int64(maxBodySize)
		if c.Request.URL.Path == "/api/v1/ai/multimodal/chat" {
			limit = maxMultimodalBodySize
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "camera=(self), microphone=(self), display-capture=(self), geolocation=(self)")
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		c.Header("Content-Security-Policy", contentSecurityPolicy(c.Request.URL.Path))
		c.Next()
	}
}

func contentSecurityPolicy(path string) string {
	scriptSrc := "script-src 'self' https://webapi.amap.com https://restapi.amap.com"
	if path == "/digital-human" || strings.HasPrefix(path, "/digital-human/") {
		scriptSrc = "script-src 'self' 'unsafe-eval' https://webapi.amap.com https://restapi.amap.com"
	}
	return "default-src 'self'; " + scriptSrc + "; style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; img-src 'self' data: blob: https://webapi.amap.com https://*.amap.com; connect-src 'self' ws: wss: https://webapi.amap.com https://restapi.amap.com; font-src 'self' data: https://cdnjs.cloudflare.com; media-src 'self' blob:;"
}

func isAllowedTrackingPage(page string) bool {
	page = strings.TrimSpace(page)
	if page == "" {
		return true
	}
	if index := strings.IndexAny(page, "?#"); index >= 0 {
		page = page[:index]
	}
	allowedPages := map[string]bool{
		"/": true, "/map": true, "/digital-human": true,
		"/dashboard": true, "/admin": true, "/login": true,
	}
	if allowedPages[page] {
		return true
	}
	return strings.HasPrefix(page, "/admin/")
}

// getRateLimitMiddleware returns the Redis-based rate limiter when Redis is
// connected, otherwise the in-memory rate limiter.
func getRateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	return pkg.RedisRateLimitMiddleware(limit, window)
}

type Handlers struct {
	ScenicSpot        *ScenicSpotHandler
	GuideContent      *GuideContentHandler
	TourRoute         *TourRouteHandler
	VisitorExperience *VisitorExperienceHandler
	VisitorQuery      *VisitorQueryHandler
	User              *UserHandler
	AI                *AIHandler
	TTS               *TTSHandler
	DigitalHuman      *DigitalHumanHandler
	OpenAIProxy       *OpenAIProxyHandler
	Admin             *AdminHandler
	ScenicProfile     *ScenicProfileHandler
	Guest             *GuestHandler
	Session           *SessionHandler
	QR                *QRHandler
	AllowedOrigins    []string
}
