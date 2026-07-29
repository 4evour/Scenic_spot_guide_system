package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scenic-guide/internal/model"
)

func TestQueryWithRAGHandlesCasualConversationLocally(t *testing.T) {
	rag := newTestRAGService(t)

	answer, trace, err := rag.QueryWithRAGTrace(context.Background(), "你好", "zh-CN")
	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if trace.Provider != "local-conversation" || trace.RetrievalMode != "conversation" {
		t.Fatalf("unexpected casual trace: %+v", trace)
	}
	if trace.ShouldAbstain || trace.Confidence < 0.9 || trace.ChunkCount != 0 {
		t.Fatalf("casual conversation should be locally answered with high confidence: %+v", trace)
	}
	if !strings.Contains(answer, "小灵") {
		t.Fatalf("unexpected casual answer: %s", answer)
	}
}

func TestQueryWithRAGStreamingHandlesCasualConversationLocally(t *testing.T) {
	rag := newTestRAGService(t)
	var streamed string

	answer, route, trace, err := rag.QueryWithRAGStreaming(context.Background(), "", "谢谢", "zh-CN", func(token string) {
		streamed += token
	})
	if err != nil {
		t.Fatalf("QueryWithRAGStreaming returned error: %v", err)
	}
	if route != nil || trace.Provider != "local-conversation" || trace.RetrievalMode != "conversation" {
		t.Fatalf("unexpected casual streaming result: route=%+v trace=%+v", route, trace)
	}
	if answer == "" || answer != streamed || !strings.Contains(answer, "不客气") {
		t.Fatalf("unexpected casual streaming answer: answer=%q streamed=%q", answer, streamed)
	}
}

func TestBuildRAGPromptIncludesEmotionGuidance(t *testing.T) {
	prompt := (&RAGService{}).BuildRAGPromptWithContext("我担心迷路怎么办？", []model.KnowledgeChunk{
		{Title: "路线", Content: "景区入口设有导览服务台。", Source: "test"},
	}, "")
	if !strings.Contains(prompt, "游客沟通状态") || !strings.Contains(prompt, "先安抚情绪") {
		t.Fatalf("prompt should include anxiety guidance: %s", prompt)
	}
}

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

func TestLocalFactAnswerKeepsComplementarySyntheticFacts(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		chunks    []model.KnowledgeChunk
		wantTerms []string
	}{
		{
			name:  "height keeps every available measurement dimension",
			query: "纪念塔有多高？",
			chunks: []model.KnowledgeChunk{{
				Title:   "纪念塔资料",
				Content: "纪念塔是园区的重要地标。纪念塔也是游客常见的拍照背景。纪念塔通高88米，佛体79米。计入台基后总高101.5米。",
			}},
			wantTerms: []string{"通高88米", "佛体79米", "总高101.5米"},
		},
		{
			name:  "location keeps administrative region and locality",
			query: "云海馆位于哪里？",
			chunks: []model.KnowledgeChunk{{
				Title:   "云海馆位置",
				Content: "云海馆是园区的核心建筑。云海馆位于甲省乙市。场馆具体坐落在滨湖区域青山镇。",
			}},
			wantTerms: []string{"甲省乙市", "滨湖区域", "青山镇"},
		},
		{
			name:  "ceremony content keeps subject and explanation",
			query: "水礼仪主要表现什么内容？",
			chunks: []model.KnowledgeChunk{{
				Title:   "水礼仪内容",
				Content: "水礼仪是园区的标志景观。水礼仪主要表现文化内容。水礼仪内容常被游客讨论。仪式主体再现先贤诞生场景。莲台升起与环形水幕共同解释迎礼过程。",
			}},
			wantTerms: []string{"先贤诞生", "莲台升起", "环形水幕"},
		},
		{
			name:  "craft answer keeps materials and process explanation",
			query: "星海宫为什么有很高的艺术价值？",
			chunks: []model.KnowledgeChunk{{
				Title:   "星海宫艺术",
				Content: "星海宫是园区的重要建筑。星海宫的艺术价值常被参观者提及。主体装饰采用榫卯木雕与釉彩玻璃。这些材料由手工雕刻、烧制和镶嵌等传统工艺完成，因而形成独特艺术价值。",
			}},
			wantTerms: []string{"榫卯木雕", "釉彩玻璃", "手工雕刻", "传统工艺"},
		},
		{
			name:  "culture answer keeps cultural subject and examples",
			query: "云坛主要展示哪类文化？",
			chunks: []model.KnowledgeChunk{{
				Title:   "云坛文化",
				Content: "云坛是园区的文化建筑。云坛主要展示文化内容。云坛文化受到游客关注。场馆以山地宗教文化为主题。空间展示四季神像、祈愿经轮和织绘卷轴，用来解释这一传统的象征意义。",
			}},
			wantTerms: []string{"山地宗教文化", "四季神像", "祈愿经轮", "织绘卷轴"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rag := newTestRAGService(t)
			rag.profile = nil
			answer := rag.generateAnswerFromChunksWithContext(tt.query, tt.chunks, "")
			for _, term := range tt.wantTerms {
				if !strings.Contains(answer, term) {
					t.Fatalf("answer = %q, missing complementary fact %q", answer, term)
				}
			}
		})
	}
}

