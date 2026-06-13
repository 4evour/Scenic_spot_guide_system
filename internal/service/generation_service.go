package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/scenic-guide/internal/model"
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
		Index int `json:"index"`
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

	// 使用 profile 配置的 prompt 模板（支持任意景区）
	if s.profile != nil && s.profile.Prompts.RAGPrompt != "" {
		prompt := s.profile.RenderPrompt(s.profile.Prompts.RAGPrompt)
		prompt = strings.ReplaceAll(prompt, "{knowledge_context}", context.String())
		prompt = strings.ReplaceAll(prompt, "{session_context}", conversation.String())
		prompt = strings.ReplaceAll(prompt, "{query}", query)

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
	return prompt
}

func (s *RAGService) QueryWithRAG(query string) (string, error) {
	answer, _, err := s.QueryWithRAGTrace(query, "")
	return answer, err
}

func (s *RAGService) QueryWithRAGTrace(query, lang string) (answer string, trace RAGTrace, err error) {
	return s.queryWithRAGTraceInternal(query, query, "", lang)
}

func (s *RAGService) queryWithRAGTraceInternal(retrievalQuery, promptQuery, sessionContext, lang string) (answer string, trace RAGTrace, err error) {
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

	cacheKey := retrievalQuery
	if sessionContext != "" {
		cacheKey = "prompt:" + promptQuery + "\nretrieval:" + retrievalQuery + "\nctx:" + sessionContext
	}
	if cachedResp, ok := s.getCachedResponse(cacheKey); ok {
		slog.Debug("RAG 查询命中缓存", "query_len", len([]rune(promptQuery)))
		trace.CacheHit = true
		trace.TotalMs = time.Since(totalStart).Milliseconds()
		trace.SlowRequest = trace.TotalMs > SlowRequestThresholdMs
		return cachedResp, trace, nil
	}

	retrievalStart := time.Now()
	chunks, err := s.RetrieveRelevantKnowledge(retrievalQuery, TopK)
	trace.RetrievalMs = time.Since(retrievalStart).Milliseconds()
	trace.ChunkCount = len(chunks)
	if err != nil {
		return "", trace, fmt.Errorf("检索相关知识失败: %v", err)
	}

	if len(chunks) == 0 {
		slog.Info("RAG 未检索到相关知识，使用通用 Chat 模式", "query_len", len([]rune(promptQuery)))
		generationStart := time.Now()
		answer, err := s.QueryGeneralChat(promptQuery, lang)
		trace.GenerationMs = time.Since(generationStart).Milliseconds()
		return answer, trace, err
	}

	slog.Info("RAG 检索命中知识库", "query_len", len([]rune(promptQuery)), "chunks", len(chunks), "mode", map[bool]string{true: "bm25", false: "embedding"}[s.useBM25])

	if s.chatAPIKey == "" {
		generationStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs = time.Since(generationStart).Milliseconds()
		s.setCachedResponse(cacheKey, answer)
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
		generationStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs = time.Since(generationStart).Milliseconds()
		s.setCachedResponse(cacheKey, answer)
		return answer, trace, nil
	}

	apiURL := s.chatBaseURL + "/chat/completions"

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		slog.Error("RAG 创建 HTTP 请求失败", "error", err)
		generationStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs = time.Since(generationStart).Milliseconds()
		s.setCachedResponse(cacheKey, answer)
		return answer, trace, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.chatAPIKey)

	generationStart := time.Now()
	resp, err := s.httpClient.Do(httpReq)
	trace.GenerationMs = time.Since(generationStart).Milliseconds()
	if err != nil {
		slog.Error("RAG 调用 Chat API 失败", "error", err)
		fallbackStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs += time.Since(fallbackStart).Milliseconds()
		s.setCachedResponse(cacheKey, answer)
		return answer, trace, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := readLimitedBody(resp.Body)
		slog.Warn("RAG Chat API 返回非 200", "status", resp.StatusCode, "body_len", len(body))
		fallbackStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs += time.Since(fallbackStart).Milliseconds()
		s.setCachedResponse(cacheKey, answer)
		return answer, trace, nil
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		slog.Error("RAG 读取 Chat API 响应失败", "error", readErr)
		fallbackStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs += time.Since(fallbackStart).Milliseconds()
		s.setCachedResponse(cacheKey, answer)
		return answer, trace, nil
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		slog.Error("RAG 解析 Chat API 响应失败", "error", err)
		fallbackStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs += time.Since(fallbackStart).Milliseconds()
		s.setCachedResponse(cacheKey, answer)
		return answer, trace, nil
	}

	if openAIResp.Error != nil {
		slog.Warn("RAG Chat API 返回业务错误",
			"type", openAIResp.Error.Type,
			"code", openAIResp.Error.Code,
			"message", openAIResp.Error.Message,
		)
		fallbackStart := time.Now()
		answer := s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
		trace.GenerationMs += time.Since(fallbackStart).Milliseconds()
		s.setCachedResponse(cacheKey, answer)
		return answer, trace, nil
	}

	if len(openAIResp.Choices) > 0 && openAIResp.Choices[0].Message.Content != "" {
		answer := openAIResp.Choices[0].Message.Content
		s.setCachedResponse(cacheKey, answer)
		return answer, trace, nil
	}

	fallbackStart := time.Now()
	answer = s.generateAnswerFromChunksWithContext(promptQuery, chunks, sessionContext)
	trace.GenerationMs += time.Since(fallbackStart).Milliseconds()
	s.setCachedResponse(cacheKey, answer)
	return answer, trace, nil
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
func (s *RAGService) CallLLMStreaming(systemPrompt, userPrompt string, onToken func(string)) (string, error) {
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
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("stream request creation failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.chatAPIKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("stream LLM call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := readLimitedBody(resp.Body)
		return "", fmt.Errorf("stream LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var fullResponse strings.Builder
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
				fullResponse.WriteString(token)
				onToken(token)
			}
		}
	}
	return fullResponse.String(), nil
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

	if isBoundaryIntent(intentText) {
		snippets := s.extractRelevantSnippets(query+" 官方最新公告 现场公示 不能编造", chunks, 3)
		if len(snippets) == 0 {
			snippets = []string{previewRunes(fullContent, 260)}
		}
		answer := "这个不能直接替您确认或承诺。根据当前资料，可以先参考：\n\n" + formatNumberedLines(snippets, 3) + "\n\n涉及开放状态、票价、演出场次、实时客流、排队时间、无人机或宠物等现场规则时，请以景区官方最新公告或现场公示为准。"
		return previewRunes(answer, 700)
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
		return previewRunes(answer, 700)
	}

	// 使用 profile 配置的兜底答案（支持任意景区）
	if s.profile != nil {
		if fallbackAnswer, ok := s.profile.GetFallbackAnswer(query); ok {
			return fallbackAnswer
		}
	}

	snippets := s.extractRelevantSnippets(query, chunks, 4)
	if len(snippets) == 0 {
		snippets = []string{previewRunes(fullContent, 500)}
	}

	scenicName := "景区"
	if s.profile != nil {
		scenicName = s.profile.Name
	}
	answer := fmt.Sprintf("根据%s景区资料：\n\n", scenicName) + strings.Join(snippets, "\n\n")
	return previewRunes(answer, 700)
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
		text  string
		score float64
	}

	snippets := make([]scoredSnippet, 0)
	seen := make(map[string]struct{})
	for _, chunk := range chunks {
		for _, sentence := range splitKnowledgeSentences(chunk.Content) {
			if len([]rune(sentence)) < 8 {
				continue
			}
			score := s.BM25Similarity(query, sentence)
			for _, cb := range s.getConditionalBoosts() {
				score += conditionalTermBoost(query, sentence, cb.QueryTerms, cb.ContentTerms)
			}
			if score <= 0 {
				continue
			}
			if _, ok := seen[sentence]; ok {
				continue
			}
			seen[sentence] = struct{}{}
			snippets = append(snippets, scoredSnippet{text: sentence, score: score})
		}
	}

	sort.Slice(snippets, func(i, j int) bool {
		return snippets[i].score > snippets[j].score
	})

	result := make([]string, 0, limit)
	for i := 0; i < min(limit, len(snippets)); i++ {
		result = append(result, snippets[i].text)
	}
	return result
}

