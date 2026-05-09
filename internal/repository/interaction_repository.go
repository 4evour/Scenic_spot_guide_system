package repository

import (
	"time"

	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type InteractionRepository struct {
	db *gorm.DB
}

func NewInteractionRepository(db *gorm.DB) *InteractionRepository {
	return &InteractionRepository{db: db}
}

// Create 记录一条交互日志
func (r *InteractionRepository) Create(log *model.InteractionLog) error {
	return r.db.Create(log).Error
}

// GetTotalCount 获取总交互次数
func (r *InteractionRepository) GetTotalCount(since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

// GetUniqueSessionCount 获取指定时间后的去重会话数
func (r *InteractionRepository) GetUniqueSessionCount(since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ?", since).
		Distinct("session_id").
		Count(&count).Error
	return count, err
}

// GetAvgResponseTime 获取平均响应时间
func (r *InteractionRepository) GetAvgResponseTime(since time.Time) (float64, error) {
	var avg float64
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ? AND response_time_ms > 0", since).
		Select("AVG(response_time_ms)").
		Scan(&avg).Error
	return avg, err
}

// GetEmotionDistribution 获取情感分布
func (r *InteractionRepository) GetEmotionDistribution(since time.Time) (map[string]int64, error) {
	type result struct {
		Emotion string
		Count   int64
	}
	var results []result
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ? AND emotion != ''", since).
		Select("emotion, COUNT(*) as count").
		Group("emotion").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	dist := make(map[string]int64)
	for _, r := range results {
		dist[r.Emotion] = r.Count
	}
	return dist, nil
}

// GetTopQuestions 获取热门问题 Top N
func (r *InteractionRepository) GetTopQuestions(since time.Time, limit int) ([]QuestionCount, error) {
	var results []QuestionCount
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ?", since).
		Select("query as question, COUNT(*) as count").
		Group("query").
		Order("count DESC").
		Limit(limit).
		Find(&results).Error
	return results, err
}

type QuestionCount struct {
	Question string
	Count    int64
}

// GetCategoryDistribution 获取问题分类分布
func (r *InteractionRepository) GetCategoryDistribution(since time.Time) ([]CategoryCount, error) {
	var results []CategoryCount
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ? AND category != ''", since).
		Select("category, COUNT(*) as count").
		Group("category").
		Order("count DESC").
		Find(&results).Error
	return results, err
}

type CategoryCount struct {
	Category string
	Count    int64
}

// GetResponseTimeDistribution 获取响应时间分布
func (r *InteractionRepository) GetResponseTimeDistribution(since time.Time) (map[string]int64, error) {
	var results []struct {
		Bucket string
		Count  int64
	}
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ? AND response_time_ms > 0", since).
		Select(`
			CASE
				WHEN response_time_ms < 1000 THEN '<1s'
				WHEN response_time_ms < 3000 THEN '1-3s'
				WHEN response_time_ms < 5000 THEN '3-5s'
				ELSE '>5s'
			END as bucket,
			COUNT(*) as count
		`).
		Group("bucket").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	dist := make(map[string]int64)
	for _, r := range results {
		dist[r.Bucket] = r.Count
	}
	return dist, nil
}

// GetHourlyStats 获取按小时统计的交互量
func (r *InteractionRepository) GetHourlyStats(since time.Time) ([]HourlyStat, error) {
	var results []HourlyStat
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ?", since).
		Select("strftime('%H', created_at) as hour, COUNT(*) as count").
		Group("hour").
		Order("hour").
		Find(&results).Error
	return results, err
}

type HourlyStat struct {
	Hour  string
	Count int64
}

type DailyEmotionStat struct {
	Date     string
	Emotion  string
	Count    int64
	Positive int64
	Negative int64
	Total    int64
}

type DailySatisfactionStat struct {
	Date             string
	Total            int64
	Positive         int64
	SatisfactionRate float64
}

// GetDailyEmotionTrend 获取每日情绪趋势
func (r *InteractionRepository) GetDailyEmotionTrend(since time.Time) ([]DailyEmotionStat, error) {
	var results []DailyEmotionStat
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ?", since).
		Select(`
			strftime('%Y-%m-%d', created_at) as date,
			SUM(CASE WHEN emotion IN ('joy', 'surprise') THEN 1 ELSE 0 END) as positive,
			SUM(CASE WHEN emotion IN ('sadness', 'fear', 'anger', 'disgust') THEN 1 ELSE 0 END) as negative,
			COUNT(*) as total
		`).
		Group("date").
		Order("date").
		Find(&results).Error
	return results, err
}

// GetDailySatisfactionTrend 获取每日满意度趋势
func (r *InteractionRepository) GetDailySatisfactionTrend(since time.Time) ([]DailySatisfactionStat, error) {
	var results []DailySatisfactionStat
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ?", since).
		Select(`
			strftime('%Y-%m-%d', created_at) as date,
			COUNT(*) as total,
			SUM(CASE WHEN emotion IN ('joy', 'surprise') THEN 1 ELSE 0 END) as positive
		`).
		Group("date").
		Order("date").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	for i := range results {
		if results[i].Total > 0 {
			results[i].SatisfactionRate = float64(results[i].Positive) / float64(results[i].Total) * 100
		}
	}
	return results, nil
}

// GetRecentConversations 获取最近N条对话
func (r *InteractionRepository) GetRecentConversations(limit int) ([]model.InteractionLog, error) {
	var logs []model.InteractionLog
	err := r.db.Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// GetTodayTotal 获取今日服务总人次 (unique sessions)
func (r *InteractionRepository) GetTodayTotal() (int64, error) {
	today := time.Now().Truncate(24 * time.Hour)
	var count int64
	err := r.db.Model(&model.InteractionLog{}).
		Where("created_at >= ?", today).
		Distinct("session_id").
		Count(&count).Error
	return count, err
}

// GetSatisfactionRate 计算满意度 (正面情绪占比)
func (r *InteractionRepository) GetSatisfactionRate(since time.Time) (float64, error) {
	dist, err := r.GetEmotionDistribution(since)
	if err != nil {
		return 0, err
	}
	var total, positive int64
	for emotion, count := range dist {
		total += count
		if emotion == "joy" || emotion == "surprise" {
			positive += count
		}
	}
	if total == 0 {
		return 0, nil
	}
	return float64(positive) / float64(total) * 100, nil
}
