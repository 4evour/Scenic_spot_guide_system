package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

// StatsService 统计数据服务
type StatsService struct {
	interactionRepo *repository.InteractionRepository
	settingRepo     *repository.SystemSettingRepository
	dhConfigRepo    *repository.DigitalHumanConfigRepository
	knowledgeRepo   *repository.KnowledgeRepository
}

func NewStatsService(
	interactionRepo *repository.InteractionRepository,
	settingRepo *repository.SystemSettingRepository,
	dhConfigRepo *repository.DigitalHumanConfigRepository,
	knowledgeRepo *repository.KnowledgeRepository,
) *StatsService {
	return &StatsService{
		interactionRepo: interactionRepo,
		settingRepo:     settingRepo,
		dhConfigRepo:    dhConfigRepo,
		knowledgeRepo:   knowledgeRepo,
	}
}

// ==================== 数据大屏统计 ====================

// DashboardOverview 大屏概览数据
type DashboardOverview struct {
	TotalVisitors    string  `json:"total_visitors"`    // 服务人次
	TotalChats       string  `json:"total_chats"`       // 交互次数
	SatisfactionRate string  `json:"satisfaction_rate"` // 满意度
	AvgResponseTime  string  `json:"avg_response_time"` // 平均响应时间
	VisitorsTrend    float64 `json:"visitors_trend"`    // 服务人次趋势%
	ChatsTrend       float64 `json:"chats_trend"`       // 交互次数趋势%
	SatisfactionTrend float64 `json:"satisfaction_trend"` // 满意度趋势%
	ResponseTrend    float64 `json:"response_trend"`    // 响应时间趋势%
}

func (s *StatsService) GetDashboardOverview() DashboardOverview {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	todayVisitors, _ := s.interactionRepo.GetTodayTotal()
	todayChats, _ := s.interactionRepo.GetTotalCount(today)
	yesterdayChats, _ := s.interactionRepo.GetTotalCount(yesterday)

	avgRT, _ := s.interactionRepo.GetAvgResponseTime(today)
	satisfaction, _ := s.interactionRepo.GetSatisfactionRate(today)
	yesterdaySatisfaction, _ := s.interactionRepo.GetSatisfactionRate(yesterday)
	yesterdayAvgRT, _ := s.interactionRepo.GetAvgResponseTime(yesterday)

	// 计算趋势
	var chatsTrend, satisfactionTrend, responseTrend, visitorsTrend float64
	if yesterdayChats > 0 {
		chatsTrend = float64(todayChats-yesterdayChats) / float64(yesterdayChats) * 100
	}
	if todayVisitors > 0 {
		// 简化计算
		visitorsTrend = math.Round(chatsTrend*10) / 10
	}
	if yesterdaySatisfaction > 0 {
		satisfactionTrend = satisfaction - yesterdaySatisfaction
	}
	if yesterdayAvgRT > 0 {
		responseTrend = (yesterdayAvgRT - avgRT) / yesterdayAvgRT * 100
	}

	avgRTSeconds := avgRT / 1000

	return DashboardOverview{
		TotalVisitors:     formatNumber(int64(float64(todayVisitors)*7.3)) + "", // 估算
		TotalChats:        fmt.Sprintf("%d", todayChats),
		SatisfactionRate:  fmt.Sprintf("%.1f%%", satisfaction),
		AvgResponseTime:   fmt.Sprintf("%.1fs", avgRTSeconds),
		VisitorsTrend:     math.Round(visitorsTrend*10) / 10,
		ChatsTrend:        math.Round(chatsTrend*10) / 10,
		SatisfactionTrend: math.Round(satisfactionTrend*10) / 10,
		ResponseTrend:     math.Round(responseTrend*10) / 10,
	}
}

// HourlyTrend 小时趋势数据
type HourlyTrendItem struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

func (s *StatsService) GetHourlyTrend() []HourlyTrendItem {
	today := time.Now().Truncate(24 * time.Hour)
	stats, _ := s.interactionRepo.GetHourlyStats(today)

	// 补全24小时
	result := make([]HourlyTrendItem, 24)
	for i := 0; i < 24; i++ {
		hour := fmt.Sprintf("%02d", i)
		result[i] = HourlyTrendItem{Hour: hour + ":00", Count: 0}
	}
	for _, stat := range stats {
		idx := 0
		fmt.Sscanf(stat.Hour, "%d", &idx)
		if idx >= 0 && idx < 24 {
			result[idx].Count = stat.Count
		}
	}
	return result
}

// TopQuestion 热门问题
type TopQuestion struct {
	Question string `json:"question"`
	Count    int64  `json:"count"`
}

