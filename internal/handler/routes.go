package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

func SetupRoutes(r *gin.Engine, handlers *Handlers) {
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
	r.GET("/dashboard", vueApp)
	r.GET("/admin", vueApp)
	r.GET("/digital-human", vueApp)
	r.GET("/map", vueApp)

	api := r.Group("/api/v1")
	api.Use(pkg.CSRFProtection())

	handlers.ScenicSpot.Routes(api)
	handlers.GuideContent.Routes(api)
	handlers.TourRoute.Routes(api)
	handlers.VisitorQuery.Routes(api)
	handlers.User.Routes(api)
	handlers.AI.Routes(api)
	handlers.TTS.Routes(api)
	handlers.DigitalHuman.Routes(api)
	handlers.Admin.Routes(api)
	handlers.ScenicProfile.Routes(api)

	// 轻量级行为追踪接口（页面访问、用户操作）
	api.POST("/track", func(c *gin.Context) {
		var req struct {
			Page    string `json:"page"`
			Action  string `json:"action"`
			Details string `json:"details"`
			UserID  uint   `json:"user_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(200, gin.H{"code": 0})
			return
		}
		if statsSvc := pkg.GetStatsService(); statsSvc != nil {
			source := "page_visit"
			if req.Action != "" {
				source = "user_action"
			}
			statsSvc.RecordInteraction(service.InteractionRecord{
				UserID:   req.UserID,
				Query:    req.Page,
				Response: req.Details,
				Category: req.Action,
				Source:   source,
			})
		}
		c.JSON(200, gin.H{"code": 0})
	})

	// OpenAI-compatible endpoint for Open-LLM-VTuber.
	r.POST("/v1/chat/completions", getRateLimitMiddleware(30, time.Minute), handlers.OpenAIProxy.ChatCompletions)

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
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With")
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

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "camera=(self), microphone=(self), display-capture=(self), geolocation=()")
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' https://webapi.amap.com https://restapi.amap.com; style-src 'self' https://cdnjs.cloudflare.com; img-src 'self' data: blob: https://webapi.amap.com https://*.amap.com; connect-src 'self' ws: wss: https://webapi.amap.com https://restapi.amap.com; font-src 'self' data: https://cdnjs.cloudflare.com; media-src 'self' blob:;")
		c.Next()
	}
}

// getRateLimitMiddleware returns the Redis-based rate limiter when Redis is
// connected, otherwise the in-memory rate limiter.
func getRateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	return pkg.RedisRateLimitMiddleware(limit, window)
}

type Handlers struct {
	ScenicSpot     *ScenicSpotHandler
	GuideContent   *GuideContentHandler
	TourRoute      *TourRouteHandler
	VisitorQuery   *VisitorQueryHandler
	User           *UserHandler
	AI             *AIHandler
	TTS            *TTSHandler
	DigitalHuman   *DigitalHumanHandler
	OpenAIProxy    *OpenAIProxyHandler
	Admin          *AdminHandler
	ScenicProfile  *ScenicProfileHandler
	AllowedOrigins []string
}
