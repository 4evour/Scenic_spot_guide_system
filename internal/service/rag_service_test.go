package service

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

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
