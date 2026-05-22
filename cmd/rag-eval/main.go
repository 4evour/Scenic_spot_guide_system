package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
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

func main() {
	knowledgeFile := flag.String("knowledge", "knowledge/lingshan_chunks.jsonl", "知识库 JSONL 文件")
	evalFile := flag.String("eval", "knowledge/lingshan_eval_qa.json", "RAG 评测问答文件")
	format := flag.String("format", "text", "输出格式：text 或 json")
	failOnMiss := flag.Bool("fail-on-miss", false, "存在未通过用例时返回非零退出码")
	topK := flag.Int("k", service.TopK, "检索 TopK")
	bench := flag.Bool("bench", false, "输出容量/延迟基准报告")
	concurrency := flag.Int("concurrency", 1, "bench 并发数")
	repeat := flag.Int("repeat", 1, "bench 每条评测重复次数")
	retrievalOnly := flag.Bool("retrieval-only", false, "只评估检索与本地片段，不调用外部生成")
	useEmbedding := flag.Bool("embedding", false, "启用配置中的 Embedding Provider 参与检索评估")
	configDir := flag.String("config", "configs", "配置目录，用于 -embedding 或端到端评估")
	reportEnv := flag.Bool("report-env", false, "在报告中记录运行环境、数据集和评估口径")
	outFile := flag.String("out", "", "将 JSON 报告写入指定文件")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	report, err := runEvaluation(*knowledgeFile, *evalFile, evaluationRunOptions{
		topK:          *topK,
		bench:         *bench,
		concurrency:   *concurrency,
		repeat:        *repeat,
		retrievalOnly: *retrievalOnly,
		useEmbedding:  *useEmbedding,
		configDir:     *configDir,
		reportEnv:     *reportEnv,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "RAG 评估失败: %v\n", err)
		os.Exit(1)
	}

	switch strings.ToLower(*format) {
	case "json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化评估报告失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	default:
		printTextReport(report)
	}

	if *outFile != "" {
		if err := writeJSONReport(*outFile, report); err != nil {
			fmt.Fprintf(os.Stderr, "写入评估报告失败: %v\n", err)
			os.Exit(1)
		}
	}

	if *failOnMiss && !report.IsPassing() {
		os.Exit(2)
	}
}

type evaluationRunOptions struct {
	topK          int
	bench         bool
	concurrency   int
	repeat        int
	retrievalOnly bool
	useEmbedding  bool
	configDir     string
	reportEnv     bool
}

var registerRAGEvalDriver sync.Once

func runEvaluation(knowledgeFile, evalFile string, options evaluationRunOptions) (*service.RAGEvaluationReport, error) {
	const driverName = "modernc-rag-eval"
	registerRAGEvalDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:rag-eval?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("初始化临时数据库失败: %w", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("迁移临时数据库失败: %w", err)
	}

	ragConfig, embeddingProvider, err := buildEvaluationProviders(options)
	if err != nil {
		return nil, err
	}

	chatAPIKey := ""
	chatModel := ""
	chatBaseURL := ""
	if !options.retrievalOnly && ragConfig != nil {
		chatAPIKey = ragConfig.AI.APIKey
		chatModel = ragConfig.AI.Model
		chatBaseURL = ragConfig.AI.BaseURL
	}

	rag := service.NewRAGService(repository.NewKnowledgeRepository(db), chatAPIKey, chatModel, chatBaseURL, embeddingProvider)
	if err := rag.LoadKnowledgeFromFile(knowledgeFile); err != nil {
		return nil, err
	}
	chunkCount := countJSONLLines(knowledgeFile)
	var report *service.RAGEvaluationReport
	if !options.bench {
		report, err = rag.EvaluateFileWithOptions(evalFile, service.EvaluationOptions{
			TopK:          options.topK,
			RetrievalOnly: options.retrievalOnly,
		})
	} else {
		report, err = runBench(rag, evalFile, options)
	}
	if err != nil {
		return nil, err
	}
	attachRunInfo(report, knowledgeFile, evalFile, chunkCount, providerName(embeddingProvider), generationProviderName(ragConfig, options), options)
	return report, nil
}

func buildEvaluationProviders(options evaluationRunOptions) (*config.Config, service.EmbeddingProvider, error) {
	if !options.useEmbedding && options.retrievalOnly {
		return nil, nil, nil
	}

	cfg, err := config.LoadConfig(options.configDir)
	if err != nil {
		if options.useEmbedding || !options.retrievalOnly {
			return nil, nil, fmt.Errorf("读取配置失败: %w", err)
		}
		return nil, nil, nil
	}

	var embeddingProvider service.EmbeddingProvider
	if options.useEmbedding {
		provider := service.NewQwenEmbeddingProvider(&cfg.Embedding)
		if !provider.IsAvailable() {
			return nil, nil, fmt.Errorf("已启用 -embedding，但配置中缺少 embedding.api_key")
		}
		embeddingProvider = provider
	}

	return cfg, embeddingProvider, nil
}