func TestLocalFactAnswerLimitsSentencesAndFinalRunes(t *testing.T) {
	longSentence := strings.Repeat("这段补充资料用于验证最终回答长度限制", 20)
	chunks := []model.KnowledgeChunk{{
		Title: "纪念塔资料",
		Content: strings.Join([]string{
			"纪念塔通高88米",
			"佛体高79米",
			"含台基总高101.5米",
			longSentence + "甲",
			longSentence + "乙",
			longSentence + "丙",
		}, "。") + "。",
	}}

	rag := newTestRAGService(t)
	rag.profile = nil
	answer := rag.generateAnswerFromChunksWithContext("我很担心，请介绍纪念塔资料", chunks, "")
	if got := len([]rune(answer)); got > 700 {
		t.Fatalf("final local answer rune count = %d, want <= 700", got)
	}
	body := answer
	if _, remainder, ok := strings.Cut(body, "：\n\n"); ok {
		body = remainder
	}
	if got := len(strings.Split(strings.TrimSpace(body), "\n\n")); got > 4 {
		t.Fatalf("local answer snippet count = %d, want <= 4: %q", got, answer)
	}
}

func TestQueryWithRAGFallsBackWhenConfiguredLLMFails(t *testing.T) {
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

	answer, _, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有什么特色？", "zh-CN")
	if err != nil {
		t.Fatalf("expected local fallback, got error: %v", err)
	}
	if strings.TrimSpace(answer) == "" {
		t.Fatal("local fallback should return a non-empty answer")
	}
}

func TestQueryWithRAGUsesSingleModelCallForFinalAnswer(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official","content":"灵山大佛通高88米。"},
		{"id":"buddha-location","title":"灵山大佛位置","source":"official","content":"灵山大佛位于无锡太湖之滨。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if enabled, ok := body["enable_thinking"].(bool); !ok || enabled {
			t.Errorf("enable_thinking = %#v, want false", body["enable_thinking"])
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"灵山大佛通高88米，位于无锡太湖之滨。"}}]}`))
	}))
	defer server.Close()

	rag.chatAPIKey = "configured-key"
	rag.chatModel = "test-model"
	rag.chatBaseURL = server.URL

	answer, trace, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有多高，在哪里？", "zh-CN")
	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("model calls = %d, want 1 final-answer call", calls)
	}
	if trace.RerankMs != 0 {
		t.Fatalf("model-based rerank should be disabled: %+v", trace)
	}
	if !strings.Contains(answer, "88米") {
		t.Fatalf("unexpected answer: %q", answer)
	}
}

func TestQueryWithRAGUsesDeepSeekNonThinkingFormat(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official","content":"灵山大佛通高88米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if _, ok := body["enable_thinking"]; ok {
			t.Errorf("DeepSeek request should not contain enable_thinking: %#v", body["enable_thinking"])
		}
		thinking, ok := body["thinking"].(map[string]interface{})
		if !ok || thinking["type"] != "disabled" {
			t.Errorf("thinking = %#v, want type=disabled", body["thinking"])
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"灵山大佛通高88米。"}}]}`))
	}))
	defer server.Close()

	rag.chatAPIKey = "configured-key"
	rag.chatModel = "deepseek-v4-flash"
	rag.chatBaseURL = server.URL

	answer, _, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有多高？", "zh-CN")
	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if !strings.Contains(answer, "88米") {
		t.Fatalf("unexpected answer: %q", answer)
	}
}

