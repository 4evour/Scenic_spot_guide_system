package service

import (
	"testing"
)

func TestDetectQuestionIntent(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"今天开放吗", "实时信息边界"},
		{"半天够吗", "路线规划"},
		{"灵山大佛的历史", ""},
		{"怎么去灵山", "属性追问"},
		{"门票多少钱", "实时信息边界"},
		{"附近有什么吃的", ""},
	}
	for _, tt := range tests {
		got := detectQuestionIntent(tt.query)
		if got != tt.expected {
			t.Errorf("detectQuestionIntent(%q) = %q, want %q", tt.query, got, tt.expected)
		}
	}
}

func TestNormalizeSessionID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc-123", "abc-123"},
		{"", ""},
		{"  spaces  ", "spaces"},
	}
	for _, tt := range tests {
		got := normalizeSessionID(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeSessionID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsFollowUpQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"那门票呢", true},
		{"灵山大佛在哪里", true},
		{"介绍一下灵山胜境", false},
	}
	for _, tt := range tests {
		got := isFollowUpQuery(tt.query)
		if got != tt.expected {
			t.Errorf("isFollowUpQuery(%q) = %v, want %v", tt.query, got, tt.expected)
		}
	}
}

func TestIsBoundaryIntent(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"今天开放吗", true},
		{"景区附近今晚还有多少间酒店空房？", true},
		{"根据游客消费样本，消费结构中占比最高的类别是什么？", false},
		{"灵山大佛的历史", false},
	}
	for _, tt := range tests {
		got := isBoundaryIntent(tt.query)
		if got != tt.expected {
			t.Errorf("isBoundaryIntent(%q) = %v, want %v", tt.query, got, tt.expected)
		}
	}
}

func TestInferConversationContext(t *testing.T) {
	svc := &RAGService{}
	ctx := svc.inferConversationContext("灵山大佛有多高", "灵山大佛高88米")
	// With nil profile, topic detection falls back to generic logic
	if ctx.Intent == "" {
		t.Log("intent is empty with nil profile (expected)")
	}
}
