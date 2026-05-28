package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	iconfig "github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/model"
)

// LLMQueryRewrite 使用 LLM 对用户查询进行智能扩展改写
// 替代硬编码的 expandQueryForRetrieval，自动适配任意景区
func (s *RAGService) LLMQueryRewrite(profile *iconfig.ScenicProfile, query string) (string, []string) {
	if s.chatAPIKey == "" || profile == nil {
		// 无 API Key 时 fallback 到配置化规则扩展
		return s.configBasedQueryExpansion(profile, query)
	}

	spotsDesc := strings.Join(profile.Keywords.Spots, "、")
	prompt := fmt.Sprintf(`你是景区知识库的检索优化助手。给定游客的原始问题和景区景点列表，请生成用于检索的扩展关键词。

景区景点：%s
游客问题：%s

规则：
1. 分析游客问题的意图（如路线推荐、景点介绍、服务查询等）
2. 根据意图补充相关的景点名称和主题词
3. 只输出扩展后的检索关键词，用空格分隔，不要解释
4. 保留原始问题中的关键词
5. 最多补充 8 个额外关键词`, spotsDesc, query)

	expanded, err := s.callLLMForTask(prompt, 100)
	if err != nil {
		slog.Debug("LLM 查询改写失败，使用配置规则", "error", err)
		return s.configBasedQueryExpansion(profile, query)
	}

	expanded = strings.TrimSpace(expanded)
	if expanded == "" || expanded == query {
		return query, nil
	}

	// 提取新增的关键词
	originalTokens := make(map[string]bool)
	for _, t := range s.bm25.Tokenize(query) {
		originalTokens[t] = true
	}
	var addedTerms []string
	for _, t := range s.bm25.Tokenize(expanded) {
		if !originalTokens[t] {
			addedTerms = append(addedTerms, t)
		}
	}

	return expanded, addedTerms
}

// configBasedQueryExpansion 基于 ScenicProfile 配置的查询扩展（无 API Key 时的 fallback）
func (s *RAGService) configBasedQueryExpansion(profile *iconfig.ScenicProfile, query string) (string, []string) {
	if profile == nil {
		return query, nil
	}

	var added []string
	seen := make(map[string]bool)

	for _, rule := range profile.Keywords.QueryExpansion {
		for _, trigger := range rule.Trigger {
			if strings.Contains(query, trigger) {
				for _, term := range strings.Fields(rule.Expand) {
					if !seen[term] {
						seen[term] = true
						added = append(added, term)
					}
				}
				break
			}
		}
	}

	if len(added) == 0 {
		return query, nil
	}
	return strings.TrimSpace(query + " " + strings.Join(added, " ")), added
}

// LLMRerank 使用 LLM 对检索结果进行智能重排序
// 替代硬编码的 light-rerank，自动适配任意景区
func (s *RAGService) LLMRerank(query string, chunks []model.KnowledgeChunk, topN int) []model.KnowledgeChunk {
	if len(chunks) <= 1 || s.chatAPIKey == "" {
		return chunks
	}

	if topN <= 0 || topN > len(chunks) {
		topN = len(chunks)
	}

	// 构建候选文档摘要（控制 token 量）
	var docSummaries []string
	for i, chunk := range chunks {
		title := chunk.Title
		if title == "" {
			title = chunk.Source
		}
		// 截取前 100 字作为摘要
		content := chunk.Content
		if len([]rune(content)) > 100 {
			content = string([]rune(content)[:100]) + "..."
		}
		docSummaries = append(docSummaries, fmt.Sprintf("[%d] %s: %s", i+1, title, content))
	}

	prompt := fmt.Sprintf(`你是搜索结果排序助手。给定用户问题和候选文档列表，请按相关性从高到低输出文档编号。

用户问题：%s

候选文档：
%s

规则：
1. 只输出编号，用逗号分隔，从最相关到最不相关
2. 只输出编号，不要解释
3. 输出格式：3,1,5,2,4（示例）`, query, strings.Join(docSummaries, "\n"))

	result, err := s.callLLMForTask(prompt, 50)
	if err != nil {
		slog.Debug("LLM Rerank 失败，保持原始排序", "error", err)
		return chunks
	}

	// 解析编号
	indices := parseRerankIndices(result, len(chunks))
	if len(indices) == 0 {
		return chunks
	}

	// 按 LLM 排序重排
	reranked := make([]model.KnowledgeChunk, 0, len(chunks))
	used := make(map[int]bool)
	for _, idx := range indices {
		if idx >= 0 && idx < len(chunks) && !used[idx] {
			reranked = append(reranked, chunks[idx])
			used[idx] = true
		}
	}
	// 追加未被 LLM 排序的文档
	for i, chunk := range chunks {
		if !used[i] {
			reranked = append(reranked, chunk)
		}
	}

	if len(reranked) > topN {
		reranked = reranked[:topN]
	}
	return reranked
}

// ProfileBasedIntentBoost 基于 ScenicProfile 配置的意图加分（替代硬编码 focusedIntentBoost）
func (s *RAGService) ProfileBasedIntentBoost(profile *iconfig.ScenicProfile, query string, chunk model.KnowledgeChunk) float64 {
	if profile == nil {
		return 0
	}

	haystack := chunk.Title + "\n" + chunk.Content
	topic := metadataString(chunk.Metadata, "topic")
	boost := 0.0

	for _, rule := range profile.Keywords.IntentBoosts {
		// 检查查询是否包含触发词
		queryMatch := false
		for _, trigger := range rule.QueryContains {
			if strings.Contains(query, trigger) {
				queryMatch = true
				break
			}
		}
		if !queryMatch {
			continue
		}

		// 检查 topic 匹配
		if rule.Topic != "" && topic != rule.Topic {
			continue
		}

		// 检查 chunk 内容是否包含目标词
		chunkMatch := len(rule.ChunkContains) == 0 // 无 chunk 条件则直接加分
		for _, target := range rule.ChunkContains {
			if strings.Contains(haystack, target) {
				chunkMatch = true
				break
			}
		}

		if chunkMatch {
			boost += rule.Boost
		}
	}

	return boost
}

// callLLMForTask 调用 LLM 完成一个简单任务（查询改写、重排序等）
func (s *RAGService) callLLMForTask(prompt string, maxTokens int) (string, error) {
	if s.chatAPIKey == "" {
		return "", fmt.Errorf("no API key")
	}

	start := time.Now()
	body := map[string]interface{}{
		"model": s.chatModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  maxTokens,
		"temperature": 0.1,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(s.chatBaseURL, "/") + "/chat/completions"
	resp, err := s.httpClient.Post(url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("LLM API returned %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	elapsed := time.Since(start).Milliseconds()
	slog.Debug("LLM 任务完成", "elapsed_ms", elapsed, "prompt_len", len(prompt))

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// parseRerankIndices 解析 LLM 返回的排序编号
func parseRerankIndices(result string, maxIndex int) []int {
	result = strings.TrimSpace(result)
	// 去掉可能的前缀文字
	if idx := strings.Index(result, "["); idx >= 0 {
		result = result[idx:]
	}
	result = strings.NewReplacer("[", "", "]", "", "(", "", ")", "", " ", "").Replace(result)

	parts := strings.Split(result, ",")
	var indices []int
	seen := make(map[int]bool)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err == nil {
			n-- // 转为 0-based 索引
			if n >= 0 && n < maxIndex && !seen[n] {
				seen[n] = true
				indices = append(indices, n)
			}
		}
	}
	return indices
}
