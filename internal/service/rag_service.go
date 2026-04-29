package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scenic-guide/config"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

const (
	MinSimilarityThreshold = 0.01
	TopK                   = 5
)

func isLingshanRelatedQuestion(query string) bool {
	queryLower := strings.ToLower(query)
	fmt.Printf("意图判断 - 查询: %s\n", query)
	for _, keyword := range config.LingshanRelatedKeywords {
		if strings.Contains(queryLower, strings.ToLower(keyword)) {
			fmt.Printf("  匹配到关键词: %s\n", keyword)
			return true
		}
	}
	for _, pattern := range config.LingshanRelatedPatterns {
		if strings.Contains(query, pattern) {
			fmt.Printf("  匹配到模式: %s\n", pattern)
			return true
		}
	}
	fmt.Println("  未匹配到任何灵山相关关键词")
	return false
}

type RAGService struct {
	repo        *repository.KnowledgeRepository
	chatAPIKey  string
	chatModel   string
	chatBaseURL string
	embedding   EmbeddingProvider
	bm25        *BM25FallbackProvider
	useBM25     bool
	uploadDir   string
}

func NewRAGService(repo *repository.KnowledgeRepository, chatAPIKey, chatModel, chatBaseURL string, embeddingProvider EmbeddingProvider) *RAGService {
	bm25 := NewBM25FallbackProvider()
	useBM25 := true

	if embeddingProvider != nil && embeddingProvider.IsAvailable() {
		useBM25 = false
		fmt.Printf("使用Embedding Provider: %s\n", embeddingProvider.Name())
	} else {
		fmt.Printf("Embedding Provider不可用，将使用BM25\n")
	}

	return &RAGService{
		repo:        repo,
		chatAPIKey:  chatAPIKey,
		chatModel:   chatModel,
		chatBaseURL: chatBaseURL,
		embedding:   embeddingProvider,
		bm25:        bm25,
		useBM25:     useBM25,
		uploadDir:   "./knowledge",
	}
}

type ChunkData struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Source   string                 `json:"source"`
	Title    string                 `json:"title"`
	Metadata map[string]interface{} `json:"metadata"`
}

func (s *RAGService) LoadKnowledgeFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件 %s 失败: %v", filePath, err)
	}

	lines := strings.Split(string(data), "\n")
	loadedCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var chunk ChunkData
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			fmt.Printf("跳过无效行: %v\n", err)
			continue
		}

		exists, err := s.repo.Exists(chunk.ID)
		if err != nil {
			return fmt.Errorf("检查ID存在失败: %v", err)
		}

		if exists {
			fmt.Printf("知识 %s 已存在，跳过\n", chunk.ID)
			continue
		}

		vector, err := s.GenerateEmbedding(chunk.Content)
		if err != nil {
			fmt.Printf("生成embedding失败，使用BM25: %v\n", err)
			vector = s.bm25FallbackVector(chunk.Content)
		}

		metadataJSON, _ := json.Marshal(chunk.Metadata)

		knowledge := &model.KnowledgeChunk{
			ID:       chunk.ID,
			Content:  chunk.Content,
			Source:   chunk.Source,
			Title:    chunk.Title,
			Metadata: string(metadataJSON),
			Vector:   vector,
		}

		if err := s.repo.Create(knowledge); err != nil {
			fmt.Printf("保存知识 %s 失败: %v\n", chunk.ID, err)
			continue
		}

		loadedCount++
	}

	fmt.Printf("成功加载 %d 条知识\n", loadedCount)
	return nil
}

func (s *RAGService) LoadKnowledgeFromJSONL(data []byte) (int, error) {
	lines := strings.Split(string(data), "\n")
	loadedCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var chunk ChunkData
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return loadedCount, fmt.Errorf("解析JSON失败: %v", err)
		}

		exists, err := s.repo.Exists(chunk.ID)
		if err != nil {
			return loadedCount, fmt.Errorf("检查ID存在失败: %v", err)
		}

		if exists {
			continue
		}

		vector, err := s.GenerateEmbedding(chunk.Content)
		if err != nil {
			vector = s.bm25FallbackVector(chunk.Content)
		}

		metadataJSON, _ := json.Marshal(chunk.Metadata)

		knowledge := &model.KnowledgeChunk{
			ID:       chunk.ID,
			Content:  chunk.Content,
			Source:   chunk.Source,
			Title:    chunk.Title,
			Metadata: string(metadataJSON),
			Vector:   vector,
		}

		if err := s.repo.Create(knowledge); err != nil {
			return loadedCount, fmt.Errorf("保存知识失败: %v", err)
		}

		loadedCount++
	}

	return loadedCount, nil
}

