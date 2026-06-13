package service

import (
	"strings"
	"testing"

	"github.com/scenic-guide/internal/model"
)

func TestBuildRAGPromptWithContext(t *testing.T) {
	svc := &RAGService{}

	chunks := []model.KnowledgeChunk{
		{Title: "灵山大佛", Content: "灵山大佛高88米", Source: "官方网站"},
		{Title: "九龙灌浴", Content: "九龙灌浴表演每天3场", Source: "景区公告"},
	}

	prompt := svc.BuildRAGPromptWithContext("灵山大佛有多高", chunks, "")
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "灵山大佛") {
		t.Error("prompt should contain chunk title")
	}
	if !strings.Contains(prompt, "88米") {
		t.Error("prompt should contain chunk content")
	}
	if !strings.Contains(prompt, "灵山大佛有多高") {
		t.Error("prompt should contain the query")
	}
}

func TestBuildRAGPromptWithContextAndSession(t *testing.T) {
	svc := &RAGService{}

	chunks := []model.KnowledgeChunk{
		{Title: "灵山大佛", Content: "灵山大佛高88米", Source: "官方网站"},
	}

	sessionCtx := "游客之前问过门票价格"
	prompt := svc.BuildRAGPromptWithContext("那开放时间呢", chunks, sessionCtx)
	if !strings.Contains(prompt, sessionCtx) {
		t.Error("prompt should contain session context")
	}
}

func TestBuildRAGPromptEmptyChunks(t *testing.T) {
	svc := &RAGService{}

	prompt := svc.BuildRAGPromptWithContext("test", nil, "")
	if prompt != "" {
		t.Errorf("expected empty prompt for nil chunks, got %q", prompt)
	}
}

func TestBuildRAGPrompt(t *testing.T) {
	svc := &RAGService{}

	chunks := []model.KnowledgeChunk{
		{Title: "测试", Content: "内容", Source: "来源"},
	}

	prompt := svc.BuildRAGPrompt("问题", chunks)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}

func TestIsRouteIntent(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"半天够吗", true},
		{"灵山大佛的历史", false},
	}
	for _, tt := range tests {
		got := isRouteIntent(tt.query)
		if got != tt.expected {
			t.Errorf("isRouteIntent(%q) = %v, want %v", tt.query, got, tt.expected)
		}
	}
}