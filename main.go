package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	// 安全护栏:dev 构建(含管理员旁路后门)禁止在 release 模式启动,
	// 防止 -tags dev 编译的二进制被误部署到生产。
	if pkg.IsDevBuild && strings.EqualFold(os.Getenv("GIN_MODE"), "release") {
		return fmt.Errorf("安全拒绝: 当前为 dev 构建(含管理员认证旁路),禁止在 GIN_MODE=release 下运行。请使用默认构建(go build .)")
	}

	slog.Info("初始化 JWT")
	if err := pkg.InitJWT(&cfg.Security); err != nil {
		return fmt.Errorf("JWT 初始化失败: %w", err)
	}
	slog.Info("JWT 初始化成功")

	if os.Getenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS") != "" || os.Getenv("SCENIC_GUIDE_DEV_ALLOW_BYPASS") != "" {
		slog.Warn("安全警告: 检测到开发管理员旁路环境变量。旁路需同时设置 SCENIC_GUIDE_DEV_ADMIN_BYPASS 与 SCENIC_GUIDE_DEV_ALLOW_BYPASS 才生效,且仅限本地开发!",
			"main_var", "SCENIC_GUIDE_DEV_ADMIN_BYPASS", "confirm_var", "SCENIC_GUIDE_DEV_ALLOW_BYPASS")
	}

	if err := pkg.InitRedis(&cfg.Redis); err != nil {
		slog.Warn("Redis 连接失败，将使用内存限流器", "error", err)
	}

	slog.Info("初始化数据库")
	err = pkg.InitDatabase(&cfg.Database)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	slog.Info("数据库连接成功", "driver", cfg.Database.Driver)

	slog.Info("执行数据库迁移")
	err = model.AutoMigrate(pkg.GetDB())
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

	// 配置可信代理:默认不信任 X-Forwarded-For(关闭后 Gin 只取 RemoteAddr 作为客户端 IP,
	// 防止攻击者伪造 XFF 头绕过基于 IP 的限流)。部署在反代后时,运维可通过
	// SCENIC_GUIDE_TRUSTED_PROXIES 显式指定可信代理 IP(逗号分隔),以正确还原真实客户端 IP。
	if trusted := strings.TrimSpace(os.Getenv("SCENIC_GUIDE_TRUSTED_PROXIES")); trusted != "" {
		proxies := strings.Split(trusted, ",")
		for i := range proxies {
			proxies[i] = strings.TrimSpace(proxies[i])
		}
		if err := r.SetTrustedProxies(proxies); err != nil {
			return fmt.Errorf("设置可信代理失败: %w", err)
		}
		slog.Info("已配置可信代理", "proxies", proxies)
	} else {
		// 显式关闭 XFF 信任,防止伪造 IP 绕过限流。
		if err := r.SetTrustedProxies(nil); err != nil {
			return fmt.Errorf("关闭可信代理失败: %w", err)
		}
		slog.Info("未配置可信代理,已关闭 X-Forwarded-For 信任(基于 IP 的限流将以 RemoteAddr 为准)")
	}

	setupHandlers := setupDI(ragService, cfg.Security.TokenExpireHours, cfg.Security.AllowedOrigins, scenicProfile)
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
	pkg.StopRateLimiters()
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

	userRepo := repository.NewUserRepository(pkg.GetDB())
	admins, err := userRepo.FindByRole("admin")
	if err != nil {
		slog.Error("查询管理员账号失败", "error", err)
		return
	}
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

	knowledgeRepo := repository.NewKnowledgeRepository(pkg.GetDB())

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

	count, _ := knowledgeRepo.Count()
	slog.Info("开始补齐默认知识库", "existing_count", count)
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
	slog.Info("知识库补齐完成", "count", count)

	return ragService
}

