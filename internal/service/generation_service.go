package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
)

// OpenAI 兼容 API 类型定义（供多处复用）
type openAIRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

type openAIResponse struct {
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *openAIError `json:"error"`
}

func (s *RAGService) BuildRAGPrompt(query string, chunks []model.KnowledgeChunk) string {
	return s.BuildRAGPromptWithContext(query, chunks, "")
}

func (s *RAGService) BuildRAGPromptWithContext(query string, chunks []model.KnowledgeChunk, sessionContext string) string {
	if len(chunks) == 0 {
		return ""
	}

	var context strings.Builder
	for i, chunk := range chunks {
		context.WriteString(fmt.Sprintf("%d. 【%s】\n%s\n来源：%s\n\n", i+1, chunk.Title, chunk.Content, chunk.Source))
	}

	var conversation strings.Builder
	if strings.TrimSpace(sessionContext) != "" {
		conversation.WriteString("【当前会话上下文】\n")
		conversation.WriteString(sessionContext)
		conversation.WriteString("\n- 回答时先自然承接上一轮主题，再回答游客当前问题。\n")
		conversation.WriteString("- 如果涉及票价、开放、演出、客流、排队、无人机、宠物等实时或现场规则，必须说明不能直接承诺，以官方最新公告或现场公示为准。\n\n")
	}

	answerStyleGuard := `【回答要求】
- 直接回答游客当前问题，不要先解释你检索到了什么。
- 知识库资料里如果出现“游客常问”“回答策略”“问答素材”“资料来源”等元说明，只把它们当内部参考，不要复述给游客。
- 用真人导游口吻组织成自然讲解，避免罗列知识库原文。
`

	// 使用 profile 配置的 prompt 模板（支持任意景区）
	if s.profile != nil && s.profile.Prompts.RAGPrompt != "" {
		prompt := s.profile.RenderPrompt(s.profile.Prompts.RAGPrompt)
		prompt = strings.ReplaceAll(prompt, "{knowledge_context}", context.String())
		prompt = strings.ReplaceAll(prompt, "{session_context}", conversation.String())
		prompt = strings.ReplaceAll(prompt, "{query}", query)
		prompt += "\n\n" + answerStyleGuard
		if guidance := EmotionGuidance(query); guidance != "" {
			prompt += "\n\n【游客沟通状态】\n" + guidance
		}

		// 注入个性化路线推荐（当查询匹配路线关键词时）
		routeHint := s.buildRouteRecommendation(query)
		if routeHint != "" {
			prompt += "\n\n【路线推荐参考】\n" + routeHint + "\n请根据游客的兴趣自然地推荐合适的路线。"
		}
		return prompt
	}

	// fallback: 无 profile 时使用通用模板
	prompt := fmt.Sprintf(`你是一位专业的景区数字人导览员，负责为游客提供导览服务。

【知识库资料】
%s

%s
【游客问题】
%s

请基于以上资料回答：`, context.String(), conversation.String(), query)
	prompt += "\n\n" + answerStyleGuard
	if guidance := EmotionGuidance(query); guidance != "" {
		prompt += "\n\n【游客沟通状态】\n" + guidance
	}
	return prompt
}

func (s *RAGService) QueryWithRAG(ctx context.Context, query string) (string, error) {
	answer, _, err := s.QueryWithRAGTrace(ctx, query, "")
	return answer, err
}

func (s *RAGService) QueryWithRAGTrace(ctx context.Context, query, lang string) (answer string, trace RAGTrace, err error) {
	return s.queryWithRAGTraceInternal(ctx, query, query, "", lang)
}

func (s *RAGService) buildCasualAnswer(query, lang string) (string, bool) {
	intent, ok := DetectCasualIntent(query)
	if !ok {
		return "", false
	}

	if strings.TrimSpace(lang) == "en-US" {
		switch intent {
		case "greeting":
			return "Hello! I am Xiaoling. I can introduce scenic spots, plan routes, or simply chat with you.", true
		case "thanks":
			return "You are welcome. Enjoy your visit! Is there anything else you would like to know?", true
		case "farewell":
			return "Goodbye, and have a pleasant trip!", true
		case "identity":
			return "I am Xiaoling, the scenic-area digital guide. I can answer questions about attractions, routes, culture, and visiting tips.", true
		case "complaint":
			return "It sounds like this has been frustrating. Tell me what happened and I will help you sort it out. For on-site issues, please contact the scenic-area service center.", true
		case "anxiety":
			return "Please do not worry. I can help you check the route and visiting tips step by step. For on-site assistance, please contact staff.", true
		case "chat":
			return "Of course. Would you like to talk about the scenic area, your travel plans, or anything else?", true
		}
	}

	switch intent {
	case "greeting":
		return "你好，我是小灵，可以帮你介绍景点、规划路线，也可以陪你聊聊。", true
	case "thanks":
		return "不客气，祝你游玩愉快。还想了解哪个景点呢？", true
	case "farewell":
		return "再见，祝你旅途愉快！", true
	case "identity":
		return "我是景区数字人导览助手小灵，可以回答景点、路线、文化和游览注意事项。", true
	case "complaint":
		return "听起来这件事让你有些不舒服。你可以告诉我具体遇到的问题，我帮你梳理；如果涉及现场处理，也可以联系景区服务中心。", true
	case "anxiety":
		return "别着急，我可以一步一步帮你确认路线和注意事项；如果需要现场协助，建议联系景区工作人员。", true
	case "chat":
		return "当然可以。你想聊景区、旅行计划，还是随便聊聊？", true
	default:
		return "", false
	}
}

