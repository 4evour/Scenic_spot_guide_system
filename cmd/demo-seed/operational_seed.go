package main

import (
	"fmt"
	"time"

	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

const demoJudgePrefix = "demo-judge-"

type demoInteractionTemplate struct {
	query    string
	response string
	category string
}

var demoInteractionTemplates = []demoInteractionTemplate{
	{query: "灵山大佛有多高？", response: "灵山大佛通高88米，是景区核心地标。", category: "景点"},
	{query: "九龙灌浴什么时候演出？", response: "演出时段请以景区当天现场公告为准，建议提前到达广场。", category: "演出"},
	{query: "亲子游怎么安排？", response: "建议串联九龙灌浴、百子戏弥勒、梵宫和文创驿站。", category: "路线"},
	{query: "梵宫有什么特色？", response: "梵宫汇集木雕、琉璃、壁画等传统工艺与佛教艺术。", category: "文化"},
	{query: "老人适合哪条路线？", response: "可选择观光车参考路线，减少连续步行并保留核心景点。", category: "路线"},
	{query: "五印坛城讲什么文化？", response: "五印坛城以藏传佛教文化为主题，展示五方五佛、转经筒和唐卡。", category: "文化"},
	{query: "景区里哪里可以休息？", response: "文创驿站和主要服务节点可供短暂停留，现场安排以当天开放为准。", category: "服务"},
	{query: "推荐一条半日路线", response: "可按九龙灌浴、灵山梵宫、灵山大佛的顺序安排经典半日游。", category: "路线"},
	{query: "灵山大佛适合拍照吗？", response: "大佛广场视野开阔，拍摄时请遵守宗教场所和现场提示。", category: "景点"},
	{query: "无障碍游览方便吗？", response: "建议优先选择主游线和观光车组合，并向游客中心确认当天无障碍设施。", category: "服务"},
	{query: "景区有什么特色餐饮？", response: "景区内可关注灵山蔬食馆等餐饮服务，营业情况以现场为准。", category: "餐饮"},
	{query: "如何避开高峰客流？", response: "可采用错峰参考步行路线，并根据现场客流灵活调整游览顺序。", category: "客流"},
}

var demoHourlyVolume = map[int]int{
	8: 1, 9: 2, 10: 3, 11: 4, 12: 5, 13: 4,
	14: 3, 15: 4, 16: 5, 17: 3, 18: 2, 19: 1,
}

func buildDemoInteractions(now time.Time) []model.InteractionLog {
	anchor := now.Truncate(time.Hour).Add(-time.Hour)
	emotions := []string{"joy", "joy", "surprise", "joy", "neutral", "thinking", "joy", "satisfaction"}
	sources := []string{"web", "digital_human", "voice"}
	logs := make([]model.InteractionLog, 0, 520)
	counter := 0

	for hourOffset := 0; hourOffset < 14*24; hourOffset++ {
		eventHour := anchor.Add(-time.Duration(hourOffset) * time.Hour)
		volume := demoHourlyVolume[eventHour.Hour()]
		for index := 0; index < volume; index++ {
			template := demoInteractionTemplates[counter%len(demoInteractionTemplates)]
			logs = append(logs, model.InteractionLog{
				SessionID:      fmt.Sprintf("%ssession-%03d", demoJudgePrefix, counter/3),
				Query:          template.query,
				Response:       template.response,
				Emotion:        emotions[counter%len(emotions)],
				ResponseTimeMs: int64(780 + (counter%9)*125),
				Category:       template.category,
				Source:         sources[counter%len(sources)],
				CreatedAt:      eventHour.Add(time.Duration(index*9) * time.Minute),
			})
			counter++
		}
	}
	return logs
}

var demoConversationTitles = []string{
	"灵山大佛文化导览",
	"亲子半日游规划",
	"梵宫艺术特色",
	"观光车轻松路线",
	"五印坛城文化讲解",
	"景区服务设施咨询",
	"错峰游览建议",
	"拍照与参观礼仪",
}

func seedOperationalDemoData(db *gorm.DB, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var visitor model.User
		if err := tx.Where("username = ?", "visitor").First(&visitor).Error; err != nil {
			return fmt.Errorf("查询演示游客失败: %w", err)
		}
		if err := replaceDemoInteractions(tx, now); err != nil {
			return err
		}
		if err := replaceDemoChatHistory(tx, visitor.ID, now); err != nil {
			return err
		}
		if err := replaceDemoSpotRatings(tx, now); err != nil {
			return err
		}
		if err := replaceDemoRouteRecommendations(tx, now); err != nil {
			return err
		}
		return nil
	})
}

func replaceDemoInteractions(db *gorm.DB, now time.Time) error {
	legacySessionIDs := []string{"demo-001", "demo-002", "demo-003", "demo-004"}
	if err := db.Where("session_id LIKE ?", demoJudgePrefix+"%").
		Or("session_id IN ?", legacySessionIDs).
		Delete(&model.InteractionLog{}).Error; err != nil {
		return fmt.Errorf("清理旧演示交互失败: %w", err)
	}
	logs := buildDemoInteractions(now)
	if err := db.CreateInBatches(logs, 100).Error; err != nil {
		return fmt.Errorf("写入演示交互失败: %w", err)
	}
	return nil
}