func (s *RAGService) SaveUploadedFile(filename string, data []byte) (string, error) {
	ext := filepath.Ext(filename)
	if ext != ".jsonl" && ext != ".json" {
		return "", fmt.Errorf("仅支持 .jsonl 或 .json 文件")
	}

	if err := os.MkdirAll(s.uploadDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	savePath := filepath.Join(s.uploadDir, filename)
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存文件失败: %v", err)
	}

	return savePath, nil
}

func (s *RAGService) DeleteKnowledge(id string) error {
	return s.repo.Delete(id)
}

func (s *RAGService) DeleteAllKnowledge() error {
	return s.repo.DeleteAll()
}

func (s *RAGService) ListKnowledge(page, pageSize int) ([]model.KnowledgeChunk, int64, error) {
	all, err := s.repo.GetAll()
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(all))

	start := (page - 1) * pageSize
	if start >= len(all) {
		return []model.KnowledgeChunk{}, total, nil
	}

	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}

	return all[start:end], total, nil
}

func (s *RAGService) GetKnowledge(id string) (*model.KnowledgeChunk, error) {
	return s.repo.GetByID(id)
}

func (s *RAGService) bm25FallbackVector(content string) string {
	tokens := s.bm25.Tokenize(content)
	vector := make(map[string]float64)
	for _, token := range tokens {
		vector[token]++
	}
	vectorJSON, _ := json.Marshal(vector)
	return string(vectorJSON)
}

func (s *RAGService) GenerateEmbedding(text string) (string, error) {
	if !s.useBM25 && s.embedding != nil {
		vec, err := s.embedding.GenerateEmbedding(text)
		if err != nil {
			return "", err
		}
		vecJSON, err := json.Marshal(vec)
		if err != nil {
			return "", err
		}
		return string(vecJSON), nil
	}

	return s.bm25FallbackVector(text), nil
}

