package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/scenic-guide/internal/model"
)

func (s *RAGService) bm25FallbackVector(content string) string {
	tokens := s.bm25.Tokenize(content)
	vector := make(map[string]float64)
	for _, token := range tokens {
		vector[token]++
	}
	vectorJSON, _ := json.Marshal(vector)
	return string(vectorJSON)
}

func (s *RAGService) GenerateEmbedding(text string) (string, error) {
	if !s.useBM25 && s.embedding != nil {
		vec, err := s.embedding.GenerateEmbedding(text)
		if err != nil {
			return "", err
		}
		vecJSON, err := json.Marshal(vec)
		if err != nil {
			return "", err
		}
		return string(vecJSON), nil
	}

	return s.bm25FallbackVector(text), nil
}

func (s *RAGService) CosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) == 0 || len(vec2) == 0 {
		return 0
	}
	// 维度不一致的向量语义上不可比:不应截断后再做点积(会得到无意义的相似度),
	// 直接返回 0。正常运行时 query 与 chunk 的 embedding 维度由同一 provider 决定、恒定相等;
	// 此分支主要用于防御维度异常的数据。
	if len(vec1) != len(vec2) {
		return 0
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

func (s *RAGService) BM25Similarity(query, content string) float64 {
	queryTokens := s.bm25.Tokenize(query)
	docTokens := s.bm25.Tokenize(content)
	return s.bm25.CalculateSimilarity(queryTokens, docTokens)
}

func (s *RAGService) RetrieveRelevantKnowledge(query string, topK int) ([]model.KnowledgeChunk, error) {
	return s.RetrieveRelevantKnowledgeWithOptions(query, RetrievalOptions{TopK: topK})
}

func (s *RAGService) RetrieveRelevantKnowledgeWithOptions(query string, options RetrievalOptions) ([]model.KnowledgeChunk, error) {
	if options.TopK <= 0 {
		options.TopK = TopK
	}
	mode := normalizeRetrievalMode(options.Mode, s.embedding != nil && s.embedding.IsAvailable(), s.useBM25)
	if options.EmbeddingWeight <= 0 && options.BM25Weight <= 0 {
		options.EmbeddingWeight = 0.6
		options.BM25Weight = 0.4
	}
	if options.RRFK <= 0 {
		options.RRFK = 60
	}

	allChunks, err := s.getCachedKnowledge()
	if err != nil {
		return nil, fmt.Errorf("获取所有知识片段失败: %v", err)
	}

	if len(allChunks) == 0 {
		return nil, nil
	}

	// Use LLM query rewrite when API key is available, fallback to config-based expansion
	var retrievalText string
	var addedTerms []string
	if !options.SkipModelEnhancement && s.chatAPIKey != "" && s.profile != nil {
		retrievalText, addedTerms = s.LLMQueryRewrite(s.profile, query)
	} else {
		retrievalText, addedTerms = s.configBasedQueryExpansion(s.profile, query)
	}
	expandedQuery := retrievalQueryExpansion{Original: query, RetrievalText: retrievalText, AddedTerms: addedTerms}
	queryTokens := s.bm25.Tokenize(expandedQuery.RetrievalText)
	var queryVec []float64
	if modeUsesEmbedding(mode) {
		vec, err := s.getCachedEmbedding(query)
		if err == nil {
			queryVec = vec
		}
	}

	if len(queryVec) == 0 && modeUsesEmbedding(mode) {
		mode = RetrievalModeBM25Local
	}

	candidateChunks := allChunks
	if mode == RetrievalModeBM25Local || mode == RetrievalModeLightRerank {
		candidateChunks = s.getBM25CandidateChunks(queryTokens, allChunks)
	}

	scoredChunks := make([]retrievalScoredChunk, 0, len(candidateChunks))
	for _, chunk := range candidateChunks {
		score := s.scoreChunkForMode(query, expandedQuery.RetrievalText, queryTokens, queryVec, chunk, mode, options)
		if score >= MinSimilarityThreshold {
			scoredChunks = append(scoredChunks, retrievalScoredChunk{chunk: chunk, similarity: score})
		}
	}

	if mode == RetrievalModeRRFFusion {
		scoredChunks = s.rrfFusionScores(query, expandedQuery.RetrievalText, queryTokens, queryVec, allChunks, options.RRFK)
	} else if mode == RetrievalModeLightRerank {
		for i := range scoredChunks {
			scoredChunks[i].similarity += s.lightRerankBoost(query, expandedQuery, queryTokens, scoredChunks[i].chunk)
		}
	}

	if len(scoredChunks) == 0 {
		return nil, nil
	}

	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].similarity > scoredChunks[j].similarity
	})

	result := make([]model.KnowledgeChunk, 0, options.TopK)
	for i := 0; i < min(options.TopK, len(scoredChunks)); i++ {
		result = append(result, scoredChunks[i].chunk)
	}

	// Apply LLM reranking when API key is available (enhances retrieval quality)
	if !options.SkipModelEnhancement && s.chatAPIKey != "" && len(result) > 1 {
		result = s.LLMRerank(query, result, options.TopK)
	}

	return result, nil
}