func (s *RAGService) queryWithRAGTraceInternal(ctx context.Context, retrievalQuery, promptQuery, sessionContext, lang string) (answer string, trace RAGTrace, err error) {
	retrievalQuery = strings.TrimSpace(retrievalQuery)
	promptQuery = strings.TrimSpace(promptQuery)
	if promptQuery == "" {
		promptQuery = retrievalQuery
	}
	totalStart := time.Now()
	trace = RAGTrace{
		TraceID:       fmt.Sprintf("rag-%d", time.Now().UnixNano()),
		Provider:      map[bool]string{true: "bm25-local", false: "embedding"}[s.useBM25],
		RetrievalMode: string(normalizeRetrievalMode(RetrievalModeDefault, s.embedding != nil && s.embedding.IsAvailable(), s.useBM25)),
	}
	if retrievalQuery != promptQuery {
		trace.RewrittenQuery = retrievalQuery
	}
	defer func() {
		trace.TotalMs = time.Since(totalStart).Milliseconds()
		trace.SlowRequest = trace.TotalMs > SlowRequestThresholdMs
		pkg.RecordRAGQueryDuration(time.Since(totalStart).Seconds())
		if trace.CacheHit {
			pkg.RecordRAGCacheHit()
		}
		logAttrs := []any{
			"trace_id", trace.TraceID,
			"provider", trace.Provider,
			"retrieval_mode", trace.RetrievalMode,
			"cache_hit", trace.CacheHit,
			"chunk_count", trace.ChunkCount,
			"retrieval_ms", trace.RetrievalMs,
			"embedding_ms", trace.EmbeddingMs,
			"generation_ms", trace.GenerationMs,
			"total_ms", trace.TotalMs,
			"query_len", len([]rune(promptQuery)),
		}
		if trace.RewrittenQuery != "" {
			logAttrs = append(logAttrs, "rewritten_query_len", len([]rune(trace.RewrittenQuery)))
		}
		if trace.SlowRequest {
			slog.Warn("RAG 查询慢请求", logAttrs...)
		} else {
			slog.Info("RAG 查询完成", logAttrs...)
		}
	}()

	if casualAnswer, ok := s.buildCasualAnswer(promptQuery, lang); ok {
		trace.Provider = "local-conversation"
		trace.RetrievalMode = "conversation"
		trace.Confidence = 0.95
		trace.ShouldAbstain = false
		generationStart := time.Now()
		answer = casualAnswer
		trace.GenerationMs = time.Since(generationStart).Milliseconds()
		return answer, trace, nil
	}

	cacheKey := retrievalQuery
	if sessionContext != "" {
		cacheKey = "prompt:" + promptQuery + "\nretrieval:" + retrievalQuery + "\nctx:" + sessionContext
	}
	if cached, ok := s.getCachedResponse(cacheKey); ok {
		slog.Debug("RAG 查询命中缓存", "query_len", len([]rune(promptQuery)))
		trace.CacheHit = true
		trace.Sources = cached.Sources
		if cached.EvidenceEvaluated {
			trace.Confidence = cached.Confidence
			trace.ShouldAbstain = cached.ShouldAbstain
		} else {
			trace.Confidence, trace.ShouldAbstain = calculateAnswerEvidence(retrievalQuery+" "+promptQuery, trace.Sources)
		}
		trace.TotalMs = time.Since(totalStart).Milliseconds()
		trace.SlowRequest = trace.TotalMs > SlowRequestThresholdMs
		if trace.ShouldAbstain && !isBoundaryIntent(promptQuery) {
			return addEmotionCare(promptQuery, s.buildNoEvidenceAnswer(lang)), trace, nil
		}
		return cached.Response, trace, nil
	}

	retrievalStart := time.Now()
	chunks, err := s.RetrieveRelevantKnowledge(retrievalQuery, TopK)
	trace.RetrievalMs = time.Since(retrievalStart).Milliseconds()
	trace.ChunkCount = len(chunks)
	trace.Sources = buildRAGSources(chunks, 3)
	trace.Confidence, trace.ShouldAbstain = calculateChunkEvidence(retrievalQuery+" "+promptQuery, chunks)
	if err != nil {
		return "", trace, fmt.Errorf("检索相关知识失败: %v", err)
	}

	if len(chunks) == 0 {
		slog.Info("RAG 未检索到相关知识，拒绝无依据生成", "query_len", len([]rune(promptQuery)))
		generationStart := time.Now()
		answer := addEmotionCare(promptQuery, s.buildNoEvidenceAnswer(lang))
		trace.GenerationMs = time.Since(generationStart).Milliseconds()
		return answer, trace, nil
	}
	if trace.ShouldAbstain && !isBoundaryIntent(promptQuery) {
		generationStart := time.Now()
		answer := addEmotionCare(promptQuery, s.buildNoEvidenceAnswer(lang))
		trace.GenerationMs = time.Since(generationStart).Milliseconds()
		return answer, trace, nil
	}

	slog.Info("RAG 检索命中知识库", "query_len", len([]rune(promptQuery)), "chunks", len(chunks), "mode", map[bool]string{true: "bm25", false: "embedding"}[s.useBM25])

	if s.chatAPIKey == "" {
		generationStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs = time.Since(generationStart).Milliseconds()
		s.setCachedRAGResponse(cacheKey, answer, trace.Sources, trace.Confidence, trace.ShouldAbstain)
		return answer, trace, nil
	}

	prompt := s.BuildRAGPromptWithContext(promptQuery, chunks, sessionContext)

	req := openAIRequest{
		Model: s.chatModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: s.getSystemPromptOrDefault(lang),
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		slog.Error("RAG 请求序列化失败", "error", err)
		return "", trace, fmt.Errorf("RAG Chat API 请求序列化失败: %w", err)
	}

	apiURL := s.chatBaseURL + "/chat/completions"

	modelStart := time.Now()
	modelResult, modelErr, _ := s.modelRequests.Do(cacheKey, func() (interface{}, error) {
		var responseBody []byte
		err := s.chatGuard.run(ctx, func(callCtx context.Context) error {
			request, err := http.NewRequestWithContext(callCtx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
			if err != nil {
				return err
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+s.chatAPIKey)
			resp, err := s.httpClient.Do(request)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				responseBody, _ := readLimitedBody(resp.Body)
				slog.Warn("RAG Chat API 返回非 200", "status", resp.StatusCode, "body_len", len(responseBody))
				return &modelHTTPError{status: resp.StatusCode}
			}
			var readErr error
			responseBody, readErr = io.ReadAll(resp.Body)
			return readErr
		})
		return responseBody, err
	})
	trace.GenerationMs = time.Since(modelStart).Milliseconds()
	if modelErr != nil {
		slog.Warn("RAG Chat API 调用失败，使用本地知识库降级", "error", modelErr)
		trace.Provider = "local-rag-fallback"
		fallbackStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs += time.Since(fallbackStart).Milliseconds()
		s.setCachedRAGResponse(cacheKey, answer, trace.Sources, trace.Confidence, trace.ShouldAbstain)
		return answer, trace, nil
	}
	body, ok := modelResult.([]byte)
	if !ok {
		return "", trace, fmt.Errorf("RAG Chat API 响应结果类型错误")
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		slog.Error("RAG 解析 Chat API 响应失败", "error", err)
		return "", trace, fmt.Errorf("RAG 解析 Chat API 响应失败: %w", err)
	}

	if openAIResp.Error != nil {
		slog.Warn("RAG Chat API 返回业务错误",
			"type", openAIResp.Error.Type,
			"code", openAIResp.Error.Code,
			"message", openAIResp.Error.Message,
		)
		return "", trace, fmt.Errorf("RAG Chat API 返回业务错误: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) > 0 && openAIResp.Choices[0].Message.Content != "" {
		answer := openAIResp.Choices[0].Message.Content
		s.setCachedRAGResponse(cacheKey, answer, trace.Sources, trace.Confidence, trace.ShouldAbstain)
		return answer, trace, nil
	}

	return "", trace, fmt.Errorf("RAG Chat API 未返回有效回答")
}

// openAIStreamChunk represents a single chunk from the OpenAI streaming API.
type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// CallLLMStreaming calls the LLM API with stream=true and invokes onToken for each token.
func (s *RAGService) CallLLMStreaming(ctx context.Context, systemPrompt, userPrompt string, onToken func(string)) (string, error) {
	reqBody := map[string]interface{}{
		"model":  s.chatModel,
		"stream": true,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("stream request marshal failed: %w", err)
	}

	apiURL := s.chatBaseURL + "/chat/completions"
	return s.chatGuard.runStreaming(ctx, func(callCtx context.Context) (string, bool, error) {
		request, err := http.NewRequestWithContext(callCtx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", false, err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+s.chatAPIKey)
		resp, err := s.httpClient.Do(request)
		if err != nil {
			return "", false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			_, _ = readLimitedBody(resp.Body)
			return "", false, &modelHTTPError{status: resp.StatusCode}
		}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var fullResponse strings.Builder
		emitted := false
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 {
				token := chunk.Choices[0].Delta.Content
				if token != "" {
					emitted = true
					fullResponse.WriteString(token)
					if onToken != nil {
						onToken(token)
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return fullResponse.String(), emitted, fmt.Errorf("stream response scan failed: %w", err)
		}
		return fullResponse.String(), emitted, nil
	})
}

const maxAPIResponseBytes = 20 << 20 // 20MB

func readLimitedBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, maxAPIResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return data, err
	}
	if int64(len(data)) > maxAPIResponseBytes {
		return data[:maxAPIResponseBytes], fmt.Errorf("response body exceeded %d bytes", maxAPIResponseBytes)
	}
	return data, nil
}

func (s *RAGService) generateAnswerFromChunks(query string, chunks []model.KnowledgeChunk) string {
	return s.generateAnswerFromChunksWithContext(query, chunks, "")
}

func (s *RAGService) generateAnswerFromChunksWithContext(query string, chunks []model.KnowledgeChunk, sessionContext string) string {
	var content strings.Builder
	for _, chunk := range chunks {
		content.WriteString(chunk.Content)
		content.WriteString("\n\n")
	}

	fullContent := content.String()
	intentText := query + "\n" + sessionContext

	carryBoundary := isFollowUpQuery(query) && strings.Contains(sessionContext, "边界状态：涉及实时信息")
	if isBoundaryIntent(query) || carryBoundary {
		answer := fmt.Sprintf("当前资料不足，无法确认%s，也不能直接替您确认或承诺。请以景区官方最新公告、官方渠道的实时查询结果或现场公示为准。", boundarySubject(query))
		if snippet := boundaryEvidenceSnippet(query, chunks); snippet != "" {
			answer += "\n\n参考边界说明：" + snippet
		}
		return finalizeLocalAnswer(query, answer)
	}

	if isRouteIntent(intentText) {
		snippets := s.extractRelevantSnippets(query+" 路线 游览 雨天 亲子 老人 休息点", chunks, 4)
		if len(snippets) == 0 {
			snippets = splitKnowledgeSentences(fullContent)
		}
		answer := "可以按这个思路安排：\n\n" + formatNumberedLines(snippets, 4)
		if containsAny(intentText, []string{"下雨", "雨天", "天气", "高温"}) {
			answer += "\n\n如果现场降雨、高温或排队变化明显，建议优先选择室内点和休息点，并按景区现场指引调整。"
		}
		return finalizeLocalAnswer(query, answer)
	}

	snippetQuery := focusedSnippetQuery(query)
	snippetLimit := 4
	snippets := s.extractRelevantSnippets(snippetQuery, chunks, snippetLimit)
	if isFocusedFactIntent(query) && len(snippets) > 1 && firstFactSnippetCoversAllDimensions(query, snippets) {
		snippets = snippets[:1]
	}
	if len(snippets) == 0 {
		// 只有检索句子无法生成时才使用 profile 兜底，避免通用短答案覆盖更具体的证据。
		if s.profile != nil {
			if fallbackAnswer, ok := s.profile.GetFallbackAnswer(query); ok {
				return finalizeLocalAnswer(query, fallbackAnswer)
			}
		}
		snippets = []string{previewRunes(fullContent, 500)}
	}

	scenicName := "景区"
	if s.profile != nil {
		scenicName = s.profile.Name
	}
	answer := fmt.Sprintf("根据%s景区资料：\n\n", scenicName) + strings.Join(snippets, "\n\n")
	return finalizeLocalAnswer(query, answer)
}

func finalizeLocalAnswer(query, answer string) string {
	return previewRunes(addEmotionCare(query, answer), 697)
}

func isFocusedFactIntent(query string) bool {
	return containsAny(query, []string{"哪个", "哪一个", "是什么", "多少", "多高", "位于哪里", "地址", "占比最高", "优先了解"})
}

func focusedSnippetQuery(query string) string {
	switch {
	case containsAny(query, []string{"多高", "高度"}):
		return query + " 高度 通高 米"
	case containsAny(query, []string{"位于哪里", "在哪里", "位置"}):
		return query + " 位于 坐落 地址 省 市 区域 镇"
	default:
		return query
	}
}

func firstFactSnippetCoversAllDimensions(query string, snippets []string) bool {
	if len(snippets) == 0 {
		return false
	}
	for _, dimension := range factIntentDimensions(query) {
		available := false
		for _, snippet := range snippets {
			if sentenceHasFactDimension(query, snippet, dimension) {
				available = true
				break
			}
		}
		if available && !sentenceHasFactDimension(query, snippets[0], dimension) {
			return false
		}
	}
	return len(factIntentDimensions(query)) > 0
}

func boundarySubject(query string) string {
	switch {
	case containsAny(query, []string{"门票", "票价", "优惠"}):
		return "今天的票价或优惠信息"
	case containsAny(query, []string{"酒店空房", "还有多少间", "房态", "剩余房间", "客房库存"}):
		return "今晚的酒店空房或客房库存"
	case containsAny(query, []string{"停车", "车位"}):
		return "当前停车余位"
	case containsAny(query, []string{"演出", "场次"}):
		return "今天的演出场次"
	case containsAny(query, []string{"几点", "开放", "开不开"}):
		return "今天的开放时间或开放状态"
	default:
		return "这个实时、库存或现场运营信息"
	}
}

func addEmotionCare(query, answer string) string {
	return ApplyVisitorEmotionCare(DetectVisitorEmotion(query), answer)
}

func (s *RAGService) buildNoEvidenceAnswer(lang string) string {
	if strings.TrimSpace(lang) == "en-US" {
		return "I could not find reliable information about this question in the current scenic-area materials. Please check the latest official notice or ask the service center."
	}
	return "当前景区资料中没有找到足够依据回答这个问题。为避免提供不准确的信息，请以景区官方最新公告、购票页面或现场工作人员答复为准。"
}

func isRouteIntent(text string) bool {
	return containsAny(text, []string{"路线", "半天", "怎么走", "先看", "下雨", "雨天", "天气", "高温", "带孩子", "亲子", "小朋友", "老人", "长辈"})
}

func formatNumberedLines(lines []string, limit int) string {
	lines = compactKeywords(lines)
	if len(lines) == 0 {
		return "1. 根据当前资料，建议先咨询景区服务中心确认更合适的安排。"
	}
	var b strings.Builder
	for i := 0; i < min(limit, len(lines)); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, line))
	}
	return strings.TrimSpace(b.String())
}