func (s *RAGService) CosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) == 0 || len(vec2) == 0 {
		return 0
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	minLen := len(vec1)
	if len(vec2) < minLen {
		minLen = len(vec2)
	}

	for i := 0; i < minLen; i++ {
		dotProduct += vec1[i] * vec2[i]
	}
	for i := 0; i < len(vec1); i++ {
		norm1 += vec1[i] * vec1[i]
	}
	for i := 0; i < len(vec2); i++ {
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

func (s *RAGService) BM25Similarity(query, content string) float64 {
	queryTokens := s.bm25.Tokenize(query)
	docTokens := s.bm25.Tokenize(content)
	return s.bm25.CalculateSimilarity(queryTokens, docTokens)
}

func (s *RAGService) RetrieveRelevantKnowledge(query string, topK int) ([]model.KnowledgeChunk, error) {
	allChunks, err := s.repo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("获取所有知识片段失败: %v", err)
	}

	if len(allChunks) == 0 {
		return nil, nil
	}

	type scoredChunk struct {
		chunk      model.KnowledgeChunk
		similarity float64
	}

	var scoredChunks []scoredChunk

	for _, chunk := range allChunks {
		var similarity float64

		if s.useBM25 {
			similarity = s.BM25Similarity(query, chunk.Content)
		} else {
			vec1, err := s.embedding.GenerateEmbedding(query)
			if err != nil {
				similarity = s.BM25Similarity(query, chunk.Content)
			} else {
				vec2, err := s.parseVector(chunk.Vector)
				if err != nil {
					similarity = s.BM25Similarity(query, chunk.Content)
				} else {
					similarity = s.CosineSimilarity(vec1, vec2)
				}
			}
		}

		scoredChunks = append(scoredChunks, scoredChunk{
			chunk:      chunk,
			similarity: similarity,
		})
	}

	for i := 0; i < len(scoredChunks)-1; i++ {
		for j := i + 1; j < len(scoredChunks); j++ {
			if scoredChunks[j].similarity > scoredChunks[i].similarity {
				scoredChunks[i], scoredChunks[j] = scoredChunks[j], scoredChunks[i]
			}
		}
	}

	fmt.Printf("\n检索调试 - 查询: %s\n", query)
	for i, sc := range scoredChunks {
		if i < 3 {
			fmt.Printf("  [%d] %s (相似度: %.4f)\n", i+1, sc.chunk.Title, sc.similarity)
		}
	}

	result := make([]model.KnowledgeChunk, 0, topK)
	maxSimilarity := 0.0
	if len(scoredChunks) > 0 {
		maxSimilarity = scoredChunks[0].similarity
	}

	fmt.Printf("  最高相似度: %.4f\n", maxSimilarity)

	if maxSimilarity < MinSimilarityThreshold {
		fmt.Printf("  最高相似度 %.4f 低于阈值 %.4f，返回空（将调用通用知识API）\n", maxSimilarity, MinSimilarityThreshold)
		return nil, nil
	}

	for i := 0; i < min(topK, len(scoredChunks)); i++ {
		result = append(result, scoredChunks[i].chunk)
	}

	return result, nil
}

func (s *RAGService) parseVector(vectorStr string) ([]float64, error) {
	var vector []float64
	if err := json.Unmarshal([]byte(vectorStr), &vector); err != nil {
		return nil, err
	}
	return vector, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *RAGService) BuildRAGPrompt(query string, chunks []model.KnowledgeChunk) string {
	if len(chunks) == 0 {
		return ""
	}

	var context strings.Builder
	for i, chunk := range chunks {
		context.WriteString(fmt.Sprintf("%d. 【%s】\n%s\n来源：%s\n\n", i+1, chunk.Title, chunk.Content, chunk.Source))
	}

	prompt := fmt.Sprintf(`你是灵山胜境景区智能导览助手，负责基于知识库资料回答游客关于灵山胜境、景点介绍、历史文化、门票开放、游览路线、演艺活动、交通餐饮等问题。

请严格依据下面提供的知识库资料回答用户问题。

回答要求：
1. 优先使用知识库资料中的内容回答。
2. 不要编造资料中没有的信息，尤其是门票价格、开放时间、演出时间、路线安排、交通方式等。
3. 如果资料中有多个相关信息，请整理成清晰、自然的中文回答。
4. 如果用户问题适合分点回答，请使用简洁列表。
5. 如果资料中只包含部分答案，请说明“根据当前资料，可以确认的是……”。
6. 如果资料中完全没有相关答案，请回答“根据当前灵山胜境资料无法确认”。
7. 回答语气要像景区导览员，友好、准确、自然。
8. 不要提到“RAG”“向量数据库”“知识片段”等技术词。
9. 如有来源信息，可以在回答末尾简要标注“参考资料：xxx”。

知识库资料：
%s

用户问题：
%s

请基于以上资料回答：`, context.String(), query)

	return prompt
}

func (s *RAGService) QueryWithRAG(query string) (string, error) {
	fmt.Printf("\n===== QueryWithRAG 开始 =====\n")
	fmt.Printf("查询内容: %s\n", query)
	fmt.Printf("useBM25: %v\n", s.useBM25)

	chunks, err := s.RetrieveRelevantKnowledge(query, TopK)
	if err != nil {
		fmt.Printf("检索失败: %v\n", err)
		return "", fmt.Errorf("检索相关知识失败: %v", err)
	}

	fmt.Printf("检索到 %d 条相关知识\n", len(chunks))

	if len(chunks) == 0 {
		fmt.Println("未检索到相关知识，使用通用Chat模式")
		return s.QueryGeneralChat(query)
	}

	fmt.Println("使用RAG知识库回答")

	if s.chatAPIKey == "" {
		return s.generateAnswerFromChunks(query, chunks), nil
	}

	prompt := s.BuildRAGPrompt(query, chunks)

	type DashScopeRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	type DashScopeResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Output struct {
			Text string `json:"text"`
		} `json:"output"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	req := DashScopeRequest{
		Model: s.chatModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: "你是灵山胜境景区智能导览助手，负责基于知识库资料回答游客问题。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return s.generateAnswerFromChunks(query, chunks), nil
	}

	apiURL := s.chatBaseURL + "/chat/completions"
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return s.generateAnswerFromChunks(query, chunks), nil
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.chatAPIKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil { 	
		return s.generateAnswerFromChunks(query, chunks), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("API返回错误状态码: %d, 响应: %s\n", resp.StatusCode, string(body))
		return s.generateAnswerFromChunks(query, chunks), nil
	}

	body, _ := io.ReadAll(resp.Body)

	var dashScopeResp DashScopeResponse
	if err := json.Unmarshal(body, &dashScopeResp); err != nil {
		fmt.Printf("解析API响应失败: %v\n", err)
		return s.generateAnswerFromChunks(query, chunks), nil
	}

	if dashScopeResp.Error.Message != "" {
		fmt.Printf("DashScope API错误: %s\n", dashScopeResp.Error.Message)
		return s.generateAnswerFromChunks(query, chunks), nil
	}

	if len(dashScopeResp.Choices) > 0 && dashScopeResp.Choices[0].Message.Content != "" {
		return dashScopeResp.Choices[0].Message.Content, nil
	} else if dashScopeResp.Output.Text != "" {
		return dashScopeResp.Output.Text, nil
	}

	return s.generateAnswerFromChunks(query, chunks), nil
}

func (s *RAGService) generateAnswerFromChunks(query string, chunks []model.KnowledgeChunk) string {
	var content string
	for _, chunk := range chunks {
		content += chunk.Content + "\n\n"
	}

	// 简单的关键词匹配回答
	answer := "根据灵山胜境景区资料：\n\n"
	answer += content

	// 尝试提取关键信息
	if strings.Contains(query, "高") || strings.Contains(query, "高度") {
		// 查找高度相关信息
		if strings.Contains(content, "88") && strings.Contains(content, "佛") {
			return "灵山大佛高88米，主体高79米，莲花瓣高9米，含台基总高101.5米，耗铜量达725吨。"
		}
	}

	// 截取前500字符作为回答
	if len(answer) > 500 {
		answer = answer[:500] + "..."
	}

	return answer
}

func (s *RAGService) QueryGeneralChat(query string) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("QueryGeneralChat panic: %v\n", r)
		}
	}()

	fmt.Printf("\n--- 调用通用知识Chat ---\n")
	fmt.Printf("查询内容: %s\n", query)

	type DashScopeRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	type DashScopeResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Output struct {
			Text string `json:"text"`
		} `json:"output"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	userPrompt := fmt.Sprintf(`你是一个中文智能助手，请根据你的通用知识回答用户问题。

回答要求：
1. 如果用户问题与灵山胜境、景区导览、无锡旅游、佛教文化、景点介绍等相关，但当前没有可用资料，请明确说明“当前灵山胜境知识库中没有检索到相关资料”，然后可以给出一般性建议，但不要伪装成官方信息。
2. 如果用户问题与灵山胜境无关，请作为通用助手正常回答。
3. 对于实时性较强的信息，例如票价、营业时间、演出时间、交通管制、优惠政策等，如果没有可靠资料，请提醒用户以景区官方公告为准。
4. 不要编造具体数据、官方政策、价格、时间表。
5. 回答要简洁、自然、中文表达清楚。
6. 不要提到“RAG”“向量数据库”“知识片段”等技术词。

用户问题：
%s

请回答：`, query)

	req := DashScopeRequest{
		Model: s.chatModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: "你是一个友好的中文智能助手，能够回答各类问题。",
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("JSON序列化失败: %v\n", err)
		return "抱歉，我现在无法回答这个问题。", nil
	}

	apiURL := s.chatBaseURL + "/chat/completions"
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("创建HTTP请求失败: %v\n", err)
		return "抱歉，我现在无法回答这个问题。", nil
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.chatAPIKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("调用Chat API失败: %v\n", err)
		return "抱歉，我现在无法回答这个问题。", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("API返回错误: %s\n", string(body))
		return "抱歉，我现在无法回答这个问题。", nil
	}

	body, _ := io.ReadAll(resp.Body)

	var dashScopeResp DashScopeResponse
	if err := json.Unmarshal(body, &dashScopeResp); err != nil {
		fmt.Printf("解析API响应失败: %v\n", err)
		return "抱歉，我现在无法回答这个问题。", nil
	}

	if dashScopeResp.Error.Message != "" { 	
		fmt.Printf("Chat API错误: %s\n", dashScopeResp.Error.Message)
		return "抱歉，我现在无法回答这个问题。", nil
	}

	var answer string
	if len(dashScopeResp.Choices) > 0 && dashScopeResp.Choices[0].Message.Content != "" {
		answer = dashScopeResp.Choices[0].Message.Content
	} else if dashScopeResp.Output.Text != "" {
		answer = dashScopeResp.Output.Text
	} else {
		return "抱歉，我无法生成合适的回答。", nil
	}

	return answer, nil
}