func providerName(provider service.EmbeddingProvider) string {
	if provider == nil || !provider.IsAvailable() {
		return "bm25-local"
	}
	return provider.Name()
}

func generationProviderName(cfg *config.Config, options evaluationRunOptions) string {
	if options.retrievalOnly || cfg == nil || strings.TrimSpace(cfg.AI.APIKey) == "" {
		return "disabled"
	}
	if strings.TrimSpace(cfg.AI.Model) != "" {
		return cfg.AI.Model
	}
	return "chat-api"
}

func printTextReport(report *service.RAGEvaluationReport) {
	fmt.Println("========== RAG 评估报告 ==========")
	fmt.Printf("用例总数: %d\n", report.Total)
	fmt.Printf("TopK: %d\n", report.TopK)
	fmt.Printf("通过用例: %d\n", report.Passed)
	fmt.Printf("未通过用例: %d\n", report.Failed)
	fmt.Printf("通过率: %.1f%%\n", report.PassRate*100)
	fmt.Printf("关键词平均覆盖率: %.1f%%\n", report.AverageKeywordCoverage*100)
	fmt.Printf("Recall@K: %.1f%%\n", report.AverageRecallAtK*100)
	fmt.Printf("MRR@K: %.3f\n", report.MRRAtK)
	fmt.Printf("检索延迟 p50/p95: %dms / %dms\n\n", report.RetrievalP50Ms, report.RetrievalP95Ms)
	if report.RunInfo.KnowledgeFile != "" {
		fmt.Printf("数据集: %s\n", report.RunInfo.KnowledgeFile)
		fmt.Printf("评测集: %s\n", report.RunInfo.EvaluationFile)
		fmt.Printf("运行环境: %s/%s %s CPU=%s 并发=%d repeat=%d retrieval-only=%t\n\n",
			report.RunInfo.OS,
			report.RunInfo.Arch,
			report.RunInfo.GoVersion,
			report.RunInfo.CPU,
			report.RunInfo.Concurrency,
			report.RunInfo.Repeat,
			report.RunInfo.RetrievalOnly,
		)
	}
	if len(report.GroupStats) > 0 {
		fmt.Println("分组统计:")
		for _, stat := range report.GroupStats {
			fmt.Printf("  %s=%s total=%d pass=%d fail=%d Recall@K=%.1f%% MRR@K=%.3f\n",
				stat.GroupBy, stat.Name, stat.Total, stat.Passed, stat.Failed, stat.AverageRecallAtK*100, stat.MRRAtK)
		}
		fmt.Println()
	}

	results := report.Results
	if report.Total > 50 {
		fmt.Println("大规模评估仅展示前 10 条失败样例；完整结果请使用 -format json。")
		results = failedResults(report.Results, 10)
	}

	for index, result := range results {
		status := "PASS"
		if !result.Passed {
			status = "MISS"
		}
		fmt.Printf("[%d/%d] %s %s\n", index+1, report.Total, status, result.Question)
		if result.Category != "" || result.Difficulty != "" {
			fmt.Printf("  分类/难度: %s / %s\n", result.Category, result.Difficulty)
		}
		if len(result.ExpectedChunkIDs) > 0 {
			fmt.Printf("  Recall@K: %.1f%% MRR: %.3f 命中排名: %d\n", result.RecallAtK*100, result.ReciprocalRank, result.FirstRelevantRank)
			fmt.Printf("  期望切片: %s\n", strings.Join(result.ExpectedChunkIDs, ", "))
			fmt.Printf("  检索切片: %s\n", strings.Join(result.RetrievedChunkIDs, ", "))
		}
		if len(result.MissingKeywords) > 0 {
			fmt.Printf("  缺失关键词: %s\n", strings.Join(result.MissingKeywords, ", "))
		}
		if result.Error != "" {
			fmt.Printf("  错误: %s\n", result.Error)
		}
		if result.ResponsePreview != "" {
			fmt.Printf("  回答预览: %s\n", result.ResponsePreview)
		}
	}
}

