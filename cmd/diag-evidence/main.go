// 检索证据诊断工具：对比「走 LLM 查询改写」与「跳过改写」两种检索路径，
// 打印 retrievalText、命中 chunk、evidence 分数，定位拒答根因。
// 用法: go run ./cmd/diag-evidence -query "九龙灌浴是什么"
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
)

func main() {
	query := flag.String("query", "九龙灌浴是什么", "查询文本")
	flag.Parse()

	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		fmt.Println("配置加载失败:", err)
		os.Exit(1)
	}
	if err := pkg.InitDatabase(&cfg.Database); err != nil {
		fmt.Println("DB 失败:", err)
		os.Exit(1)
	}

	scenicID := os.Getenv("SCENIC_GUIDE_SCENIC_ID")
	if scenicID == "" {
		scenicID = "lingshan"
	}
	profile, err := config.LoadScenicProfile(scenicID)
	if err != nil {
		fmt.Println("景区配置失败:", err)
		os.Exit(1)
	}
	knowledgeRepo := repository.NewKnowledgeRepository(pkg.GetDB())
	rag := service.NewRAGService(knowledgeRepo, cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL, nil, profile)
	for _, f := range []string{"./knowledge/lingshan_chunks.jsonl", "./knowledge/real/lingshan_real_chunks.jsonl"} {
		if e := rag.LoadKnowledgeFromFile(f); e != nil {
			fmt.Println("加载知识库失败", f, e)
		}
	}

	fmt.Printf("\n===== query = %q  chatAPIKey存在=%v =====\n", *query, cfg.AI.APIKey != "")

	for _, skip := range []bool{false, true} {
		label := "走LLM改写"
		if skip {
			label = "跳过改写(本地扩展)"
		}
		t0 := time.Now()
		chunks, err := rag.RetrieveRelevantKnowledgeWithOptions(*query, service.RetrievalOptions{
			TopK:                 8,
			Mode:                 service.RetrievalModeBM25Local,
			SkipModelEnhancement: skip,
		})
		dur := time.Since(t0)
		fmt.Printf("\n----- [%s] 耗时 %v, 命中 %d 条 -----\n", label, dur, len(chunks))
		if err != nil {
			fmt.Println("  错误:", err)
			continue
		}
		// 用原始 query 和 "retrievalQuery+query" 两种口径算 evidence，对比拒答阈值 0.24
		conf1, abst1 := service.CalculateChunkEvidencePublic(*query, chunks)
		conf2, abst2 := service.CalculateChunkEvidencePublic(*query+" "+*query, chunks)
		fmt.Printf("  evidence(原始query): conf=%.3f abstain=%v\n", conf1, abst1)
		fmt.Printf("  evidence(query+query): conf=%.3f abstain=%v  [阈值0.24]\n", conf2, abst2)
		for i, c := range chunks {
			if i >= 3 {
				break
			}
			title := c.Title
			if len([]rune(title)) > 24 {
				title = string([]rune(title)[:24])
			}
			preview := c.Content
			if len([]rune(preview)) > 60 {
				preview = string([]rune(preview)[:60])
			}
			fmt.Printf("  [%d] id=%s title=%s\n      preview=%s...\n", i, c.ID, title, preview)
		}
	}
}