func (s *RAGService) extractRelevantSnippets(query string, chunks []model.KnowledgeChunk, limit int) []string {
	type scoredSnippet struct {
		text       string
		score      float64
		structural bool
		chunkIndex int
	}

	snippets := make([]scoredSnippet, 0)
	seen := make(map[string]struct{})
	for chunkIndex, chunk := range chunks {
		for _, sentence := range splitKnowledgeSentences(chunk.Content) {
			if len([]rune(sentence)) < 8 {
				continue
			}
			if isKnowledgeMetaSentence(sentence) {
				continue
			}
			score := s.BM25Similarity(query, sentence)
			for _, cb := range s.getConditionalBoosts() {
				score += conditionalTermBoost(query, sentence, cb.QueryTerms, cb.ContentTerms)
			}
			score += min(s.BM25Similarity(query, chunk.Title), 5)
			score += factIntentSentenceBoost(query, sentence)
			if score <= 0 {
				continue
			}
			if _, ok := seen[sentence]; ok {
				continue
			}
			seen[sentence] = struct{}{}
			snippets = append(snippets, scoredSnippet{
				text:       sentence,
				score:      score,
				structural: isStructuralKnowledgeSentence(query, sentence),
				chunkIndex: chunkIndex,
			})
		}
	}

	sort.Slice(snippets, func(i, j int) bool {
		return snippets[i].score > snippets[j].score
	})
	result := make([]string, 0, limit)
	selected := make(map[string]struct{}, limit)
	appendSnippet := func(snippet scoredSnippet) bool {
		if _, ok := selected[snippet.text]; ok {
			return false
		}
		selected[snippet.text] = struct{}{}
		result = append(result, snippet.text)
		return len(result) == limit
	}
	preferredChunkIndex := s.preferredDetailedChunkIndex(query, chunks)
	if preferredChunkIndex >= 0 && containsAny(query, []string{"年份", "时间线", "工程参数", "组成", "元素", "文化定位", "适合", "工艺", "吉祥颂", "无障碍", "轮椅", "母婴", "太湖山水", "佛教文化结合", "拍照", "礼仪", "博览馆", "万佛殿", "哪类佛教文化"}) {
		preferred := make([]string, 0, 6)
		for _, sentence := range splitKnowledgeSentences(chunks[preferredChunkIndex].Content) {
			if len([]rune(sentence)) >= 8 && !isKnowledgeMetaSentence(sentence) {
				preferred = append(preferred, sentence)
			}
		}
		if len(preferred) > 0 {
			return preferred
		}
	}
	for _, dimension := range factIntentDimensions(query) {
		foundInPreferredChunk := false
		if preferredChunkIndex >= 0 {
			for _, snippet := range snippets {
				if snippet.structural || snippet.chunkIndex != preferredChunkIndex || !sentenceHasFactDimension(query, snippet.text, dimension) {
					continue
				}
				foundInPreferredChunk = true
				if appendSnippet(snippet) {
					return result
				}
				break
			}
		}
		if foundInPreferredChunk {
			continue
		}
		for _, snippet := range snippets {
			if snippet.structural || !sentenceHasFactDimension(query, snippet.text, dimension) {
				continue
			}
			if appendSnippet(snippet) {
				return result
			}
			break
		}
	}
	if preferredChunkIndex >= 0 {
		for _, snippet := range snippets {
			if snippet.structural || snippet.chunkIndex != preferredChunkIndex {
				continue
			}
			if appendSnippet(snippet) {
				return result
			}
		}
	}
	if requiresComplementaryChunkEvidence(query) {
		for chunkIndex := 0; chunkIndex < min(2, len(chunks)); chunkIndex++ {
			for _, snippet := range snippets {
				if snippet.structural || snippet.chunkIndex != chunkIndex {
					continue
				}
				if appendSnippet(snippet) {
					return result
				}
				break
			}
		}
	}
	for _, snippet := range snippets {
		if snippet.structural {
			continue
		}
		if appendSnippet(snippet) {
			return result
		}
	}
	for _, snippet := range snippets {
		if !snippet.structural {
			continue
		}
		if appendSnippet(snippet) {
			break
		}
	}
	return result
}

