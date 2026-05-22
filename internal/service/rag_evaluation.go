package service

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/scenic-guide/internal/model"
)

type RAGEvaluationCase struct {
	Question         string   `json:"question"`
	Answer           string   `json:"answer,omitempty"`
	Source           string   `json:"source,omitempty"`
	ExpectedKeywords []string `json:"expected_keywords,omitempty"`
	ExpectedChunkIDs []string `json:"expected_chunk_ids,omitempty"`
	Category         string   `json:"category,omitempty"`
	Difficulty       string   `json:"difficulty,omitempty"`
	SourceType       string   `json:"source_type,omitempty"`
}

type EvaluationOptions struct {
	TopK          int
	RetrievalOnly bool
}

type RAGEvaluationResult struct {
	Question           string   `json:"question"`
	Category           string   `json:"category,omitempty"`
	Difficulty         string   `json:"difficulty,omitempty"`
	SourceType         string   `json:"source_type,omitempty"`
	Passed             bool     `json:"passed"`
	FailureReason      string   `json:"failure_reason,omitempty"`
	KeywordCoverage    float64  `json:"keyword_coverage"`
	RecallAtK          float64  `json:"recall_at_k"`
	ReciprocalRank     float64  `json:"reciprocal_rank"`
	FirstRelevantRank  int      `json:"first_relevant_rank"`
	MatchedKeywords    []string `json:"matched_keywords"`
	MissingKeywords    []string `json:"missing_keywords"`
	ExpectedChunkIDs   []string `json:"expected_chunk_ids,omitempty"`
	RetrievedChunkIDs  []string `json:"retrieved_chunk_ids"`
	RetrievalLatencyMs int64    `json:"retrieval_latency_ms"`
	ResponsePreview    string   `json:"response_preview"`
	Error              string   `json:"error,omitempty"`
}

type RAGEvaluationGroupStats struct {
	GroupBy                string   `json:"group_by"`
	Name                   string   `json:"name"`
	Total                  int      `json:"total"`
	Passed                 int      `json:"passed"`
	Failed                 int      `json:"failed"`
	PassRate               float64  `json:"pass_rate"`
	AverageKeywordCoverage float64  `json:"average_keyword_coverage"`
	AverageRecallAtK       float64  `json:"average_recall_at_k"`
	MRRAtK                 float64  `json:"mrr_at_k"`
	Failures               []string `json:"failures,omitempty"`
}

type RAGEvaluationRunInfo struct {
	KnowledgeFile      string `json:"knowledge_file,omitempty"`
	EvaluationFile     string `json:"evaluation_file,omitempty"`
	KnowledgeChunks    int    `json:"knowledge_chunks"`
	EvaluationCases    int    `json:"evaluation_cases"`
	TopK               int    `json:"top_k"`
	Concurrency        int    `json:"concurrency"`
	Repeat             int    `json:"repeat"`
	RetrievalOnly      bool   `json:"retrieval_only"`
	EmbeddingProvider  string `json:"embedding_provider,omitempty"`
	GenerationProvider string `json:"generation_provider,omitempty"`
	OS                 string `json:"os,omitempty"`
	Arch               string `json:"arch,omitempty"`
	CPU                string `json:"cpu,omitempty"`
	GoVersion          string `json:"go_version,omitempty"`
}

type RAGEvaluationReport struct {
	Total                  int                       `json:"total"`
	Passed                 int                       `json:"passed"`
	Failed                 int                       `json:"failed"`
	TopK                   int                       `json:"top_k"`
	PassRate               float64                   `json:"pass_rate"`
	AverageKeywordCoverage float64                   `json:"average_keyword_coverage"`
	AverageRecallAtK       float64                   `json:"average_recall_at_k"`
	MRRAtK                 float64                   `json:"mrr_at_k"`
	RetrievalP50Ms         int64                     `json:"retrieval_p50_ms"`
	RetrievalP95Ms         int64                     `json:"retrieval_p95_ms"`
	RetrievalOnly          bool                      `json:"retrieval_only"`
	StartedAt              time.Time                 `json:"started_at"`
	FinishedAt             time.Time                 `json:"finished_at"`
	RunInfo                RAGEvaluationRunInfo      `json:"run_info,omitempty"`
	GroupStats             []RAGEvaluationGroupStats `json:"group_stats,omitempty"`
	Results                []RAGEvaluationResult     `json:"results"`
}

func (r RAGEvaluationReport) IsPassing() bool {
	return r.Total > 0 && r.Passed == r.Total
}

func (s *RAGService) EvaluateFile(evalFile string) (*RAGEvaluationReport, error) {
	data, err := os.ReadFile(evalFile)
	if err != nil {
		return nil, fmt.Errorf("读取评估文件失败: %v", err)
	}

	var cases []RAGEvaluationCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("解析评估文件失败: %v", err)
	}

	return s.EvaluateQuestions(cases)
}

func (s *RAGService) EvaluateQuestions(cases []RAGEvaluationCase) (*RAGEvaluationReport, error) {
	return s.EvaluateQuestionsWithOptions(cases, EvaluationOptions{TopK: TopK})
}

