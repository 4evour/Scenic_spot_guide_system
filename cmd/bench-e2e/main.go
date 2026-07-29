// 端到端延迟基准：文本输入 → 检索 → LLM 流式首字(TTFT) → TTS 流式首音
// 绕过 HTTP 服务里 BM25 长驻索引的拒答 bug，直接复用已验证正常的检索路径。
//
// 用法:
//
//	go run ./cmd/bench-e2e -samples 20
//	go run ./cmd/bench-e2e -samples 30 -warmup 3
//
// 测量阶段(ms):
//
//	retrieval_ms        检索(BM25, 跳过 LLM 改写)
//	ttft_ms             模型首字时间 (Time-To-First-Token), 从调用 LLM 到第一个 token
//	generation_total_ms 模型生成完整回答耗时
//	tts_first_byte_ms   TTS 首字节时间, 从请求 TTS 到第一个音频字节
//	tts_total_ms        TTS 合成完整音频耗时
//	answer_chars        回答字符数
//	audio_bytes         音频字节数
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
)

var defaultQuestions = []string{
	"九龙灌浴是什么表演？",
	"五印坛城在哪里？",
	"阿育王柱有多高？",
	"灵山梵宫有什么特色？",
	"祥符禅寺是做什么的？",
	"佛足坛在什么位置？",
	"五智门有什么看点？",
	"带孩子适合怎么游览？",
	"景区有哪些文化景点？",
	"半天时间怎么安排？",
}

type sample struct {
	Question          string
	RetrievalMs       int64
	TTFTMs            int64 // 模型首字
	GenerationTotalMs int64
	TTSFirstByteMs    int64 // 首音
	TTSTotalMs        int64
	AnswerChars       int
	AudioBytes        int
	Err               string
}