func attachRunInfo(report *service.RAGEvaluationReport, knowledgeFile, evalFile string, knowledgeChunks int, embeddingProvider, generationProvider string, options evaluationRunOptions) {
	concurrency := options.concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	repeat := options.repeat
	if repeat < 1 {
		repeat = 1
	}
	report.RunInfo = service.RAGEvaluationRunInfo{
		KnowledgeFile:      knowledgeFile,
		EvaluationFile:     evalFile,
		KnowledgeChunks:    knowledgeChunks,
		EvaluationCases:    report.Total,
		TopK:               options.topK,
		Concurrency:        concurrency,
		Repeat:             repeat,
		RetrievalOnly:      options.retrievalOnly,
		EmbeddingProvider:  embeddingProvider,
		GenerationProvider: generationProvider,
	}
	if options.reportEnv {
		report.RunInfo.OS = runtime.GOOS
		report.RunInfo.Arch = runtime.GOARCH
		report.RunInfo.CPU = runtime.GOARCH
		report.RunInfo.GoVersion = runtime.Version()
	}
}

func countJSONLLines(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	return count
}

func writeJSONReport(outputPath string, report *service.RAGEvaluationReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, append(data, '\n'), 0o644)
}

func runBench(rag *service.RAGService, evalFile string, options evaluationRunOptions) (*service.RAGEvaluationReport, error) {
	data, err := os.ReadFile(evalFile)
	if err != nil {
		return nil, fmt.Errorf("读取评估文件失败: %v", err)
	}
	var baseCases []service.RAGEvaluationCase
	if err := json.Unmarshal(data, &baseCases); err != nil {
		return nil, fmt.Errorf("解析评估文件失败: %v", err)
	}

	repeat := options.repeat
	if repeat < 1 {
		repeat = 1
	}
	cases := make([]service.RAGEvaluationCase, 0, len(baseCases)*repeat)
	for i := 0; i < repeat; i++ {
		cases = append(cases, baseCases...)
	}

	if len(baseCases) > 0 {
		_, _ = rag.RetrieveRelevantKnowledge(baseCases[0].Question, options.topK)
	}

	concurrency := options.concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency == 1 {
		return rag.EvaluateQuestionsWithOptions(cases, service.EvaluationOptions{
			TopK:          options.topK,
			RetrievalOnly: options.retrievalOnly,
		})
	}

	startedAt := time.Now()
	results := make([]service.RAGEvaluationResult, len(cases))
	jobs := make(chan int)
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				report, err := rag.EvaluateQuestionsWithOptions([]service.RAGEvaluationCase{cases[index]}, service.EvaluationOptions{
					TopK:          options.topK,
					RetrievalOnly: options.retrievalOnly,
				})
				if err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					errMu.Unlock()
					continue
				}
				if len(report.Results) == 1 {
					results[index] = report.Results[0]
				}
			}
		}()
	}

	for index := range cases {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	return summarizeResults(results, options.topK, options.retrievalOnly, startedAt), nil
}

func failedResults(results []service.RAGEvaluationResult, limit int) []service.RAGEvaluationResult {
	failed := make([]service.RAGEvaluationResult, 0, limit)
	for _, result := range results {
		if result.Passed {
			continue
		}
		failed = append(failed, result)
		if len(failed) >= limit {
			break
		}
	}
	return failed
}

func summarizeResults(results []service.RAGEvaluationResult, topK int, retrievalOnly bool, startedAt time.Time) *service.RAGEvaluationReport {
	report := &service.RAGEvaluationReport{
		Total:         len(results),
		TopK:          topK,
		RetrievalOnly: retrievalOnly,
		StartedAt:     startedAt,
		FinishedAt:    time.Now(),
		Results:       results,
	}

	var coverageSum float64
	var recallSum float64
	var mrrSum float64
	latencies := make([]time.Duration, 0, len(results))
	for _, result := range results {
		if result.Passed {
			report.Passed++
		}
		coverageSum += result.KeywordCoverage
		recallSum += result.RecallAtK
		mrrSum += result.ReciprocalRank
		latencies = append(latencies, time.Duration(result.RetrievalLatencyMs)*time.Millisecond)
	}
	report.Failed = report.Total - report.Passed
	if report.Total > 0 {
		report.PassRate = float64(report.Passed) / float64(report.Total)
		report.AverageKeywordCoverage = coverageSum / float64(report.Total)
		report.AverageRecallAtK = recallSum / float64(report.Total)
		report.MRRAtK = mrrSum / float64(report.Total)
		report.RetrievalP50Ms = percentileMs(latencies, 0.50)
		report.RetrievalP95Ms = percentileMs(latencies, 0.95)
	}
	report.GroupStats = service.SummarizeEvaluationGroups(results)
	return report
}

func percentileMs(values []time.Duration, percentile float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	index := int(float64(len(sorted))*percentile+0.999999) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index].Milliseconds()
}