func normalizeRetrievalMode(mode RetrievalMode, hasEmbedding, useBM25 bool) RetrievalMode {
	switch mode {
	case RetrievalModeBM25Local, RetrievalModeEmbedding, RetrievalModeHybridWeighted, RetrievalModeRRFFusion, RetrievalModeLightRerank:
		if mode != RetrievalModeBM25Local && mode != RetrievalModeLightRerank && !hasEmbedding {
			return RetrievalModeBM25Local
		}
		return mode
	default:
		if useBM25 || !hasEmbedding {
			return RetrievalModeBM25Local
		}
		return RetrievalModeEmbedding
	}
}

func modeUsesEmbedding(mode RetrievalMode) bool {
	return mode == RetrievalModeEmbedding || mode == RetrievalModeHybridWeighted || mode == RetrievalModeRRFFusion
}

func (s *RAGService) scoreChunkForMode(originalQuery, retrievalQuery string, queryTokens []string, queryVec []float64, chunk model.KnowledgeChunk, mode RetrievalMode, options RetrievalOptions) float64 {
	bm25Score := s.bm25.CalculateSimilarity(queryTokens, s.getCachedChunkTokens(chunk)) + s.lexicalBoost(retrievalQuery, chunk) + s.ProfileBasedIntentBoost(s.profile, originalQuery, chunk)*0.35
	if mode == RetrievalModeBM25Local || mode == RetrievalModeLightRerank || len(queryVec) == 0 {
		return bm25Score
	}

	semanticScore, ok := s.semanticSimilarity(queryVec, chunk)
	if !ok {
		return bm25Score
	}

	switch mode {
	case RetrievalModeEmbedding:
		return semanticScore*0.72 + math.Log1p(bm25Score)*0.28
	case RetrievalModeHybridWeighted:
		total := options.EmbeddingWeight + options.BM25Weight
		if total <= 0 {
			total = 1
		}
		return semanticScore*(options.EmbeddingWeight/total) + math.Log1p(bm25Score)*(options.BM25Weight/total)
	default:
		return semanticScore*0.72 + math.Log1p(bm25Score)*0.28
	}
}

func (s *RAGService) semanticSimilarity(queryVec []float64, chunk model.KnowledgeChunk) (float64, bool) {
	vec, err := s.parseVector(chunk.Vector)
	if err != nil || len(vec) == 0 {
		return 0, false
	}
	return s.CosineSimilarity(queryVec, vec), true
}

