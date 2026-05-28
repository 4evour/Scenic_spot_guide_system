package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "服务启动失败: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	pkg.InitLogger(cfg.Logging.Level)
	slog.Info("启动景区导览服务", "log_level", cfg.Logging.Level)
	slog.Info("配置加载成功")

	slog.Info("初始化 JWT")
	if err := pkg.InitJWT(&cfg.Security); err != nil {
		return fmt.Errorf("JWT 初始化失败: %w", err)
	}
	slog.Info("JWT 初始化成功")

	if os.Getenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS") != "" {
		slog.Warn("安全警告: 开发管理员旁路已启用，生产环境请务必禁用!", "env_var", "SCENIC_GUIDE_DEV_ADMIN_BYPASS")
	}

	slog.Info("初始化数据库")
	err = pkg.InitDatabase(&cfg.Database)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	slog.Info("数据库连接成功", "driver", cfg.Database.Driver)

	slog.Info("执行数据库迁移")
	err = model.AutoMigrate(pkg.DB)
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	slog.Info("数据库迁移成功")

	ensureAdminUser()

	// 加载景区配置
	scenicID := os.Getenv("SCENIC_GUIDE_SCENIC_ID")
	if scenicID == "" {
		scenicID = "lingshan"
	}
	scenicProfile, err := config.LoadScenicProfile(scenicID)
	if err != nil {
		slog.Warn("加载景区配置失败，使用默认配置", "scenic_id", scenicID, "error", err)
	} else {
		slog.Info("景区配置加载成功", "scenic_id", scenicID, "name", scenicProfile.Name)
	}

	slog.Info("初始化 RAG 知识库")
	ragService := initRAG(cfg, scenicProfile)
	if ragService != nil {
		slog.Info("RAG 知识库初始化成功")
	} else {
		slog.Warn("RAG 知识库初始化失败，将使用基础 AI 服务")
	}

	slog.Info("设置路由")
	r := gin.Default()
	r.Use(gin.Recovery())

	setupHandlers := setupDI(ragService, cfg.Security.TokenExpireHours)
	handler.SetupRoutes(r, setupHandlers)
	slog.Info("路由设置成功")

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	slog.Info("启动 HTTP 服务", "addr", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
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
		slog.Info("收到退出信号，正在关闭服务", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("服务器关闭失败: %w", err)
	}
	slog.Info("服务已优雅关闭")
	return nil
}

func ensureAdminUser() {
	username := os.Getenv("SCENIC_GUIDE_ADMIN_USERNAME")
	password := os.Getenv("SCENIC_GUIDE_ADMIN_PASSWORD")

	if username == "" {
		username = "admin"
	}
	if password == "" {
		slog.Warn("未设置管理员密码，跳过自动创建管理员账号；如需启动时创建管理员，请设置 SCENIC_GUIDE_ADMIN_PASSWORD 环境变量")
		return
	}

	userRepo := repository.NewUserRepository(pkg.DB)
	admins, _ := userRepo.FindByRole("admin")
	if len(admins) > 0 {
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("生成管理员密码失败", "error", err)
		return
	}
	admin := &model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    username + "@scenic.local",
		Role:     "admin",
	}
	if err := userRepo.Create(admin); err != nil {
		slog.Error("自动创建管理员失败", "error", err)
		return
	}
	slog.Info("已自动创建管理员账号", "username", username)
}

func initRAG(cfg *config.Config, profile *config.ScenicProfile) *service.RAGService {
	if cfg.AI.APIKey == "" {
		slog.Warn("AI API Key 未配置，RAG 将使用本地检索和规则兜底")
	}

	knowledgeRepo := repository.NewKnowledgeRepository(pkg.DB)

	var embeddingProvider service.EmbeddingProvider
	if cfg.Embedding.APIKey != "" {
		embeddingProvider = service.NewQwenEmbeddingProvider(&cfg.Embedding)
		if embeddingProvider.IsAvailable() {
			slog.Info("Embedding Provider 可用", "provider", embeddingProvider.Name())
		} else {
			slog.Warn("Embedding Provider 不可用，将使用 BM25")
			embeddingProvider = nil
		}
	} else {
		slog.Info("未配置 Embedding API Key，将使用 BM25")
	}

	ragService := service.NewRAGService(knowledgeRepo, cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL, embeddingProvider, profile)

	// 检查现有知识库
	count, _ := knowledgeRepo.Count()
	if count > 0 {
		slog.Info("知识库已有数据，无需重新加载", "count", count)
	} else {
		slog.Info("知识库为空，开始加载默认知识库")
		// 从景区配置读取知识库路径，无配置时使用默认路径
		knowledgeFiles := []string{
			"./knowledge/lingshan_chunks.jsonl",
			"./knowledge/real/lingshan_real_chunks.jsonl",
		}
		if profile != nil {
			knowledgeFiles = nil
			if profile.Knowledge.ChunksFile != "" {
				knowledgeFiles = append(knowledgeFiles, profile.Knowledge.ChunksFile)
			}
			if profile.Knowledge.RealChunksFile != "" {
				knowledgeFiles = append(knowledgeFiles, profile.Knowledge.RealChunksFile)
			}
		}
		for _, file := range knowledgeFiles {
			if err := ragService.LoadKnowledgeFromFile(file); err != nil {
				slog.Error("加载知识库文件失败", "file", file, "error", err)
			}
		}
		count, _ = knowledgeRepo.Count()
		slog.Info("知识库加载完成", "count", count)
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
