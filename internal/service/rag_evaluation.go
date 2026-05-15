package service

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

type RAGEvaluationCase struct {
	Question         string   `json:"question"`
	Answer           string   `json:"answer,omitempty"`
	Source           string   `json:"source,omitempty"`
	ExpectedKeywords []string `json:"expected_keywords,omitempty"`
}

type RAGEvaluationResult struct {
	Question        string   `json:"question"`
	Passed          bool     `json:"passed"`
	KeywordCoverage float64  `json:"keyword_coverage"`
	MatchedKeywords []string `json:"matched_keywords"`
	MissingKeywords []string `json:"missing_keywords"`
	ResponsePreview string   `json:"response_preview"`
	Error           string   `json:"error,omitempty"`
}

type RAGEvaluationReport struct {
	Total                  int                   `json:"total"`
	Passed                 int                   `json:"passed"`
	Failed                 int                   `json:"failed"`
	PassRate               float64               `json:"pass_rate"`
	AverageKeywordCoverage float64               `json:"average_keyword_coverage"`
	StartedAt              time.Time             `json:"started_at"`
	FinishedAt             time.Time             `json:"finished_at"`
	Results                []RAGEvaluationResult `json:"results"`
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
	report := &RAGEvaluationReport{
		Total:     len(cases),
		StartedAt: time.Now(),
		Results:   make([]RAGEvaluationResult, 0, len(cases)),
	}

	var coverageSum float64
	for _, item := range cases {
		keywords := normalizeEvaluationKeywords(item)
		result := RAGEvaluationResult{
			Question: item.Question,
		}

		response, err := s.QueryWithRAG(item.Question)
		if err != nil {
			result.Error = err.Error()
			result.MissingKeywords = keywords
			report.Results = append(report.Results, result)
			continue
		}

		result.ResponsePreview = previewRunes(response, 120)
		result.MatchedKeywords, result.MissingKeywords = matchKeywords(response, keywords)
		result.KeywordCoverage = keywordCoverage(len(result.MatchedKeywords), len(keywords))
		result.Passed = len(keywords) > 0 && len(result.MissingKeywords) == 0
		if result.Passed {
			report.Passed++
		}
		coverageSum += result.KeywordCoverage
		report.Results = append(report.Results, result)
	}

	report.Failed = report.Total - report.Passed
	if report.Total > 0 {
		report.PassRate = float64(report.Passed) / float64(report.Total)
		report.AverageKeywordCoverage = coverageSum / float64(report.Total)
	}
	report.FinishedAt = time.Now()
	return report, nil
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
