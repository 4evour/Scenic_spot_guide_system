package service

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

var registerRAGTestDriver sync.Once

func newTestRAGService(t *testing.T) *RAGService {
	t.Helper()

	const driverName = "modernc-rag-test"
	registerRAGTestDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	repo := repository.NewKnowledgeRepository(db)
	return NewRAGService(repo, "", "", "", nil)
}

type staticEmbeddingProvider struct {
	vectors map[string][]float64
}

func (p staticEmbeddingProvider) GenerateEmbedding(text string) ([]float64, error) {
	if vec, ok := p.vectors[text]; ok {
		return vec, nil
	}
	return []float64{0, 1}, nil
}

func (p staticEmbeddingProvider) Name() string {
	return "static-test"
}

func (p staticEmbeddingProvider) IsAvailable() bool {
	return true
}

func newTestRAGServiceWithEmbedding(t *testing.T, provider EmbeddingProvider) *RAGService {
	t.Helper()

	const driverName = "modernc-rag-test"
	registerRAGTestDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	repo := repository.NewKnowledgeRepository(db)
	return NewRAGService(repo, "", "", "", provider)
}

func TestRAGServiceLoadsJSONAndRetrievesWithBM25(t *testing.T) {
	rag := newTestRAGService(t)
	data := []byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"test","content":"灵山大佛通高88米，佛体79米，莲花瓣9米。","metadata":{"category":"景点"}},
		{"id":"palace","title":"灵山梵宫","source":"test","content":"灵山梵宫汇集东阳木雕、琉璃、油画等传统工艺。","metadata":{"category":"文化"}}
	]`)

	loaded, err := rag.LoadKnowledgeJSON(data)
	if err != nil {
		t.Fatalf("LoadKnowledgeJSON returned error: %v", err)
	}
	if loaded != 2 {
		t.Fatalf("loaded = %d, want 2", loaded)
	}

	chunks, err := rag.RetrieveRelevantKnowledge("灵山大佛有多高", 1)
	if err != nil {
		t.Fatalf("RetrieveRelevantKnowledge returned error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("retrieved chunks = %d, want 1", len(chunks))
	}
	if chunks[0].ID != "buddha-height" {
		t.Fatalf("top chunk id = %q, want buddha-height", chunks[0].ID)
	}
}

func TestRAGServiceUsesHybridScoreWhenEmbeddingAvailable(t *testing.T) {
	rag := newTestRAGServiceWithEmbedding(t, staticEmbeddingProvider{
		vectors: map[string][]float64{
			"灵山梵宫有什么工艺？": {1, 0},
		},
	})
	data := []byte(`[
		{"id":"generic-vector","title":"通用介绍","source":"test","content":"这里介绍游客服务中心和普通导览信息。","vector":"[1,0]"},
		{"id":"palace-craft","title":"灵山梵宫工艺","source":"test","content":"灵山梵宫汇集东阳木雕、敦煌壁画、扬州漆器等传统工艺。","vector":"[0.95,0.05]"}
	]`)

	if _, err := rag.LoadKnowledgeJSON(data); err != nil {
		t.Fatalf("LoadKnowledgeJSON returned error: %v", err)
	}

	chunks, err := rag.RetrieveRelevantKnowledge("灵山梵宫有什么工艺？", 1)
	if err != nil {
		t.Fatalf("RetrieveRelevantKnowledge returned error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("retrieved chunks = %d, want 1", len(chunks))
	}
	if chunks[0].ID != "palace-craft" {
		t.Fatalf("top chunk id = %q, want palace-craft", chunks[0].ID)
	}
}

func TestRAGServiceEvaluateQuestionsReportsKeywordCoverage(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"test","content":"灵山大佛高88米，主体高79米，莲花瓣高9米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	report, err := rag.EvaluateQuestions([]RAGEvaluationCase{
		{
			Question:         "灵山大佛有多高？",
			ExpectedKeywords: []string{"88米", "79米"},
		},
	})
	if err != nil {
		t.Fatalf("EvaluateQuestions returned error: %v", err)
	}
	if report.Total != 1 || report.Passed != 1 || report.Failed != 0 {
		t.Fatalf("unexpected report counts: total=%d passed=%d failed=%d", report.Total, report.Passed, report.Failed)
	}
	if report.AverageKeywordCoverage != 1 {
		t.Fatalf("coverage = %.2f, want 1.00", report.AverageKeywordCoverage)
	}
	if report.AverageRecallAtK != 1 {
		t.Fatalf("recall@k = %.2f, want 1.00", report.AverageRecallAtK)
	}
	if report.MRRAtK != 1 {
		t.Fatalf("mrr@k = %.2f, want 1.00", report.MRRAtK)
	}
	if report.RetrievalP95Ms <= 0 {
		t.Fatalf("retrieval p95 should be recorded")
	}
}

func TestRAGServiceEvaluateFileSupportsAnswerFallbackKeywords(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"location","title":"灵山胜境位置","source":"test","content":"灵山胜境位于江苏省无锡市太湖西北部的马山镇。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	evalPath := filepath.Join(t.TempDir(), "eval.json")
	body, err := json.Marshal([]RAGEvaluationCase{
		{
			Question: "灵山胜境位于哪里？",
			Answer:   "灵山胜境位于江苏省无锡市太湖西北部的马山镇。",
		},
	})
	if err != nil {
		t.Fatalf("marshal eval: %v", err)
	}
	if err := os.WriteFile(evalPath, body, 0o600); err != nil {
		t.Fatalf("write eval: %v", err)
	}

	report, err := rag.EvaluateFile(evalPath)
	if err != nil {
		t.Fatalf("EvaluateFile returned error: %v", err)
	}
	if report.Total != 1 {
		t.Fatalf("total = %d, want 1", report.Total)
	}
	if len(report.Results[0].MatchedKeywords) == 0 {
		t.Fatalf("expected derived keywords to match at least one answer fact")
	}
}

func TestRAGServiceEvaluateQuestionsReportsRetrievalMiss(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"palace","title":"灵山梵宫","source":"test","content":"灵山梵宫汇集东阳木雕、敦煌壁画、扬州漆器等传统工艺。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	report, err := rag.EvaluateQuestionsWithOptions([]RAGEvaluationCase{
		{
			Question:         "灵山梵宫有什么工艺？",
			ExpectedKeywords: []string{"东阳木雕"},
			ExpectedChunkIDs: []string{"missing-id"},
			Category:         "文化",
			Difficulty:       "medium",
		},
	}, EvaluationOptions{TopK: 1, RetrievalOnly: true})
	if err != nil {
		t.Fatalf("EvaluateQuestionsWithOptions returned error: %v", err)
	}
	if report.Results[0].RecallAtK != 0 {
		t.Fatalf("recall@k = %.2f, want 0", report.Results[0].RecallAtK)
	}
	if report.Results[0].FirstRelevantRank != 0 {
		t.Fatalf("first relevant rank = %d, want 0", report.Results[0].FirstRelevantRank)
	}
	if report.Results[0].Category != "文化" || report.Results[0].Difficulty != "medium" {
		t.Fatalf("category/difficulty not propagated: %+v", report.Results[0])
	}
}

func TestRAGServiceEvaluateQuestionsReportsGroupsAndFailureReasons(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official","content":"灵山大佛通高88米，佛体79米，莲花瓣9米。","metadata":{"source_type":"official"}},
		{"id":"palace-art","title":"灵山梵宫艺术","source":"official","content":"灵山梵宫可欣赏东阳木雕、琉璃巨制、敦煌壁画等艺术瑰宝。","metadata":{"source_type":"official"}}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	report, err := rag.EvaluateQuestionsWithOptions([]RAGEvaluationCase{
		{
			Question:         "灵山大佛有多高？",
			ExpectedKeywords: []string{"88米"},
			ExpectedChunkIDs: []string{"buddha-height"},
			Category:         "closed_real",
			Difficulty:       "easy",
			SourceType:       "official",
		},
		{
			Question:         "梵宫能看哪些工艺？",
			ExpectedKeywords: []string{"东阳木雕"},
			ExpectedChunkIDs: []string{"missing-palace"},
			Category:         "open_real",
			Difficulty:       "medium",
			SourceType:       "official",
		},
	}, EvaluationOptions{TopK: 1, RetrievalOnly: true})
	if err != nil {
		t.Fatalf("EvaluateQuestionsWithOptions returned error: %v", err)
	}

	if len(report.GroupStats) == 0 {
		t.Fatalf("expected grouped stats")
	}
	if got := report.Results[0].SourceType; got != "official" {
		t.Fatalf("source type = %q, want official", got)
	}
	if got := report.Results[1].FailureReason; got != "retrieval_miss" {
		t.Fatalf("failure reason = %q, want retrieval_miss", got)
	}

	categoryStats := findGroupStat(report.GroupStats, "category", "closed_real")
	if categoryStats.Total != 1 || categoryStats.Passed != 1 || categoryStats.AverageRecallAtK != 1 {
		t.Fatalf("unexpected category stats: %+v", categoryStats)
	}
	sourceStats := findGroupStat(report.GroupStats, "source_type", "official")
	if sourceStats.Total != 2 || sourceStats.Failed != 1 || len(sourceStats.Failures) != 1 {
		t.Fatalf("unexpected source stats: %+v", sourceStats)
	}
}

func findGroupStat(stats []RAGEvaluationGroupStats, groupBy, name string) RAGEvaluationGroupStats {
	for _, stat := range stats {
		if stat.GroupBy == groupBy && stat.Name == name {
			return stat
		}
	}
	return RAGEvaluationGroupStats{}
}

func TestPercentileDuration(t *testing.T) {
	values := []time.Duration{
		10 * time.Millisecond,
		30 * time.Millisecond,
		20 * time.Millisecond,
	}

	if got := percentileDuration(values, 0.50).Milliseconds(); got != 20 {
		t.Fatalf("p50 = %dms, want 20ms", got)
	}
	if got := percentileDuration(values, 0.95).Milliseconds(); got != 30 {
		t.Fatalf("p95 = %dms, want 30ms", got)
	}
}