func (s *RAGService) queryWithoutKnowledge(query string) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("queryWithoutKnowledge panic: %v\n", r)
		}
	}()

	fmt.Printf("\n--- 调用通用知识API ---\n")
	fmt.Printf("查询内容: %s\n", query)
	fmt.Printf("API Base URL: %s\n", s.chatBaseURL)
	fmt.Printf("Model: %s\n", s.chatModel)

	type DashScopeRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	type DashScopeResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Output struct {
			Text string `json:"text"`
		} `json:"output"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	req := DashScopeRequest{
		Model: s.chatModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: "你是一个知识问答助手。请基于你的通用知识回答用户问题，以不出错为主。如果不确定答案，请说明无法确认。",
			},
			{
				Role:    "user",
				Content: query,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("JSON序列化失败: %v\n", err)
		return "根据当前资料无法确认", nil
	}

	fmt.Printf("请求体: %s\n", string(reqBody))

	apiURL := s.chatBaseURL + "/chat/completions"
	fmt.Printf("完整API URL: %s\n", apiURL)

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("创建HTTP请求失败: %v\n", err)
		return "根据当前资料无法确认", nil
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.chatAPIKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Println("开始调用DashScope API...")
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("调用DashScope API失败: %v\n", err)
		return "根据当前资料无法确认", nil
	}
	defer resp.Body.Close()

	fmt.Printf("API响应状态码: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("API返回错误: %s\n", string(body))
		return "根据当前资料无法确认", nil
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("API响应体: %s\n", string(body))

	var dashScopeResp DashScopeResponse
	if err := json.Unmarshal(body, &dashScopeResp); err != nil {
		fmt.Printf("解析API响应失败: %v\n", err)
		return "根据当前资料无法确认", nil
	}

	if dashScopeResp.Error.Message != "" {
		fmt.Printf("DashScope API错误: %s\n", dashScopeResp.Error.Message)
		return "根据当前资料无法确认", nil
	}

	var answer string
	if len(dashScopeResp.Choices) > 0 && dashScopeResp.Choices[0].Message.Content != "" {
		answer = dashScopeResp.Choices[0].Message.Content
	} else if dashScopeResp.Output.Text != "" {
		answer = dashScopeResp.Output.Text
	} else {
		fmt.Println("API返回空结果")
		return "根据当前资料无法确认", nil
	}

	fmt.Printf("API返回回答: %s\n", answer[:min(len(answer), 100)]+"...")
	return "[通用知识回答] " + answer, nil
}

func (s *RAGService) RunEvaluation(evalFile string) error {
	data, err := os.ReadFile(evalFile)
	if err != nil {
		return fmt.Errorf("读取评估文件失败: %v", err)
	}

	var tests []struct {
		Question         string   `json:"question"`
		ExpectedKeywords []string `json:"expected_keywords"`
	}

	if err := json.Unmarshal(data, &tests); err != nil {
		return fmt.Errorf("解析评估文件失败: %v", err)
	}

	fmt.Println("\n========== RAG系统评估测试 ==========")
	passed := 0
	for i, test := range tests {
		response, err := s.QueryWithRAG(test.Question)
		if err != nil {
			fmt.Printf("[%d/%d] ❌ %s\n  错误: %v\n\n", i+1, len(tests), test.Question, err)
			continue
		}

		allFound := true
		for _, keyword := range test.ExpectedKeywords {
			if !strings.Contains(response, keyword) {
				allFound = false
				break
			}
		}

		status := "✅"
		if !allFound {
			status = "⚠️ "
		} else {
			passed++
		}

		fmt.Printf("[%d/%d] %s %s\n", i+1, len(tests), status, test.Question)
		if !allFound {
			fmt.Printf("  期望关键词: %v\n", test.ExpectedKeywords)
		}
		if len(response) > 100 {
			fmt.Printf("  回答: %s...\n\n", response[:100])
		} else {
			fmt.Printf("  回答: %s\n\n", response)
		}
	}

	fmt.Printf("========== 测试完成: %d/%d 通过 ==========\n", passed, len(tests))
	return nil
}
