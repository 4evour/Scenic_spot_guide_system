package service

import (
	"fmt"
	"log/slog"
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
	TotalVisitors     string  `json:"total_visitors"`     // 服务人次
	WeeklyVisitors    string  `json:"weekly_visitors"`    // 本周服务人次
	TotalChats        string  `json:"total_chats"`        // 交互次数
	WeeklyChats       string  `json:"weekly_chats"`       // 本周交互次数
	SatisfactionRate  string  `json:"satisfaction_rate"`  // 满意度
	AvgResponseTime   string  `json:"avg_response_time"`  // 平均响应时间
	VisitorsTrend     float64 `json:"visitors_trend"`     // 服务人次趋势%
	ChatsTrend        float64 `json:"chats_trend"`        // 交互次数趋势%
	SatisfactionTrend float64 `json:"satisfaction_trend"` // 满意度趋势%
	ResponseTrend     float64 `json:"response_trend"`     // 响应时间趋势%
}

func (s *StatsService) GetDashboardOverview() DashboardOverview {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)
	weekStart := today.AddDate(0, 0, -int(today.Weekday()+6)%7)

	todayVisitors, err := s.interactionRepo.GetUniqueSessionCount(today)
	if err != nil {
		slog.Error("查询今日访问量失败", "error", err)
	}
	yesterdayVisitors, err := s.interactionRepo.GetUniqueSessionCount(yesterday)
	if err != nil {
		slog.Error("查询昨日访问量失败", "error", err)
	}
	weeklyVisitors, err := s.interactionRepo.GetUniqueSessionCount(weekStart)
	if err != nil {
		slog.Error("查询本周访问量失败", "error", err)
	}
	todayChats, err := s.interactionRepo.GetTotalCount(today)
	if err != nil {
		slog.Error("查询今日交互次数失败", "error", err)
	}
	weeklyChats, err := s.interactionRepo.GetTotalCount(weekStart)
	if err != nil {
		slog.Error("查询本周交互次数失败", "error", err)
	}
	yesterdayChats, err := s.interactionRepo.GetTotalCount(yesterday)
	if err != nil {
		slog.Error("查询昨日交互次数失败", "error", err)
	}

	avgRT, err := s.interactionRepo.GetAvgResponseTime(today)
	if err != nil {
		slog.Error("查询平均响应时间失败", "error", err)
	}
	satisfaction, err := s.interactionRepo.GetSatisfactionRate(today)
	if err != nil {
		slog.Error("查询满意度失败", "error", err)
	}
	yesterdaySatisfaction, err := s.interactionRepo.GetSatisfactionRate(yesterday)
	if err != nil {
		slog.Error("查询昨日满意度失败", "error", err)
	}
	yesterdayAvgRT, err := s.interactionRepo.GetAvgResponseTime(yesterday)
	if err != nil {
		slog.Error("查询昨日平均响应时间失败", "error", err)
	}

	// 计算趋势
	var chatsTrend, satisfactionTrend, responseTrend, visitorsTrend float64
	if yesterdayChats > 0 {
		chatsTrend = float64(todayChats-yesterdayChats) / float64(yesterdayChats) * 100
	}
	if yesterdayVisitors > 0 {
		visitorsTrend = float64(todayVisitors-yesterdayVisitors) / float64(yesterdayVisitors) * 100
	}
	if yesterdaySatisfaction > 0 {
		satisfactionTrend = satisfaction - yesterdaySatisfaction
	}
	if yesterdayAvgRT > 0 {
		responseTrend = (yesterdayAvgRT - avgRT) / yesterdayAvgRT * 100
	}

	avgRTSeconds := avgRT / 1000

	return DashboardOverview{
		TotalVisitors:     formatNumber(todayVisitors),
		WeeklyVisitors:    formatNumber(weeklyVisitors),
		TotalChats:        fmt.Sprintf("%d", todayChats),
		WeeklyChats:       fmt.Sprintf("%d", weeklyChats),
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
	stats, err := s.interactionRepo.GetHourlyStats(today)
	if err != nil {
		slog.Error("查询小时趋势失败", "error", err)
	}

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
	questions, err := s.interactionRepo.GetTopQuestions(today, limit)
	if err != nil {
		slog.Error("查询热门问题失败", "error", err)
	}
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
	cats, err := s.interactionRepo.GetCategoryDistribution(today)
	if err != nil {
		slog.Error("查询分类分布失败", "error", err)
	}

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

type SatisfactionTrendItem struct {
	Date  string  `json:"date"`
	Rate  float64 `json:"rate"`
	Total int64   `json:"total"`
}

func (s *StatsService) GetResponseTimeDistribution() []ResponseTimeDistItem {
	today := time.Now().Truncate(24 * time.Hour)
	dist, err := s.interactionRepo.GetResponseTimeDistribution(today)
	if err != nil {
		slog.Error("查询响应时间分布失败", "error", err)
	}

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

func (s *StatsService) GetSatisfactionTrend() []SatisfactionTrendItem {
	since := time.Now().AddDate(0, 0, -6).Truncate(24 * time.Hour)
	stats, err := s.interactionRepo.GetDailySatisfactionTrend(since)
	if err != nil {
		slog.Error("查询满意度趋势失败", "error", err)
	}
	byDate := make(map[string]repository.DailySatisfactionStat)
	for _, stat := range stats {
		byDate[stat.Date] = stat
	}

	result := make([]SatisfactionTrendItem, 0, 7)
	for i := 0; i < 7; i++ {
		day := since.AddDate(0, 0, i).Format("2006-01-02")
		stat := byDate[day]
		result = append(result, SatisfactionTrendItem{
			Date:  day,
			Rate:  math.Round(stat.SatisfactionRate*10) / 10,
			Total: stat.Total,
		})
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
	logs, err := s.interactionRepo.GetRecentConversations(limit)
	if err != nil {
		slog.Error("查询最近对话失败", "error", err)
	}
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
	AttentionAnalysis   []AttentionItem    `json:"attention_analysis"`
	EmotionDistribution []EmotionItem      `json:"emotion_distribution"`
	EmotionTrend        []EmotionTrendItem `json:"emotion_trend"`
	Suggestions         []SuggestionItem   `json:"suggestions"`
	PeakHours           []PeakHourItem     `json:"peak_hours"`
	Summary             ReportSummary      `json:"summary"`
}

type AttentionItem struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type EmotionItem struct {
	Label   string  `json:"label"`
	Icon    string  `json:"icon"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type SuggestionItem struct {
	Content string `json:"content"`
}

type PeakHourItem struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

type EmotionTrendItem struct {
	Date         string  `json:"date"`
	PositiveRate float64 `json:"positive_rate"`
	NegativeRate float64 `json:"negative_rate"`
	Total        int64   `json:"total"`
}

type ReportSummary struct {
	TotalInteractions int64   `json:"total_interactions"`
	SatisfactionRate  float64 `json:"satisfaction_rate"`
	NegativeRate      float64 `json:"negative_rate"`
	TopConcern        string  `json:"top_concern"`
	PeakHour          string  `json:"peak_hour"`
}

func (s *StatsService) GetVisitorReport() VisitorReport {
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	totalInteractions, err := s.interactionRepo.GetTotalCount(weekAgo)
	if err != nil {
		slog.Error("查询交互总数失败", "error", err)
	}

	// 关注点分析 - 从分类分布获取
	cats, err := s.interactionRepo.GetCategoryDistribution(weekAgo)
	if err != nil {
		slog.Error("查询分类分布失败", "error", err)
	}
	var total int64
	for _, c := range cats {
		total += c.Count
	}

	attentionLabels := map[string]string{
		"历史": "历史文化",
		"景点": "自然风光",
		"路线": "路线规划",
		"美食": "美食推荐",
		"门票": "票务咨询",
		"拍照": "拍照打卡",
		"活动": "亲子活动",
		"时间": "开放时间",
		"交通": "交通出行",
	}

	attention := make([]AttentionItem, 0)
	topConcern := "暂无数据"
	var topConcernValue float64
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
		value := math.Round(pct)
		if value > topConcernValue {
			topConcernValue = value
			topConcern = label
		}
		attention = append(attention, AttentionItem{Label: label, Value: value})
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
		topConcern = "拍照打卡"
	}

	// 情感分布
	dist, err := s.interactionRepo.GetEmotionDistribution(weekAgo)
	if err != nil {
		slog.Error("查询情感分布失败", "error", err)
	}
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
	satisfactionRate := findEmotionPercent(emotions, "正面")
	negativeRate := findEmotionPercent(emotions, "负面")

	emotionTrend := s.buildEmotionTrend(weekAgo)

	// 热门时段
	hourlyStats, err := s.interactionRepo.GetHourlyStats(weekAgo)
	if err != nil {
		slog.Error("查询小时统计失败", "error", err)
	}
	var peakHours []PeakHourItem
	peakHour := "暂无数据"
	var maxPeakCount int64
	for _, hs := range hourlyStats {
		item := PeakHourItem{Hour: hs.Hour + ":00", Count: hs.Count}
		if hs.Count > maxPeakCount {
			maxPeakCount = hs.Count
			peakHour = item.Hour
		}
		peakHours = append(peakHours, item)
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
		peakHour = "11:00"
	}

	suggestions := buildServiceSuggestions(attention, emotions, peakHour, totalInteractions)

	return VisitorReport{
		AttentionAnalysis:   attention,
		EmotionDistribution: emotions,
		EmotionTrend:        emotionTrend,
		Suggestions:         suggestions,
		PeakHours:           peakHours,
		Summary: ReportSummary{
			TotalInteractions: totalInteractions,
			SatisfactionRate:  satisfactionRate,
			NegativeRate:      negativeRate,
			TopConcern:        topConcern,
			PeakHour:          peakHour,
		},
	}
}

func (s *StatsService) buildEmotionTrend(since time.Time) []EmotionTrendItem {
	stats, err := s.interactionRepo.GetDailyEmotionTrend(since)
	if err != nil {
		slog.Error("查询情感趋势失败", "error", err)
	}
	byDate := make(map[string]repository.DailyEmotionStat)
	for _, item := range stats {
		byDate[item.Date] = item
	}

	result := make([]EmotionTrendItem, 0, 7)
	start := time.Now().AddDate(0, 0, -6)
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		stat := byDate[day]
		item := EmotionTrendItem{Date: day, Total: stat.Total}
		if stat.Total > 0 {
			item.PositiveRate = math.Round(float64(stat.Positive) / float64(stat.Total) * 100)
			item.NegativeRate = math.Round(float64(stat.Negative) / float64(stat.Total) * 100)
		}
		result = append(result, item)
	}

	return result
}

func findEmotionPercent(items []EmotionItem, label string) float64 {
	for _, item := range items {
		if item.Label == label {
			return item.Percent
		}
	}
	return 0
}

func buildServiceSuggestions(attention []AttentionItem, emotions []EmotionItem, peakHour string, totalInteractions int64) []SuggestionItem {
	suggestions := make([]SuggestionItem, 0, 5)

	if totalInteractions == 0 {
		return []SuggestionItem{
			{Content: "近 7 天暂无有效交互记录，建议先引导游客通过数字人入口咨询路线、票务和服务设施问题。"},
			{Content: "运营初期可预置讲解词、常见问答和游客服务知识，提升首批问答覆盖率。"},
		}
	}

	sort.Slice(attention, func(i, j int) bool {
		return attention[i].Value > attention[j].Value
	})
	if len(attention) > 0 {
		top := attention[0]
		suggestions = append(suggestions, SuggestionItem{
			Content: fmt.Sprintf("游客最关注「%s」（%.0f%%），建议在首页、数字人欢迎语和知识库中优先强化相关指引。", top.Label, top.Value),
		})
	}

	negativeRate := findEmotionPercent(emotions, "负面")
	if negativeRate >= 20 {
		suggestions = append(suggestions, SuggestionItem{
			Content: fmt.Sprintf("负面情绪占比 %.0f%%，建议复盘低分对话并补充票务、排队、交通等易引发焦虑的问题答案。", negativeRate),
		})
	} else {
		suggestions = append(suggestions, SuggestionItem{
			Content: "当前负面情绪占比较低，可继续保持亲切讲解语气，并重点优化高频问题的回答精度。",
		})
	}

	if peakHour != "" && peakHour != "暂无数据" {
		suggestions = append(suggestions, SuggestionItem{
			Content: fmt.Sprintf("交互高峰集中在 %s 附近，建议该时段加强入口引导、路线推荐和现场服务人员联动。", peakHour),
		})
	}

	for _, item := range attention {
		if strings.Contains(item.Label, "票务") || strings.Contains(item.Label, "交通") {
			suggestions = append(suggestions, SuggestionItem{
				Content: "票务或交通咨询热度较高，建议同步检查开放时间、停车、接驳和优惠政策信息是否最新。",
			})
			break
		}
	}

	return suggestions
}

// ==================== 数字人配置 ====================

type DigitalHumanSettings struct {
	Name           string  `json:"name"`
	Appearance     string  `json:"appearance"`
	Costume        string  `json:"costume"`
	Style          string  `json:"style"`
	Color          string  `json:"color"`
	CultureTheme   string  `json:"culture_theme"`
	VoiceType      string  `json:"voice_type"`
	VoiceTone      string  `json:"voice_tone"`
	Speed          float64 `json:"speed"`
	Volume         int     `json:"volume"`
	Greeting       string  `json:"greeting"`
	DefaultEmotion string  `json:"default_emotion"`
	EmotionLevel   int     `json:"emotion_level"`
}

func (s *StatsService) GetDigitalHumanConfig() DigitalHumanSettings {
	config, err := s.dhConfigRepo.Get()
	if err != nil {
		return DigitalHumanSettings{
			Name: "小灵", Appearance: "亲和型国风讲解员", Costume: "古典汉服",
			Style: "古典汉服", Color: "#D4AF37", CultureTheme: "灵山佛教文化与江南山水意境",
			VoiceType: "温柔女声", VoiceTone: "温暖、端庄、亲切", Speed: 0.8, Volume: 80,
			Greeting:       "欢迎来到灵山胜境，我是您的数字导览员小灵。",
			DefaultEmotion: "joy", EmotionLevel: 3,
		}
	}
	return DigitalHumanSettings{
		Name: config.Name, Appearance: config.Appearance, Costume: config.Costume,
		Style: config.Style, Color: config.Color, CultureTheme: config.CultureTheme,
		VoiceType: config.VoiceType, VoiceTone: config.VoiceTone, Speed: config.Speed, Volume: config.Volume,
		Greeting:       config.Greeting,
		DefaultEmotion: config.DefaultEmotion, EmotionLevel: config.EmotionLevel,
	}
}

func (s *StatsService) UpdateDigitalHumanConfig(settings DigitalHumanSettings) error {
	config, err := s.dhConfigRepo.Get()
	if err != nil {
		// 获取失败时构造默认记录并创建
		config = &model.DigitalHumanConfig{
			Name: "小灵", Appearance: "亲和型国风讲解员", Costume: "古典汉服",
		}
	}
	config.Name = settings.Name
	config.Appearance = settings.Appearance
	config.Costume = settings.Costume
	config.Style = settings.Style
	config.Color = settings.Color
	config.CultureTheme = settings.CultureTheme
	config.VoiceType = settings.VoiceType
	config.VoiceTone = settings.VoiceTone
	config.Speed = settings.Speed
	config.Volume = settings.Volume
	config.Greeting = settings.Greeting
	config.DefaultEmotion = settings.DefaultEmotion
	config.EmotionLevel = settings.EmotionLevel
	return s.dhConfigRepo.Update(config)
}

// ==================== 知识库统计 ====================

type KnowledgeStats struct {
	TotalCount int64 `json:"total_count"`
}

func (s *StatsService) GetKnowledgeStats() KnowledgeStats {
	count, err := s.knowledgeRepo.Count()
	if err != nil {
		slog.Error("查询知识库数量失败", "error", err)
	}
	return KnowledgeStats{TotalCount: count}
}

// ==================== 系统设置 ====================

type SystemSettings struct {
	ScenicName     string `json:"scenic_name"`
	ScenicDesc     string `json:"scenic_desc"`
	ServiceHotline string `json:"service_hotline"`
	EnableLogin    bool   `json:"enable_login"`
	EnableVoice    bool   `json:"enable_voice"`
	EnableFilter   bool   `json:"enable_filter"`
	BackupFreq     string `json:"backup_frequency"`
	DataRetention  string `json:"data_retention"`
}

func parseBoolOrDefault(val string, def bool) bool {
	if val == "" {
		return def
	}
	return val == "true"
}

func (s *StatsService) GetSystemSettings() SystemSettings {
	settings := SystemSettings{
		ScenicName:     "灵山胜境",
		ScenicDesc:     "灵山胜境是著名的佛教文化景区...",
		ServiceHotline: "400-xxx-xxxx",
		EnableLogin:    true,
		EnableVoice:    true,
		EnableFilter:   false,
		BackupFreq:     "每日",
		DataRetention:  "30",
	}
	if val, err := s.settingRepo.Get("scenic_name"); err == nil {
		settings.ScenicName = val
	}
	if val, err := s.settingRepo.Get("scenic_desc"); err == nil {
		settings.ScenicDesc = val
	}
	if val, err := s.settingRepo.Get("service_hotline"); err == nil {
		settings.ServiceHotline = val
	}
	if val, err := s.settingRepo.Get("enable_login"); err == nil {
		settings.EnableLogin = parseBoolOrDefault(val, true)
	}
	if val, err := s.settingRepo.Get("enable_voice"); err == nil {
		settings.EnableVoice = parseBoolOrDefault(val, true)
	}
	if val, err := s.settingRepo.Get("enable_filter"); err == nil {
		settings.EnableFilter = parseBoolOrDefault(val, false)
	}
	if val, err := s.settingRepo.Get("backup_frequency"); err == nil {
		settings.BackupFreq = val
	}
	if val, err := s.settingRepo.Get("data_retention"); err == nil {
		settings.DataRetention = val
	}
	return settings
}

func (s *StatsService) UpdateSystemSettings(settings SystemSettings) error {
	sets := []struct {
		key string
		val string
	}{
		{"scenic_name", settings.ScenicName},
		{"scenic_desc", settings.ScenicDesc},
		{"service_hotline", settings.ServiceHotline},
		{"enable_login", fmt.Sprintf("%t", settings.EnableLogin)},
		{"enable_voice", fmt.Sprintf("%t", settings.EnableVoice)},
		{"enable_filter", fmt.Sprintf("%t", settings.EnableFilter)},
		{"backup_frequency", settings.BackupFreq},
		{"data_retention", settings.DataRetention},
	}
	var firstErr error
	for _, entry := range sets {
		if err := s.settingRepo.Set(entry.key, entry.val); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
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
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
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
