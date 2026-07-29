// Package main 实现 cmd/rag-audit：针对真实游客问法评测集，逐题产出
// "问题 -> 完整本地回答 -> Top-K 引用知识库切片原文" 的可审核对照材料。
//
// 与 cmd/rag-eval 的区别：rag-eval 面向指标评估，并可选择保留完整回答；
// 本工具还会保留每条引用切片原文和覆盖统计，方便人工审核检索准确性与回答质量。
//
// 全程离线：默认使用本地 BM25 检索 + 本地规则生成，不读取任何 API Key，
// 不依赖外部 LLM / Embedding 服务，可完整复现。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

// auditCase 与 knowledge/real 下的灵山评测 JSON 结构对齐。
type auditCase struct {
	Question         string   `json:"question"`
	Category         string   `json:"category"`
	Difficulty       string   `json:"difficulty"`
	SourceType       string   `json:"source_type"`
	ExpectedKeywords []string `json:"expected_keywords"`
	ExpectedChunkIDs []string `json:"expected_chunk_ids"`
}

// citedChunk 是一条引用切片的完整原文，用于审核时核对内容。
type citedChunk struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	Source            string `json:"source"`
	KnowledgeCategory string `json:"knowledge_category,omitempty"`
	SpotID            uint   `json:"spot_id,omitempty"`
	SpotCategory      string `json:"spot_category,omitempty"`
	Content           string `json:"content"`
	IsExpected        bool   `json:"is_expected"` // 是否属于该题的 expected_chunk_ids
}

// auditResult 是单题的审核记录。
type auditResult struct {
	Index             int          `json:"index"`
	Question          string       `json:"question"`
	Category          string       `json:"category"`
	Difficulty        string       `json:"difficulty"`
	SourceType        string       `json:"source_type"`
	ExpectedKeywords  []string     `json:"expected_keywords"`
	ExpectedChunkIDs  []string     `json:"expected_chunk_ids"`
	Answer            string       `json:"answer"`
	AnswerMode        string       `json:"answer_mode"` // local-rule / configured-llm
	ShouldAbstain     bool         `json:"should_abstain"`
	Confidence        float64      `json:"confidence"`
	RetrievalMode     string       `json:"retrieval_mode"`
	RetrievedChunkIDs []string     `json:"retrieved_chunk_ids"`
	CitedChunks       []citedChunk `json:"cited_chunks"`
	RetrievalLatency  int64        `json:"retrieval_latency_ms"`
	TotalLatency      int64        `json:"total_latency_ms"`
	MatchedKeywords   []string     `json:"matched_keywords"`
	MissingKeywords   []string     `json:"missing_keywords"`
	RecallHit         bool         `json:"recall_hit"` // Top-K 是否命中任一 expected chunk
	FirstRelevantRank int          `json:"first_relevant_rank"`
	LLMError          string       `json:"llm_error,omitempty"`
}

type auditReport struct {
	KnowledgeFile   string        `json:"knowledge_file"`
	EvalFile        string        `json:"eval_file"`
	TopK            int           `json:"top_k"`
	GenerationMode  string        `json:"generation_mode"`
	GenerationModel string        `json:"generation_model,omitempty"`
	Total           int           `json:"total"`
	RecallHitCount  int           `json:"recall_hit_count"`
	RecallRate      float64       `json:"recall_rate"`
	StartedAt       time.Time     `json:"started_at"`
	FinishedAt      time.Time     `json:"finished_at"`
	ChunkCoverage   chunkCoverage `json:"chunk_coverage"`
	GroupStats      []groupStat   `json:"group_stats"`
	Results         []auditResult `json:"results"`
}

type chunkCoverage struct {
	TotalChunks       int      `json:"total_chunks"`
	CoveredChunks     int      `json:"covered_chunks"`
	CoverageRate      float64  `json:"coverage_rate"`
	UncoveredChunkIDs []string `json:"uncovered_chunk_ids"`
}

type groupStat struct {
	GroupBy    string  `json:"group_by"`
	Name       string  `json:"name"`
	Total      int     `json:"total"`
	RecallHit  int     `json:"recall_hit"`
	RecallRate float64 `json:"recall_rate"`
	KeywordAvg float64 `json:"keyword_coverage_avg"`
}