func (s *RAGService) EvaluateFileWithOptions(evalFile string, options EvaluationOptions) (*RAGEvaluationReport, error) {
	data, err := os.ReadFile(evalFile)
	if err != nil {
		return nil, fmt.Errorf("读取评估文件失败: %v", err)
	}

	var cases []RAGEvaluationCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("解析评估文件失败: %v", err)
	}

	return s.EvaluateQuestionsWithOptions(cases, options)
}

func (s *RAGService) EvaluateQuestionsWithOptions(cases []RAGEvaluationCase, options EvaluationOptions) (*RAGEvaluationReport, error) {
	if options.TopK <= 0 {
		options.TopK = TopK
	}

	report := &RAGEvaluationReport{
		Total:         len(cases),
		TopK:          options.TopK,
		RetrievalOnly: options.RetrievalOnly,
		StartedAt:     time.Now(),
		Results:       make([]RAGEvaluationResult, 0, len(cases)),
	}

	var coverageSum float64
	var recallSum float64
	var reciprocalRankSum float64
	latencies := make([]time.Duration, 0, len(cases))
	for _, item := range cases {
		keywords := normalizeEvaluationKeywords(item)
		result := RAGEvaluationResult{
			Question:         item.Question,
			Category:         item.Category,
			Difficulty:       item.Difficulty,
			SourceType:       item.SourceType,
			ExpectedChunkIDs: compactKeywords(item.ExpectedChunkIDs),
		}

		start := time.Now()
		chunks, err := s.RetrieveRelevantKnowledge(item.Question, options.TopK)
		latency := time.Since(start)
		if latency == 0 {
			latency = time.Nanosecond
		}
		latencies = append(latencies, latency)
		result.RetrievalLatencyMs = durationMillisCeil(latency)
		if err != nil {
			result.Error = err.Error()
			result.MissingKeywords = keywords
			result.FailureReason = "retrieval_error"
			report.Results = append(report.Results, result)
			continue
		}

		result.RetrievedChunkIDs = chunkIDs(chunks)
		result.RecallAtK, result.ReciprocalRank, result.FirstRelevantRank = retrievalMetrics(result.ExpectedChunkIDs, result.RetrievedChunkIDs)

		response := ""
		if options.RetrievalOnly {
			response = joinChunkContent(chunks)
		} else {
			response, err = s.QueryWithRAG(item.Question)
			if err != nil {
				result.Error = err.Error()
				result.MissingKeywords = keywords
				result.FailureReason = "generation_error"
				report.Results = append(report.Results, result)
				continue
			}
		}

		result.ResponsePreview = previewRunes(response, 120)
		result.MatchedKeywords, result.MissingKeywords = matchKeywords(response, keywords)
		result.KeywordCoverage = keywordCoverage(len(result.MatchedKeywords), len(keywords))
		result.Passed = len(keywords) > 0 && len(result.MissingKeywords) == 0 && retrievalPassed(result)
		if !result.Passed {
			result.FailureReason = classifyEvaluationFailure(result, len(keywords))
		}
		if result.Passed {
			report.Passed++
		}
		coverageSum += result.KeywordCoverage
		recallSum += result.RecallAtK
		reciprocalRankSum += result.ReciprocalRank
		report.Results = append(report.Results, result)
	}

	report.Failed = report.Total - report.Passed
	if report.Total > 0 {
		report.PassRate = float64(report.Passed) / float64(report.Total)
		report.AverageKeywordCoverage = coverageSum / float64(report.Total)
		report.AverageRecallAtK = recallSum / float64(report.Total)
		report.MRRAtK = reciprocalRankSum / float64(report.Total)
		report.RetrievalP50Ms = durationMillisCeil(percentileDuration(latencies, 0.50))
		report.RetrievalP95Ms = durationMillisCeil(percentileDuration(latencies, 0.95))
	}
	report.GroupStats = summarizeEvaluationGroups(report.Results)
	report.FinishedAt = time.Now()
	return report, nil
}

func classifyEvaluationFailure(result RAGEvaluationResult, keywordCount int) string {
	if result.Error != "" {
		return "evaluation_error"
	}
	if len(result.ExpectedChunkIDs) > 0 && result.RecallAtK == 0 {
		return "retrieval_miss"
	}
	if keywordCount > 0 && len(result.MissingKeywords) > 0 {
		return "keyword_miss"
	}
	return "not_passed"
}

func summarizeEvaluationGroups(results []RAGEvaluationResult) []RAGEvaluationGroupStats {
	stats := make([]RAGEvaluationGroupStats, 0)
	stats = append(stats, summarizeEvaluationGroup(results, "category", func(result RAGEvaluationResult) string {
		return result.Category
	})...)
	stats = append(stats, summarizeEvaluationGroup(results, "difficulty", func(result RAGEvaluationResult) string {
		return result.Difficulty
	})...)
	stats = append(stats, summarizeEvaluationGroup(results, "source_type", func(result RAGEvaluationResult) string {
		return result.SourceType
	})...)
	return stats
}

func SummarizeEvaluationGroups(results []RAGEvaluationResult) []RAGEvaluationGroupStats {
	return summarizeEvaluationGroups(results)
}