func (s *RAGService) preferredDetailedChunkIndex(query string, chunks []model.KnowledgeChunk) int {
	if !containsAny(query, []string{"主要表现", "什么内容", "哪类", "文化", "年份", "时间线", "工程参数", "组成", "元素", "历史出处", "意象", "适合", "工艺", "吉祥颂", "无障碍", "轮椅", "母婴", "太湖山水", "佛教文化结合", "拍照", "礼仪", "博览馆", "万佛殿"}) {
		return -1
	}

	preferredTitles := []string{"详细介绍"}
	switch {
	case containsAny(query, []string{"年份", "时间线", "奠基", "建设"}):
		preferredTitles = []string{"时间线", "历史沿革"}
	case containsAny(query, []string{"工程参数", "参数", "用铜量", "登云道"}):
		preferredTitles = []string{"工程参数"}
	case containsAny(query, []string{"组成", "元素"}):
		preferredTitles = []string{"组成元素", "详细介绍"}
	case containsAny(query, []string{"历史出处", "千年古刹"}):
		preferredTitles = []string{"历史出处", "历史沿革"}
	case containsAny(query, []string{"意象", "空间"}):
		preferredTitles = []string{"空间意象", "文化意象"}
	case containsAny(query, []string{"博览馆", "万佛殿"}):
		preferredTitles = []string{"佛教文化博览馆与万佛殿", "详细介绍"}
	case containsAny(query, []string{"体现了哪类佛教文化", "体现哪类佛教文化", "哪类佛教文化"}):
		for i, chunk := range chunks {
			if strings.Contains(chunk.Title, "详细介绍") {
				return i
			}
		}
		preferredTitles = []string{"详细介绍", "文化建筑", "文化主题", "文化定位", "文化内涵", "文化意象"}
	case containsAny(query, []string{"适合", "文化定位", "什么文化", "哪类"}):
		preferredTitles = []string{"文化建筑", "文化主题", "文化定位", "文化内涵", "文化意象", "详细介绍"}
	case containsAny(query, []string{"工艺", "吉祥颂"}):
		preferredTitles = []string{"艺术工艺", "文化建筑", "详细介绍"}
	case containsAny(query, []string{"无障碍", "轮椅", "母婴"}):
		preferredTitles = []string{"无障碍", "服务边界", "服务设施"}
	case containsAny(query, []string{"太湖山水", "佛教文化结合"}):
		preferredTitles = []string{"太湖山水和佛教文化结合", "文化结合", "空间关系", "景区概况"}
	case containsAny(query, []string{"拍照", "礼仪"}):
		preferredTitles = []string{"拍照礼仪边界", "拍照提示"}
	}
	if containsAny(query, []string{"主要表现", "什么内容"}) {
		preferredTitles = []string{"文化定位", "文化内涵", "概览"}
	}
	bestIndex := -1
	bestScore := 0.0
	for i, chunk := range chunks {
		if !containsAny(chunk.Title, preferredTitles) {
			continue
		}
		score := s.BM25Similarity(query, chunk.Title)
		if score > bestScore {
			bestIndex = i
			bestScore = score
		}
	}
	return bestIndex
}

