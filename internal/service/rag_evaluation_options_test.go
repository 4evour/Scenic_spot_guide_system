package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEvaluateQuestionsRejectsUnknownGenerationMode(t *testing.T) {
	rag := newTestRAGService(t)

	_, err := rag.EvaluateQuestionsWithOptions(nil, EvaluationOptions{
		GenerationMode: EvaluationGenerationMode("unknown"),
	})
	if err == nil {
		t.Fatal("EvaluateQuestionsWithOptions accepted an unknown generation mode")
	}
	if !strings.Contains(err.Error(), "generation mode") {
		t.Fatalf("error = %q, want generation mode validation", err)
	}
}

func TestEvaluateQuestionsCanIncludeFullResponse(t *testing.T) {
	rag := newTestRAGService(t)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"height","title":"灵山大佛高度","source":"official","content":"灵山大佛通高88米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	report, err := rag.EvaluateQuestionsWithOptions([]RAGEvaluationCase{{
		Question:         "灵山大佛有多高？",
		ExpectedKeywords: []string{"88米"},
		ExpectedChunkIDs: []string{"height"},
	}}, EvaluationOptions{
		TopK:            1,
		IncludeResponse: true,
	})
	if err != nil {
		t.Fatalf("EvaluateQuestionsWithOptions returned error: %v", err)
	}
	if len(report.Results) != 1 || !strings.Contains(report.Results[0].Response, "88米") {
		t.Fatalf("full response was not retained: %+v", report.Results)
	}
	if report.Results[0].ResponsePreview != "" {
		t.Fatalf("preview should be omitted when full response is retained: %+v", report.Results[0])
	}
	data, err := json.Marshal(report.Results[0])
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(data), `"response_preview"`) {
		t.Fatalf("serialized full-response result should not duplicate preview: %s", data)
	}

	previewReport, err := rag.EvaluateQuestionsWithOptions([]RAGEvaluationCase{{
		Question:         "灵山大佛有多高？",
		ExpectedKeywords: []string{"88米"},
		ExpectedChunkIDs: []string{"height"},
	}}, EvaluationOptions{TopK: 1})
	if err != nil {
		t.Fatalf("default preview evaluation returned error: %v", err)
	}
	if previewReport.Results[0].ResponsePreview == "" || previewReport.Results[0].Response != "" {
		t.Fatalf("default report should retain only preview: %+v", previewReport.Results[0])
	}
}
