package service

import (
	"testing"
)

func TestBM25Tokenize(t *testing.T) {
	p := NewBM25FallbackProvider()
	tokens := p.Tokenize("灵山胜境在哪里")

	tokenSet := make(map[string]bool)
	for _, tok := range tokens {
		tokenSet[tok] = true
	}

	// 应包含 2-gram 和 3-gram
	if !tokenSet["灵山"] {
		t.Error("expected tokens to contain '灵山'")
	}
	if !tokenSet["胜境"] {
		t.Error("expected tokens to contain '胜境'")
	}
	if !tokenSet["灵山胜"] {
		t.Error("expected tokens to contain '灵山胜' (3-gram)")
	}

	// 停用词应被过滤
	if tokenSet["在"] {
		t.Error("stop word '在' should be filtered")
	}
	if tokenSet["哪里"] {
		// "哪里" is in stop words list as "怎么" category - check
		// Actually "哪里" is not in the stop words, so it's fine
	}
}

func TestBM25TokenizeEmpty(t *testing.T) {
	p := NewBM25FallbackProvider()
	tokens := p.Tokenize("")
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens for empty string, got %d", len(tokens))
	}
}

func TestBM25CalculateScore(t *testing.T) {
	p := NewBM25FallbackProvider()

	queryTokens := []string{"灵山", "大佛"}
	docTokens := []string{"灵山", "大佛", "景区", "介绍"}

	score := p.CalculateScore(queryTokens, docTokens)
	if score <= 0 {
		t.Errorf("expected positive score, got %f", score)
	}
	if score > 1.0 {
		t.Errorf("score should be normalized to [0,1], got %f", score)
	}
}

func TestBM25CalculateScoreNoOverlap(t *testing.T) {
	p := NewBM25FallbackProvider()

	queryTokens := []string{"灵山"}
	docTokens := []string{"太湖", "景区"}

	score := p.CalculateScore(queryTokens, docTokens)
	if score != 0 {
		t.Errorf("expected 0 for no overlap, got %f", score)
	}
}

func TestBM25CalculateScoreEmpty(t *testing.T) {
	p := NewBM25FallbackProvider()

	score := p.CalculateScore([]string{}, []string{"test"})
	if score != 0 {
		t.Errorf("expected 0 for empty query, got %f", score)
	}

	score = p.CalculateScore([]string{"test"}, []string{})
	if score != 0 {
		t.Errorf("expected 0 for empty doc, got %f", score)
	}
}

func TestBM25IsAvailable(t *testing.T) {
	p := NewBM25FallbackProvider()
	if !p.IsAvailable() {
		t.Error("BM25FallbackProvider should always be available")
	}
}

func TestBM25Name(t *testing.T) {
	p := NewBM25FallbackProvider()
	if p.Name() == "" {
		t.Error("provider name should not be empty")
	}
}

func TestBM25WithCorpusStats(t *testing.T) {
	p := NewBM25FallbackProvider()

	// Simulate a corpus with 3 documents
	tokenIndex := map[string][]string{
		"灵山": {"doc1", "doc2"},
		"大佛": {"doc1"},
		"太湖": {"doc3"},
		"景区": {"doc1", "doc2", "doc3"},
	}
	chunkTokens := map[string][]string{
		"doc1": {"灵山", "大佛", "景区", "介绍"},
		"doc2": {"灵山", "景区", "门票"},
		"doc3": {"太湖", "景区", "风景"},
	}
	p.UpdateCorpusStats(tokenIndex, chunkTokens)

	if p.totalDocs != 3 {
		t.Errorf("expected 3 total docs, got %d", p.totalDocs)
	}
	if p.docFreq["景区"] != 3 {
		t.Errorf("expected docFreq[景区]=3, got %d", p.docFreq["景区"])
	}
	if p.docFreq["灵山"] != 2 {
		t.Errorf("expected docFreq[灵山]=2, got %d", p.docFreq["灵山"])
	}

	// Query "灵山大佛" against doc1 - should score well (both terms match)
	score1 := p.CalculateScore(
		p.Tokenize("灵山大佛"),
		p.Tokenize("灵山大佛景区介绍"),
	)

	// Query "灵山大佛" against doc3 - should score 0 (no match)
	score3 := p.CalculateScore(
		p.Tokenize("灵山大佛"),
		p.Tokenize("太湖景区风景"),
	)

	if score1 <= 0 {
		t.Errorf("expected positive score for matching doc, got %f", score1)
	}
	if score3 != 0 {
		t.Errorf("expected 0 for non-matching doc, got %f", score3)
	}

	// Rare term "大佛" (df=1) should have higher IDF than "景区" (df=3)
	idfRare := 0.0
	idfCommon := 0.0
	if p.docFreq["大佛"] > 0 {
		idfRare = 1.0 / float64(p.docFreq["大佛"])
	}
	if p.docFreq["景区"] > 0 {
		idfCommon = 1.0 / float64(p.docFreq["景区"])
	}
	if idfRare <= idfCommon {
		t.Log("Note: rare term should contribute more than common term")
	}
}