func boundaryEvidenceSnippet(query string, chunks []model.KnowledgeChunk) string {
	if containsAny(query, []string{"消费", "样本", "营收"}) && len(chunks) > 0 {
		return strings.Join(splitKnowledgeSentences(chunks[0].Content), " ")
	}
	markers := []string{"不能确认", "资料不足", "官方最新公告", "现场公示", "现场管理", "现场指引", "现场咨询", "实时运营信息", "宠物入园", "无人机拍摄", "现场规定", "仅描述公开资料包样本", "不代表景区当前实时营收"}
	for _, chunk := range chunks {
		var selected []string
		for _, sentence := range splitKnowledgeSentences(chunk.Content) {
			if containsAny(sentence, markers) {
				selected = append(selected, strings.TrimSpace(sentence))
			}
			if len(selected) == 2 {
				break
			}
		}
		if len(selected) > 0 {
			return strings.Join(selected, " ")
		}
	}
	return ""
}

func requiresComplementaryChunkEvidence(query string) bool {
	return containsAny(query, []string{"为什么", "为何", "原因"})
}

func isStructuralKnowledgeSentence(query, sentence string) bool {
	trimmed := strings.TrimSpace(sentence)
	if strings.HasPrefix(trimmed, "景点名称：") || strings.HasPrefix(trimmed, "备注：") {
		return true
	}
	if strings.HasPrefix(trimmed, "具体位置：") && !containsAny(query, []string{"位于哪里", "在哪里", "位置", "地址"}) {
		return true
	}
	return len([]rune(trimmed)) <= 32 && strings.Contains(trimmed, "：") && !containsAny(trimmed, []string{"，", "、", "（", "("})
}