func summarizeEvaluationGroup(results []RAGEvaluationResult, groupBy string, nameOf func(RAGEvaluationResult) string) []RAGEvaluationGroupStats {
	type aggregate struct {
		stat        RAGEvaluationGroupStats
		coverageSum float64
		recallSum   float64
		mrrSum      float64
	}

	groups := make(map[string]*aggregate)
	for _, result := range results {
		name := strings.TrimSpace(nameOf(result))
		if name == "" {
			continue
		}
		if _, ok := groups[name]; !ok {
			groups[name] = &aggregate{stat: RAGEvaluationGroupStats{GroupBy: groupBy, Name: name}}
		}
		group := groups[name]
		group.stat.Total++
		if result.Passed {
			group.stat.Passed++
		} else {
			group.stat.Failed++
			if len(group.stat.Failures) < 10 {
				group.stat.Failures = append(group.stat.Failures, result.Question)
			}
		}
		group.coverageSum += result.KeywordCoverage
		group.recallSum += result.RecallAtK
		group.mrrSum += result.ReciprocalRank
	}

	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)

	stats := make([]RAGEvaluationGroupStats, 0, len(names))
	for _, name := range names {
		group := groups[name]
		if group.stat.Total > 0 {
			group.stat.PassRate = float64(group.stat.Passed) / float64(group.stat.Total)
			group.stat.AverageKeywordCoverage = group.coverageSum / float64(group.stat.Total)
			group.stat.AverageRecallAtK = group.recallSum / float64(group.stat.Total)
			group.stat.MRRAtK = group.mrrSum / float64(group.stat.Total)
		}
		stats = append(stats, group.stat)
	}
	return stats
}

func normalizeEvaluationKeywords(item RAGEvaluationCase) []string {
	if len(item.ExpectedKeywords) > 0 {
		return compactKeywords(item.ExpectedKeywords)
	}
	return deriveKeywordsFromAnswer(item.Answer)
}

func compactKeywords(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	keywords := make([]string, 0, len(input))
	for _, keyword := range input {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		if _, ok := seen[keyword]; ok {
			continue
		}
		seen[keyword] = struct{}{}
		keywords = append(keywords, keyword)
	}
	return keywords
}

func matchKeywords(response string, keywords []string) ([]string, []string) {
	matched := make([]string, 0, len(keywords))
	missing := make([]string, 0)
	for _, keyword := range keywords {
		if strings.Contains(response, keyword) {
			matched = append(matched, keyword)
		} else {
			missing = append(missing, keyword)
		}
	}
	return matched, missing
}

func keywordCoverage(matched, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(matched) / float64(total)
}

func previewRunes(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func deriveKeywordsFromAnswer(answer string) []string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil
	}

	candidates := make([]string, 0)
	numberPattern := regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?`)
	candidates = append(candidates, numberPattern.FindAllString(answer, -1)...)

	for _, part := range strings.FieldsFunc(answer, func(r rune) bool {
		return strings.ContainsRune("，。；、！？,.!?;:： \n\t\r（）()“”\"'《》", r)
	}) {
		part = strings.TrimSpace(part)
		if len([]rune(part)) >= 2 && len([]rune(part)) <= 32 {
			candidates = append(candidates, part)
		}
	}

	keywords := compactKeywords(candidates)
	if len(keywords) > 6 {
		keywords = keywords[:6]
	}
	return keywords
}

func chunkIDs(chunks []model.KnowledgeChunk) []string {
	ids := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		ids = append(ids, chunk.ID)
	}
	return ids
}

func retrievalMetrics(expectedIDs, retrievedIDs []string) (float64, float64, int) {
	if len(expectedIDs) == 0 {
		return 1, 1, 1
	}

	expected := make(map[string]struct{}, len(expectedIDs))
	for _, id := range expectedIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			expected[id] = struct{}{}
		}
	}
	if len(expected) == 0 {
		return 1, 1, 1
	}

	matched := 0
	firstRank := 0
	seen := make(map[string]struct{})
	for index, id := range retrievedIDs {
		if _, ok := expected[id]; !ok {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		matched++
		if firstRank == 0 {
			firstRank = index + 1
		}
	}

	recall := float64(matched) / float64(len(expected))
	reciprocalRank := 0.0
	if firstRank > 0 {
		reciprocalRank = 1 / float64(firstRank)
	}
	return recall, reciprocalRank, firstRank
}

func retrievalPassed(result RAGEvaluationResult) bool {
	return len(result.ExpectedChunkIDs) == 0 || result.RecallAtK > 0
}

func joinChunkContent(chunks []model.KnowledgeChunk) string {
	var builder strings.Builder
	for _, chunk := range chunks {
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(chunk.Title)
		builder.WriteString("\n")
		builder.WriteString(chunk.Content)
	}
	return builder.String()
}

func percentileDuration(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	if percentile <= 0 {
		return sorted[0]
	}
	if percentile >= 1 {
		return sorted[len(sorted)-1]
	}
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func durationMillisCeil(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	ms := duration.Milliseconds()
	if duration%time.Millisecond != 0 {
		ms++
	}
	return ms
}
