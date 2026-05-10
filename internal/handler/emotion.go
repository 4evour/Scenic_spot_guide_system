package handler

import "strings"

func detectEmotion(text string) string {
	textLower := strings.ToLower(text)

	if strings.Contains(textLower, "抱歉") || strings.Contains(textLower, "对不起") || strings.Contains(textLower, "无法") {
		return "sadness"
	}
	if strings.Contains(textLower, "欢迎") || strings.Contains(textLower, "您好") || strings.Contains(textLower, "很高兴") {
		return "joy"
	}
	if strings.Contains(textLower, "推荐") || strings.Contains(textLower, "建议") || strings.Contains(textLower, "最佳") {
		return "joy"
	}
	if strings.Contains(textLower, "注意") || strings.Contains(textLower, "提醒") || strings.Contains(textLower, "警告") {
		return "surprise"
	}
	if strings.Contains(textLower, "不好") || strings.Contains(textLower, "糟糕") || strings.Contains(textLower, "问题") {
		return "fear"
	}
	return "neutral"
}
