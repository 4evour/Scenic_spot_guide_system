package handler

import (
	"math"
	"testing"
)

func TestDetectEmotion(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// anger (12 keywords: 愤怒/生气/讨厌/烦/滚/无聊/差劲/恶心/可恶/过分/烦人/火大)
		{"anger_furious", "太愤怒了", "anger"},
		{"anger_angry", "我很生气", "anger"},
		{"anger_hate", "真讨厌", "anger"},
		{"anger_annoyed", "真烦人", "anger"},
		{"anger_leave", "滚开", "anger"},
		{"anger_bored", "好无聊啊", "anger"},
		{"anger_terrible", "做得太差劲了", "anger"},

		// disgust (7 keywords)
		{"disgust_disgusting", "真令人作呕", "disgust"},
		{"disgust_nausea", "让人反胃", "disgust"},
		{"disgust_loathe", "我很厌恶这种行为", "disgust"},

		// sadness (15 keywords)
		{"sad_apologize", "抱歉，我无法回答这个问题", "sadness"},
		{"sad_sorry", "对不起，我不太确定", "sadness"},
		{"sad_cannot", "无法提供该信息", "sadness"},
		{"sad_regret", "遗憾地通知您", "sadness"},
		{"sad_bad", "这个回答不好", "sadness"},
		{"sad_terrible", "太糟糕了", "sadness"},
		{"sad_disappoint", "让您失望了", "sadness"},
		{"sad_sad", "很难过听到这个消息", "sadness"},
		{"sad_heartbroken", "令人伤心", "sadness"},
		{"sad_pity", "真可惜啊", "sadness"},
		{"sad_unfortunate", "很不幸", "sadness"},
		{"sad_helpless", "我也很无奈", "sadness"},
		{"sad_noway", "那没办法了", "sadness"},

		// fear (10 keywords)
		{"fear_scared", "我害怕", "fear"},
		{"fear_worried", "担心安全问题", "fear"},
		{"fear_terrifying", "太恐怖了", "fear"},
		{"fear_scary", "好可怕", "fear"},
		{"fear_frightening", "有点吓人", "fear"},
		{"fear_danger", "这里有危险", "fear"},
		{"fear_careful", "小心台阶", "fear"},
		{"fear_panic", "感到惊慌", "fear"},
		{"fear_nervous", "我很紧张", "fear"},

		// surprise (15 keywords)
		{"surprise_wow", "哇，灵山大佛真的好壮观", "surprise"},
		{"surprise_surprise", "给您一个惊喜", "surprise"},
		{"surprise_notice", "请注意安全", "surprise"},
		{"surprise_remind", "温馨提醒您", "surprise"},
		{"surprise_warn", "警告：此处禁止攀爬", "surprise"},
		{"surprise_unexpected", "竟然有这么多人", "surprise"},
		{"surprise_actually", "居然不知道这个", "surprise"},
		{"surprise_omg", "天哪太美了", "surprise"},
		{"surprise_unbelievable", "不可思议的景观", "surprise"},
		{"surprise_shocked", "令人震惊的消息", "surprise"},
		{"surprise_incredible", "难以置信", "surprise"},
		{"surprise_sudden", "突然下起了雨", "surprise"},
		{"surprise_unexpected2", "意外发现了", "surprise"},
		{"surprise_startled", "吓一跳！真没想到", "surprise"},

		// joy (25 keywords)
		{"joy_welcome", "欢迎来到灵山胜境", "joy"},
		{"joy_hello", "您好，有什么可以帮您", "joy"},
		{"joy_glad", "很高兴为您服务", "joy"},
		{"joy_recommend", "推荐您去梵宫看看", "joy"},
		{"joy_suggest", "建议您早上出发", "joy"},
		{"joy_best", "最佳游览时间是上午", "joy"},
		{"joy_wonderful", "精彩的演出等您来看", "joy"},
		{"joy_beautiful", "美丽的风景让人流连忘返", "joy"},
		{"joy_spectacular", "壮观的灵山大佛", "joy"},
		{"joy_happy", "开心地游玩吧", "joy"},
		{"joy_great", "太棒了这个地方", "joy"},
		{"joy_nice", "真不错", "joy"},
		{"joy_excellent", "好极了", "joy"},
		{"joy_perfect", "完美的一天", "joy"},
		{"joy_awesome", "给你点个赞", "joy"},
		{"joy_amazing", "太厉害了", "joy"},
		{"joy_good", "真好", "joy"},
		{"joy_like", "我很喜欢这里", "joy"},
		{"joy_pleasant", "愉快的旅程", "joy"},
		{"joy_blessed", "幸福的感觉", "joy"},
		{"joy_thanks", "谢谢你的帮助", "joy"},
		{"joy_grateful", "感谢你的回答", "joy"},

		// neutral
		{"neutral_question", "灵山大佛有多高？", "neutral"},
		{"neutral_factual", "灵山景区位于江苏省无锡市", "neutral"},
		{"neutral_empty", "", "neutral"},
		{"neutral_no_keyword", "今天天气怎么样", "neutral"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectEmotion(tt.text)
			if got != tt.want {
				t.Errorf("detectEmotion(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestDetectEmotion_PriorityOrder(t *testing.T) {
	// anger 检测优先级最高
	text := "愤怒，欢迎来到灵山"
	got := detectEmotion(text)
	if got != "anger" {
		t.Errorf("expected anger to take priority, got %q", got)
	}

	// sadness 先于 joy
	text = "抱歉，欢迎来到灵山"
	got = detectEmotion(text)
	if got != "sadness" {
		t.Errorf("expected sadness to take priority over joy, got %q", got)
	}

	// surprise 先于 joy
	text = "注意，推荐您去看看"
	got = detectEmotion(text)
	if got != "surprise" {
		t.Errorf("expected surprise to take priority over joy, got %q", got)
	}

	// fear 先于 joy
	text = "小心台阶，推荐您去那边"
	got = detectEmotion(text)
	if got != "fear" {
		t.Errorf("expected fear to take priority over joy, got %q", got)
	}
}

func TestDetectEmotionWithIntensity(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		wantEmotion   string
		minIntensity  float64
		maxIntensity  float64
	}{
		{"strong_joy", "太棒了！完美！真不错！", "joy", 0.15, 0.95},
		{"weak_joy", "推荐您去看看", "joy", 0.05, 0.45},
		{"strong_sadness", "抱歉，我真的很糟糕，太可惜了", "sadness", 0.10, 0.80},
		{"empty", "", "neutral", 0, 0},
		{"neutral_no_match", "灵山景区今天天气晴朗", "neutral", 0, 0},
		{"anger_strong", "滚！真烦人！太差劲了！", "anger", 0.15, 0.95},
		{"fear_single", "危险", "fear", 0.10, 0.70},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emotion, intensity := detectEmotionWithIntensity(tt.text)
			if emotion != tt.wantEmotion {
				t.Errorf("emotion = %q, want %q", emotion, tt.wantEmotion)
			}
			if intensity < tt.minIntensity-0.01 || intensity > tt.maxIntensity+0.01 {
				t.Errorf("intensity = %.3f, want in [%.3f, %.3f]", intensity, tt.minIntensity, tt.maxIntensity)
			}
		})
	}
}

func TestDetectEmotionWithIntensity_Consistency(t *testing.T) {
	// 同一文本多次检测结果应一致
	text := "欢迎来到灵山景区，这里很壮观，真不错！"
	emotion1, intensity1 := detectEmotionWithIntensity(text)
	for i := 0; i < 100; i++ {
		e, in := detectEmotionWithIntensity(text)
		if e != emotion1 {
			t.Errorf("emotion inconsistent: got %q on iteration %d, want %q", e, i, emotion1)
		}
		if math.Abs(in-intensity1) > 1e-10 {
			t.Errorf("intensity inconsistent: got %.10f on iteration %d, want %.10f", in, i, intensity1)
		}
	}
}