func factIntentSentenceBoost(query, sentence string) float64 {
	boost := 0.0

	switch {
	case containsAny(query, []string{"多高", "高度", "多少米", "多长", "多重"}):
		if hasMeasurement(sentence) {
			boost += 8
		}
		if containsAny(sentence, []string{"通高", "总高", "佛体", "高度", "长", "重"}) {
			boost += 3
		}
	case containsAny(query, []string{"为什么", "为何", "原因"}):
		hasExplanation := containsAny(sentence, []string{"凭借", "因为", "由于", "因而", "采用", "运用", "汇集", "融合", "结合", "形成", "体现"})
		hasCraft := containsAny(sentence, []string{"工艺", "技艺", "艺术", "制作", "雕刻", "烧制", "镶嵌", "材料", "传统"})
		if hasExplanation {
			boost += 6
		}
		if hasCraft {
			boost += 6
		}
		if hasExplanation && hasCraft {
			boost += 6
		}
	case containsAny(query, []string{"主要表现", "什么内容", "哪类", "文化"}):
		hasSubject := containsAny(sentence, []string{"文化", "宗教", "礼仪", "仪式", "传统", "主题", "主体"})
		hasExplanation := isCultureExplanationSentence(sentence)
		if hasSubject {
			boost += 8
		}
		if hasExplanation {
			boost += 8
		}
		if hasSubject && hasExplanation {
			boost += 6
		}
	}

	if strings.HasPrefix(strings.TrimSpace(sentence), "具体位置：") {
		boost -= 2
	}
	return boost
}

func factIntentDimensions(query string) []string {
	switch {
	case containsAny(query, []string{"多高", "高度", "多少米", "多长", "多重"}):
		return []string{"overall_measurement", "body_measurement", "total_measurement", "measurement"}
	case containsAny(query, []string{"位于哪里", "在哪里", "位置", "地址"}):
		return []string{"administrative_location", "region_location", "locality_location"}
	case containsAny(query, []string{"为什么", "为何", "原因"}):
		return []string{"craft_subject", "craft_explanation"}
	case containsAny(query, []string{"主要表现", "什么内容", "哪类", "文化"}):
		return []string{"culture_subject", "culture_explanation", "concrete_examples", "ritual_subject", "culture_practice"}
	default:
		return nil
	}
}

