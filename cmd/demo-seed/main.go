package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	configDir := flag.String("config", "./configs", "配置目录")
	adminPassword := flag.String("admin-password", "", "演示管理员密码 (必填)")
	flag.Parse()

	if *adminPassword == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --admin-password is required. Do not use default passwords.")
		fmt.Fprintln(os.Stderr, "Example: go run ./cmd/demo-seed --admin-password YourSecurePassword123")
		os.Exit(1)
	}

	if err := seedDemoData(*configDir, *adminPassword); err != nil {
		fmt.Printf("演示数据初始化失败: %v\n", err)
		os.Exit(1)
		return
	}
	fmt.Println("演示数据初始化完成")
}

func seedDemoData(configDir, adminPassword string) error {
	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	if err := pkg.InitDatabase(&cfg.Database); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	if err := model.AutoMigrate(pkg.GetDB()); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	if err := seedUsers(adminPassword); err != nil {
		return err
	}
	if err := seedScenicSpots(); err != nil {
		return err
	}
	if err := seedTourRoutes(); err != nil {
		return err
	}
	if err := seedInteractions(); err != nil {
		return err
	}

	rag := service.NewRAGService(repository.NewKnowledgeRepository(pkg.GetDB()), "", "", "", nil, nil)
	if err := seedKnowledgeFiles(rag, []string{
		"./knowledge/lingshan_chunks.jsonl",
		"./knowledge/real/lingshan_real_chunks.jsonl",
	}); err != nil {
		return err
	}
	return nil
}

func seedKnowledgeFiles(rag *service.RAGService, files []string) error {
	for _, file := range files {
		if err := rag.LoadKnowledgeFromFile(file); err != nil {
			return fmt.Errorf("导入演示知识库失败: %w", err)
		}
	}
	return nil
}

func seedUsers(adminPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成管理员密码失败: %w", err)
	}

	admin := model.User{
		Username: "admin",
		Password: string(hash),
		Email:    "admin@example.com",
		Role:     "admin",
	}
	if err := pkg.GetDB().Where("username = ?", admin.Username).
		Assign(model.User{Password: admin.Password, Email: admin.Email, Role: admin.Role}).
		FirstOrCreate(&admin).Error; err != nil {
		return fmt.Errorf("写入演示管理员失败: %w", err)
	}

	visitor := model.User{
		Username: "visitor",
		Password: string(hash),
		Email:    "visitor@example.com",
		Role:     "visitor",
	}
	if err := pkg.GetDB().Where("username = ?", visitor.Username).
		Assign(model.User{Password: visitor.Password, Email: visitor.Email, Role: visitor.Role}).
		FirstOrCreate(&visitor).Error; err != nil {
		return fmt.Errorf("写入演示游客失败: %w", err)
	}
	return nil
}

