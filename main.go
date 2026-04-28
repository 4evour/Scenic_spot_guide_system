package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/handler"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
)

func main() {
	fmt.Println("=== 启动景区导览服务 ===")
	
	fmt.Println("步骤1: 加载配置...")
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		return
	}
	fmt.Println("配置加载成功")

	fmt.Println("步骤2: 初始化日志...")
	pkg.InitLogger(cfg.Logging.Level)
	fmt.Println("日志初始化成功")

	fmt.Println("步骤2.5: 初始化JWT...")
	pkg.InitJWT(&cfg.Security)
	fmt.Println("JWT初始化成功")

	fmt.Println("步骤3: 初始化数据库...")
	err = pkg.InitDatabase(&cfg.Database)
	if err != nil {
		fmt.Printf("数据库连接失败: %v\n", err)
		return
	}
	fmt.Println("数据库连接成功")

	fmt.Println("步骤4: 数据库迁移...")
	err = model.AutoMigrate(pkg.DB)
	if err != nil {
		fmt.Printf("数据库迁移失败: %v\n", err)
		return
	}
	fmt.Println("数据库迁移成功")

	fmt.Println("步骤5: 设置路由...")
	r := gin.Default()
	r.Use(gin.Recovery())

	setupHandlers := setupDI()
	handler.SetupRoutes(r, setupHandlers)
	fmt.Println("路由设置成功")

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("步骤6: 启动服务器，监听地址: %s\n", addr)
	err = r.Run(addr)
	if err != nil {
		fmt.Printf("服务器启动失败: %v\n", err)
	}
}

func setupDI() *handler.Handlers {
	scenicSpotRepo := repository.NewScenicSpotRepository(pkg.DB)
	scenicSpotService := service.NewScenicSpotService(scenicSpotRepo)
	scenicSpotHandler := handler.NewScenicSpotHandler(scenicSpotService)

	guideContentRepo := repository.NewGuideContentRepository(pkg.DB)
	guideContentService := service.NewGuideContentService(guideContentRepo)
	guideContentHandler := handler.NewGuideContentHandler(guideContentService)

	tourRouteRepo := repository.NewTourRouteRepository(pkg.DB)
	tourRouteService := service.NewTourRouteService(tourRouteRepo)
	tourRouteHandler := handler.NewTourRouteHandler(tourRouteService)

	visitorQueryRepo := repository.NewVisitorQueryRepository(pkg.DB)
	visitorQueryService := service.NewVisitorQueryService(visitorQueryRepo)
	visitorQueryHandler := handler.NewVisitorQueryHandler(visitorQueryService)

	userRepo := repository.NewUserRepository(pkg.DB)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	aiHandler := handler.NewAIHandler()
	ttsHandler := handler.NewTTSHandler()

	return &handler.Handlers{
		ScenicSpot:   scenicSpotHandler,
		GuideContent: guideContentHandler,
		TourRoute:    tourRouteHandler,
		VisitorQuery: visitorQueryHandler,
		User:         userHandler,
		AI:           aiHandler,
		TTS:          ttsHandler,
	}
}