func sentenceHasFactDimension(query, sentence, dimension string) bool {
	trimmed := strings.TrimSpace(sentence)
	switch dimension {
	case "overall_measurement":
		return hasMeasurement(trimmed) && containsAny(trimmed, []string{"通高", "整体", "全高"})
	case "body_measurement":
		return hasMeasurement(trimmed) && containsAny(trimmed, []string{"主体", "本体", "佛体", "雕像"})
	case "total_measurement":
		return hasMeasurement(trimmed) && containsAny(trimmed, []string{"总高", "台基", "底座", "基座"})
	case "measurement":
		return hasMeasurement(trimmed)
	case "administrative_location":
		return containsAny(trimmed, []string{"省", "市", "县"})
	case "region_location":
		return containsAny(trimmed, []string{"区域", "片区", "城区", "新区", "景区"})
	case "locality_location":
		return containsAny(trimmed, []string{"镇", "乡", "街道"})
	case "craft_subject":
		return containsAny(trimmed, []string{"工艺", "技艺", "艺术", "制作", "雕刻", "烧制", "镶嵌", "材料", "传统"})
	case "craft_explanation":
		return containsAny(trimmed, []string{"凭借", "因为", "由于", "因而", "采用", "运用", "汇集", "融合", "结合", "形成", "体现"})
	case "culture_subject":
		return containsAny(trimmed, []string{"文化", "宗教", "礼仪", "仪式", "传统", "主题", "主体"})
	case "culture_explanation":
		return isCultureExplanationSentence(trimmed)
	case "concrete_examples":
		return containsAny(trimmed, []string{"、", "与", "和"}) && sentenceHasFactDimension(query, trimmed, "culture_explanation")
	case "ritual_subject":
		return containsAny(trimmed, []string{"供奉", "祭祀", "礼拜"})
	case "culture_practice":
		return containsAny(trimmed, []string{"游客", "参观者", "体验", "参与", "转动", "使用"}) &&
			sentenceHasFactDimension(query, trimmed, "culture_explanation")
	default:
		return false
	}
}

func isCultureExplanationSentence(sentence string) bool {
	if containsAny(sentence, []string{"主题", "再现", "解释", "介绍", "体现", "象征", "寓意", "包括", "涵盖", "设有", "供奉", "陈列", "场景", "意义", "过程"}) {
		return true
	}
	return strings.Contains(sentence, "展示") && containsAny(sentence, []string{"、", "与", "和"})
}

func hasMeasurement(sentence string) bool {
	hasDigit := false
	for _, r := range sentence {
		if r >= '0' && r <= '9' {
			hasDigit = true
			break
		}
	}
	return hasDigit && containsAny(sentence, []string{"米", "m", "吨", "级", "%"})
}

func isKnowledgeMetaSentence(sentence string) bool {
	return containsAny(sentence, []string{"游客常问", "常见问法", "问答素材", "回答策略", "资料来源"})
}

func splitKnowledgeSentences(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return compactKeywords(strings.FieldsFunc(content, func(r rune) bool {
		return strings.ContainsRune("。！？；\n", r)
	}))
}

func (s *RAGService) QueryGeneralChat(ctx context.Context, query, lang string) (string, error) {
	cacheKey := "general:" + lang + ":" + query
	if cached, ok := s.getCachedResponse(cacheKey); ok {
		slog.Debug("通用 Chat 命中查询缓存", "query_len", len([]rune(query)))
		return cached.Response, nil
	}

	if strings.TrimSpace(s.chatAPIKey) == "" {
		answer, sources := s.localRAGFallback(query, lang)
		s.setCachedResponse(cacheKey, answer, sources)
		return answer, nil
	}

	defer func() {
		if r := recover(); r != nil {
			slog.Error("QueryGeneralChat panic", "panic", r)
		}
	}()

	slog.Info("调用通用 Chat API", "query_len", len([]rune(query)), "base_url", s.chatBaseURL, "model", s.chatModel)

	// 使用 profile 配置的通用 Chat Prompt（支持任意景区）
	var userPrompt string
	if s.profile != nil && s.profile.Prompts.GeneralChatPrompt != "" {
		userPrompt = s.profile.RenderPrompt(s.profile.Prompts.GeneralChatPrompt)
		userPrompt = strings.ReplaceAll(userPrompt, "{query}", query)
	} else {
		userPrompt = fmt.Sprintf("你是一位专业的景区数字人导览员。\n\n【游客问题】\n%s\n\n请回答：", query)
	}

	req := openAIRequest{
		Model: s.chatModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: s.getSystemPromptOrDefault(lang),
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		slog.Error("通用 Chat 请求序列化失败", "error", err)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("JSON序列化失败: %v", err)
	}

	modelResult, modelErr, _ := s.modelRequests.Do(cacheKey, func() (interface{}, error) {
		return s.callChatCompletion(ctx, reqBody)
	})
	if modelErr != nil {
		slog.Warn("通用 Chat API 调用失败，使用本地知识库降级", "error", modelErr)
		answer, sources := s.localRAGFallback(query, lang)
		s.setCachedResponse(cacheKey, answer, sources)
		return answer, nil
	}
	body, ok := modelResult.([]byte)
	if !ok {
		return "", fmt.Errorf("通用 Chat API 响应结果类型错误")
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		slog.Error("通用 Chat API 响应解析失败", "error", err)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("解析API响应失败: %v", err)
	}

	if openAIResp.Error != nil {
		slog.Warn("通用 Chat API 返回业务错误",
			"type", openAIResp.Error.Type,
			"code", openAIResp.Error.Code,
			"message", openAIResp.Error.Message,
		)
		answer, sources := s.localRAGFallback(query, lang)
		s.setCachedResponse(cacheKey, answer, sources)
		return answer, nil
	}

	var answer string
	if len(openAIResp.Choices) > 0 && openAIResp.Choices[0].Message.Content != "" {
		answer = openAIResp.Choices[0].Message.Content
	} else {
		slog.Warn("通用 Chat API 返回空结果")
		return "抱歉，我无法生成合适的回答。", fmt.Errorf("API返回空结果")
	}

	s.setCachedResponse(cacheKey, answer, nil)
	slog.Info("通用 Chat API 返回回答", "answer_len", len([]rune(answer)))
	return answer, nil
}

