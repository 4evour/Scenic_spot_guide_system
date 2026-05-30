package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	iconfig "github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

var registerRAGTestDriver sync.Once

func newTestRAGService(t *testing.T) *RAGService {
	t.Helper()

	const driverName = "modernc-rag-test"
	registerRAGTestDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	repo := repository.NewKnowledgeRepository(db)
	profile := newTestScenicProfile()
	return NewRAGService(repo, "", "", "", nil, profile)
}

// newTestScenicProfile 创建测试用灵山景区配置（与 lingshan.yaml 保持一致）
func newTestScenicProfile() *iconfig.ScenicProfile {
	return &iconfig.ScenicProfile{
		ID:        "lingshan",
		Name:      "灵山胜境",
		ShortName: "灵山",
		DigitalHuman: iconfig.DHProfile{
			Name:        "小灵",
			Personality: "热情、专业、温柔",
		},
		Keywords: iconfig.KeywordConfig{
			Spots: []string{"灵山大佛", "九龙灌浴", "灵山梵宫", "五印坛城", "曼飞龙塔", "佛手广场", "百子戏弥勒", "祥符禅寺"},
			RelatedKeywords: []string{
				"灵山", "大佛", "九龙灌", "佛教", "佛文化", "祥符寺",
				"灵山胜境", "拈花湾", "门票", "票价", "开放时间",
				"景点", "景区", "导览", "路线", "交通", "停车场",
				"表演", "演出", "梵宫", "五印坛城", "曼飞龙塔",
				"九龙灌浴", "灌浴", "太子像", "莲花", "喷泉",
				"灵山梵宫", "灵山大佛", "佛足坛", "五明桥", "大照壁",
			},
			RelatedPatterns: []string{"灵山", "拈花湾", "祥符寺"},
			TopicEntities:   []string{"灵山大佛", "九龙灌浴", "灵山梵宫", "五印坛城", "曼飞龙塔", "佛手广场", "百子戏弥勒", "祥符禅寺"},
			ConditionalBoosts: []iconfig.ConditionalBoost{
				{QueryTerms: []string{"哪里", "位于", "地址", "位置"}, ContentTerms: []string{"位于", "地处", "江苏", "无锡", "马山", "太湖"}},
				{QueryTerms: []string{"表现", "内容", "讲什么", "展示"}, ContentTerms: []string{"释迦牟尼", "花开见佛", "九龙沐浴", "佛陀诞生", "再现", "展示", "场景", "核心", "文化内涵"}},
				{QueryTerms: []string{"为什么", "称为", "特色"}, ContentTerms: []string{"被誉为", "内部汇集", "传统工艺", "艺术"}},
				{QueryTerms: []string{"哪类", "什么文化", "佛教文化"}, ContentTerms: []string{"藏传佛教", "五方五佛", "转经筒", "唐卡"}},
				{QueryTerms: []string{"五印坛城"}, ContentTerms: []string{"藏传佛教", "五方五佛", "转经筒", "唐卡", "曼陀罗"}},
			},
			QueryExpansion: []iconfig.ExpansionRule{
				{Trigger: []string{"半天", "优先", "先看"}, Expand: "初次到访 主线 九龙灌浴 佛手广场 祥符禅寺 灵山大佛"},
				{Trigger: []string{"中轴线"}, Expand: "初次到访 主线 中轴游览线 九龙灌浴 佛手广场 祥符禅寺 灵山大佛"},
				{Trigger: []string{"带孩子", "小朋友", "亲子"}, Expand: "百子戏弥勒 佛手广场 九龙灌浴 亲子游客"},
				{Trigger: []string{"拍照", "轻松点位"}, Expand: "佛手广场 百子戏弥勒 适合拍照"},
				{Trigger: []string{"大佛之外", "文化建筑"}, Expand: "五印坛城 曼飞龙塔 灵山梵宫 佛教文化建筑"},
				{Trigger: []string{"木雕", "壁画", "琉璃", "工艺"}, Expand: "灵山梵宫 艺术工艺"},
				{Trigger: []string{"藏式", "藏传"}, Expand: "五印坛城 藏传佛教"},
				{Trigger: []string{"喷水", "花开见佛", "喷泉"}, Expand: "九龙灌浴"},
				{Trigger: []string{"演艺", "剧场", "演出"}, Expand: "九龙灌浴 吉祥颂 演出场次 官方最新公告"},
				{Trigger: []string{"今天", "现在", "现场", "开不开", "排队", "人多", "无人机", "宠物", "能不能替代公告", "替代官方公告"}, Expand: "实时信息 官方最新公告 现场公示 不能编造"},
				{Trigger: []string{"容易过期", "最容易过期"}, Expand: "门票价格 开放时间 演出场次 停车余位 临时闭园 临时检修 优惠政策 实时信息"},
				{Trigger: []string{"无人机", "宠物"}, Expand: "安全禁忌 宠物入园 无人机拍摄 现场规定 正式规定 现场管理 不能替代"},
				{Trigger: []string{"排队", "人多", "客流"}, Expand: "排队时间 实时客流 今日游客多不多 拥堵程度 天气应用 地图热力"},
				{Trigger: []string{"导览服务", "现场设施", "服务设施"}, Expand: "导览服务 休息点 洗手间 现场指引 官方最新公告 服务开放情况"},
				{Trigger: []string{"只看大佛", "商业化游乐", "普通景区"}, Expand: "太湖山水 佛教文化 文化建筑 演艺体验 礼佛空间 九龙灌浴 祥符禅寺 灵山梵宫"},
				{Trigger: []string{"天气", "高温", "雨天", "路线"}, Expand: "降雨 高温 室内点 雨天路线"},
			},
			IntentBoosts: []iconfig.IntentBoostRule{
				{QueryContains: []string{"半天", "优先", "先看", "路线", "中轴线"}, Topic: "route", ChunkContains: []string{"路线", "主线", "初次", "半天", "中轴"}, Boost: 1.6},
				{QueryContains: []string{"带孩子", "小朋友", "亲子", "拍照", "轻松"}, Topic: "family", ChunkContains: []string{"百子戏弥勒", "佛手广场", "亲子", "拍照"}, Boost: 1.4},
				{QueryContains: []string{"今天", "现在", "现场", "开不开", "排队", "人多", "无人机", "宠物"}, Topic: "boundary", ChunkContains: []string{"实时", "官方最新公告", "现场公示", "不能编造", "资料不足"}, Boost: 2.0},
				{QueryContains: []string{"餐厅", "素食", "简餐", "吃饭", "开不开"}, Topic: "service", ChunkContains: []string{"餐饮", "餐厅", "素食", "简餐", "菜单"}, Boost: 5.2},
				{QueryContains: []string{"无人机", "宠物"}, ChunkContains: []string{"无人机", "宠物", "携带物品", "正式规定", "现场管理"}, Boost: 8.0},
				{QueryContains: []string{"排队", "人多", "客流"}, ChunkContains: []string{"排队时间", "实时客流", "今日游客多不多", "拥堵程度"}, Boost: 4.0},
				{QueryContains: []string{"大佛之外", "文化建筑", "三大语系"}, ChunkContains: []string{"五印坛城", "曼飞龙塔", "灵山梵宫", "佛教三大语系"}, Boost: 1.8},
				{QueryContains: []string{"木雕", "壁画", "琉璃", "工艺"}, Topic: "fangong", ChunkContains: []string{"灵山梵宫", "木雕", "壁画", "琉璃"}, Boost: 1.8},
				{QueryContains: []string{"藏式", "藏传"}, Topic: "wuyin", ChunkContains: []string{"五印坛城", "藏传佛教", "藏式"}, Boost: 1.8},
				{QueryContains: []string{"喷水", "花开见佛", "喷泉"}, Topic: "jiulong", ChunkContains: []string{"九龙灌浴", "喷水", "花开见佛"}, Boost: 1.8},
				{QueryContains: []string{"演艺", "剧场", "演出"}, Topic: "show", ChunkContains: []string{"吉祥颂", "九龙灌浴", "演出场次"}, Boost: 1.4},
				{QueryContains: []string{"天气", "高温", "雨天"}, Topic: "route", ChunkContains: []string{"天气", "高温", "雨天", "室内点", "实时客流"}, Boost: 1.5},
				{QueryContains: []string{"导览服务", "现场设施", "服务设施"}, Topic: "service", ChunkContains: []string{"导览服务", "休息点", "洗手间", "现场指引", "服务中心"}, Boost: 4.0},
			},
		},
		Prompts: iconfig.PromptConfig{
			SystemRole: "你是灵山胜境景区的AI数字人导览员\"小灵\"，负责为游客提供专业、热情的导览服务。",
			FallbackAnswers: map[string]string{
				"灵山大佛高度": "灵山大佛高88米，主体高79米，是世界上最高的青铜立佛之一。",
				"五印坛城":     "五印坛城以藏传佛教文化为主题，展示五方五佛、转经筒和唐卡艺术。",
			},
			FollowUpRewrite: map[string]string{
				"路线规划": "初次到访 主线",
				"天气路线": "雨天路线 降雨 高温 室内点 现场天气调整",
				"亲子路线": "亲子游客 百子戏弥勒 佛手广场 九龙灌浴",
				"老人路线": "老人游客 轻松路线 休息点 不要安排太满",
				"补充推荐": "补充推荐 大佛之外 文化建筑 灵山梵宫 五印坛城",
			},
		},
	}
}

