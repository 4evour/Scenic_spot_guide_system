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

	fmt.Println("步骤4.5: 初始化RAG知识库...")
	ragService := initRAG(cfg)
	if ragService != nil {
		fmt.Println("RAG知识库初始化成功")
	} else {
		fmt.Println("RAG知识库初始化失败，将使用基础AI服务")
	}

	fmt.Println("步骤5: 设置路由...")
	r := gin.Default()
	r.Use(gin.Recovery())

	setupHandlers := setupDI(ragService)
	handler.SetupRoutes(r, setupHandlers)
	fmt.Println("路由设置成功")

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("步骤6: 启动服务器，监听地址: %s\n", addr)
	err = r.Run(addr)
	if err != nil {
		fmt.Printf("服务器启动失败: %v\n", err)
	}
}

func initRAG(cfg *config.Config) *service.RAGService {
	if cfg.AI.APIKey == "" {
		fmt.Println("警告: AI API Key未配置，跳过RAG初始化")
		return nil
	}

	knowledgeRepo := repository.NewKnowledgeRepository(pkg.DB)

	var embeddingProvider service.EmbeddingProvider
	// 暂时注释掉embedding provider的使用，先用BM25测试
	/*
		if cfg.Embedding.APIKey != "" {
			embeddingProvider = service.NewQwenEmbeddingProvider(&cfg.Embedding)
			if embeddingProvider.IsAvailable() {
				fmt.Printf("Embedding Provider [%s] 可用\n", embeddingProvider.Name())
			} else {
				fmt.Println("Embedding Provider不可用，将使用BM25")
				embeddingProvider = nil
			}
		} else {
			fmt.Println("未配置Embedding API Key，将使用BM25")
		}
	*/
	fmt.Println("使用BM25作为fallback方案")
	embeddingProvider = nil

	ragService := service.NewRAGService(knowledgeRepo, cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL, embeddingProvider)

	var err error
	fmt.Println("清空旧知识库...")
	if err = ragService.DeleteAllKnowledge(); err != nil {
		fmt.Printf("清空知识库失败: %v\n", err)
	}

	fmt.Println("开始加载新知识文件...")
	err = ragService.LoadKnowledgeFromFile("./knowledge/lingshan_chunks.jsonl")
	if err != nil {
		fmt.Printf("加载知识库失败: %v\n", err)
		return ragService
	}
	fmt.Println("知识库加载完成")

	newCount, err := knowledgeRepo.Count()
	if err == nil {
		fmt.Printf("当前知识库共有 %d 条知识\n", newCount)
	}

	return ragService
}

func setupDI(ragService *service.RAGService) *handler.Handlers {
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

	aiHandler := handler.NewAIHandler(ragService)
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
