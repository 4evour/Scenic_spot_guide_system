package service

import "testing"

func TestDetectVisitorEmotion(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		category VisitorEmotionCategory
	}{
		{name: "satisfaction", text: "谢谢你的帮助", category: VisitorEmotionSatisfaction},
		{name: "question", text: "灵山大佛多高？", category: VisitorEmotionQuestion},
		{name: "complaint", text: "太贵了，想投诉", category: VisitorEmotionComplaint},
		{name: "anxiety", text: "我担心迷路怎么办", category: VisitorEmotionAnxiety},
		{name: "excitement", text: "哇，太壮观了", category: VisitorEmotionExcitement},
		{name: "neutral", text: "我明天上午到景区", category: VisitorEmotionNeutral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectVisitorEmotion(tt.text)
			if result.Category != tt.category {
				t.Fatalf("category = %q, want %q; evidence=%v", result.Category, tt.category, result.Evidence)
			}
		})
	}
}

func TestDetectCasualIntent(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		intent string
		casual bool
	}{
		{name: "greeting", text: "你好", intent: "greeting", casual: true},
		{name: "thanks", text: "谢谢你", intent: "thanks", casual: true},
		{name: "identity", text: "你是谁", intent: "identity", casual: true},
		{name: "scenic chat", text: "我想闲聊一下景区", intent: "chat", casual: true},
		{name: "complaint", text: "太贵了", intent: "complaint", casual: true},
		{name: "scenic complaint", text: "门票太贵了", intent: "", casual: false},
		{name: "mixed scenic question", text: "你好，灵山大佛多高？", intent: "", casual: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, casual := DetectCasualIntent(tt.text)
			if intent != tt.intent || casual != tt.casual {
				t.Fatalf("intent=%q casual=%v, want intent=%q casual=%v", intent, casual, tt.intent, tt.casual)
			}
		})
	}
}

func TestDetectVisitorEmotionWithVoice(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		features *VoiceAcousticFeatures
		want     VisitorEmotionCategory
		modality string
	}{
		{
			name: "high energy voice",
			text: "这个景色很特别",
			features: &VoiceAcousticFeatures{
				DurationMs: 1800, SampleCount: 20, RMSMean: 0.08, RMSPeak: 0.22, RMSVariation: 0.05,
				PauseRatio: 0.1, PitchMeanHz: 210, PitchVariationHz: 60, SpeechRateCharsPerSecond: 7,
			},
			want: VisitorEmotionExcitement, modality: "text+acoustic",
		},
		{
			name: "text complaint wins",
			text: "排队太久了，我要投诉",
			features: &VoiceAcousticFeatures{
				DurationMs: 2200, SampleCount: 25, RMSMean: 0.07, RMSPeak: 0.2, RMSVariation: 0.05,
				PauseRatio: 0.12, PitchMeanHz: 205, PitchVariationHz: 50, SpeechRateCharsPerSecond: 6.5,
			},
			want: VisitorEmotionComplaint, modality: "text+acoustic",
		},
		{name: "insufficient samples", text: "普通问题", features: &VoiceAcousticFeatures{DurationMs: 100, SampleCount: 1}, want: VisitorEmotionNeutral, modality: "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectVisitorEmotionWithVoice(tt.text, tt.features)
			if got.Category != tt.want || got.Modality != tt.modality {
				t.Fatalf("emotion = %+v, want category=%q modality=%q", got, tt.want, tt.modality)
			}
		})
	}
}
