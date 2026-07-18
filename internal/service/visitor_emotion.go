package service

import (
	"math"
	"strings"
)

type VisitorEmotionCategory string

const (
	VisitorEmotionSatisfaction VisitorEmotionCategory = "satisfaction"
	VisitorEmotionQuestion     VisitorEmotionCategory = "question"
	VisitorEmotionComplaint    VisitorEmotionCategory = "complaint"
	VisitorEmotionAnxiety      VisitorEmotionCategory = "anxiety"
	VisitorEmotionExcitement   VisitorEmotionCategory = "excitement"
	VisitorEmotionNeutral      VisitorEmotionCategory = "neutral"
)

type VisitorEmotionResult struct {
	Category              VisitorEmotionCategory `json:"category"`
	LegacyToken           string                 `json:"legacy_token"`
	Confidence            float64                `json:"confidence"`
	Intensity             float64                `json:"intensity"`
	Evidence              []string               `json:"evidence,omitempty"`
	RecommendHumanService bool                   `json:"recommend_human_service"`
	Modality              string                 `json:"modality"`
	AcousticConfidence    float64                `json:"acoustic_confidence,omitempty"`
}

type VoiceAcousticFeatures struct {
	DurationMs               float64 `json:"duration_ms"`
	SampleCount              int     `json:"sample_count"`
	RMSMean                  float64 `json:"rms_mean"`
	RMSPeak                  float64 `json:"rms_peak"`
	RMSVariation             float64 `json:"rms_variation"`
	PauseRatio               float64 `json:"pause_ratio"`
	PitchMeanHz              float64 `json:"pitch_mean_hz"`
	PitchVariationHz         float64 `json:"pitch_variation_hz"`
	SpeechRateCharsPerSecond float64 `json:"speech_rate_chars_per_second"`
	RepetitionRatio          float64 `json:"repetition_ratio"`
}

type visitorEmotionRule struct {
	Category VisitorEmotionCategory
	Token    string
	Terms    []string
	Weight   float64
}

var visitorEmotionRules = []visitorEmotionRule{
	{Category: VisitorEmotionComplaint, Token: "anger", Weight: 1.0, Terms: []string{"投诉", "生气", "愤怒", "气死", "垃圾", "太差", "差劲", "坑人", "骗人", "失望", "不满", "不舒服", "太贵", "排队太久", "怎么这么慢"}},
	{Category: VisitorEmotionAnxiety, Token: "fear", Weight: 0.9, Terms: []string{"焦虑", "担心", "害怕", "紧张", "怎么办", "来不及", "赶不上", "迷路", "找不到", "安全吗", "危险", "不放心", "怕"}},
	{Category: VisitorEmotionExcitement, Token: "surprise", Weight: 0.8, Terms: []string{"期待", "兴奋", "哇", "惊喜", "太棒", "好开心", "迫不及待", "好壮观", "真精彩", "喜欢"}},
	{Category: VisitorEmotionSatisfaction, Token: "joy", Weight: 0.7, Terms: []string{"谢谢", "感谢", "辛苦了", "多谢", "满意", "很好", "不错", "有帮助", "明白了"}},
}

var visitorQuestionTerms = []string{"请问", "怎么", "哪里", "在哪", "多少", "几时", "几点", "能不能", "可以吗", "是否", "有没有", "为什么", "哪一个", "哪条", "？", "?"}

func DetectVisitorEmotion(text string) VisitorEmotionResult {
	text = strings.TrimSpace(text)
	if text == "" {
		return VisitorEmotionResult{Category: VisitorEmotionNeutral, LegacyToken: "neutral", Confidence: 0, Intensity: 0, Modality: "text"}
	}

	best := visitorEmotionRule{}
	bestScore := 0.0
	var evidence []string
	for _, rule := range visitorEmotionRules {
		score := 0.0
		matches := make([]string, 0, len(rule.Terms))
		for _, term := range rule.Terms {
			if strings.Contains(text, term) {
				score += rule.Weight
				matches = append(matches, term)
			}
		}
		if score > bestScore {
			best = rule
			bestScore = score
			evidence = matches
		}
	}

	if bestScore == 0 {
		for _, term := range visitorQuestionTerms {
			if strings.Contains(text, term) {
				return VisitorEmotionResult{
					Category:    VisitorEmotionQuestion,
					LegacyToken: "neutral",
					Confidence:  0.72,
					Intensity:   0.35,
					Evidence:    []string{term},
					Modality:    "text",
				}
			}
		}
		return VisitorEmotionResult{Category: VisitorEmotionNeutral, LegacyToken: "neutral", Confidence: 0.55, Intensity: 0.15, Modality: "text"}
	}

	confidence := 0.55 + 0.12*float64(len(evidence)-1)
	if confidence > 0.95 {
		confidence = 0.95
	}
	intensity := bestScore / (bestScore + 1.5)
	return VisitorEmotionResult{
		Category:              best.Category,
		LegacyToken:           best.Token,
		Confidence:            confidence,
		Intensity:             intensity,
		Evidence:              evidence,
		RecommendHumanService: best.Category == VisitorEmotionComplaint && intensity >= 0.55,
		Modality:              "text",
	}
}

