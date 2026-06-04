package handler

import "testing"

func TestDetectEmotion(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		// sadness
		{"sad_apologize", "抱歉，我无法回答这个问题", "sadness"},
		{"sad_sorry", "对不起，我不太确定", "sadness"},
		{"sad_cannot", "无法提供该信息", "sadness"},
		{"sad_regret", "遗憾地通知您", "sadness"},
		{"sad_bad", "这个回答不好", "sadness"},
		{"sad_terrible", "太糟糕了", "sadness"},
		{"sad_disappoint", "让您失望了", "sadness"},
		{"sad_sad", "很难过听到这个消息", "sadness"},

		// surprise
		{"surprise_wow", "哇，灵山大佛真的好壮观", "surprise"},
		{"surprise_surprise", "给您一个惊喜", "surprise"},
		{"surprise_notice", "请注意安全", "surprise"},
		{"surprise_remind", "温馨提醒您", "surprise"},
		{"surprise_warn", "警告：此处禁止攀爬", "surprise"},
		{"surprise_unexpected", "竟然有这么多人", "surprise"},
		{"surprise_actually", "居然不知道这个", "surprise"},

		// joy
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
	// sadness keywords are checked first, so they take priority
	text := "抱歉，欢迎来到灵山"
	got := detectEmotion(text)
	if got != "sadness" {
		t.Errorf("expected sadness to take priority over joy, got %q", got)
	}

	// surprise keywords are checked before joy
	text = "注意，推荐您去看看"
	got = detectEmotion(text)
	if got != "surprise" {
		t.Errorf("expected surprise to take priority over joy, got %q", got)
	}
}