func splitKnowledgeSentences(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return compactKeywords(strings.FieldsFunc(content, func(r rune) bool {
		return strings.ContainsRune("。！？；\n", r)
	}))
}

func (s *RAGService) QueryGeneralChat(query, lang string) (string, error) {
	if cachedResp, ok := s.getCachedResponse(query); ok {
		slog.Debug("通用 Chat 命中查询缓存", "query_len", len([]rune(query)))
		return cachedResp, nil
	}

	if strings.TrimSpace(s.chatAPIKey) == "" {
		answer := "当前知识库没有检索到足够匹配的资料，并且 AI API Key 尚未配置。您可以先在管理后台补充相关知识，或配置 AI API Key 后再启用通用问答。"
		s.setCachedResponse(query, answer)
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

	apiURL := s.chatBaseURL + "/chat/completions"
	slog.Debug("通用 Chat 请求体已生成", "body_len", len(reqBody), "api_url", apiURL)

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		slog.Error("通用 Chat 创建 HTTP 请求失败", "error", err)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.chatAPIKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("通用 Chat API 调用失败", "error", err)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("调用DeepSeek API失败: %v", err)
	}
	defer resp.Body.Close()

	slog.Debug("通用 Chat API 已响应", "status", resp.StatusCode)

	body, readErr := readLimitedBody(resp.Body)
	if readErr != nil {
		slog.Error("通用 Chat API 响应读取失败", "error", readErr)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("读取响应体失败: %v", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Warn("通用 Chat API 返回非 200", "status", resp.StatusCode, "body_len", len(body))
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("API返回错误状态码: %d, 响应长度: %d bytes", resp.StatusCode, len(body))
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
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("DeepSeek API错误: %s - %s",
			openAIResp.Error.Code, openAIResp.Error.Message)
	}

	var answer string
	if len(openAIResp.Choices) > 0 && openAIResp.Choices[0].Message.Content != "" {
		answer = openAIResp.Choices[0].Message.Content
	} else {
		slog.Warn("通用 Chat API 返回空结果")
		return "抱歉，我无法生成合适的回答。", fmt.Errorf("API返回空结果")
	}

	s.setCachedResponse(query, answer)
	slog.Info("通用 Chat API 返回回答", "answer_len", len([]rune(answer)))
	return answer, nil
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
			}
		}

		if relevant {
			result += fmt.Sprintf("- %s：%s（途经：%s）\n", routeName, routeDesc, routeSpots)
		}
	}

	return result
}

// contains 检查字符串是否包含子串

