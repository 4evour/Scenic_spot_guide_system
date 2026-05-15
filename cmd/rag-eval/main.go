package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

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
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	report, err := runEvaluation(*knowledgeFile, *evalFile)
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

	if *failOnMiss && !report.IsPassing() {
		os.Exit(2)
	}
}

func runEvaluation(knowledgeFile, evalFile string) (*service.RAGEvaluationReport, error) {
	const driverName = "modernc-rag-eval"
	sql.Register(driverName, &sqlite3.Driver{})

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

	rag := service.NewRAGService(repository.NewKnowledgeRepository(db), "", "", "", nil)
	if err := rag.LoadKnowledgeFromFile(knowledgeFile); err != nil {
		return nil, err
	}
	return rag.EvaluateFile(evalFile)
}

func printTextReport(report *service.RAGEvaluationReport) {
	fmt.Println("========== RAG 评估报告 ==========")
	fmt.Printf("用例总数: %d\n", report.Total)
	fmt.Printf("通过用例: %d\n", report.Passed)
	fmt.Printf("未通过用例: %d\n", report.Failed)
	fmt.Printf("通过率: %.1f%%\n", report.PassRate*100)
	fmt.Printf("关键词平均覆盖率: %.1f%%\n\n", report.AverageKeywordCoverage*100)

	for index, result := range report.Results {
		status := "PASS"
		if !result.Passed {
			status = "MISS"
		}
		fmt.Printf("[%d/%d] %s %s\n", index+1, report.Total, status, result.Question)
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
