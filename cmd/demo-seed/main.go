package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/geolocation"
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
	if err := seedScenicSpots(configDir); err != nil {
		return err
	}
	if err := seedTourRoutes(); err != nil {
		return err
	}
	if err := seedOperationalDemoData(pkg.GetDB(), time.Now()); err != nil {
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

func seedScenicSpots(configDir string) error {
	spots, err := loadDemoScenicSpots(filepath.Join(configDir, "scenic_spot_coordinates.json"))
	if err != nil {
		return fmt.Errorf("加载景点坐标校准失败: %w", err)
	}
	for _, spot := range spots {
		if err := pkg.GetDB().Where("name = ?", spot.Name).Assign(spot).FirstOrCreate(&spot).Error; err != nil {
			return fmt.Errorf("写入演示景点失败: %w", err)
		}
	}
	return nil
}

func loadDemoScenicSpots(calibrationPath string) ([]model.ScenicSpot, error) {
	calibration, err := geolocation.LoadCalibration(calibrationPath)
	if err != nil {
		return nil, err
	}
	coordinates := make(map[string]geolocation.SpotCalibration, len(calibration.Spots))
	for _, item := range calibration.Spots {
		coordinates[item.Name] = item
	}
	spots := demoScenicSpotTemplates()
	for index := range spots {
		coordinate, ok := coordinates[spots[index].Name]
		if !ok {
			return nil, fmt.Errorf("景点 %q 缺少坐标校准", spots[index].Name)
		}
		spots[index].Location = coordinate.ReturnedAddress
		spots[index].Longitude = coordinate.Longitude
		spots[index].Latitude = coordinate.Latitude
		spots[index].GeofenceEnabled = spots[index].GeofenceEnabled && coordinate.Verified && coordinate.GeofenceEnabled
	}
	return spots, nil
}

func demoScenicSpotTemplates() []model.ScenicSpot {
	return []model.ScenicSpot{
		routeScenicSpot("南门", "景区步行路线的主要入口。", "出入口"),
		routeScenicSpot("灵山大照壁", "位于景区入口区域的标志性照壁。", "文化景观"),
		routeScenicSpot("五明桥", "连接入口区域与核心游线的石桥景观。", "文化景观"),
		routeScenicSpot("胜境门楼", "连接入口与景区主游线的门楼节点。", "文化景观"),
		routeScenicSpot("佛足坛", "以佛足印文化为主题的游览节点。", "文化景观"),
		{
			Name:                    "灵山大佛",
			Description:             "高88米的青铜立佛，是灵山胜境核心景点。",
			Category:                "核心景点",
			Rating:                  4.9,
			Price:                   0,
			GeofenceEnabled:         true,
			GeofenceRadiusM:         100,
			GeofenceCooldownMinutes: 180,
			GeofenceIntroText:       "[joy]欢迎来到灵山大佛。眼前这尊释迦牟尼青铜立像高88米，面向太湖，是灵山胜境最醒目的核心地标。参观时可以从大佛广场仰望整体造像，再沿中轴线了解佛教文化、礼佛动线和太湖山水共同构成的庄严景观。",
			QRCode:                  "SPOT-0001",
			QREnabled:               true,
			QRIntroText:             "[joy]欢迎来到灵山大佛。眼前这尊释迦牟尼青铜立像高88米，面向太湖，是灵山胜境最醒目的核心地标。参观时可以从大佛广场仰望整体造像，再沿中轴线了解佛教文化、礼佛动线和太湖山水共同构成的庄严景观。",
		},
		{
			Name:                    "九龙灌浴",
			Description:             "再现释迦牟尼诞生时九龙沐浴的动态音乐喷泉演出。",
			Category:                "演艺体验",
			Rating:                  4.8,
			Price:                   0,
			GeofenceEnabled:         true,
			GeofenceRadiusM:         100,
			GeofenceCooldownMinutes: 120,
			GeofenceIntroText:       "[joy]欢迎来到九龙灌浴。这里用大型动态音乐群雕和喷泉演绎佛陀诞生时“花开见佛、九龙灌浴”的故事，是灵山胜境很有代表性的演艺体验。演出时段请以景区现场公告为准，建议提前到达广场区域，选择视野开阔的位置观看。",
			QRCode:                  "SPOT-0002",
			QREnabled:               true,
			QRIntroText:             "[joy]欢迎来到九龙灌浴。这里用大型动态音乐群雕和喷泉演绎佛陀诞生时“花开见佛、九龙灌浴”的故事，是灵山胜境很有代表性的演艺体验。演出时段请以景区现场公告为准，建议提前到达广场区域，选择视野开阔的位置观看。",
		},
		routeScenicSpot("降魔浮雕", "讲述释迦牟尼降伏心魔、彻悟成佛故事的浮雕景观。", "文化景观"),
		routeScenicSpot("三圣殿", "景区佛教文化建筑节点。", "文化建筑"),
		routeScenicSpot("阿育王柱", "以四方狮子等佛教文化意象构成的石柱景观。", "文化景观"),
		routeScenicSpot("天下第一掌", "按灵山大佛右手复制的佛手文化景观。", "文化景观"),
		routeScenicSpot("百子戏弥勒", "以弥勒与百子群雕呈现欢喜、包容寓意的景观。", "文化景观"),
		routeScenicSpot("灵山蔬食馆", "景区内提供蔬食餐饮服务的设施。", "餐饮服务"),
		routeScenicSpot("祥符禅寺", "灵山胜境内的佛教文化建筑与参访节点。", "文化建筑"),
		routeScenicSpot("杏坛广场", "连接祥符禅寺与灵山大佛游线的广场节点。", "游览节点"),
		{
			Name:                    "灵山梵宫",
			Description:             "汇集东阳木雕、琉璃、油画等传统工艺的佛教艺术殿堂。",
			Category:                "文化建筑",
			Rating:                  4.9,
			Price:                   0,
			GeofenceEnabled:         true,
			GeofenceRadiusM:         110,
			GeofenceCooldownMinutes: 180,
			GeofenceIntroText:       "[joy]欢迎来到灵山梵宫。梵宫是一座集中展示佛教文化与传统工艺的艺术殿堂，内部可见木雕、壁画、琉璃、漆器等多种装饰细节。参观时建议放慢脚步，重点观察穹顶、廊柱和大型艺术空间，感受建筑、音乐演艺与佛教叙事结合的沉浸氛围。",
			QRCode:                  "SPOT-0003",
			QREnabled:               true,
			QRIntroText:             "[joy]欢迎来到灵山梵宫。梵宫是一座集中展示佛教文化与传统工艺的艺术殿堂，内部可见木雕、壁画、琉璃、漆器等多种装饰细节。参观时建议放慢脚步，重点观察穹顶、廊柱和大型艺术空间，感受建筑、音乐演艺与佛教叙事结合的沉浸氛围。",
		},
		{
			Name:                    "五印坛城",
			Description:             "以藏传佛教文化为主题，展示五方五佛、转经筒和唐卡艺术。",
			Category:                "文化建筑",
			Rating:                  4.7,
			Price:                   0,
			GeofenceEnabled:         true,
			GeofenceRadiusM:         100,
			GeofenceCooldownMinutes: 180,
			GeofenceIntroText:       "[joy]欢迎来到五印坛城。这里以藏传佛教文化为主题，通过五方五佛意象、转经筒、唐卡和法器展示，呈现坛城文化中的秩序感与象征意义。参观时可以从建筑色彩、装饰纹样和展陈器物入手，理解不同佛教艺术传统在灵山景区中的表达。",
			QRCode:                  "SPOT-0004",
			QREnabled:               true,
			QRIntroText:             "[joy]欢迎来到五印坛城。这里以藏传佛教文化为主题，通过五方五佛意象、转经筒、唐卡和法器展示，呈现坛城文化中的秩序感与象征意义。参观时可以从建筑色彩、装饰纹样和展陈器物入手，理解不同佛教艺术传统在灵山景区中的表达。",
		},
		routeScenicSpot("曼飞龙塔", "展示南传佛教建筑风格的塔群景观。", "文化建筑"),
		routeScenicSpot("出口", "景区路线结束和离园节点。", "出入口"),
		{
			Name:                    "文创驿站",
			Description:             "提供文创商品、饮品和游客休憩服务。",
			Category:                "服务设施",
			Rating:                  4.5,
			Price:                   0,
			GeofenceEnabled:         false,
			GeofenceRadiusM:         80,
			GeofenceCooldownMinutes: 1440,
			QRCode:                  "SPOT-0005",
			QREnabled:               true,
			QRIntroText:             "[joy]欢迎来到文创驿站。这里主要提供文创商品、饮品补给和短暂停留休憩服务，适合在游览结束前整理行程、挑选纪念品或稍作休息。若您想把灵山胜境的文化记忆带走，可以优先查看带有景区元素的文创产品。",
		},
	}
}

func routeScenicSpot(name, description, category string) model.ScenicSpot {
	return model.ScenicSpot{
		Name:                    name,
		Description:             description,
		Category:                category,
		Price:                   0,
		GeofenceEnabled:         false,
		GeofenceRadiusM:         80,
		GeofenceCooldownMinutes: 1440,
	}
}

func seedTourRoutes() error {
	routes := []model.TourRoute{
		{Name: "完整参考步行路线", Description: "南门至出口串联景区主轴和核心文化节点（第三方转载待官方复核）。", Spots: "南门,佛足坛,九龙灌浴,降魔浮雕,三圣殿,阿育王柱,天下第一掌,百子戏弥勒,祥符禅寺,杏坛广场,灵山大佛,灵山梵宫,曼飞龙塔,五印坛城,出口", Duration: 300, Difficulty: "medium", Rating: 4.8},
		{Name: "错峰参考步行路线", Description: "优先安排文化点，尽量避开主轴集中客流（第三方转载待官方复核）。", Spots: "南门,灵山大照壁,胜境门楼,佛足坛,五印坛城,曼飞龙塔,灵山梵宫,阿育王柱,天下第一掌,百子戏弥勒,祥符禅寺,杏坛广场,灵山大佛,三圣殿,降魔浮雕,九龙灌浴", Duration: 300, Difficulty: "medium", Rating: 4.7},
		{Name: "观光车参考路线", Description: "减少步行的观光车与重点步行节点组合（第三方转载待官方复核）。", Spots: "灵山大照壁,佛足坛,杏坛广场,灵山大佛,祥符禅寺,天下第一掌,百子戏弥勒,阿育王柱,灵山蔬食馆,灵山梵宫,九龙灌浴,降魔浮雕,曼飞龙塔,五印坛城,出口", Duration: 240, Difficulty: "easy", Rating: 4.6},
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