func (s *RAGService) rrfFusionScores(originalQuery, retrievalQuery string, queryTokens []string, queryVec []float64, chunks []model.KnowledgeChunk, rrfK float64) []retrievalScoredChunk {
	type fused struct {
		chunk model.KnowledgeChunk
		score float64
	}
	fusedByID := make(map[string]*fused, len(chunks))
	addRanking := func(scored []retrievalScoredChunk) {
		sort.Slice(scored, func(i, j int) bool {
			if scored[i].similarity == scored[j].similarity {
				return scored[i].chunk.ID < scored[j].chunk.ID
			}
			return scored[i].similarity > scored[j].similarity
		})
		for rank, item := range scored {
			if item.similarity < MinSimilarityThreshold {
				continue
			}
			entry := fusedByID[item.chunk.ID]
			if entry == nil {
				entry = &fused{chunk: item.chunk}
				fusedByID[item.chunk.ID] = entry
			}
			entry.score += 1 / (rrfK + float64(rank+1))
		}
	}

	bm25Scored := make([]retrievalScoredChunk, 0, len(chunks))
	embeddingScored := make([]retrievalScoredChunk, 0, len(chunks))
	for _, chunk := range chunks {
		bm25Score := s.bm25.CalculateSimilarity(queryTokens, s.getCachedChunkTokens(chunk)) + s.lexicalBoost(retrievalQuery, chunk) + s.ProfileBasedIntentBoost(s.profile, originalQuery, chunk)*0.35
		bm25Scored = append(bm25Scored, retrievalScoredChunk{chunk: chunk, similarity: bm25Score})
		if semanticScore, ok := s.semanticSimilarity(queryVec, chunk); ok {
			embeddingScored = append(embeddingScored, retrievalScoredChunk{chunk: chunk, similarity: semanticScore})
		}
	}
	addRanking(bm25Scored)
	addRanking(embeddingScored)

	result := make([]retrievalScoredChunk, 0, len(fusedByID))
	for _, item := range fusedByID {
		expandedQuery := retrievalQueryExpansion{Original: originalQuery, RetrievalText: retrievalQuery}
		result = append(result, retrievalScoredChunk{chunk: item.chunk, similarity: item.score + s.lightRerankBoost(originalQuery, expandedQuery, queryTokens, item.chunk)*0.002})
	}
	return result
}

func (s *RAGService) lightRerankBoost(query string, expandedQuery retrievalQueryExpansion, queryTokens []string, chunk model.KnowledgeChunk) float64 {
	title := chunk.Title
	content := chunk.Content
	haystack := title + "\n" + content
	boost := 0.0

	for _, token := range queryTokens {
		if len([]rune(token)) < 2 {
			continue
		}
		if strings.Contains(title, token) {
			boost += 1.5
		}
		if strings.Contains(content, token) {
			boost += 0.4
		}
	}
	for _, term := range expandedQuery.AddedTerms {
		if len([]rune(term)) < 2 {
			continue
		}
		if strings.Contains(title, term) {
			boost += 0.55
		} else if strings.Contains(content, term) {
			boost += 0.18
		}
	}
	for _, keyword := range s.getRelatedKeywords() {
		if len([]rune(keyword)) < 3 {
			continue
		}
		if strings.Contains(query, keyword) && strings.Contains(title, keyword) {
			boost += 2.0
		} else if strings.Contains(query, keyword) && strings.Contains(haystack, keyword) {
			boost += 0.8
		}
	}

	if sourceType := metadataString(chunk.Metadata, "source_type"); sourceType == "official" || sourceType == "government" {
		boost += 0.2
	}
	boost += s.ProfileBasedIntentBoost(s.profile, query, chunk)
	return boost
}