func replaceDemoChatHistory(db *gorm.DB, userID uint, now time.Time) error {
	var sessionIDs []uint
	if err := db.Model(&model.ChatSession{}).
		Where("session_id LIKE ?", demoJudgePrefix+"chat-%").
		Pluck("id", &sessionIDs).Error; err != nil {
		return fmt.Errorf("查询旧演示会话失败: %w", err)
	}
	if len(sessionIDs) > 0 {
		if err := db.Where("chat_session_id IN ?", sessionIDs).Delete(&model.ChatMessage{}).Error; err != nil {
			return fmt.Errorf("清理旧演示消息失败: %w", err)
		}
	}
	if err := db.Where("session_id LIKE ?", demoJudgePrefix+"chat-%").Delete(&model.ChatSession{}).Error; err != nil {
		return fmt.Errorf("清理旧演示会话失败: %w", err)
	}

	for index, title := range demoConversationTitles {
		createdAt := now.Add(-time.Duration((index+1)*20) * time.Hour)
		session := model.ChatSession{
			UserID:       userID,
			SessionID:    fmt.Sprintf("%schat-%02d", demoJudgePrefix, index+1),
			Title:        title,
			Source:       []string{"digital_human", "web"}[index%2],
			MessageCount: 6,
			LastActiveAt: createdAt.Add(10 * time.Minute),
			CreatedAt:    createdAt,
			UpdatedAt:    createdAt.Add(10 * time.Minute),
		}
		if err := db.Create(&session).Error; err != nil {
			return fmt.Errorf("写入演示会话失败: %w", err)
		}

		messages := make([]model.ChatMessage, 0, 6)
		for turn := 0; turn < 3; turn++ {
			template := demoInteractionTemplates[(index*3+turn)%len(demoInteractionTemplates)]
			messageTime := createdAt.Add(time.Duration(turn*4) * time.Minute)
			messages = append(messages,
				model.ChatMessage{ChatSessionID: session.ID, UserID: userID, Role: "user", Content: template.query, CreatedAt: messageTime},
				model.ChatMessage{ChatSessionID: session.ID, UserID: userID, Role: "assistant", Content: template.response, Emotion: "joy", ResponseTimeMs: int64(900 + turn*180), CreatedAt: messageTime.Add(time.Minute)},
			)
		}
		if err := db.Create(&messages).Error; err != nil {
			return fmt.Errorf("写入演示消息失败: %w", err)
		}
	}
	return nil
}

func replaceDemoSpotRatings(db *gorm.DB, now time.Time) error {
	if err := db.Where("session_id LIKE ?", demoJudgePrefix+"rating-%").Delete(&model.VisitorSpotRating{}).Error; err != nil {
		return fmt.Errorf("清理旧演示景点评分失败: %w", err)
	}
	var spots []model.ScenicSpot
	if err := db.Order("id").Limit(12).Find(&spots).Error; err != nil {
		return fmt.Errorf("查询演示景点失败: %w", err)
	}
	if len(spots) == 0 {
		return fmt.Errorf("写入演示景点评分失败: 未找到景点")
	}

	comments := []string{"文化氛围很有特色", "路线清晰，适合家庭游览", "讲解内容详细", "拍照视野很好"}
	ratings := make([]model.VisitorSpotRating, 0, 12)
	for index := 0; index < 12; index++ {
		score := 5
		if index%4 == 3 {
			score = 4
		}
		ratings = append(ratings, model.VisitorSpotRating{
			SessionID:      fmt.Sprintf("%srating-%02d", demoJudgePrefix, index+1),
			SpotID:         spots[index%len(spots)].ID,
			OverallRating:  score,
			CultureRating:  5,
			PhotoRating:    4 + index%2,
			FacilityRating: 4,
			Comment:        comments[index%len(comments)],
			Tags:           `["文化体验","游览友好"]`,
			Sentiment:      "positive",
			CreatedAt:      now.Add(-time.Duration(index+1) * 8 * time.Hour),
			UpdatedAt:      now.Add(-time.Duration(index+1) * 8 * time.Hour),
		})
	}
	if err := db.Create(&ratings).Error; err != nil {
		return fmt.Errorf("写入演示景点评分失败: %w", err)
	}
	return nil
}

func replaceDemoRouteRecommendations(db *gorm.DB, now time.Time) error {
	if err := db.Where("session_id LIKE ?", demoJudgePrefix+"route-%").Delete(&model.RouteRecommendationLog{}).Error; err != nil {
		return fmt.Errorf("清理旧演示路线推荐失败: %w", err)
	}
	var routes []model.TourRoute
	if err := db.Order("id").Find(&routes).Error; err != nil {
		return fmt.Errorf("查询演示路线失败: %w", err)
	}
	if len(routes) == 0 {
		return fmt.Errorf("写入演示路线推荐失败: 未找到路线")
	}

	profiles := []string{"family", "culture", "senior", "photography"}
	recommendations := make([]model.RouteRecommendationLog, 0, 16)
	for index := 0; index < 16; index++ {
		route := routes[index%len(routes)]
		recommendations = append(recommendations, model.RouteRecommendationLog{
			SessionID:     fmt.Sprintf("%sroute-%02d", demoJudgePrefix, index+1),
			ProfileType:   profiles[index%len(profiles)],
			RouteName:     route.Name,
			SpotIDs:       "[]",
			InterestTags:  `["文化","轻松游览"]`,
			TotalDuration: route.Duration,
			ScoreSummary:  "根据游览时长、同行人群和兴趣偏好生成的演示推荐。",
			CreatedAt:     now.Add(-time.Duration(index+1) * 6 * time.Hour),
		})
	}
	if err := db.Create(&recommendations).Error; err != nil {
		return fmt.Errorf("写入演示路线推荐失败: %w", err)
	}
	return nil
}
