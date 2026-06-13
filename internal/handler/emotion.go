package handler

import (
	"math"
	"strings"
)

// emotionKeyword 带权重的情绪关键词
type emotionKeyword struct {
	word   string
	weight float64 // 0.0–1.0
}

// emotionEntry 一组同情绪的关键词表
type emotionEntry struct {
	name     string
	keywords []emotionKeyword
}

// 情绪关键词库 (按检测优先级排序)
// sadness 优先 → 先道歉再讲内容符合景区导览习惯
var emotionEntries = []emotionEntry{
	{
		name: "anger",
		keywords: []emotionKeyword{
			{"愤怒", 0.9}, {"生气", 0.8}, {"讨厌", 0.8}, {"烦", 0.7},
			{"滚", 0.9}, {"无聊", 0.6}, {"差劲", 0.8}, {"恶心", 0.8},
			{"可恶", 0.8}, {"过分", 0.7}, {"烦人", 0.7}, {"火大", 0.8},
		},
	},
	{
		name: "disgust",
		keywords: []emotionKeyword{
			{"恶心", 0.9}, {"反胃", 0.8}, {"厌恶", 0.8}, {"嫌弃", 0.7},
			{"龌龊", 0.8}, {"肮脏", 0.8}, {"令人作呕", 0.9},
		},
	},
	{
		name: "sadness",
		keywords: []emotionKeyword{
			{"抱歉", 0.6}, {"对不起", 0.7}, {"无法", 0.5}, {"遗憾", 0.6},
			{"不好", 0.5}, {"糟糕", 0.7}, {"失望", 0.7}, {"难过", 0.7},
			{"伤心", 0.7}, {"可惜", 0.6}, {"悲痛", 0.8}, {"遗憾地", 0.6},
			{"不幸", 0.7}, {"无奈", 0.6}, {"没办法", 0.6},
		},
	},
	{
		name: "fear",
		keywords: []emotionKeyword{
			{"害怕", 0.8}, {"担心", 0.6}, {"恐怖", 0.8}, {"可怕", 0.8},
			{"吓人", 0.7}, {"危险", 0.7}, {"小心", 0.5}, {"恐惧", 0.8},
			{"惊慌", 0.8}, {"紧张", 0.5},
		},
	},
	{
		name: "surprise",
		keywords: []emotionKeyword{
			{"哇", 0.7}, {"惊喜", 0.7}, {"注意", 0.5}, {"提醒", 0.5},
			{"警告", 0.6}, {"竟然", 0.7}, {"居然", 0.7}, {"天哪", 0.8},
			{"不可思议", 0.8}, {"震惊", 0.8}, {"难以置信", 0.8}, {"突然", 0.5},
			{"意外", 0.6}, {"吓一跳", 0.7}, {"没想到", 0.7},
		},
	},
	{
		name: "joy",
		keywords: []emotionKeyword{
			{"欢迎", 0.6}, {"您好", 0.6}, {"很高兴", 0.8}, {"推荐", 0.5},
			{"建议", 0.4}, {"最佳", 0.5}, {"精彩", 0.7}, {"美丽", 0.7},
			{"壮观", 0.7}, {"开心", 0.8}, {"太棒了", 0.9}, {"真不错", 0.8},
			{"好极了", 0.8}, {"完美", 0.7}, {"赞", 0.7}, {"厉害", 0.7},
			{"真好", 0.7}, {"喜欢", 0.7}, {"愉快", 0.7}, {"幸福", 0.8},
			{"很棒", 0.8}, {"太好了", 0.8}, {"不错", 0.5}, {"谢谢", 0.5},
			{"感谢", 0.5},
		},
	},
}

// detectEmotion 基于关键词的简单情绪检测
func detectEmotion(text string) string {
	emotion, _ := detectEmotionWithIntensity(text)
	return emotion
}

// detectEmotionWithIntensity 基于关键词加权检测，同时返回强度和情绪
// intensity 范围 0.0–1.0，关键词权重加权平均
func detectEmotionWithIntensity(text string) (emotion string, intensity float64) {
	if text == "" {
		return "neutral", 0
	}

	textLen := float64(len([]rune(text)))
	if textLen == 0 {
		return "neutral", 0
	}

	type match struct {
		emotion string
		weight  float64
		count   int
	}

	var matches []match

	for _, entry := range emotionEntries {
		m := match{emotion: entry.name}
		for _, kw := range entry.keywords {
			if strings.Contains(text, kw.word) {
				m.weight += kw.weight
				m.count++
			}
		}
		if m.count > 0 {
			matches = append(matches, m)
		}
	}

	if len(matches) == 0 {
		return "neutral", 0
	}

	// 选权重最高的情绪
	best := matches[0]
	for _, m := range matches[1:] {
		if m.weight > best.weight {
			best = m
		}
	}

	// 强度 = 总权重 / (关键词数 + 文本长度阻尼)
	// 用 sqrt 防止长文本中少量关键词被过度稀释
	damping := math.Sqrt(textLen) * 0.15
	rawIntensity := best.weight / (float64(best.count) + damping)
	intensity = math.Min(rawIntensity, 1.0)
	if intensity < 0.05 {
		return "neutral", 0
	}

	return best.emotion, intensity
}