func metadataString(metadataJSON, key string) string {
	if strings.TrimSpace(metadataJSON) == "" {
		return ""
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func (s *RAGService) rebuildBM25IndexLocked(chunks []model.KnowledgeChunk) {
	s.tokenCache = make(map[string][]string, len(chunks))
	s.tokenIndex = make(map[string][]string)
	s.chunkByID = make(map[string]model.KnowledgeChunk, len(chunks))
	for _, chunk := range chunks {
		tokens := s.bm25.Tokenize(chunk.Title + "\n" + chunk.Content)
		s.tokenCache[chunk.ID] = tokens
		s.chunkByID[chunk.ID] = chunk
		seen := make(map[string]struct{}, len(tokens))
		for _, token := range tokens {
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			s.tokenIndex[token] = append(s.tokenIndex[token], chunk.ID)
		}
	}
	// Update BM25 corpus-level statistics for proper IDF and doc-length normalization
	s.bm25.UpdateCorpusStats(s.tokenIndex, s.tokenCache)
}

func (s *RAGService) getBM25CandidateChunks(queryTokens []string, fallback []model.KnowledgeChunk) []model.KnowledgeChunk {
	s.cacheMutex.RLock()
	tokenIndex := s.tokenIndex
	chunkByID := s.chunkByID
	s.cacheMutex.RUnlock()
	if len(tokenIndex) == 0 || len(chunkByID) == 0 {
		return fallback
	}

	counts := make(map[string]int)
	for _, token := range queryTokens {
		for _, id := range tokenIndex[token] {
			counts[id]++
		}
	}
	if len(counts) == 0 {
		return fallback
	}

	type candidate struct {
		id    string
		count int
	}
	candidates := make([]candidate, 0, len(counts))
	for id, count := range counts {
		candidates = append(candidates, candidate{id: id, count: count})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].count == candidates[j].count {
			return candidates[i].id < candidates[j].id
		}
		return candidates[i].count > candidates[j].count
	})

	const maxCandidates = 600
	limit := min(len(candidates), maxCandidates)
	chunks := make([]model.KnowledgeChunk, 0, limit)
	for i := 0; i < limit; i++ {
		if chunk, ok := chunkByID[candidates[i].id]; ok {
			chunks = append(chunks, chunk)
		}
	}
	return chunks
}

func (s *RAGService) getCachedChunkTokens(chunk model.KnowledgeChunk) []string {
	s.cacheMutex.RLock()
	tokens, ok := s.tokenCache[chunk.ID]
	s.cacheMutex.RUnlock()
	if ok {
		return tokens
	}

	tokens = s.bm25.Tokenize(chunk.Title + "\n" + chunk.Content)
	s.cacheMutex.Lock()
	if s.tokenCache == nil {
		s.tokenCache = make(map[string][]string)
	}
	s.tokenCache[chunk.ID] = tokens
	s.cacheMutex.Unlock()
	return tokens
}

func (s *RAGService) parseVector(vectorStr string) ([]float64, error) {
	var vector []float64
	if err := json.Unmarshal([]byte(vectorStr), &vector); err != nil {
		return nil, err
	}
	return vector, nil
}

func (s *RAGService) lexicalBoost(query string, chunk model.KnowledgeChunk) float64 {
	haystack := chunk.Title + "\n" + chunk.Content
	boost := 0.0

	for _, keyword := range s.getRelatedKeywords() {
		if len([]rune(keyword)) < 3 {
			continue
		}
		if strings.Contains(query, keyword) && strings.Contains(haystack, keyword) {
			boost += 2.5
		}
	}

	// 配置化条件加分（替代硬编码）
	for _, cb := range s.getConditionalBoosts() {
		boost += conditionalTermBoost(query, haystack, cb.QueryTerms, cb.ContentTerms)
	}

	return boost
}

func containsAny(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func conditionalTermBoost(query, haystack string, queryTerms, contentTerms []string) float64 {
	queryMatched := false
	for _, term := range queryTerms {
		if strings.Contains(query, term) {
			queryMatched = true
			break
		}
	}
	if !queryMatched {
		return 0
	}

	boost := 0.0
	for _, term := range contentTerms {
		if strings.Contains(haystack, term) {
			boost += 1.2
		}
	}
	return boost
}
