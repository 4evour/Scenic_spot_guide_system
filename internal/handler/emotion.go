package handler

import "strings"

// detectEmotion 基于关键词的简单情绪检测
// 注意：中文关键词无需 ToLower，因为中文没有大小写之分
func detectEmotion(text string) string {
	// 负面情绪
	negativeWords := []string{"抱歉", "对不起", "无法", "遗憾", "不好", "糟糕", "失望", "难过"}
	for _, w := range negativeWords {
		if strings.Contains(text, w) {
			return "sadness"
		}
	}
	// 惊喜/注意
	surpriseWords := []string{"哇", "惊喜", "注意", "提醒", "警告", "竟然", "居然"}
	for _, w := range surpriseWords {
		if strings.Contains(text, w) {
			return "surprise"
		}
	}
	// 正面情绪
	positiveWords := []string{"欢迎", "您好", "很高兴", "推荐", "建议", "最佳", "精彩", "美丽", "壮观", "开心"}
	for _, w := range positiveWords {
		if strings.Contains(text, w) {
			return "joy"
		}
	}
	return "neutral"
}