type staticEmbeddingProvider struct {
	vectors map[string][]float64
}

func (p staticEmbeddingProvider) GenerateEmbedding(text string) ([]float64, error) {
	if vec, ok := p.vectors[text]; ok {
		return vec, nil
	}
	return []float64{0, 1}, nil
}

func (p staticEmbeddingProvider) Name() string {
	return "static-test"
}

func (p staticEmbeddingProvider) IsAvailable() bool {
	return true
}

func newTestRAGServiceWithEmbedding(t *testing.T, provider EmbeddingProvider) *RAGService {
	t.Helper()

	const driverName = "modernc-rag-test"
	registerRAGTestDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	repo := repository.NewKnowledgeRepository(db)
	profile := newTestScenicProfile()
	return NewRAGService(repo, "", "", "", provider, profile)
}

func TestRAGServiceLoadsJSONAndRetrievesWithBM25(t *testing.T) {
	rag := newTestRAGService(t)
	data := []byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"test","content":"灵山大佛通高88米，佛体79米，莲花瓣9米。","metadata":{"category":"景点"}},
		{"id":"palace","title":"灵山梵宫","source":"test","content":"灵山梵宫汇集东阳木雕、琉璃、油画等传统工艺。","metadata":{"category":"文化"}}
	]`)

	loaded, err := rag.LoadKnowledgeJSON(data)
	if err != nil {
		t.Fatalf("LoadKnowledgeJSON returned error: %v", err)
	}
	if loaded != 2 {
		t.Fatalf("loaded = %d, want 2", loaded)
	}

	chunks, err := rag.RetrieveRelevantKnowledge("灵山大佛有多高", 1)
	if err != nil {
		t.Fatalf("RetrieveRelevantKnowledge returned error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("retrieved chunks = %d, want 1", len(chunks))
	}
	if chunks[0].ID != "buddha-height" {
		t.Fatalf("top chunk id = %q, want buddha-height", chunks[0].ID)
	}
}

func TestRAGServiceUsesHybridScoreWhenEmbeddingAvailable(t *testing.T) {
	rag := newTestRAGServiceWithEmbedding(t, staticEmbeddingProvider{
		vectors: map[string][]float64{
			"灵山梵宫有什么工艺？": {1, 0},
		},
	})
	data := []byte(`[
		{"id":"generic-vector","title":"通用介绍","source":"test","content":"这里介绍游客服务中心和普通导览信息。","vector":"[1,0]"},
		{"id":"palace-craft","title":"灵山梵宫工艺","source":"test","content":"灵山梵宫汇集东阳木雕、敦煌壁画、扬州漆器等传统工艺。","vector":"[0.95,0.05]"}
	]`)

	if _, err := rag.LoadKnowledgeJSON(data); err != nil {
		t.Fatalf("LoadKnowledgeJSON returned error: %v", err)
	}

	chunks, err := rag.RetrieveRelevantKnowledge("灵山梵宫有什么工艺？", 1)
	if err != nil {
		t.Fatalf("RetrieveRelevantKnowledge returned error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("retrieved chunks = %d, want 1", len(chunks))
	}
	if chunks[0].ID != "palace-craft" {
		t.Fatalf("top chunk id = %q, want palace-craft", chunks[0].ID)
	}
}

func TestRAGServiceRetrievesWithRRFMode(t *testing.T) {
	rag := newTestRAGServiceWithEmbedding(t, staticEmbeddingProvider{
		vectors: map[string][]float64{
			"灵山梵宫有什么工艺？": {1, 0},
		},
	})
	data := []byte(`[
		{"id":"semantic-match","title":"游客服务","source":"test","content":"这里介绍普通游客服务和咨询方式。","vector":"[1,0]"},
		{"id":"lexical-match","title":"灵山梵宫工艺","source":"test","content":"灵山梵宫汇集东阳木雕、敦煌壁画、扬州漆器等传统工艺。","vector":"[0,1]"},
		{"id":"irrelevant","title":"交通指南","source":"test","content":"无锡火车站可乘公交前往灵山。","vector":"[0,1]"}
	]`)

	if _, err := rag.LoadKnowledgeJSON(data); err != nil {
		t.Fatalf("LoadKnowledgeJSON returned error: %v", err)
	}

	chunks, err := rag.RetrieveRelevantKnowledgeWithOptions("灵山梵宫有什么工艺？", RetrievalOptions{
		TopK: 1,
		Mode: RetrievalModeRRFFusion,
	})
	if err != nil {
		t.Fatalf("RetrieveRelevantKnowledgeWithOptions returned error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("retrieved chunks = %d, want 1", len(chunks))
	}
	if chunks[0].ID != "lexical-match" {
		t.Fatalf("top chunk id = %q, want lexical-match", chunks[0].ID)
	}
}

func TestRAGServiceRetrievesWithWeightedHybridMode(t *testing.T) {
	rag := newTestRAGServiceWithEmbedding(t, staticEmbeddingProvider{
		vectors: map[string][]float64{
			"灵山梵宫有什么工艺？": {1, 0},
		},
	})
	data := []byte(`[
		{"id":"semantic-match","title":"游客服务","source":"test","content":"这里介绍普通游客服务和咨询方式。","vector":"[1,0]"},
		{"id":"lexical-match","title":"灵山梵宫工艺","source":"test","content":"灵山梵宫汇集东阳木雕、敦煌壁画、扬州漆器等传统工艺。","vector":"[0,1]"}
	]`)

	if _, err := rag.LoadKnowledgeJSON(data); err != nil {
		t.Fatalf("LoadKnowledgeJSON returned error: %v", err)
	}

	chunks, err := rag.RetrieveRelevantKnowledgeWithOptions("灵山梵宫有什么工艺？", RetrievalOptions{
		TopK:            1,
		Mode:            RetrievalModeHybridWeighted,
		EmbeddingWeight: 0.2,
		BM25Weight:      0.8,
	})
	if err != nil {
		t.Fatalf("RetrieveRelevantKnowledgeWithOptions returned error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("retrieved chunks = %d, want 1", len(chunks))
	}
	if chunks[0].ID != "lexical-match" {
		t.Fatalf("top chunk id = %q, want lexical-match", chunks[0].ID)
	}
}

func TestRAGServiceLightRerankPromotesTitleAndEntityMatches(t *testing.T) {
	rag := newTestRAGService(t)
	data := []byte(`[
		{"id":"broad-match","title":"景区综合介绍","source":"test","content":"灵山大佛、九龙灌浴、灵山梵宫、五印坛城都是灵山胜境的主体景观。"},
		{"id":"title-match","title":"九龙灌浴表演","source":"test","content":"九龙灌浴通过音乐、喷泉和群雕动态演绎佛陀诞生时花开见佛的故事。"}
	]`)

	if _, err := rag.LoadKnowledgeJSON(data); err != nil {
		t.Fatalf("LoadKnowledgeJSON returned error: %v", err)
	}

	chunks, err := rag.RetrieveRelevantKnowledgeWithOptions("九龙灌浴讲什么？", RetrievalOptions{
		TopK: 1,
		Mode: RetrievalModeLightRerank,
	})
	if err != nil {
		t.Fatalf("RetrieveRelevantKnowledgeWithOptions returned error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("retrieved chunks = %d, want 1", len(chunks))
	}
	if chunks[0].ID != "title-match" {
		t.Fatalf("top chunk id = %q, want title-match", chunks[0].ID)
	}
}

func TestExpandQueryForRetrievalAddsFocusedTerms(t *testing.T) {
	rag := newTestRAGService(t)
	rag.profile = &iconfig.ScenicProfile{
		Keywords: iconfig.KeywordConfig{
			QueryExpansion: []iconfig.ExpansionRule{
				{Trigger: []string{"半天", "优先", "先看"}, Expand: "初次到访 主线 九龙灌浴 佛手广场 祥符禅寺 灵山大佛"},
				{Trigger: []string{"带孩子", "小朋友", "亲子"}, Expand: "百子戏弥勒 佛手广场 九龙灌浴 亲子游客"},
			},
		},
	}
	retrievalText, addedTerms := rag.configBasedQueryExpansion(rag.profile, "带孩子半天游灵山优先看哪些点？")
	_ = addedTerms
	for _, term := range []string{"初次到访", "九龙灌浴", "佛手广场", "百子戏弥勒", "亲子游客"} {
		if !strings.Contains(retrievalText, term) {
			t.Fatalf("retrieval text %q does not contain expanded term %q", retrievalText, term)
		}
	}
}

func TestQueryExpansionDoesNotChangePromptQuestion(t *testing.T) {
	rag := newTestRAGService(t)
	original := "半天游灵山优先看哪些点？"
	prompt := rag.BuildRAGPrompt(original, []model.KnowledgeChunk{
		{ID: "route", Title: "测试路线", Content: "测试内容", Source: "test"},
	})
	if !strings.Contains(prompt, original) {
		t.Fatalf("prompt should contain original query")
	}
	if strings.Contains(prompt, "初次到访 主线") {
		t.Fatalf("prompt leaked retrieval expansion: %s", prompt)
	}
}

func TestRAGServiceFocusedFailureRegressionSet(t *testing.T) {
	rag := newTestRAGService(t)
	data, err := os.ReadFile(filepath.Join("..", "..", "knowledge", "real", "lingshan_real_chunks.jsonl"))
	if err != nil {
		t.Fatalf("read real chunks: %v", err)
	}
	if _, err := rag.LoadKnowledgeJSON(data); err != nil {
		t.Fatalf("load real chunks: %v", err)
	}

	cases := []struct {
		name        string
		query       string
		expectedIDs []string
		topN        int
	}{
		{name: "half_day_main_route", query: "半天游灵山优先看哪些点？", expectedIDs: []string{"real-route-001"}, topN: 8},
		{name: "family_route", query: "带孩子去灵山有哪些点比较合适？", expectedIDs: []string{"real-route-002", "real-foshou-mile-001", "real-mile-001"}, topN: 8},
		{name: "culture_buildings_beyond_buddha", query: "灵山大佛之外还有哪些文化建筑？", expectedIDs: []string{"real-wuyin-002", "real-manfeilong-001"}, topN: 8},
		{name: "fangong_craft", query: "如果游客说“想看木雕壁画琉璃”，应该推荐哪里？", expectedIDs: []string{"real-fangong-002"}, topN: 8},
		{name: "tibetan_style", query: "游客说想看“藏式风格”的内容，应该联想到哪里？", expectedIDs: []string{"real-wuyin-001", "real-wuyin-003"}, topN: 8},
		{name: "water_show", query: "如果游客只问“喷水表演在哪”，应该召回哪个知识？", expectedIDs: []string{"real-jiulong-001", "real-jiulong-002"}, topN: 8},
		{name: "restaurant_open_now", query: "游客问餐厅现在开不开，能不能直接回答？", expectedIDs: []string{"real-food-rest-001", "real-service-004"}, topN: 8},
		{name: "official_notice_boundary", query: "小灵能不能替代官方公告？", expectedIDs: []string{"real-negative-001", "real-negative-002", "real-risk-002"}, topN: 8},
		{name: "pet_drone_boundary", query: "宠物入园和无人机拍摄能不能直接承诺？", expectedIDs: []string{"real-boundary-safety-001"}, topN: 8},
		{name: "queue_time_boundary", query: "游客问现场排队多久，知识库能回答吗？", expectedIDs: []string{"real-boundary-weather-001"}, topN: 8},
		{name: "weather_route", query: "路线里为什么要考虑天气？", expectedIDs: []string{"real-season-001", "real-route-007"}, topN: 8},
		{name: "guide_service_boundary", query: "游客问“有没有导览服务”，可以怎么回答？", expectedIDs: []string{"real-service-002", "real-food-rest-001"}, topN: 8},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := rag.RetrieveRelevantKnowledgeWithOptions(tc.query, RetrievalOptions{
				TopK: tc.topN,
				Mode: RetrievalModeLightRerank,
			})
			if err != nil {
				t.Fatalf("RetrieveRelevantKnowledgeWithOptions returned error: %v", err)
			}
			got := chunkIDs(chunks)
			if !hasAnyID(got, tc.expectedIDs) {
				t.Fatalf("retrieved ids = %v, want one of %v", got, tc.expectedIDs)
			}
		})
	}
}

func TestRAGServiceRewritesFollowUpWithShortTermContext(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"dafo-height","title":"灵山大佛高度","source":"test","content":"灵山大佛通高88米，是灵山胜境的标志性景观。"},
		{"id":"jiulong-story","title":"九龙灌浴故事","source":"test","content":"九龙灌浴通过音乐、喷泉和群雕动态演绎佛陀诞生时花开见佛的故事。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	response, err := rag.QueryWithRAGInSession("s1", "灵山大佛是什么？")
	if err != nil {
		t.Fatalf("QueryWithRAGInSession returned error: %v", err)
	}
	if !strings.Contains(response, "灵山大佛") {
		t.Fatalf("expected first answer to mention 灵山大佛: %s", response)
	}

	rewritten := rag.RewriteFollowUpQuery("s1", "它有多高？")
	if !strings.Contains(rewritten, "灵山大佛") || !strings.Contains(rewritten, "它有多高") {
		t.Fatalf("rewritten query = %q, want context and original follow-up", rewritten)
	}
	if strings.Contains(rewritten, response[:min(len(response), 20)]) {
		t.Fatalf("rewritten query should use topic metadata instead of raw previous answer: %q", rewritten)
	}
}

func TestRAGServiceRewritesRouteFollowUpWithSessionContext(t *testing.T) {
	rag := newTestRAGService(t)
	rag.appendSessionTurn("route-session", "半天游灵山怎么走？", "可以先看九龙灌浴、佛手广场、祥符禅寺和灵山大佛。")

	rewritten := rag.RewriteFollowUpQuery("route-session", "下雨呢？")
	for _, want := range []string{"路线", "雨天路线", "降雨", "室内点"} {
		if !strings.Contains(rewritten, want) {
			t.Fatalf("rewritten route follow-up = %q, want %q", rewritten, want)
		}
	}
}

func TestRAGServiceRewritesBoundaryFollowUpWithSessionContext(t *testing.T) {
	rag := newTestRAGService(t)
	rag.appendSessionTurn("boundary-session", "九龙灌浴值得看吗？", "九龙灌浴是灵山胜境的重要动态景观。")

	rewritten := rag.RewriteFollowUpQuery("boundary-session", "现在人多吗？")
	for _, want := range []string{"九龙灌浴", "实时客流", "排队时间", "官方最新公告", "现场公示"} {
		if !strings.Contains(rewritten, want) {
			t.Fatalf("rewritten boundary follow-up = %q, want %q", rewritten, want)
		}
	}
}

func TestRAGServiceCapsSessionHistory(t *testing.T) {
	rag := newTestRAGService(t)
	for i := 0; i < MaxSessionHistorySize+25; i++ {
		rag.appendSessionTurn(fmt.Sprintf("session-%04d", i), "灵山大佛是什么？", "灵山大佛是灵山胜境的标志性景观。")
	}

	rag.cacheMutex.RLock()
	sessionCount := len(rag.sessionHistory)
	_, oldestExists := rag.sessionHistory["session-0000"]
	rag.cacheMutex.RUnlock()

	if sessionCount > MaxSessionHistorySize {
		t.Fatalf("session history size = %d, want <= %d", sessionCount, MaxSessionHistorySize)
	}
	if oldestExists {
		t.Fatalf("oldest session should be evicted when session history exceeds cap")
	}
}

func TestRAGPromptIncludesSessionContextWithoutChangingQuestion(t *testing.T) {
	rag := newTestRAGService(t)
	prompt := rag.BuildRAGPromptWithContext("它有多高？", []model.KnowledgeChunk{
		{ID: "height", Title: "灵山大佛高度", Content: "灵山大佛通高88米。", Source: "test"},
	}, "上一轮主题：灵山大佛；当前意图：属性追问")

	if !strings.Contains(prompt, "【当前会话上下文】") || !strings.Contains(prompt, "上一轮主题：灵山大佛") {
		t.Fatalf("prompt should contain concise session context: %s", prompt)
	}
	if !strings.Contains(prompt, "【游客问题】\n它有多高？") {
		t.Fatalf("prompt should keep original user question: %s", prompt)
	}
	if strings.Contains(prompt, "灵山大佛 有多高") {
		t.Fatalf("prompt leaked rewritten retrieval query: %s", prompt)
	}
}

func TestFallbackAnswerFormatsRouteAndBoundary(t *testing.T) {
	rag := newTestRAGService(t)
	chunks := []model.KnowledgeChunk{
		{ID: "route", Title: "雨天路线", Source: "test", Content: "雨天可优先安排灵山梵宫、五印坛城等室内或停留更稳的点位，露天点位根据现场天气调整。"},
		{ID: "boundary", Title: "实时边界", Source: "test", Content: "实时客流、排队时间、开放状态需要以官方最新公告或现场公示为准，小灵不能替代现场确认。"},
	}

	routeAnswer := rag.generateAnswerFromChunksWithContext("下雨怎么办？", chunks, "上一轮主题：半天游路线；当前意图：天气路线")
	if !strings.Contains(routeAnswer, "可以按这个思路安排") || !strings.Contains(routeAnswer, "1.") {
		t.Fatalf("route fallback answer should be structured, got: %s", routeAnswer)
	}

	boundaryAnswer := rag.generateAnswerFromChunksWithContext("现在人多吗？", chunks, "上一轮主题：九龙灌浴；当前意图：实时信息边界")
	for _, want := range []string{"不能直接替您确认", "官方最新公告", "现场公示"} {
		if !strings.Contains(boundaryAnswer, want) {
			t.Fatalf("boundary fallback answer = %q, want %q", boundaryAnswer, want)
		}
	}
}

func TestRAGServiceEvaluateQuestionsReportsKeywordCoverage(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"test","content":"灵山大佛高88米，主体高79米，莲花瓣高9米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	report, err := rag.EvaluateQuestions([]RAGEvaluationCase{
		{
			Question:         "灵山大佛有多高？",
			ExpectedKeywords: []string{"88米", "79米"},
		},
	})
	if err != nil {
		t.Fatalf("EvaluateQuestions returned error: %v", err)
	}
	if report.Total != 1 || report.Passed != 1 || report.Failed != 0 {
		t.Fatalf("unexpected report counts: total=%d passed=%d failed=%d", report.Total, report.Passed, report.Failed)
	}
	if report.AverageKeywordCoverage != 1 {
		t.Fatalf("coverage = %.2f, want 1.00", report.AverageKeywordCoverage)
	}
	if report.AverageRecallAtK != 1 {
		t.Fatalf("recall@k = %.2f, want 1.00", report.AverageRecallAtK)
	}
	if report.MRRAtK != 1 {
		t.Fatalf("mrr@k = %.2f, want 1.00", report.MRRAtK)
	}
	if report.RetrievalP95Ms <= 0 {
		t.Fatalf("retrieval p95 should be recorded")
	}
}

func TestRAGServiceEvaluateFileSupportsAnswerFallbackKeywords(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"location","title":"灵山胜境位置","source":"test","content":"灵山胜境位于江苏省无锡市太湖西北部的马山镇。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	evalPath := filepath.Join(t.TempDir(), "eval.json")
	body, err := json.Marshal([]RAGEvaluationCase{
		{
			Question: "灵山胜境位于哪里？",
			Answer:   "灵山胜境位于江苏省无锡市太湖西北部的马山镇。",
		},
	})
	if err != nil {
		t.Fatalf("marshal eval: %v", err)
	}
	if err := os.WriteFile(evalPath, body, 0o600); err != nil {
		t.Fatalf("write eval: %v", err)
	}

	report, err := rag.EvaluateFile(evalPath)
	if err != nil {
		t.Fatalf("EvaluateFile returned error: %v", err)
	}
	if report.Total != 1 {
		t.Fatalf("total = %d, want 1", report.Total)
	}
	if len(report.Results[0].MatchedKeywords) == 0 {
		t.Fatalf("expected derived keywords to match at least one answer fact")
	}
}

func TestRAGServiceEvaluateQuestionsReportsRetrievalMiss(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"palace","title":"灵山梵宫","source":"test","content":"灵山梵宫汇集东阳木雕、敦煌壁画、扬州漆器等传统工艺。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	report, err := rag.EvaluateQuestionsWithOptions([]RAGEvaluationCase{
		{
			Question:         "灵山梵宫有什么工艺？",
			ExpectedKeywords: []string{"东阳木雕"},
			ExpectedChunkIDs: []string{"missing-id"},
			Category:         "文化",
			Difficulty:       "medium",
		},
	}, EvaluationOptions{TopK: 1, RetrievalOnly: true})
	if err != nil {
		t.Fatalf("EvaluateQuestionsWithOptions returned error: %v", err)
	}
	if report.Results[0].RecallAtK != 0 {
		t.Fatalf("recall@k = %.2f, want 0", report.Results[0].RecallAtK)
	}
	if report.Results[0].FirstRelevantRank != 0 {
		t.Fatalf("first relevant rank = %d, want 0", report.Results[0].FirstRelevantRank)
	}
	if report.Results[0].Category != "文化" || report.Results[0].Difficulty != "medium" {
		t.Fatalf("category/difficulty not propagated: %+v", report.Results[0])
	}
}

func TestRAGServiceEvaluateQuestionsReportsGroupsAndFailureReasons(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official","content":"灵山大佛通高88米，佛体79米，莲花瓣9米。","metadata":{"source_type":"official"}},
		{"id":"palace-art","title":"灵山梵宫艺术","source":"official","content":"灵山梵宫可欣赏东阳木雕、琉璃巨制、敦煌壁画等艺术瑰宝。","metadata":{"source_type":"official"}}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	report, err := rag.EvaluateQuestionsWithOptions([]RAGEvaluationCase{
		{
			Question:         "灵山大佛有多高？",
			ExpectedKeywords: []string{"88米"},
			ExpectedChunkIDs: []string{"buddha-height"},
			Category:         "closed_real",
			Difficulty:       "easy",
			SourceType:       "official",
		},
		{
			Question:         "梵宫能看哪些工艺？",
			ExpectedKeywords: []string{"东阳木雕"},
			ExpectedChunkIDs: []string{"missing-palace"},
			Category:         "closed_real",
			Difficulty:       "medium",
			SourceType:       "official",
		},
	}, EvaluationOptions{TopK: 1, RetrievalOnly: true})
	if err != nil {
		t.Fatalf("EvaluateQuestionsWithOptions returned error: %v", err)
	}

	if len(report.GroupStats) == 0 {
		t.Fatalf("expected grouped stats")
	}
	if got := report.Results[0].SourceType; got != "official" {
		t.Fatalf("source type = %q, want official", got)
	}
	if got := report.Results[1].FailureReason; got != "retrieval_miss" {
		t.Fatalf("failure reason = %q, want retrieval_miss", got)
	}

	categoryStats := findGroupStat(report.GroupStats, "category", "closed_real")
	if categoryStats.Total != 2 || categoryStats.Passed != 1 || categoryStats.AverageRecallAtK != 0.5 {
		t.Fatalf("unexpected category stats: %+v", categoryStats)
	}
	difficultyStats := findGroupStat(report.GroupStats, "difficulty", "easy")
	if difficultyStats.Total != 1 || difficultyStats.Passed != 1 || difficultyStats.AverageRecallAtK != 1 {
		t.Fatalf("unexpected difficulty stats: %+v", difficultyStats)
	}
	sourceStats := findGroupStat(report.GroupStats, "source_type", "official")
	if sourceStats.Total != 2 || sourceStats.Failed != 1 || len(sourceStats.Failures) != 1 {
		t.Fatalf("unexpected source stats: %+v", sourceStats)
	}
	if len(report.FailureStats) == 0 {
		t.Fatalf("expected failure stats")
	}
}

func TestRAGServiceClassifiesOpenQuestionRetrievalMiss(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"palace","title":"灵山梵宫","source":"test","content":"灵山梵宫汇集东阳木雕、敦煌壁画、扬州漆器等传统工艺。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	report, err := rag.EvaluateQuestionsWithOptions([]RAGEvaluationCase{
		{
			Question:         "带孩子去灵山有哪些点比较合适？",
			ExpectedKeywords: []string{"儿童"},
			ExpectedChunkIDs: []string{"missing-family-route"},
			Category:         "open_real",
		},
	}, EvaluationOptions{TopK: 1, RetrievalOnly: true})
	if err != nil {
		t.Fatalf("EvaluateQuestionsWithOptions returned error: %v", err)
	}

	if got := report.Results[0].FailureReason; got != "open_question_retrieval_miss" {
		t.Fatalf("failure reason = %q, want open_question_retrieval_miss", got)
	}
	if len(report.FailureStats) != 1 || report.FailureStats[0].Reason != "open_question_retrieval_miss" {
		t.Fatalf("unexpected failure stats: %+v", report.FailureStats)
	}
}

func findGroupStat(stats []RAGEvaluationGroupStats, groupBy, name string) RAGEvaluationGroupStats {
	for _, stat := range stats {
		if stat.GroupBy == groupBy && stat.Name == name {
			return stat
		}
	}
	return RAGEvaluationGroupStats{}
}

func hasAnyID(got []string, expected []string) bool {
	for _, id := range got {
		for _, want := range expected {
			if id == want {
				return true
			}
		}
	}
	return false
}

func TestPercentileDuration(t *testing.T) {
	values := []time.Duration{
		10 * time.Millisecond,
		30 * time.Millisecond,
		20 * time.Millisecond,
	}

	if got := percentileDuration(values, 0.50).Milliseconds(); got != 20 {
		t.Fatalf("p50 = %dms, want 20ms", got)
	}
	if got := percentileDuration(values, 0.95).Milliseconds(); got != 30 {
		t.Fatalf("p95 = %dms, want 30ms", got)
	}
}
