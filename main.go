package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/handler"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "服务启动失败: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("=== 启动景区导览服务 ===")

	fmt.Println("步骤1: 加载配置...")
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	fmt.Println("配置加载成功")

	fmt.Println("步骤2: 初始化日志...")
	pkg.InitLogger(cfg.Logging.Level)
	fmt.Println("日志初始化成功")

	fmt.Println("步骤2.5: 初始化JWT...")
	if err := pkg.InitJWT(&cfg.Security); err != nil {
		return fmt.Errorf("JWT 初始化失败: %w", err)
	}
	fmt.Println("JWT初始化成功")

	fmt.Println("步骤3: 初始化数据库...")
	err = pkg.InitDatabase(&cfg.Database)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	fmt.Println("数据库连接成功")

	fmt.Println("步骤4: 数据库迁移...")
	err = model.AutoMigrate(pkg.DB)
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
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

	setupHandlers := setupDI(ragService, cfg.Security.TokenExpireHours)
	handler.SetupRoutes(r, setupHandlers)
	fmt.Println("路由设置成功")

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("步骤6: 启动服务器，监听地址: %s\n", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("服务器启动失败: %w", err)
	case sig := <-shutdownSignals:
		fmt.Printf("收到退出信号: %s，正在关闭服务...\n", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("服务器关闭失败: %w", err)
	}
	fmt.Println("服务已优雅关闭")
	return nil
}

func initRAG(cfg *config.Config) *service.RAGService {
	if cfg.AI.APIKey == "" {
		fmt.Println("警告: AI API Key未配置，跳过RAG初始化")
	}

	knowledgeRepo := repository.NewKnowledgeRepository(pkg.DB)

	var embeddingProvider service.EmbeddingProvider
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

	ragService := service.NewRAGService(knowledgeRepo, cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL, embeddingProvider)

	// 检查现有知识库
	count, _ := knowledgeRepo.Count()
	if count > 0 {
		fmt.Printf("当前知识库已有 %d 条知识，无需重新加载\n", count)
	} else {
		fmt.Println("知识库为空，开始加载知识库...")
		err := ragService.LoadKnowledgeFromFile("./knowledge/lingshan_chunks.jsonl")
		if err != nil {
			fmt.Printf("加载知识库失败: %v\n", err)
		} else {
			count, _ = knowledgeRepo.Count()
			fmt.Printf("知识库加载完成，共 %d 条知识\n", count)
		}
	}

	return ragService
}

func setupDI(ragService *service.RAGService, tokenExpireHours int) *handler.Handlers {
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
	userHandler := handler.NewUserHandler(userService, tokenExpireHours)

	aiHandler := handler.NewAIHandler(ragService)
	ttsHandler := handler.NewTTSHandler()
	digitalHumanHandler := handler.NewDigitalHumanHandler(ragService, tourRouteService, visitorQueryService)
	openAIProxyHandler := handler.NewOpenAIProxyHandler(ragService)

	// 初始化统计服务
	interactionRepo := repository.NewInteractionRepository(pkg.DB)
	knowledgeRepo := repository.NewKnowledgeRepository(pkg.DB)
	settingRepo := repository.NewSystemSettingRepository(pkg.DB)
	dhConfigRepo := repository.NewDigitalHumanConfigRepository(pkg.DB)
	statsService := service.NewStatsService(interactionRepo, settingRepo, dhConfigRepo, knowledgeRepo)
	pkg.StatsService = statsService
	adminHandler := handler.NewAdminHandler(statsService)

	return &handler.Handlers{
		ScenicSpot:   scenicSpotHandler,
		GuideContent: guideContentHandler,
		TourRoute:    tourRouteHandler,
		VisitorQuery: visitorQueryHandler,
		User:         userHandler,
		AI:           aiHandler,
		TTS:          ttsHandler,
		DigitalHuman: digitalHumanHandler,
		OpenAIProxy:  openAIProxyHandler,
		Admin:        adminHandler,
	}
}