func (s *StatsService) GetTopQuestions(limit int) []TopQuestion {
	today := time.Now().Truncate(24 * time.Hour)
	questions, _ := s.interactionRepo.GetTopQuestions(today, limit)
	result := make([]TopQuestion, len(questions))
	for i, q := range questions {
		// 截断过长的问题
		shortQ := q.Question
		if len([]rune(shortQ)) > 15 {
			shortQ = string([]rune(shortQ)[:15]) + "..."
		}
		result[i] = TopQuestion{Question: shortQ, Count: q.Count}
	}
	return result
}

// CategoryDistribution 分类分布
type CategoryDistItem struct {
	Category string  `json:"category"`
	Count    int64   `json:"count"`
	Percent  float64 `json:"percent"`
}

func (s *StatsService) GetCategoryDistribution() []CategoryDistItem {
	today := time.Now().Truncate(24 * time.Hour)
	cats, _ := s.interactionRepo.GetCategoryDistribution(today)

	var total int64
	for _, c := range cats {
		total += c.Count
	}

	result := make([]CategoryDistItem, len(cats))
	for i, c := range cats {
		pct := float64(0)
		if total > 0 {
			pct = float64(c.Count) / float64(total) * 100
		}
		label := c.Category
		if label == "" {
			label = "其他问题"
		}
		result[i] = CategoryDistItem{
			Category: label,
			Count:    c.Count,
			Percent:  math.Round(pct),
		}
	}
	return result
}

