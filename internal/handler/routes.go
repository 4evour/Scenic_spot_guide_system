package handler

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, handlers *Handlers) {
	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	api := r.Group("/api/v1")

	handlers.ScenicSpot.Routes(api)
	handlers.GuideContent.Routes(api)
	handlers.TourRoute.Routes(api)
	handlers.VisitorQuery.Routes(api)
	handlers.User.Routes(api)
	handlers.AI.Routes(api)
	handlers.TTS.Routes(api)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "景区导览服务运行正常",
		})
	})
}

type Handlers struct {
	ScenicSpot   *ScenicSpotHandler
	GuideContent *GuideContentHandler
	TourRoute    *TourRouteHandler
	VisitorQuery *VisitorQueryHandler
	User         *UserHandler
	AI           *AIHandler
	TTS          *TTSHandler
}