func DetectVisitorEmotionWithVoice(text string, features *VoiceAcousticFeatures) VisitorEmotionResult {
	result := DetectVisitorEmotion(text)
	category, token, confidence, intensity, evidence, ok := detectAcousticEmotion(features)
	if !ok {
		return result
	}

	result.Modality = "text+acoustic"
	result.AcousticConfidence = confidence
	result.Evidence = append(result.Evidence, evidence...)
	if result.Category == VisitorEmotionNeutral || result.Category == VisitorEmotionQuestion {
		result.Category = category
		result.LegacyToken = token
		result.Confidence = confidence
		result.Intensity = intensity
	} else {
		result.Confidence = math.Min(0.98, result.Confidence+confidence*0.12)
		result.Intensity = math.Max(result.Intensity, intensity)
	}
	result.RecommendHumanService = result.RecommendHumanService ||
		(result.Category == VisitorEmotionComplaint && result.Intensity >= 0.55) ||
		(result.Category == VisitorEmotionAnxiety && result.Intensity >= 0.7)
	return result
}

func detectAcousticEmotion(features *VoiceAcousticFeatures) (VisitorEmotionCategory, string, float64, float64, []string, bool) {
	if features == nil || features.SampleCount < 5 || features.DurationMs < 300 || features.DurationMs > 120000 {
		return "", "", 0, 0, nil, false
	}
	values := []float64{
		features.RMSMean, features.RMSPeak, features.RMSVariation, features.PauseRatio,
		features.PitchMeanHz, features.PitchVariationHz, features.SpeechRateCharsPerSecond, features.RepetitionRatio,
	}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return "", "", 0, 0, nil, false
		}
	}
	if features.SampleCount > 5000 || features.RMSMean > 1 || features.RMSPeak > 1 || features.RMSVariation > 1 ||
		features.PauseRatio > 1 || features.RepetitionRatio > 1 || features.PitchMeanHz > 1000 ||
		features.PitchVariationHz > 1000 || features.SpeechRateCharsPerSecond > 50 {
		return "", "", 0, 0, nil, false
	}

	forceful := 0
	if features.RMSPeak >= 0.25 {
		forceful++
	}
	if features.RMSVariation >= 0.07 {
		forceful++
	}
	if features.SpeechRateCharsPerSecond >= 7.5 || features.RepetitionRatio >= 0.15 {
		forceful++
	}
	if forceful >= 3 {
		return VisitorEmotionComplaint, "anger", 0.68, 0.72, []string{"voice:forceful"}, true
	}

	anxious := 0
	if features.PauseRatio >= 0.4 {
		anxious++
	}
	if features.PitchMeanHz >= 190 && features.PitchVariationHz >= 55 {
		anxious++
	}
	if features.RepetitionRatio >= 0.12 {
		anxious++
	}
	if anxious >= 2 {
		return VisitorEmotionAnxiety, "fear", 0.64, 0.68, []string{"voice:tension"}, true
	}

	excited := 0
	if features.RMSPeak >= 0.18 {
		excited++
	}
	if features.SpeechRateCharsPerSecond >= 6 {
		excited++
	}
	if features.PitchVariationHz >= 45 {
		excited++
	}
	if features.PauseRatio <= 0.25 {
		excited++
	}
	if excited >= 3 {
		return VisitorEmotionExcitement, "surprise", 0.66, 0.65, []string{"voice:high_energy"}, true
	}
	return "", "", 0, 0, nil, false
}

func DetectCasualIntent(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	if containsAny(text, []string{"随便聊", "闲聊", "讲个故事", "讲个笑话", "无聊"}) {
		return "chat", true
	}
	if containsAny(text, []string{"景区", "灵山", "大佛", "梵宫", "门票", "票价", "路线", "景点", "游览", "开放", "停车", "九龙", "五印", "餐厅", "导览"}) {
		return "", false
	}
	if containsAny(text, []string{"你好", "您好", "嗨", "哈喽", "早上好", "晚上好", "在吗", "hello", "hi"}) {
		return "greeting", true
	}
	if containsAny(text, []string{"谢谢", "感谢", "辛苦了", "多谢"}) {
		return "thanks", true
	}
	if containsAny(text, []string{"再见", "拜拜", "回头见", "先这样"}) {
		return "farewell", true
	}
	if containsAny(text, []string{"你是谁", "你叫什么", "你能做什么", "你会什么", "介绍一下你", "陪我聊聊天", "陪我聊天", "聊聊天", "你好吗"}) {
		return "identity", true
	}

	emotion := DetectVisitorEmotion(text)
	if emotion.Category == VisitorEmotionComplaint {
		return "complaint", true
	}
	if emotion.Category == VisitorEmotionAnxiety {
		return "anxiety", true
	}
	return "", false
}

func EmotionGuidance(text string) string {
	emotion := DetectVisitorEmotion(text)
	switch emotion.Category {
	case VisitorEmotionComplaint:
		return "游客可能在抱怨或不满。先承认感受、简短道歉，再给出可执行的解决路径；涉及现场问题时建议联系景区服务中心。"
	case VisitorEmotionAnxiety:
		return "游客可能焦虑或担心。先安抚情绪，再分步骤回答；不要使用绝对承诺，必要时建议联系工作人员。"
	case VisitorEmotionExcitement:
		return "游客表现出期待或兴奋。保持积极、热情的导览语气，但事实和运营信息仍必须以知识库和官方公告为准。"
	case VisitorEmotionSatisfaction:
		return "游客表现出满意或感谢。简洁回应并自然提供下一步帮助，不要重复堆砌信息。"
	default:
		return ""
	}
}

func ApplyVisitorEmotionCare(emotion VisitorEmotionResult, answer string) string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return answer
	}
	switch emotion.Category {
	case VisitorEmotionComplaint:
		if strings.HasPrefix(answer, "听起来") {
			return answer
		}
		return "听起来这件事让你有些不舒服。" + answer
	case VisitorEmotionAnxiety:
		if strings.HasPrefix(answer, "别着急") {
			return answer
		}
		return "别着急，" + answer
	default:
		return answer
	}
}