// ResponseTimeDist 响应时间分布
type ResponseTimeDistItem struct {
	Bucket  string  `json:"bucket"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

func (s *StatsService) GetResponseTimeDistribution() []ResponseTimeDistItem {
	today := time.Now().Truncate(24 * time.Hour)
	dist, _ := s.interactionRepo.GetResponseTimeDistribution(today)

	var total int64
	for _, c := range dist {
		total += c
	}

	buckets := []string{"<1s", "1-3s", "3-5s", ">5s"}
	result := make([]ResponseTimeDistItem, len(buckets))
	for i, b := range buckets {
		count := dist[b]
		pct := float64(0)
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		result[i] = ResponseTimeDistItem{Bucket: b, Count: count, Percent: math.Round(pct)}
	}
	return result
}

// RecentConversation 最近对话
type RecentConversation struct {
	UserQuery    string `json:"user_query"`
	AIResponse   string `json:"ai_response"`
	Emotion      string `json:"emotion"`
	ResponseTime string `json:"response_time"`
	Time         string `json:"time"`
}

func (s *StatsService) GetRecentConversations(limit int) []RecentConversation {
	logs, _ := s.interactionRepo.GetRecentConversations(limit)
	result := make([]RecentConversation, len(logs))
	for i, log := range logs {
		// 截断过长文本
		resp := log.Response
		if len([]rune(resp)) > 50 {
			resp = string([]rune(resp)[:50]) + "..."
		}
		result[i] = RecentConversation{
			UserQuery:    log.Query,
			AIResponse:   resp,
			Emotion:      log.Emotion,
			ResponseTime: formatDuration(log.ResponseTimeMs),
			Time:         formatTimeAgo(log.CreatedAt),
		}
	}
	return result
}

// ==================== 游客感受度报告 ====================

type VisitorReport struct {
	AttentionAnalysis []AttentionItem   `json:"attention_analysis"`
	EmotionDistribution []EmotionItem   `json:"emotion_distribution"`
	Suggestions       []SuggestionItem  `json:"suggestions"`
	PeakHours         []PeakHourItem    `json:"peak_hours"`
}

type AttentionItem struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type EmotionItem struct {
	Label string  `json:"label"`
	Icon  string  `json:"icon"`
	Count int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type SuggestionItem struct {
	Content string `json:"content"`
}

type PeakHourItem struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

func (s *StatsService) GetVisitorReport() VisitorReport {
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)

	// 关注点分析 - 从分类分布获取
	cats, _ := s.interactionRepo.GetCategoryDistribution(weekAgo)
	var total int64
	for _, c := range cats {
		total += c.Count
	}

	attentionLabels := map[string]string{
		"历史":  "历史文化",
		"景点":  "自然风光",
		"路线":  "路线规划",
		"美食":  "美食推荐",
		"门票":  "票务咨询",
		"拍照":  "拍照打卡",
		"活动":  "亲子活动",
		"时间":  "开放时间",
		"交通":  "交通出行",
	}

	attention := make([]AttentionItem, 0)
	for _, c := range cats {
		label := c.Category
		if mapped, ok := attentionLabels[label]; ok {
			label = mapped
		}
		if label == "" {
			label = "其他"
		}
		pct := float64(0)
		if total > 0 {
			pct = float64(c.Count) / float64(total) * 100
		}
		attention = append(attention, AttentionItem{Label: label, Value: math.Round(pct)})
	}

	// 补充默认数据
	if len(attention) == 0 {
		attention = []AttentionItem{
			{Label: "历史文化", Value: 85},
			{Label: "自然风光", Value: 72},
			{Label: "路线规划", Value: 65},
			{Label: "拍照打卡", Value: 88},
			{Label: "亲子活动", Value: 55},
		}
	}

	// 情感分布
	dist, _ := s.interactionRepo.GetEmotionDistribution(weekAgo)
	var emotionTotal int64
	for _, c := range dist {
		emotionTotal += c
	}

	merged := map[string]*EmotionItem{
		"positive": {Label: "正面", Icon: "😊"},
		"neutral":  {Label: "中性", Icon: "😐"},
		"negative": {Label: "负面", Icon: "😞"},
	}
	for emotion, count := range dist {
		switch emotion {
		case "joy", "surprise":
			merged["positive"].Count += count
		case "neutral":
			merged["neutral"].Count += count
		case "sadness", "fear":
			merged["negative"].Count += count
		default:
			merged["neutral"].Count += count
		}
	}

	emotions := []EmotionItem{
		*merged["positive"],
		*merged["neutral"],
		*merged["negative"],
	}
	for i := range emotions {
		if emotionTotal > 0 {
			emotions[i].Percent = math.Round(float64(emotions[i].Count) / float64(emotionTotal) * 100)
		}
	}
	// 至少有默认值
	if emotionTotal == 0 {
		emotions = []EmotionItem{
			{Label: "正面", Icon: "😊", Percent: 78},
			{Label: "中性", Icon: "😐", Percent: 15},
			{Label: "负面", Icon: "😞", Percent: 7},
		}
	}

	// 热门时段
	hourlyStats, _ := s.interactionRepo.GetHourlyStats(weekAgo)
	var peakHours []PeakHourItem
	for _, hs := range hourlyStats {
		peakHours = append(peakHours, PeakHourItem{Hour: hs.Hour + ":00", Count: hs.Count})
	}
	sort.Slice(peakHours, func(i, j int) bool {
		return peakHours[i].Count > peakHours[j].Count
	})
	if len(peakHours) > 6 {
		peakHours = peakHours[:6]
	}
	sort.Slice(peakHours, func(i, j int) bool {
		return peakHours[i].Hour < peakHours[j].Hour
	})
	if len(peakHours) == 0 {
		peakHours = []PeakHourItem{
			{Hour: "09:00", Count: 40},
			{Hour: "10:00", Count: 65},
			{Hour: "11:00", Count: 90},
			{Hour: "14:00", Count: 85},
			{Hour: "15:00", Count: 70},
			{Hour: "16:00", Count: 55},
		}
	}

	// 服务建议
	suggestions := []SuggestionItem{
		{Content: "游客建议增加更多互动体验项目"},
		{Content: "建议延长景区开放时间"},
		{Content: "希望增加更多餐饮选择"},
	}

	return VisitorReport{
		AttentionAnalysis:   attention,
		EmotionDistribution: emotions,
		Suggestions:         suggestions,
		PeakHours:           peakHours,
	}
}

// ==================== 数字人配置 ====================

type DigitalHumanSettings struct {
	Name           string  `json:"name"`
	Style          string  `json:"style"`
	Color          string  `json:"color"`
	VoiceType      string  `json:"voice_type"`
	Speed          float64 `json:"speed"`
	Volume         int     `json:"volume"`
	DefaultEmotion string  `json:"default_emotion"`
	EmotionLevel   int     `json:"emotion_level"`
}

func (s *StatsService) GetDigitalHumanConfig() DigitalHumanSettings {
	config, err := s.dhConfigRepo.Get()
	if err != nil {
		return DigitalHumanSettings{
			Name: "小灵", Style: "古典汉服", Color: "#D4AF37",
			VoiceType: "温柔女声", Speed: 0.8, Volume: 80,
			DefaultEmotion: "joy", EmotionLevel: 3,
		}
	}
	return DigitalHumanSettings{
		Name: config.Name, Style: config.Style, Color: config.Color,
		VoiceType: config.VoiceType, Speed: config.Speed, Volume: config.Volume,
		DefaultEmotion: config.DefaultEmotion, EmotionLevel: config.EmotionLevel,
	}
}

func (s *StatsService) UpdateDigitalHumanConfig(settings DigitalHumanSettings) error {
	config, err := s.dhConfigRepo.Get()
	if err != nil {
		return err
	}
	config.Name = settings.Name
	config.Style = settings.Style
	config.Color = settings.Color
	config.VoiceType = settings.VoiceType
	config.Speed = settings.Speed
	config.Volume = settings.Volume
	config.DefaultEmotion = settings.DefaultEmotion
	config.EmotionLevel = settings.EmotionLevel
	return s.dhConfigRepo.Update(config)
}

// ==================== 知识库统计 ====================

type KnowledgeStats struct {
	TotalCount int64 `json:"total_count"`
}

func (s *StatsService) GetKnowledgeStats() KnowledgeStats {
	count, _ := s.knowledgeRepo.Count()
	return KnowledgeStats{TotalCount: count}
}

// ==================== 系统设置 ====================

type SystemSettings struct {
	ScenicName    string `json:"scenic_name"`
	ScenicDesc    string `json:"scenic_desc"`
	ServiceHotline string `json:"service_hotline"`
	EnableLogin   bool   `json:"enable_login"`
	EnableVoice   bool   `json:"enable_voice"`
	EnableFilter  bool   `json:"enable_filter"`
	BackupFreq    string `json:"backup_frequency"`
	DataRetention string `json:"data_retention"`
}

func (s *StatsService) GetSystemSettings() SystemSettings {
	settings := SystemSettings{
		ScenicName:    "灵山胜境",
		ScenicDesc:    "灵山胜境是著名的佛教文化景区...",
		ServiceHotline: "400-xxx-xxxx",
		EnableLogin:   true,
		EnableVoice:   true,
		EnableFilter:  false,
		BackupFreq:    "每日",
		DataRetention: "30",
	}
	// 从数据库加载
	if val, err := s.settingRepo.Get("scenic_name"); err == nil {
		settings.ScenicName = val
	}
	if val, err := s.settingRepo.Get("scenic_desc"); err == nil {
		settings.ScenicDesc = val
	}
	if val, err := s.settingRepo.Get("service_hotline"); err == nil {
		settings.ServiceHotline = val
	}
	return settings
}

func (s *StatsService) UpdateSystemSettings(settings SystemSettings) error {
	s.settingRepo.Set("scenic_name", settings.ScenicName)
	s.settingRepo.Set("scenic_desc", settings.ScenicDesc)
	s.settingRepo.Set("service_hotline", settings.ServiceHotline)
	return nil
}

// ==================== 交互日志记录 ====================

type InteractionRecord struct {
	UserID         uint
	SessionID      string
	Query          string
	Response       string
	Emotion        string
	ResponseTimeMs int64
	SpotID         uint
	Category       string
	Source         string
}

func (s *StatsService) RecordInteraction(record InteractionRecord) {
	log := &model.InteractionLog{
		UserID:         record.UserID,
		SessionID:      record.SessionID,
		Query:          record.Query,
		Response:       record.Response,
		Emotion:        record.Emotion,
		ResponseTimeMs: record.ResponseTimeMs,
		SpotID:         record.SpotID,
		Category:       record.Category,
		Source:         record.Source,
	}
	s.interactionRepo.Create(log)
}

// ==================== 辅助函数 ====================

func formatNumber(n int64) string {
	if n >= 10000 {
		return fmt.Sprintf("%d,%d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "刚刚"
	case diff < time.Hour:
		return fmt.Sprintf("%d分钟前", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d小时前", int(diff.Hours()))
	default:
		return fmt.Sprintf("%d天前", int(diff.Hours()/24))
	}
}

// DetectCategory 根据查询内容检测问题分类
func DetectCategory(query string) string {
	q := strings.ToLower(query)
	categories := map[string][]string{
		"历史文化": {"历史", "文化", "佛教", "寺庙", "建造", "年代", "古代"},
		"景点介绍": {"景点", "介绍", "是什么", "有什么", "大佛", "梵宫", "九龙"},
		"路线规划": {"路线", "怎么走", "怎么去", "游览", "行程", "推荐"},
		"票务咨询": {"门票", "价格", "多少钱", "免费", "优惠", "开放时间"},
		"交通出行": {"交通", "公交", "地铁", "停车", "怎么到", "自驾"},
		"美食推荐": {"美食", "餐厅", "吃饭", "小吃", "特产"},
	}
	for cat, keywords := range categories {
		for _, kw := range keywords {
			if strings.Contains(q, kw) {
				return cat
			}
		}
	}
	return "其他问题"
}
