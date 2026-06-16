package service

import (
	"net/http"
	"net/http/httptest"
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

func TestBuildRAGPromptInstructsGuideAnswerNotKnowledgeMeta(t *testing.T) {
	svc := &RAGService{}

	prompt := svc.BuildRAGPromptWithContext("灵山大佛有什么特色？", []model.KnowledgeChunk{
		{
			Title:   "灵山大佛问答素材",
			Content: "游客常问灵山大佛多高、是不是景区最代表性的景点。",
			Source:  "test",
		},
	}, "")

	for _, want := range []string{"直接回答游客当前问题", "不要复述", "游客常问"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt should contain guide-answer constraint %q, got: %s", want, prompt)
		}
	}
}

func TestQueryWithRAGReturnsErrorWhenConfiguredLLMFails(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"meta-answer","title":"灵山大佛问答素材","source":"test","content":"游客常问灵山大佛多高、是不是景区最代表性的景点。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()
	rag.chatAPIKey = "configured-key"
	rag.chatModel = "deepseek-chat"
	rag.chatBaseURL = server.URL

	answer, _, err := rag.QueryWithRAGTrace("灵山大佛有什么特色？", "zh-CN")
	if err == nil {
		t.Fatalf("expected LLM failure error, got answer: %s", answer)
	}
	if strings.Contains(answer, "游客常问") {
		t.Fatalf("failed LLM should not return knowledge meta as answer: %s", answer)
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
