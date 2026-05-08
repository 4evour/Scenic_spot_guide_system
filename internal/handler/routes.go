package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, handlers *Handlers) {
	r.Static("/static", "./static")
	r.Any("/vtuber-ws/*path", vtuberWebSocketProxy("http://127.0.0.1:12393"))

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

	api := r.Group("/api/v1")

	handlers.ScenicSpot.Routes(api)
	handlers.GuideContent.Routes(api)
	handlers.TourRoute.Routes(api)
	handlers.VisitorQuery.Routes(api)
	handlers.User.Routes(api)
	handlers.AI.Routes(api)
	handlers.TTS.Routes(api)
	handlers.DigitalHuman.Routes(api)
	handlers.Admin.Routes(api)

	// OpenAI-compatible endpoint for Open-LLM-VTuber.
	r.POST("/v1/chat/completions", handlers.OpenAIProxy.ChatCompletions)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "scenic guide service is running",
		})
	})
}

func vtuberWebSocketProxy(target string) gin.HandlerFunc {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/vtuber-ws")
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

type Handlers struct {
	ScenicSpot   *ScenicSpotHandler
	GuideContent *GuideContentHandler
	TourRoute    *TourRouteHandler
	VisitorQuery *VisitorQueryHandler
	User         *UserHandler
	AI           *AIHandler
	TTS          *TTSHandler
	DigitalHuman *DigitalHumanHandler
	OpenAIProxy  *OpenAIProxyHandler
	Admin        *AdminHandler
}
