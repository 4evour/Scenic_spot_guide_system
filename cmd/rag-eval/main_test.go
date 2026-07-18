package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/scenic-guide/internal/service"
)

func TestAttachRunInfoRecordsDatasetAndEnvironment(t *testing.T) {
	report := &service.RAGEvaluationReport{Total: 2, TopK: 8}
	options := evaluationRunOptions{
		topK:            8,
		concurrency:     4,
		repeat:          3,
		retrievalOnly:   true,
		reportEnv:       true,
		retrievalMode:   service.RetrievalModeRRFFusion,
		embeddingWeight: 0.6,
		bm25Weight:      0.4,
		rrfK:            60,
	}

	attachRunInfo(report, "knowledge/real/lingshan_real_chunks.jsonl", "knowledge/real/lingshan_real_eval_open.json", 12, "bm25-local", "disabled", options)

	if report.RunInfo.KnowledgeFile != "knowledge/real/lingshan_real_chunks.jsonl" {
		t.Fatalf("knowledge file = %q", report.RunInfo.KnowledgeFile)
	}
	if report.RunInfo.KnowledgeChunks != 12 || report.RunInfo.EvaluationCases != 2 {
		t.Fatalf("unexpected dataset sizes: %+v", report.RunInfo)
	}
	if report.RunInfo.Concurrency != 4 || report.RunInfo.Repeat != 3 || !report.RunInfo.RetrievalOnly {
		t.Fatalf("unexpected run options: %+v", report.RunInfo)
	}
	if report.RunInfo.OS == "" || report.RunInfo.GoVersion == "" || report.RunInfo.CPU == "" {
		t.Fatalf("environment should be recorded: %+v", report.RunInfo)
	}
	if report.RunInfo.EmbeddingProvider != "bm25-local" || report.RunInfo.GenerationProvider != "disabled" {
		t.Fatalf("unexpected providers: %+v", report.RunInfo)
	}
	if report.RunInfo.RetrievalMode != string(service.RetrievalModeRRFFusion) || report.RunInfo.RRFK != 60 {
		t.Fatalf("retrieval mode should be recorded: %+v", report.RunInfo)
	}
}

func TestWriteJSONReportCreatesFile(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "report.json")
	report := &service.RAGEvaluationReport{Total: 1, TopK: 8}

	if err := writeJSONReport(outputPath, report); err != nil {
		t.Fatalf("writeJSONReport returned error: %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected report content")
	}
}

func TestSummarizeResultsIncludesGroupStats(t *testing.T) {
	report := summarizeResults([]service.RAGEvaluationResult{
		{
			Question:        "事实问答",
			Category:        "closed_real",
			Difficulty:      "easy",
			SourceType:      "official",
			Passed:          true,
			KeywordCoverage: 1,
			RecallAtK:       1,
			ReciprocalRank:  1,
		},
	}, 8, true, nowForTest())

	if len(report.GroupStats) == 0 {
		t.Fatalf("expected group stats")
	}
}

func TestParseCompareModesTrimsAndDefaults(t *testing.T) {
	modes := parseCompareModes(" bm25-local, rrf-fusion ,, light-rerank ")
	if len(modes) != 3 {
		t.Fatalf("modes length = %d, want 3", len(modes))
	}
	if modes[0] != service.RetrievalModeBM25Local || modes[1] != service.RetrievalModeRRFFusion || modes[2] != service.RetrievalModeLightRerank {
		t.Fatalf("unexpected modes: %+v", modes)
	}

	defaults := parseCompareModes("")
	if len(defaults) != 0 {
		t.Fatalf("empty compare modes should return no modes: %+v", defaults)
	}
}

func TestComparisonReportIsNotPassingWhenAnyModeFails(t *testing.T) {
	report := comparisonReport{
		Modes: []comparisonModeReport{
			{Name: "bm25-local", Report: &service.RAGEvaluationReport{Total: 1, Passed: 1}},
			{Name: "rrf-fusion", Report: &service.RAGEvaluationReport{Total: 1, Passed: 0}},
		},
	}

	if report.IsPassing() {
		t.Fatalf("comparison report should fail when any mode fails")
	}
}

func TestBuildEvaluationProvidersDefaultsToLocalWithoutConfig(t *testing.T) {
	missingConfigDir := filepath.Join(t.TempDir(), "missing")
	options := evaluationRunOptions{
		generationMode: service.EvaluationGenerationModeLocal,
		configDir:      missingConfigDir,
	}

	cfg, provider, err := buildEvaluationProviders(options)
	if err != nil {
		t.Fatalf("buildEvaluationProviders returned error for local mode: %v", err)
	}
	if cfg != nil || provider != nil {
		t.Fatalf("local mode should not load providers: cfg=%v provider=%v", cfg, provider)
	}
	if got := generationProviderName(cfg, options); got != string(service.EvaluationGenerationModeLocal) {
		t.Fatalf("generation provider = %q, want %q", got, service.EvaluationGenerationModeLocal)
	}
}

func TestRunEvaluationDefaultsToLocalWithoutConfigOrEnvironment(t *testing.T) {
	clearScenicGuideEnvironment(t)

	workspace := t.TempDir()
	knowledgeFile := filepath.Join(workspace, "knowledge.jsonl")
	evalFile := filepath.Join(workspace, "eval.json")
	if err := os.WriteFile(knowledgeFile, []byte(`{"id":"local-only","title":"本地评测","source":"test","content":"本地 BM25 评测无需外部生成服务。"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write knowledge file: %v", err)
	}
	if err := os.WriteFile(evalFile, []byte(`[{"question":"本地评测需要什么？","expected_keywords":["无需外部生成服务"],"expected_chunk_ids":["local-only"]}]`), 0o600); err != nil {
		t.Fatalf("write evaluation file: %v", err)
	}

	report, err := runEvaluation(knowledgeFile, evalFile, evaluationRunOptions{
		topK:      1,
		configDir: filepath.Join(workspace, "missing-config"),
	})
	if err != nil {
		t.Fatalf("runEvaluation returned error in default local mode: %v", err)
	}
	if got := report.RunInfo.GenerationProvider; got != string(service.EvaluationGenerationModeLocal) {
		t.Fatalf("generation provider = %q, want local", got)
	}
	if report.RunInfo.EmbeddingProvider != "bm25-local" {
		t.Fatalf("embedding provider = %q, want bm25-local", report.RunInfo.EmbeddingProvider)
	}
}

func clearScenicGuideEnvironment(t *testing.T) {
	t.Helper()
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(key, "SCENIC_GUIDE_") {
			continue
		}
		value := os.Getenv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if err := os.Setenv(key, value); err != nil {
				t.Errorf("restore %s: %v", key, err)
			}
		})
	}
}

func nowForTest() time.Time {
	return time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
}