func TestQueryWithRAGStreamingUsesSingleModelCall(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official","content":"灵山大佛通高88米。"},
		{"id":"buddha-location","title":"灵山大佛位置","source":"official","content":"灵山大佛位于无锡太湖之滨。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if enabled, ok := body["enable_thinking"].(bool); !ok || enabled {
			t.Errorf("enable_thinking = %#v, want false", body["enable_thinking"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"灵山大佛通高88米。\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	rag.chatAPIKey = "configured-key"
	rag.chatModel = "test-model"
	rag.chatBaseURL = server.URL

	answer, _, trace, err := rag.QueryWithRAGStreaming(context.Background(), "", "灵山大佛有多高，在哪里？", "zh-CN", nil)
	if err != nil {
		t.Fatalf("QueryWithRAGStreaming returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("model calls = %d, want 1 streaming final-answer call", calls)
	}
	if trace.RerankMs != 0 {
		t.Fatalf("model-based rerank should be disabled: %+v", trace)
	}
	if !strings.Contains(answer, "88米") {
		t.Fatalf("unexpected answer: %q", answer)
	}
}

func TestQueryWithRAGDeduplicatesConcurrentModelCalls(t *testing.T) {
	rag := newTestRAGService(t)
	rag.profile = nil
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official","content":"灵山大佛通高88米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls int
	var callsMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callsMu.Lock()
		calls++
		callsMu.Unlock()
		select {
		case <-entered:
		default:
			close(entered)
		}
		<-release
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"灵山大佛通高88米。"}}]}`))
	}))
	defer server.Close()
	rag.chatAPIKey = "configured-key"
	rag.chatModel = "test-model"
	rag.chatBaseURL = server.URL

	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, _, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛多高？", "zh-CN")
			results <- err
		}()
	}
	<-entered
	time.Sleep(50 * time.Millisecond)
	close(release)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("query returned error: %v", err)
		}
	}
	callsMu.Lock()
	defer callsMu.Unlock()
	if calls != 1 {
		t.Fatalf("model calls = %d, want 1", calls)
	}
}

func TestQueryWithRAGStreamingFallsBackBeforeFirstToken(t *testing.T) {
	rag := newTestRAGService(t)
	rag.profile = nil
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official","content":"灵山大佛通高88米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()
	rag.chatAPIKey = "configured-key"
	rag.chatModel = "test-model"
	rag.chatBaseURL = server.URL
	rag.chatGuard.cfg.RetryBackoff = time.Millisecond

	var streamed strings.Builder
	answer, _, trace, err := rag.QueryWithRAGStreaming(context.Background(), "", "灵山大佛多高？", "zh-CN", func(token string) {
		streamed.WriteString(token)
	})
	if err != nil {
		t.Fatalf("expected streaming fallback, got error: %v", err)
	}
	if trace.Provider != "local-rag-fallback" || answer == "" || streamed.String() != answer {
		t.Fatalf("unexpected streaming fallback: answer=%q streamed=%q trace=%+v", answer, streamed.String(), trace)
	}
}

func TestQueryWithRAGRecordsPrometheusMetrics(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"test","content":"灵山大佛通高88米，主体高79米，莲花瓣高9米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	durationBefore, cacheBefore := ragMetricSnapshot(t)
	if _, _, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有多高？", "zh-CN"); err != nil {
		t.Fatalf("first QueryWithRAGTrace returned error: %v", err)
	}
	durationAfterFirst, cacheAfterFirst := ragMetricSnapshot(t)
	if durationAfterFirst <= durationBefore {
		t.Fatalf("RAG duration observations did not increase: before=%d after=%d", durationBefore, durationAfterFirst)
	}
	if cacheAfterFirst != cacheBefore {
		t.Fatalf("first query should not count as cache hit: before=%.0f after=%.0f", cacheBefore, cacheAfterFirst)
	}

	if _, trace, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有多高？", "zh-CN"); err != nil {
		t.Fatalf("second QueryWithRAGTrace returned error: %v", err)
	} else if !trace.CacheHit {
		t.Fatalf("second query should hit cache")
	}
	durationAfterSecond, cacheAfterSecond := ragMetricSnapshot(t)
	if durationAfterSecond <= durationAfterFirst {
		t.Fatalf("cached RAG duration observation did not increase: before=%d after=%d", durationAfterFirst, durationAfterSecond)
	}
	if cacheAfterSecond <= cacheAfterFirst {
		t.Fatalf("RAG cache hit counter did not increase: before=%.0f after=%.0f", cacheAfterFirst, cacheAfterSecond)
	}
}

func TestQueryWithRAGTraceReturnsSources(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official-guide","content":"灵山大佛通高88米，主体高79米，莲花瓣高9米。","knowledge_category":"景点讲解","spot_id":2,"spot_category":"核心景点"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	_, trace, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有多高？", "zh-CN")
	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if len(trace.Sources) != 1 {
		t.Fatalf("sources len = %d, want 1: %+v", len(trace.Sources), trace.Sources)
	}
	source := trace.Sources[0]
	if source.ID != "buddha-height" || source.Title != "灵山大佛高度" || source.Source != "official-guide" {
		t.Fatalf("unexpected source identity: %+v", source)
	}
	if source.KnowledgeCategory != "景点讲解" || source.SpotID != 2 || source.SpotCategory != "核心景点" {
		t.Fatalf("unexpected source metadata: %+v", source)
	}
	if source.Preview == "" || strings.Contains(source.Preview, "\n") {
		t.Fatalf("unexpected source preview: %q", source.Preview)
	}
}

type delayedEmbeddingProvider struct {
	delay time.Duration
}

func (p delayedEmbeddingProvider) GenerateEmbedding(string) ([]float64, error) {
	time.Sleep(p.delay)
	return []float64{1, 0}, nil
}

func (delayedEmbeddingProvider) Name() string {
	return "delayed-test"
}

func (delayedEmbeddingProvider) IsAvailable() bool {
	return true
}

func TestQueryWithRAGTraceReportsRetrievalPhases(t *testing.T) {
	const delay = 20 * time.Millisecond
	rag := newTestRAGServiceWithEmbedding(t, delayedEmbeddingProvider{delay: delay})
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official-guide","content":"灵山大佛通高88米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	_, trace, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有多高？", "zh-CN")
	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if trace.EmbeddingMs < delay.Milliseconds() {
		t.Fatalf("embedding_ms = %d, want at least %d", trace.EmbeddingMs, delay.Milliseconds())
	}
	if trace.QueryEnhancementMs < 0 || trace.ScoringMs < 0 || trace.RerankMs < 0 {
		t.Fatalf("retrieval phase timings must not be negative: %+v", trace)
	}
	if trace.RetrievalMs < trace.EmbeddingMs {
		t.Fatalf("retrieval_ms = %d, want >= embedding_ms %d", trace.RetrievalMs, trace.EmbeddingMs)
	}
}

func TestQueryWithRAGTraceReturnsSourcesOnCacheHit(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official-guide","content":"灵山大佛通高88米，主体高79米，莲花瓣高9米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	if _, _, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有多高？", "zh-CN"); err != nil {
		t.Fatalf("first QueryWithRAGTrace returned error: %v", err)
	}
	_, trace, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有多高？", "zh-CN")
	if err != nil {
		t.Fatalf("second QueryWithRAGTrace returned error: %v", err)
	}
	if !trace.CacheHit {
		t.Fatalf("second query should hit cache")
	}
	if len(trace.Sources) != 1 || trace.Sources[0].ID != "buddha-height" {
		t.Fatalf("cached query sources = %+v, want buddha-height", trace.Sources)
	}
}

func TestQueryWithRAGTraceAbstainsWithoutEvidence(t *testing.T) {
	rag := newTestRAGService(t)

	answer, trace, err := rag.QueryWithRAGTrace(context.Background(), "景区今天还有多少停车位？", "zh-CN")
	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if !trace.ShouldAbstain {
		t.Fatalf("trace should abstain without retrieved evidence: %+v", trace)
	}
	if trace.Confidence != 0 || len(trace.Sources) != 0 {
		t.Fatalf("unexpected evidence signal: confidence=%v sources=%+v", trace.Confidence, trace.Sources)
	}
	if !strings.Contains(answer, "没有找到足够依据") {
		t.Fatalf("unexpected no-evidence answer: %s", answer)
	}
}

func TestQueryWithRAGTraceRejectsSeveralWeaklyRelatedCachedSources(t *testing.T) {
	rag := newTestRAGService(t)
	query := "灵山大佛有多高？"
	rag.setCachedResponse(query, "灵山大佛高999米。", []RAGSource{
		{ID: "parking", Title: "停车服务", Source: "official", Preview: "景区停车场提供车辆停放服务。"},
		{ID: "dining", Title: "餐饮服务", Source: "official", Preview: "景区内设有素食餐厅和简餐点。"},
		{ID: "toilet", Title: "公共设施", Source: "official", Preview: "景区游客中心附近设有洗手间。"},
	})

	answer, trace, err := rag.QueryWithRAGTrace(context.Background(), query, "zh-CN")
	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if !trace.ShouldAbstain || trace.Confidence >= 0.5 {
		t.Fatalf("weak sources must remain low confidence and abstain: %+v", trace)
	}
	if strings.Contains(answer, "999米") || !strings.Contains(answer, "没有找到足够依据") {
		t.Fatalf("weak cached evidence must not be returned as an answer: %s", answer)
	}
}

func TestCalculateAnswerEvidenceAcceptsEntityRelevantSource(t *testing.T) {
	confidence, shouldAbstain := calculateAnswerEvidence("灵山大佛有多高？", []RAGSource{
		{ID: "buddha-height", Title: "灵山大佛高度", Source: "official", Preview: "灵山大佛通高88米。"},
	})

	if shouldAbstain || confidence < 0.6 {
		t.Fatalf("entity-relevant evidence should be answerable: confidence=%v should_abstain=%v", confidence, shouldAbstain)
	}
}

func TestCalculateChunkEvidenceUsesRelevantFourthChunkFullContent(t *testing.T) {
	query := "灵山大佛有多高？"
	chunks := []model.KnowledgeChunk{
		{ID: "parking", Title: "停车服务", Source: "official", Content: "景区停车场提供车辆停放服务。"},
		{ID: "dining", Title: "餐饮服务", Source: "official", Content: "景区内设有素食餐厅和简餐点。"},
		{ID: "toilet", Title: "公共设施", Source: "official", Content: "游客中心附近设有洗手间。"},
		{ID: "height", Title: "造像资料", Source: "official", Content: strings.Repeat("历史资料背景说明。", 20) + "灵山大佛通高88米。"},
	}

	if _, shouldAbstain := calculateAnswerEvidence(query, buildRAGSources(chunks, 3)); !shouldAbstain {
		t.Fatal("display-only sources should not contain the fourth chunk evidence")
	}
	confidence, shouldAbstain := calculateChunkEvidence(query, chunks)
	if shouldAbstain || confidence < 0.5 {
		t.Fatalf("fourth full chunk should be answerable: confidence=%v should_abstain=%v", confidence, shouldAbstain)
	}
}

func TestQueryWithRAGTraceUsesRelevantFourthChunkFullEvidence(t *testing.T) {
	rag, query, retrieved := newFourthRankEvidenceRAGService(t)

	answer, trace, err := rag.QueryWithRAGTrace(context.Background(), query, "zh-CN")
	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if trace.ShouldAbstain {
		t.Fatalf("fourth-ranked full evidence must prevent abstention: %+v", trace)
	}
	if strings.Contains(answer, "没有找到足够依据") || !strings.Contains(answer, "88米") {
		t.Fatalf("answer must use the fourth-ranked full evidence: %q", answer)
	}
	if len(trace.Sources) > 3 {
		t.Fatalf("display sources len = %d, want <= 3: %+v", len(trace.Sources), trace.Sources)
	}
	for _, source := range trace.Sources {
		if source.ID == retrieved[3].ID {
			t.Fatalf("fourth-ranked evidence must not leak into the three-item display sources: %+v", trace.Sources)
		}
	}
}

func TestQueryWithRAGStreamingUsesRelevantFourthChunkFullEvidence(t *testing.T) {
	rag, query, retrieved := newFourthRankEvidenceRAGService(t)
	var streamed strings.Builder

	answer, _, trace, err := rag.QueryWithRAGStreaming(context.Background(), "", query, "zh-CN", func(token string) {
		streamed.WriteString(token)
	})
	if err != nil {
		t.Fatalf("QueryWithRAGStreaming returned error: %v", err)
	}
	if trace.ShouldAbstain {
		t.Fatalf("fourth-ranked full evidence must prevent streaming abstention: %+v", trace)
	}
	if answer == "" || streamed.String() != answer || strings.Contains(answer, "没有找到足够依据") || !strings.Contains(answer, "88米") {
		t.Fatalf("streaming answer must use the fourth-ranked full evidence: answer=%q streamed=%q", answer, streamed.String())
	}
	if len(trace.Sources) > 3 {
		t.Fatalf("display sources len = %d, want <= 3: %+v", len(trace.Sources), trace.Sources)
	}
	for _, source := range trace.Sources {
		if source.ID == retrieved[3].ID {
			t.Fatalf("fourth-ranked evidence must not leak into the three-item display sources: %+v", trace.Sources)
		}
	}
}

func newFourthRankEvidenceRAGService(t *testing.T) (*RAGService, string, []model.KnowledgeChunk) {
	t.Helper()

	const query = "灵山大佛雕像有多高？"
	rag := newTestRAGServiceWithEmbedding(t, staticEmbeddingProvider{
		vectors: map[string][]float64{
			query: {1, 0},
			"景区停车场提供车辆停放服务。":       {1, 0},
			"景区内设有素食餐厅和简餐点。":       {0.999, 0.04},
			"游客中心附近设有洗手间。":         {0.998, 0.06},
			"灵山大佛造像的官方档案记载其通高88米。": {0.05, 0.9987},
		},
	})
	rag.profile = nil
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"parking","title":"停车服务","source":"official","content":"景区停车场提供车辆停放服务。"},
		{"id":"dining","title":"餐饮服务","source":"official","content":"景区内设有素食餐厅和简餐点。"},
		{"id":"toilet","title":"公共设施","source":"official","content":"游客中心附近设有洗手间。"},
		{"id":"height","title":"造像档案","source":"official","content":"灵山大佛造像的官方档案记载其通高88米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	retrieved, err := rag.RetrieveRelevantKnowledge(query, TopK)
	if err != nil {
		t.Fatalf("RetrieveRelevantKnowledge returned error: %v", err)
	}
	if len(retrieved) != 4 {
		t.Fatalf("retrieved chunks len = %d, want 4: %+v", len(retrieved), retrieved)
	}
	if retrieved[3].ID != "height" {
		t.Fatalf("retrieved order = %v, want height evidence fourth", chunkIDs(retrieved))
	}
	if !strings.Contains(retrieved[3].Content, "88米") {
		t.Fatalf("fourth chunk does not contain the expected full evidence: %+v", retrieved[3])
	}
	for i := 0; i < 3; i++ {
		if strings.Contains(retrieved[i].Content, "灵山大佛") || strings.Contains(retrieved[i].Content, "88米") {
			t.Fatalf("retrieved chunk %d must be unrelated question text: %+v", i+1, retrieved[i])
		}
	}

	return rag, query, retrieved
}

func TestCalculateChunkEvidenceAbstainsForUnrelatedChunks(t *testing.T) {
	confidence, shouldAbstain := calculateChunkEvidence("灵山大佛有多高？", []model.KnowledgeChunk{
		{ID: "parking", Title: "停车服务", Source: "official", Content: "景区停车场提供车辆停放服务。"},
		{ID: "dining", Title: "餐饮服务", Source: "official", Content: "景区内设有素食餐厅和简餐点。"},
		{ID: "toilet", Title: "公共设施", Source: "official", Content: "游客中心附近设有洗手间。"},
	})

	if !shouldAbstain || confidence >= 0.5 {
		t.Fatalf("unrelated full chunks must abstain: confidence=%v should_abstain=%v", confidence, shouldAbstain)
	}
}

func TestQueryWithRAGTraceAbstainsForRealtimeBoundary(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"ticket-boundary","title":"门票与开放时间边界","source":"official","content":"今日票价和开放时间以官方最新公告为准。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	answer, trace, err := rag.QueryWithRAGTrace(context.Background(), "今天景区几点开门？", "zh-CN")
	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if !trace.ShouldAbstain || len(trace.Sources) == 0 {
		t.Fatalf("realtime boundary should abstain with sources: %+v", trace)
	}
	if !strings.Contains(answer, "官方") {
		t.Fatalf("boundary answer should direct to official information: %s", answer)
	}
}

func ragMetricSnapshot(t *testing.T) (uint64, float64) {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	var durationCount uint64
	var cacheHits float64
	for _, family := range families {
		switch family.GetName() {
		case "rag_query_duration_seconds":
			for _, metric := range family.GetMetric() {
				if metric.GetHistogram() != nil {
					durationCount += metric.GetHistogram().GetSampleCount()
				}
			}
		case "rag_cache_hits_total":
			for _, metric := range family.GetMetric() {
				if metric.GetCounter() != nil {
					cacheHits += metric.GetCounter().GetValue()
				}
			}
		}
	}
	return durationCount, cacheHits
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