var registerDriver sync.Once

func main() {
	knowledgeFile := flag.String("knowledge", "knowledge/real/lingshan_real_chunks.jsonl", "知识库 JSONL 文件")
	evalFile := flag.String("eval", "knowledge/real/lingshan_eval_50.json", "评测问答文件")
	topK := flag.Int("k", service.TopK, "检索 Top-K（决定每题引用切片数量上限）")
	configDir := flag.String("config", "configs", "配置目录（用于加载景区 profile；configured 模式读取 AI 配置）")
	generationMode := flag.String("generation-mode", string(service.EvaluationGenerationModeLocal), "生成模式：local（本地规则，不读 Key）或 configured（调用真实 LLM）")
	jsonOut := flag.String("json", "", "JSON 审核报告输出路径（留空则不输出）")
	mdOut := flag.String("md", "", "Markdown 审核报告输出路径（留空则不输出）")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	mode := service.EvaluationGenerationMode(strings.TrimSpace(*generationMode))
	if mode != service.EvaluationGenerationModeLocal && mode != service.EvaluationGenerationModeConfigured {
		fmt.Fprintf(os.Stderr, "不支持的 generation-mode: %s（可选 local / configured）\n", *generationMode)
		os.Exit(1)
	}

	rag, genModel, err := buildRAGService(*knowledgeFile, *configDir, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 RAG 服务失败: %v\n", err)
		os.Exit(1)
	}
	if mode == service.EvaluationGenerationModeConfigured && genModel == "" {
		fmt.Fprintf(os.Stderr, "configured 模式未检测到有效 LLM 配置：请设置 SCENIC_GUIDE_AI_API_KEY / SCENIC_GUIDE_AI_MODEL / SCENIC_GUIDE_AI_BASE_URL\n")
		os.Exit(1)
	}

	cases, err := loadCases(*evalFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取评测文件失败: %v\n", err)
		os.Exit(1)
	}

	report := runAudit(rag, cases, *knowledgeFile, *evalFile, *topK, string(mode), genModel)

	if *jsonOut != "" {
		if err := writeJSON(*jsonOut, report); err != nil {
			fmt.Fprintf(os.Stderr, "写入 JSON 报告失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "JSON 审核报告已写入: %s\n", *jsonOut)
	}
	if *mdOut != "" {
		if err := writeMarkdown(*mdOut, report); err != nil {
			fmt.Fprintf(os.Stderr, "写入 Markdown 报告失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Markdown 审核报告已写入: %s\n", *mdOut)
	}

	// 控制台摘要
	fmt.Fprintf(os.Stderr, "\n=== 审核摘要 ===\n")
	fmt.Fprintf(os.Stderr, "总题数: %d\n", report.Total)
	fmt.Fprintf(os.Stderr, "Top-%d 命中 expected chunk 的题数: %d (%.1f%%)\n", *topK, report.RecallHitCount, report.RecallRate*100)
	fmt.Fprintf(os.Stderr, "知识库切片总数: %d, 被引用切片: %d (%.1f%%)\n",
		report.ChunkCoverage.TotalChunks, report.ChunkCoverage.CoveredChunks, report.ChunkCoverage.CoverageRate*100)
	if len(report.ChunkCoverage.UncoveredChunkIDs) > 0 {
		fmt.Fprintf(os.Stderr, "未被任何题目引用的切片 (%d 条):\n  - %s\n",
			len(report.ChunkCoverage.UncoveredChunkIDs), strings.Join(report.ChunkCoverage.UncoveredChunkIDs, "\n  - "))
	}
}

// buildRAGService 装配 RAG 服务。
//   - local 模式：不读任何 Key，检索用 BM25，生成用本地规则（可完整离线复现）。
//   - configured 模式：从环境变量读取 LLM 配置并注入，生成走真实外部模型；
//     检索仍用 BM25（不接 Embedding，避免引入语义检索变量）。
//
// 返回的 genModel 用于报告标注；configured 模式但未配置 Key 时返回空模型名，由调用方拦截。
func buildRAGService(knowledgeFile, configDir string, mode service.EvaluationGenerationMode) (*service.RAGService, string, error) {
	const driverName = "modernc-rag-audit"
	registerDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:rag-audit?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		return nil, "", fmt.Errorf("初始化临时数据库失败: %w", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		return nil, "", fmt.Errorf("迁移临时数据库失败: %w", err)
	}

	// 加载景区 profile（导游口吻 prompt），失败不阻断。
	scenicID := strings.TrimSpace(os.Getenv("SCENIC_GUIDE_SCENIC_ID"))
	if scenicID == "" {
		scenicID = "lingshan"
	}
	profile, perr := config.LoadScenicProfile(scenicID)
	if perr != nil {
		slog.Debug("rag-audit 未加载景区 profile，继续使用通用检索", "scenic_id", scenicID, "error", perr)
		profile = nil
	}

	chatAPIKey := ""
	chatModel := ""
	chatBaseURL := ""
	if mode == service.EvaluationGenerationModeConfigured {
		// 优先从环境变量读取（与项目 SCENIC_GUIDE_ 前缀约定一致）。
		chatAPIKey = strings.TrimSpace(os.Getenv("SCENIC_GUIDE_AI_API_KEY"))
		chatModel = strings.TrimSpace(os.Getenv("SCENIC_GUIDE_AI_MODEL"))
		chatBaseURL = strings.TrimSpace(os.Getenv("SCENIC_GUIDE_AI_BASE_URL"))
		// 环境变量缺失时回退到配置文件（config.yaml 不进版本库，可能存在于本机）。
		if chatAPIKey == "" || chatModel == "" || chatBaseURL == "" {
			if cfg, cerr := config.LoadConfig(configDir); cerr == nil && cfg != nil {
				if chatAPIKey == "" {
					chatAPIKey = strings.TrimSpace(cfg.AI.APIKey)
				}
				if chatModel == "" {
					chatModel = strings.TrimSpace(cfg.AI.Model)
				}
				if chatBaseURL == "" {
					chatBaseURL = strings.TrimSpace(cfg.AI.BaseURL)
				}
			}
		}
	}

	rag := service.NewRAGService(repository.NewKnowledgeRepository(db), chatAPIKey, chatModel, chatBaseURL, nil, profile)
	if err := rag.LoadKnowledgeFromFile(knowledgeFile); err != nil {
		return nil, "", fmt.Errorf("加载知识库失败: %w", err)
	}
	return rag, chatModel, nil
}

func loadCases(evalFile string) ([]auditCase, error) {
	data, err := os.ReadFile(evalFile)
	if err != nil {
		return nil, err
	}
	var cases []auditCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, err
	}
	return cases, nil
}

func runAudit(rag *service.RAGService, cases []auditCase, knowledgeFile, evalFile string, topK int, mode, genModel string) auditReport {
	report := auditReport{
		KnowledgeFile:   knowledgeFile,
		EvalFile:        evalFile,
		TopK:            topK,
		GenerationMode:  mode,
		GenerationModel: genModel,
		Total:           len(cases),
		StartedAt:       time.Now(),
	}
	results := make([]auditResult, 0, len(cases))
	citedChunkSet := make(map[string]bool)
	expectedIDsAgg := make(map[string]bool)
	recallHitCount := 0
	groupAgg := make(map[string]map[string]*groupAggState) // groupBy -> name -> state

	for i, c := range cases {
		// configured 模式调用真实 LLM，超时放宽到 60s；local 模式本地生成极快，10s 足够。
		timeout := 10 * time.Second
		if mode == string(service.EvaluationGenerationModeConfigured) {
			timeout = 60 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)

		// 检索：始终用纯 BM25（SkipModelEnhancement），保证两种生成模式下检索结果一致，
		// 这样 configured 报告与 local 报告可以逐题对照，独立审核"生成质量"而非"检索改写"。
		retrievalStart := time.Now()
		retrieved, rerr := rag.RetrieveRelevantKnowledgeWithOptions(c.Question, service.RetrievalOptions{
			TopK:                 topK,
			SkipModelEnhancement: true,
		})
		retrievalMs := time.Since(retrievalStart).Milliseconds()

		// 生成
		answer := ""
		answerMode := "local-rule"
		shouldAbstain := false
		confidence := 0.0
		retrievalMode := "bm25-local"
		llmErr := ""
		totalStart := time.Now()
		if rerr != nil {
			answer = fmt.Sprintf("[检索失败: %v]", rerr)
		} else {
			confidence, shouldAbstain = service.CalculateChunkEvidencePublic(c.Question, retrieved)
			if len(retrieved) == 0 {
				answer = "[未检索到相关知识]"
			} else if mode == string(service.EvaluationGenerationModeConfigured) && rag.HasConfiguredLLM() {
				prompt := rag.BuildRAGPromptWithContext(c.Question, retrieved, "")
				out, gerr := rag.CallLLM(ctx, rag.GetSystemPromptOrDefaultPublic(""), prompt)
				if gerr != nil {
					llmErr = gerr.Error()
					// LLM 失败时回退到本地生成，便于审核者看到兜底内容
					out = rag.LocalGeneratePublic(c.Question, retrieved)
					answerMode = "local-rule(fallback)"
				} else {
					answerMode = "configured-llm"
				}
				answer = out
			} else {
				answer = rag.LocalGeneratePublic(c.Question, retrieved)
			}
		}
		cancel()
		totalMs := time.Since(totalStart).Milliseconds()

		expectedSet := make(map[string]bool, len(c.ExpectedChunkIDs))
		for _, id := range c.ExpectedChunkIDs {
			expectedSet[id] = true
			expectedIDsAgg[id] = true
		}

		cited := make([]citedChunk, 0, len(retrieved))
		retrievedIDs := make([]string, 0, len(retrieved))
		firstRelevant := 0
		recallHit := false
		for rank, ch := range retrieved {
			isExp := expectedSet[ch.ID]
			if isExp {
				recallHit = true
				if firstRelevant == 0 {
					firstRelevant = rank + 1
				}
			}
			citedChunkSet[ch.ID] = true
			retrievedIDs = append(retrievedIDs, ch.ID)
			cited = append(cited, citedChunk{
				ID:                ch.ID,
				Title:             ch.Title,
				Source:            ch.Source,
				KnowledgeCategory: ch.KnowledgeCategory,
				SpotID:            ch.SpotID,
				SpotCategory:      ch.SpotCategory,
				Content:           ch.Content,
				IsExpected:        isExp,
			})
		}

		matched, missing := matchKeywords(answer, c.ExpectedKeywords)

		res := auditResult{
			Index:             i + 1,
			Question:          c.Question,
			Category:          c.Category,
			Difficulty:        c.Difficulty,
			SourceType:        c.SourceType,
			ExpectedKeywords:  c.ExpectedKeywords,
			ExpectedChunkIDs:  c.ExpectedChunkIDs,
			Answer:            answer,
			AnswerMode:        answerMode,
			ShouldAbstain:     shouldAbstain,
			Confidence:        confidence,
			RetrievalMode:     retrievalMode,
			RetrievedChunkIDs: retrievedIDs,
			CitedChunks:       cited,
			RetrievalLatency:  retrievalMs,
			TotalLatency:      totalMs,
			MatchedKeywords:   matched,
			MissingKeywords:   missing,
			RecallHit:         recallHit,
			FirstRelevantRank: firstRelevant,
			LLMError:          llmErr,
		}
		results = append(results, res)
		if recallHit {
			recallHitCount++
		}
		accumulateGroup(groupAgg, "category", c.Category, recallHit, matched, c.ExpectedKeywords)
		accumulateGroup(groupAgg, "source_type", c.SourceType, recallHit, matched, c.ExpectedKeywords)
		accumulateGroup(groupAgg, "difficulty", c.Difficulty, recallHit, matched, c.ExpectedKeywords)
	}

	report.FinishedAt = time.Now()
	report.Results = results
	report.RecallHitCount = recallHitCount
	if report.Total > 0 {
		report.RecallRate = float64(recallHitCount) / float64(report.Total)
	}
	report.ChunkCoverage = computeCoverage(knowledgeFile, citedChunkSet)
	report.GroupStats = flattenGroupStats(groupAgg)
	return report
}

type groupAggState struct {
	Total      int
	RecallHit  int
	KeywordSum float64
}

func accumulateGroup(agg map[string]map[string]*groupAggState, groupBy, name string, recallHit bool, matched []string, expected []string) {
	if name == "" {
		name = "(未分类)"
	}
	byName, ok := agg[groupBy]
	if !ok {
		byName = make(map[string]*groupAggState)
		agg[groupBy] = byName
	}
	st, ok := byName[name]
	if !ok {
		st = &groupAggState{}
		byName[name] = st
	}
	st.Total++
	if recallHit {
		st.RecallHit++
	}
	if len(expected) > 0 {
		st.KeywordSum += float64(len(matched)) / float64(len(expected))
	}
}

func flattenGroupStats(agg map[string]map[string]*groupAggState) []groupStat {
	out := make([]groupStat, 0)
	for groupBy, byName := range agg {
		names := make([]string, 0, len(byName))
		for name := range byName {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			st := byName[name]
			gs := groupStat{
				GroupBy:   groupBy,
				Name:      name,
				Total:     st.Total,
				RecallHit: st.RecallHit,
			}
			if st.Total > 0 {
				gs.RecallRate = float64(st.RecallHit) / float64(st.Total)
			}
			if st.Total > 0 {
				gs.KeywordAvg = st.KeywordSum / float64(st.Total)
			}
			out = append(out, gs)
		}
	}
	return out
}

// computeCoverage 统计知识库中有多少切片被至少一道题引用。
// 分母直接从 JSONL 文件读取所有 chunk ID，保证与知识库实际内容一致。
func computeCoverage(knowledgeFile string, citedSet map[string]bool) chunkCoverage {
	totalIDs := readAllChunkIDs(knowledgeFile)
	cov := chunkCoverage{TotalChunks: len(totalIDs)}
	covered := 0
	uncovered := make([]string, 0)
	for _, id := range totalIDs {
		if citedSet[id] {
			covered++
		} else {
			uncovered = append(uncovered, id)
		}
	}
	cov.CoveredChunks = covered
	cov.UncoveredChunkIDs = uncovered
	if cov.TotalChunks > 0 {
		cov.CoverageRate = float64(covered) / float64(cov.TotalChunks)
	}
	return cov
}

// readAllChunkIDs 直接从 JSONL 文件读取所有 chunk ID，用于覆盖率分母。
func readAllChunkIDs(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	ids := make([]string, 0)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.ID != "" {
			ids = append(ids, entry.ID)
		}
	}
	return ids
}

func matchKeywords(answer string, expected []string) (matched, missing []string) {
	for _, kw := range expected {
		if strings.Contains(answer, kw) {
			matched = append(matched, kw)
		} else {
			missing = append(missing, kw)
		}
	}
	return matched, missing
}

func writeJSON(path string, report auditReport) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeMarkdown(path string, report auditReport) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# RAG 问答审核报告\n\n")
	modeDesc := "本地 BM25 检索 + 本地规则生成（不调用任何外部 LLM / API Key，可离线复现）"
	if report.GenerationMode == string(service.EvaluationGenerationModeConfigured) {
		modeDesc = fmt.Sprintf("本地 BM25 检索 + 真实 LLM 生成（模型: %s）", report.GenerationModel)
	}
	fmt.Fprintf(&b, "> 本报告由 `cmd/rag-audit` 生成，生成模式: **%s**\n", modeDesc)
	fmt.Fprintf(&b, "> 用于人工审核「问题 → 完整回答 → 引用知识库切片」三者是否对应。\n\n")
	fmt.Fprintf(&b, "- 评测题集: `%s`\n", report.EvalFile)
	fmt.Fprintf(&b, "- 知识库: `%s`\n", report.KnowledgeFile)
	fmt.Fprintf(&b, "- 检索 Top-K: %d\n", report.TopK)
	fmt.Fprintf(&b, "- 生成时间: %s ~ %s\n", report.StartedAt.Format("2006-01-02 15:04:05"), report.FinishedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- 题目总数: %d\n", report.Total)
	fmt.Fprintf(&b, "- Top-%d 命中 expected chunk: %d / %d (%.1f%%)\n", report.TopK, report.RecallHitCount, report.Total, report.RecallRate*100)
	fmt.Fprintf(&b, "- 知识库切片总数: %d，被引用: %d (%.1f%%)\n",
		report.ChunkCoverage.TotalChunks, report.ChunkCoverage.CoveredChunks, report.ChunkCoverage.CoverageRate*100)
	if len(report.ChunkCoverage.UncoveredChunkIDs) > 0 {
		fmt.Fprintf(&b, "\n**未被任何题目引用的切片** (%d 条)：\n\n", len(report.ChunkCoverage.UncoveredChunkIDs))
		for _, id := range report.ChunkCoverage.UncoveredChunkIDs {
			fmt.Fprintf(&b, "- `%s`\n", id)
		}
	}

	// 分组统计
	byGroup := make(map[string][]groupStat)
	for _, gs := range report.GroupStats {
		byGroup[gs.GroupBy] = append(byGroup[gs.GroupBy], gs)
	}
	for _, groupBy := range []string{"category", "source_type", "difficulty"} {
		stats := byGroup[groupBy]
		if len(stats) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## 按 %s 分组\n\n", groupBy)
		b.WriteString("| 分组 | 题数 | 命中 expected | 命中率 | 关键词平均覆盖 |\n")
		b.WriteString("|---|---|---|---|---|\n")
		for _, gs := range stats {
			fmt.Fprintf(&b, "| %s | %d | %d | %.1f%% | %.1f%% |\n",
				gs.Name, gs.Total, gs.RecallHit, gs.RecallRate*100, gs.KeywordAvg*100)
		}
	}

	fmt.Fprintf(&b, "\n---\n\n## 逐题审核（共 %d 题）\n\n", report.Total)
	for _, r := range report.Results {
		writeQuestionMarkdown(&b, report.TopK, r)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeQuestionMarkdown(b *strings.Builder, topK int, r auditResult) {
	fmt.Fprintf(b, "### Q%d. %s\n\n", r.Index, r.Question)
	fmt.Fprintf(b, "- **类别**: %s | **难度**: %s | **来源**: %s\n", r.Category, r.Difficulty, r.SourceType)
	hitMark := "❌ 未命中"
	if r.RecallHit {
		hitMark = fmt.Sprintf("✅ 命中（首位相关 rank=%d）", r.FirstRelevantRank)
	}
	fmt.Fprintf(b, "- **检索命中**: %s | **检索耗时**: %dms | **总耗时**: %dms | **检索模式**: %s\n",
		hitMark, r.RetrievalLatency, r.TotalLatency, r.RetrievalMode)
	if len(r.MatchedKeywords) > 0 || len(r.MissingKeywords) > 0 {
		fmt.Fprintf(b, "- **关键词覆盖**: 命中 %s", sliceJoin(r.MatchedKeywords))
		if len(r.MissingKeywords) > 0 {
			fmt.Fprintf(b, "；缺失 %s", sliceJoin(r.MissingKeywords))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "- **期望引用 chunk**: `%s`\n", strings.Join(r.ExpectedChunkIDs, "`, `"))
	fmt.Fprintf(b, "- **拒答标志**: should_abstain=%v, confidence=%.3f\n", r.ShouldAbstain, r.Confidence)
	fmt.Fprintf(b, "\n**【回答】**\n\n```\n%s\n```\n\n", r.Answer)
	fmt.Fprintf(b, "**【Top-%d 引用知识库切片】**\n\n", topK)
	for i, ch := range r.CitedChunks {
		mark := "  "
		if ch.IsExpected {
			mark = "✅"
		}
		fmt.Fprintf(b, "%s **%d. `%s`** — %s （来源: %s",
			mark, i+1, ch.ID, ch.Title, ch.Source)
		if ch.KnowledgeCategory != "" {
			fmt.Fprintf(b, " / 类别: %s", ch.KnowledgeCategory)
		}
		if ch.SpotCategory != "" {
			fmt.Fprintf(b, " / 景点分类: %s", ch.SpotCategory)
		}
		b.WriteString("）\n\n")
		fmt.Fprintf(b, "```\n%s\n```\n\n", ch.Content)
	}
	b.WriteString("---\n\n")
}

func sliceJoin(s []string) string {
	if len(s) == 0 {
		return "(无)"
	}
	return strings.Join(s, "、")
}