func seedScenicSpots() error {
	spots := []model.ScenicSpot{
		{
			Name:        "灵山大佛",
			Description: "高88米的青铜立佛，是灵山胜境核心景点。",
			Location:    "大佛广场",
			Category:    "核心景点",
			Rating:      4.9,
			Price:       0,
			QRCode:      "SPOT-0001",
			QREnabled:   true,
			QRIntroText: "[joy]欢迎来到灵山大佛。眼前这尊释迦牟尼青铜立像高88米，面向太湖，是灵山胜境最醒目的核心地标。参观时可以从大佛广场仰望整体造像，再沿中轴线了解佛教文化、礼佛动线和太湖山水共同构成的庄严景观。",
		},
		{
			Name:        "九龙灌浴",
			Description: "再现释迦牟尼诞生时九龙沐浴的动态音乐喷泉演出。",
			Location:    "佛足坛前",
			Category:    "演艺体验",
			Rating:      4.8,
			Price:       0,
			QRCode:      "SPOT-0002",
			QREnabled:   true,
			QRIntroText: "[joy]欢迎来到九龙灌浴。这里用大型动态音乐群雕和喷泉演绎佛陀诞生时“花开见佛、九龙灌浴”的故事，是灵山胜境很有代表性的演艺体验。演出时段请以景区现场公告为准，建议提前到达广场区域，选择视野开阔的位置观看。",
		},
		{
			Name:        "灵山梵宫",
			Description: "汇集东阳木雕、琉璃、油画等传统工艺的佛教艺术殿堂。",
			Location:    "香水海畔",
			Category:    "文化建筑",
			Rating:      4.9,
			Price:       0,
			QRCode:      "SPOT-0003",
			QREnabled:   true,
			QRIntroText: "[joy]欢迎来到灵山梵宫。梵宫是一座集中展示佛教文化与传统工艺的艺术殿堂，内部可见木雕、壁画、琉璃、漆器等多种装饰细节。参观时建议放慢脚步，重点观察穹顶、廊柱和大型艺术空间，感受建筑、音乐演艺与佛教叙事结合的沉浸氛围。",
		},
		{
			Name:        "五印坛城",
			Description: "以藏传佛教文化为主题，展示五方五佛、转经筒和唐卡艺术。",
			Location:    "梵宫东侧",
			Category:    "文化建筑",
			Rating:      4.7,
			Price:       0,
			QRCode:      "SPOT-0004",
			QREnabled:   true,
			QRIntroText: "[joy]欢迎来到五印坛城。这里以藏传佛教文化为主题，通过五方五佛意象、转经筒、唐卡和法器展示，呈现坛城文化中的秩序感与象征意义。参观时可以从建筑色彩、装饰纹样和展陈器物入手，理解不同佛教艺术传统在灵山景区中的表达。",
		},
		{
			Name:        "文创驿站",
			Description: "提供文创商品、饮品和游客休憩服务。",
			Location:    "出口商业区",
			Category:    "服务设施",
			Rating:      4.5,
			Price:       0,
			QRCode:      "SPOT-0005",
			QREnabled:   true,
			QRIntroText: "[joy]欢迎来到文创驿站。这里主要提供文创商品、饮品补给和短暂停留休憩服务，适合在游览结束前整理行程、挑选纪念品或稍作休息。若您想把灵山胜境的文化记忆带走，可以优先查看带有景区元素的文创产品。",
		},
	}
	for _, spot := range spots {
		if err := pkg.GetDB().Where("name = ?", spot.Name).Assign(spot).FirstOrCreate(&spot).Error; err != nil {
			return fmt.Errorf("写入演示景点失败: %w", err)
		}
	}
	return nil
}

func seedTourRoutes() error {
	routes := []model.TourRoute{
		{Name: "经典半日路线", Description: "入口 -> 九龙灌浴 -> 灵山梵宫 -> 灵山大佛", Spots: "九龙灌浴,灵山梵宫,灵山大佛", Duration: 180, Difficulty: "easy", Rating: 4.8},
		{Name: "文化深度路线", Description: "梵宫、五印坛城与祥符禅寺串联，适合文化讲解。", Spots: "灵山梵宫,五印坛城,祥符禅寺", Duration: 240, Difficulty: "medium", Rating: 4.7},
	}
	for _, route := range routes {
		if err := pkg.GetDB().Where("name = ?", route.Name).Assign(route).FirstOrCreate(&route).Error; err != nil {
			return fmt.Errorf("写入演示路线失败: %w", err)
		}
	}
	return nil
}

func seedInteractions() error {
	if err := pkg.GetDB().Where("session_id LIKE ?", "demo-%").Delete(&model.InteractionLog{}).Error; err != nil {
		return fmt.Errorf("清理旧演示交互失败: %w", err)
	}

	now := time.Now()
	logs := []model.InteractionLog{
		{SessionID: "demo-001", Query: "灵山大佛有多高？", Response: "灵山大佛高88米。", Emotion: "joy", ResponseTimeMs: 980, Category: "景点", Source: "web", CreatedAt: now.Add(-50 * time.Minute)},
		{SessionID: "demo-002", Query: "亲子游怎么安排？", Response: "建议选择九龙灌浴、梵宫和文创驿站。", Emotion: "joy", ResponseTimeMs: 1350, Category: "路线", Source: "digital_human", CreatedAt: now.Add(-40 * time.Minute)},
		{SessionID: "demo-003", Query: "梵宫有什么特色？", Response: "梵宫汇集东阳木雕、琉璃和油画等艺术。", Emotion: "surprise", ResponseTimeMs: 1680, Category: "历史", Source: "voice", CreatedAt: now.Add(-25 * time.Minute)},
		{SessionID: "demo-004", Query: "五印坛城讲什么文化？", Response: "五印坛城主要体现藏传佛教文化。", Emotion: "joy", ResponseTimeMs: 1120, Category: "历史", Source: "web", CreatedAt: now.Add(-10 * time.Minute)},
	}
	if err := pkg.GetDB().Create(&logs).Error; err != nil {
		return fmt.Errorf("写入演示交互失败: %w", err)
	}
	return nil
}