func setupDI(ragService *service.RAGService, tokenExpireHours int, allowedOrigins []string, scenicProfile *config.ScenicProfile) *handler.Handlers {
	db := pkg.GetDB()

	// 初始化统计服务（先于 handler 创建，以便注入）
	interactionRepo := repository.NewInteractionRepository(db)
	knowledgeRepo := repository.NewKnowledgeRepository(db)
	settingRepo := repository.NewSystemSettingRepository(db)
	dhConfigRepo := repository.NewDigitalHumanConfigRepository(db)
	statsService := service.NewStatsService(interactionRepo, settingRepo, dhConfigRepo, knowledgeRepo)
	pkg.SetStatsService(statsService)

	// 会话持久化仓储
	chatSessionRepo := repository.NewChatSessionRepository(db)
	chatMessageRepo := repository.NewChatMessageRepository(db)

	// 会话持久化服务
	chatSessionService := service.NewChatSessionService(chatSessionRepo, chatMessageRepo)
	insightService := service.NewVisitorInsightService(db, ragService)

	// 将会话持久化服务注入 RAGService
	if ragService != nil {
		ragService.SetChatSessionService(chatSessionService)
	}

	scenicSpotRepo := repository.NewScenicSpotRepository(db)
	scenicSpotService := service.NewScenicSpotService(scenicSpotRepo)
	scenicSpotHandler := handler.NewScenicSpotHandler(scenicSpotService)

	guideContentRepo := repository.NewGuideContentRepository(db)
	guideContentService := service.NewGuideContentService(guideContentRepo)
	guideContentHandler := handler.NewGuideContentHandler(guideContentService)

	tourRouteRepo := repository.NewTourRouteRepository(db)
	tourRouteService := service.NewTourRouteService(tourRouteRepo)
	tourRouteHandler := handler.NewTourRouteHandler(tourRouteService)

	visitorQueryRepo := repository.NewVisitorQueryRepository(db)
	visitorQueryService := service.NewVisitorQueryService(visitorQueryRepo)
	visitorQueryHandler := handler.NewVisitorQueryHandler(visitorQueryService)

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService, tokenExpireHours)

	// 游客服务
	guestService := service.NewGuestService(userRepo, chatSessionRepo, tokenExpireHours)
	guestHandler := handler.NewGuestHandler(guestService, tokenExpireHours)

	// 会话管理 Handler
	sessionHandler := handler.NewSessionHandler(chatSessionService)

	// 二维码扫码导览 Handler
	qrHandler := handler.NewQRHandler(scenicSpotService, ragService, statsService)

	aiHandler := handler.NewAIHandler(ragService, statsService, insightService)
	ttsHandler := handler.NewTTSHandler()
	digitalHumanHandler := handler.NewDigitalHumanHandler(ragService, tourRouteService, statsService, insightService)
	openAIProxyHandler := handler.NewOpenAIProxyHandler(ragService, statsService)
	adminHandler := handler.NewAdminHandler(statsService, "docs/eval-results", insightService)
	scenicProfileHandler := handler.NewScenicProfileHandler(scenicProfile)

	// Default allowed origins for local development
	origins := allowedOrigins
	if len(origins) == 0 {
		origins = []string{
			"http://127.0.0.1:8080",
			"http://localhost:8080",
			"http://127.0.0.1:5173",
			"http://localhost:5173",
		}
	}

	return &handler.Handlers{
		ScenicSpot:     scenicSpotHandler,
		GuideContent:   guideContentHandler,
		TourRoute:      tourRouteHandler,
		VisitorQuery:   visitorQueryHandler,
		User:           userHandler,
		AI:             aiHandler,
		TTS:            ttsHandler,
		DigitalHuman:   digitalHumanHandler,
		OpenAIProxy:    openAIProxyHandler,
		Admin:          adminHandler,
		ScenicProfile:  scenicProfileHandler,
		Guest:          guestHandler,
		Session:        sessionHandler,
		QR:             qrHandler,
		AllowedOrigins: origins,
	}
}
