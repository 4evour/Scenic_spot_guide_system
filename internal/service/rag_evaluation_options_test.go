package service

import (
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