func main() {
	samples := flag.Int("samples", 20, "每问题采样轮数(总样本 = questions × min(samples,len))")
	warmup := flag.Int("warmup", 2, "预热轮数(不计入统计)")
	voice := flag.String("voice", "female_xiaoxiao", "TTS 语音")
	rate := flag.String("rate", "+0%", "TTS 语速")
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
	tts := service.NewEdgeTTSService(30 * time.Second)
	systemPrompt := profile.GetSystemPrompt()

	fmt.Printf("模型=%s  语音=%s  systemPrompt_len=%d  样本/问=%d  预热=%d\n\n",
		cfg.AI.Model, *voice, len([]rune(systemPrompt)), *samples, *warmup)

	runOne := func(q string) sample {
		s := sample{Question: q}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		// 1) 检索 (跳过 LLM 改写: 既快又准)
		t0 := time.Now()
		chunks, err := rag.RetrieveRelevantKnowledgeWithOptions(q, service.RetrievalOptions{
			TopK: 8, Mode: service.RetrievalModeBM25Local, SkipModelEnhancement: true,
		})
		s.RetrievalMs = time.Since(t0).Milliseconds()
		if err != nil || len(chunks) == 0 {
			s.Err = fmt.Sprintf("检索失败: %v chunks=%d", err, len(chunks))
			return s
		}

		// 2) LLM 流式生成, 量首字时间
		prompt := rag.BuildRAGPromptWithContext(q, chunks, "")
		var answer strings.Builder
		firstToken := make(chan int64, 1)
		t1 := time.Now()
		var firstT int64
		_, err = rag.CallLLMStreaming(ctx, systemPrompt, prompt, func(tok string) {
			if answer.Len() == 0 {
				firstT = time.Since(t1).Milliseconds()
				select {
				case firstToken <- firstT:
				default:
				}
			}
			answer.WriteString(tok)
		})
		s.GenerationTotalMs = time.Since(t1).Milliseconds()
		select {
		case s.TTFTMs = <-firstToken:
		default:
			s.TTFTMs = firstT
		}
		if err != nil && answer.Len() == 0 {
			s.Err = fmt.Sprintf("LLM 生成失败: %v", err)
			return s
		}
		s.AnswerChars = len([]rune(answer.String()))

		// 3) TTS 流式, 量首字节
		ttsCtx, ttsCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer ttsCancel()
		// 取前 200 字做 TTS(完整长文本会分多段, 这里测首音延迟用前缀即可反映真实首音)
		ttsText := answer.String()
		if r := []rune(ttsText); len(r) > 200 {
			ttsText = string(r[:200])
		}
		t2 := time.Now()
		ch, errCh := tts.SynthesizeStream(ttsCtx, ttsText, *voice, *rate)
		var firstByte int64
		gotByte := false
		for chunk := range ch {
			if !gotByte {
				firstByte = time.Since(t2).Milliseconds()
				gotByte = true
			}
			s.AudioBytes += len(chunk)
		}
		if e := <-errCh; e != nil && !gotByte {
			s.Err = fmt.Sprintf("TTS 失败: %v", e)
			return s
		}
		s.TTSFirstByteMs = firstByte
		s.TTSTotalMs = time.Since(t2).Milliseconds()
		return s
	}

	// 预热
	for w := 0; w < *warmup; w++ {
		runOne(defaultQuestions[w%len(defaultQuestions)])
	}
	fmt.Println("预热完成, 开始正式采样...")

	var results []sample
	totalQ := *samples
	if totalQ > len(defaultQuestions)*3 {
		totalQ = len(defaultQuestions) * 3
	}
	for i := 0; i < totalQ; i++ {
		q := defaultQuestions[i%len(defaultQuestions)]
		s := runOne(q)
		results = append(results, s)
		status := "OK"
		if s.Err != "" {
			status = "ERR: " + s.Err
		}
		fmt.Printf("[%2d] %-18s | 检索 %4dms | TTFT %4dms | 生成 %5dms | 首音 %4dms | TTS总 %5dms | 答%4d字 | %s\n",
			i+1, trunc(q, 16), s.RetrievalMs, s.TTFTMs, s.GenerationTotalMs, s.TTSFirstByteMs, s.TTSTotalMs, s.AnswerChars, status)
		time.Sleep(300 * time.Millisecond)
	}

	printStats(results)
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func printStats(results []sample) {
	ok := []sample{}
	for _, s := range results {
		if s.Err == "" && s.AnswerChars > 0 {
			ok = append(ok, s)
		}
	}
	fmt.Println("\n================ 端到端延迟统计 ================")
	fmt.Printf("样本: 总 %d, 成功 %d, 失败 %d\n", len(results), len(ok), len(results)-len(ok))
	if len(ok) == 0 {
		fmt.Println("无成功样本")
		return
	}
	p := func(name string, get func(s sample) int64) {
		vals := []int64{}
		for _, s := range ok {
			v := get(s)
			if v > 0 {
				vals = append(vals, v)
			}
		}
		if len(vals) == 0 {
			fmt.Printf("  %-22s 无数据\n", name)
			return
		}
		sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
		sum := int64(0)
		for _, v := range vals {
			sum += v
		}
		avg := sum / int64(len(vals))
		p50 := vals[(len(vals)-1)*50/100]
		p95 := vals[(len(vals)-1)*95/100]
		fmt.Printf("  %-22s avg=%5dms  p50=%5dms  p95=%5dms  max=%5dms  (n=%d)\n", name, avg, p50, p95, vals[len(vals)-1], len(vals))
	}
	p("检索(BM25)", func(s sample) int64 { return s.RetrievalMs })
	p("模型首字 TTFT", func(s sample) int64 { return s.TTFTMs })
	p("模型生成总耗时", func(s sample) int64 { return s.GenerationTotalMs })
	p("TTS 首音(首字节)", func(s sample) int64 { return s.TTSFirstByteMs })
	p("TTS 合成总耗时", func(s sample) int64 { return s.TTSTotalMs })

	// 端到端 = 检索 + TTFT(到首字) 或 检索+生成(到拿到完整回答)
	e2eFirst := []int64{}
	e2eFull := []int64{}
	for _, s := range ok {
		if s.TTFTMs > 0 {
			e2eFirst = append(e2eFirst, s.RetrievalMs+s.TTFTMs)
		}
		e2eFull = append(e2eFull, s.RetrievalMs+s.GenerationTotalMs)
	}
	sumLine := func(name string, vals []int64) {
		if len(vals) == 0 {
			return
		}
		sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
		sum := int64(0)
		for _, v := range vals {
			sum += v
		}
		fmt.Printf("  %-22s avg=%5dms  p50=%5dms  p95=%5dms  max=%5dms  (n=%d)\n",
			name, sum/int64(len(vals)), vals[(len(vals)-1)*50/100], vals[(len(vals)-1)*95/100], vals[len(vals)-1], len(vals))
	}
	fmt.Println("  -------- 复合指标 --------")
	sumLine("输入→首字(检索+TTFT)", e2eFirst)
	sumLine("输入→完整回答(检索+生成)", e2eFull)
	fmt.Println("  注: 输入→首音 = 输入→完整回答 + TTS首音(因当前非流式, 需等回答全部生成完才发起TTS)")
	fmt.Println("  注: ASR(语音转文字)在浏览器侧, 用 webkitSpeechRecognition, 本程序无法测量, 见报告说明")
}