func (s *RAGService) HasConfiguredLLM() bool {
	return s != nil && strings.TrimSpace(s.chatAPIKey) != "" && strings.TrimSpace(s.chatBaseURL) != ""
}

func (s *RAGService) CallLLM(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !s.HasConfiguredLLM() {
		return "", fmt.Errorf("AI API Key 未配置，无法执行 AI 分析")
	}
	req := openAIRequest{
		Model: s.chatModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("AI 分析请求序列化失败: %w", err)
	}
	body, err := s.callChatCompletion(ctx, reqBody)
	if err != nil {
		return "", fmt.Errorf("AI 分析调用失败: %w", err)
	}
	var parsed openAIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("AI 分析响应解析失败: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("AI 分析返回错误: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("AI 分析响应为空")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func (s *RAGService) callChatCompletion(ctx context.Context, requestBody []byte) ([]byte, error) {
	if s == nil || s.chatGuard == nil {
		return nil, fmt.Errorf("Chat 模型保护器未初始化")
	}

	endpoint := strings.TrimRight(s.chatBaseURL, "/") + "/chat/completions"
	var responseBody []byte
	err := s.chatGuard.run(ctx, func(callCtx context.Context) error {
		req, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.chatAPIKey)
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			_, _ = readLimitedBody(resp.Body)
			return &modelHTTPError{status: resp.StatusCode}
		}
		responseBody, err = readLimitedBody(resp.Body)
		return err
	})
	return responseBody, err
}

func (s *RAGService) localRAGFallback(query, lang string) (string, []RAGSource) {
	chunks, err := s.RetrieveRelevantKnowledgeWithOptions(query, RetrievalOptions{
		TopK:                 TopK,
		Mode:                 RetrievalModeBM25Local,
		SkipModelEnhancement: true,
	})
	if err != nil {
		slog.Warn("本地 RAG 降级检索失败", "error", err, "query_len", len([]rune(query)))
		return addEmotionCare(query, s.buildNoEvidenceAnswer(lang)), nil
	}
	if len(chunks) == 0 {
		return addEmotionCare(query, s.buildNoEvidenceAnswer(lang)), nil
	}
	sources := buildRAGSources(chunks, 3)
	return s.generateAnswerFromChunks(query, chunks), sources
}

// buildRouteRecommendation 根据用户查询匹配路线关键词，返回路线推荐信息
func (s *RAGService) buildRouteRecommendation(query string) string {
	if s.profile == nil || len(s.profile.Routes) == 0 {
		return ""
	}

	// 路线匹配关键词映射
	routeKeywords := map[string][]string{
		"历史":  {"历史", "文化", "古迹", "建筑", "佛教", "寺庙"},
		"自然":  {"自然", "风景", "拍照", "山水", "湖"},
		"亲子":  {"亲子", "儿童", "孩子", "家庭", "小朋友", "带孩子"},
		"美食":  {"美食", "小吃", "吃饭", "餐厅", "素食"},
		"路线":  {"路线", "推荐", "怎么玩", "怎么走", "规划", "行程"},
		"观光车": {"观光车", "摆渡车"},
		"错峰":  {"错峰", "避开人流", "人多"},
		"轻松":  {"轻松", "老人", "长辈", "不累"},
		"深度":  {"深度", "详细", "全面", "全部"},
	}

	// 检查查询匹配哪些路线类别
	queryLower := query
	var matchedCategories []string
	for category, keywords := range routeKeywords {
		for _, kw := range keywords {
			if strings.Contains(queryLower, kw) {
				matchedCategories = append(matchedCategories, category)
				break
			}
		}
	}

	if len(matchedCategories) == 0 {
		return ""
	}

	// 构建路线推荐信息
	var result string
	for _, route := range s.profile.Routes {
		routeName := route.Name
		routeDesc := route.Description
		routeSpots := route.Spots

		// 根据匹配的类别筛选路线
		relevant := false
		for _, cat := range matchedCategories {
			switch cat {
			case "历史", "深度":
				if strings.Contains(routeName, "文化") || strings.Contains(routeName, "历史") || strings.Contains(routeName, "深度") {
					relevant = true
				}
			case "自然":
				if strings.Contains(routeName, "自然") || strings.Contains(routeName, "风光") {
					relevant = true
				}
			case "亲子":
				if strings.Contains(routeName, "亲子") || strings.Contains(routeName, "欢乐") {
					relevant = true
				}
			case "美食":
				if strings.Contains(routeName, "美食") || strings.Contains(routeName, "体验") {
					relevant = true
				}
			case "轻松":
				if strings.Contains(routeDesc, "轻松") || route.Difficulty == "easy" {
					relevant = true
				}
			case "路线":
				relevant = true // 用户问路线推荐，所有路线都相关
			case "观光车":
				relevant = route.RouteType == "sightseeing_bus"
			case "错峰":
				relevant = route.RouteType == "off_peak_walking"
			}
		}

		if relevant {
			result += fmt.Sprintf("- %s：%s（途经：%s）\n", routeName, routeDesc, routeSpots)
		}
	}

	return result
}
