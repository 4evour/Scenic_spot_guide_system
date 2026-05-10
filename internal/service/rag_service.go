package service

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/scenic-guide/config"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

const (
	MinSimilarityThreshold = 0.01
	TopK                   = 5
	CacheTTL               = 5 * time.Minute
	MaxCacheSize           = 1000
)

type CacheEntry struct {
	Response   string
	ExpireTime time.Time
}

type TourRouteStep struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
}

type TourRoute struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Steps       []TourRouteStep `json:"steps"`
	Duration    string          `json:"duration"`
}

func isLingshanRelatedQuestion(query string) bool {
	queryLower := strings.ToLower(query)
	for _, keyword := range config.LingshanRelatedKeywords {
		if strings.Contains(queryLower, strings.ToLower(keyword)) {
			return true
		}
	}
	for _, pattern := range config.LingshanRelatedPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}
	return false
}

type RAGService struct {
	repo           *repository.KnowledgeRepository
	chatAPIKey     string
	chatModel      string
	chatBaseURL    string
	embedding      EmbeddingProvider
	bm25           *BM25FallbackProvider
	useBM25        bool
	uploadDir      string
	httpClient     *http.Client
	queryCache     map[string]CacheEntry
	embeddingCache map[string][]float64
	knowledgeCache []model.KnowledgeChunk
	cacheMutex     sync.RWMutex
	lastCacheTime  time.Time
}

func NewRAGService(repo *repository.KnowledgeRepository, chatAPIKey, chatModel, chatBaseURL string, embeddingProvider EmbeddingProvider) *RAGService {
	bm25 := NewBM25FallbackProvider()
	useBM25 := true

	if embeddingProvider != nil && embeddingProvider.IsAvailable() {
		useBM25 = false
	}

	return &RAGService{
		repo:           repo,
		chatAPIKey:     chatAPIKey,
		chatModel:      chatModel,
		chatBaseURL:    chatBaseURL,
		embedding:      embeddingProvider,
		bm25:           bm25,
		useBM25:        useBM25,
		uploadDir:      "./knowledge",
		httpClient:     createHTTPClient(),
		queryCache:     make(map[string]CacheEntry),
		embeddingCache: make(map[string][]float64),
		knowledgeCache: nil,
		lastCacheTime:  time.Now(),
	}
}

func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        200,
			IdleConnTimeout:     60 * time.Second,
			MaxIdleConnsPerHost: 20,
			DisableCompression:  false,
			ForceAttemptHTTP2:   true,
		},
	}
}

func (s *RAGService) getCachedKnowledge() ([]model.KnowledgeChunk, error) {
	s.cacheMutex.RLock()
	now := time.Now()
	if s.knowledgeCache != nil && now.Sub(s.lastCacheTime) < CacheTTL {
		chunks := make([]model.KnowledgeChunk, len(s.knowledgeCache))
		copy(chunks, s.knowledgeCache)
		s.cacheMutex.RUnlock()
		return chunks, nil
	}
	s.cacheMutex.RUnlock()

	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	if s.knowledgeCache != nil && now.Sub(s.lastCacheTime) < CacheTTL {
		chunks := make([]model.KnowledgeChunk, len(s.knowledgeCache))
		copy(chunks, s.knowledgeCache)
		return chunks, nil
	}

	allChunks, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	s.knowledgeCache = allChunks
	s.lastCacheTime = now
	return allChunks, nil
}

func (s *RAGService) getCachedEmbedding(text string) ([]float64, error) {
	s.cacheMutex.RLock()
	if vec, ok := s.embeddingCache[text]; ok {
		s.cacheMutex.RUnlock()
		return vec, nil
	}
	s.cacheMutex.RUnlock()

	vec, err := s.embedding.GenerateEmbedding(text)
	if err != nil {
		return nil, err
	}

	s.cacheMutex.Lock()
	if len(s.embeddingCache) >= MaxCacheSize {
		s.embeddingCache = make(map[string][]float64)
	}
	s.embeddingCache[text] = vec
	s.cacheMutex.Unlock()

	return vec, nil
}

func (s *RAGService) getCachedResponse(query string) (string, bool) {
	s.cacheMutex.RLock()
	entry, ok := s.queryCache[query]
	s.cacheMutex.RUnlock()

	if !ok {
		return "", false
	}

	if time.Now().After(entry.ExpireTime) {
		s.cacheMutex.Lock()
		delete(s.queryCache, query)
		s.cacheMutex.Unlock()
		return "", false
	}

	return entry.Response, true
}

func (s *RAGService) setCachedResponse(query, response string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	if len(s.queryCache) >= MaxCacheSize {
		s.queryCache = make(map[string]CacheEntry)
	}

	s.queryCache[query] = CacheEntry{
		Response:   response,
		ExpireTime: time.Now().Add(CacheTTL),
	}
}

type ChunkData struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Source   string                 `json:"source"`
	Title    string                 `json:"title"`
	Metadata map[string]interface{} `json:"metadata"`
}

type KnowledgeUpsertInput struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Source   string                 `json:"source"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

func (s *RAGService) invalidateKnowledgeCaches() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.queryCache = make(map[string]CacheEntry)
	s.knowledgeCache = nil
	s.lastCacheTime = time.Time{}
}

func normalizeKnowledgeChunk(chunk *ChunkData) {
	chunk.ID = strings.TrimSpace(chunk.ID)
	chunk.Title = strings.TrimSpace(chunk.Title)
	chunk.Source = strings.TrimSpace(chunk.Source)
	chunk.Content = strings.TrimSpace(chunk.Content)

	if chunk.ID == "" {
		sum := sha1.Sum([]byte(chunk.Title + "\n" + chunk.Source + "\n" + chunk.Content))
		chunk.ID = fmt.Sprintf("chunk-%x", sum[:8])
	}
	if chunk.Title == "" {
		runes := []rune(chunk.Content)
		if len(runes) > 24 {
			runes = runes[:24]
		}
		chunk.Title = string(runes)
	}
	if chunk.Source == "" {
		chunk.Source = "admin"
	}
	if chunk.Metadata == nil {
		chunk.Metadata = map[string]interface{}{}
	}
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
			continue
		}
		normalizeKnowledgeChunk(&chunk)
		if chunk.Content == "" {
			continue
		}

		exists, err := s.repo.Exists(chunk.ID)
		if err != nil {
			return fmt.Errorf("检查ID存在失败: %v", err)
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
			continue
		}

		loadedCount++
	}

	return nil
}

func (s *RAGService) LoadKnowledgeFromJSONL(data []byte) (int, error) {
	return s.LoadKnowledgeJSON(data)
}

func (s *RAGService) upsertChunkData(chunk *ChunkData) (*model.KnowledgeChunk, error) {
	normalizeKnowledgeChunk(chunk)
	if chunk.Content == "" {
		return nil, fmt.Errorf("knowledge content cannot be empty")
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

	exists, err := s.repo.Exists(chunk.ID)
	if err != nil {
		return nil, fmt.Errorf("check knowledge id failed: %v", err)
	}
	if exists {
		if err := s.repo.Update(knowledge); err != nil {
			return nil, fmt.Errorf("update knowledge failed: %v", err)
		}
		return knowledge, nil
	}

	if err := s.repo.Create(knowledge); err != nil {
		return nil, fmt.Errorf("create knowledge failed: %v", err)
	}
	return knowledge, nil
}

func (s *RAGService) CreateKnowledge(input KnowledgeUpsertInput) (*model.KnowledgeChunk, error) {
	chunk := ChunkData{
		ID:       input.ID,
		Title:    input.Title,
		Source:   input.Source,
		Content:  input.Content,
		Metadata: input.Metadata,
	}
	knowledge, err := s.upsertChunkData(&chunk)
	if err != nil {
		return nil, err
	}
	s.invalidateKnowledgeCaches()
	return knowledge, nil
}

func (s *RAGService) UpdateKnowledge(id string, input KnowledgeUpsertInput) (*model.KnowledgeChunk, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("knowledge id cannot be empty")
	}
	input.ID = id
	return s.CreateKnowledge(input)
}

func (s *RAGService) LoadKnowledgeDocument(filename string, data []byte, category string) (int, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jsonl":
		return s.LoadKnowledgeJSONLines(data)
	case ".json":
		return s.LoadKnowledgeJSON(data)
	case ".md", ".markdown", ".txt":
		return s.LoadPlainTextKnowledge(filename, string(data), category)
	default:
		return 0, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func (s *RAGService) LoadKnowledgeJSON(data []byte) (int, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		var chunks []ChunkData
		if err := json.Unmarshal([]byte(trimmed), &chunks); err != nil {
			return 0, fmt.Errorf("parse JSON failed: %v", err)
		}
		loadedCount := 0
		for _, chunk := range chunks {
			if _, err := s.upsertChunkData(&chunk); err != nil {
				return loadedCount, err
			}
			loadedCount++
		}
		if loadedCount > 0 {
			s.invalidateKnowledgeCaches()
		}
		return loadedCount, nil
	}
	return s.LoadKnowledgeJSONLines(data)
}

func (s *RAGService) LoadKnowledgeJSONLines(data []byte) (int, error) {
	lines := strings.Split(string(data), "\n")
	loadedCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var chunk ChunkData
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return loadedCount, fmt.Errorf("parse JSON line failed: %v", err)
		}
		if _, err := s.upsertChunkData(&chunk); err != nil {
			return loadedCount, err
		}
		loadedCount++
	}
	if loadedCount > 0 {
		s.invalidateKnowledgeCaches()
	}
	return loadedCount, nil
}

func (s *RAGService) LoadPlainTextKnowledge(filename, content, category string) (int, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return 0, fmt.Errorf("knowledge content cannot be empty")
	}

	title := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	paragraphs := splitKnowledgeParagraphs(content, 1200)
	loadedCount := 0
	for i, paragraph := range paragraphs {
		metadata := map[string]interface{}{
			"filename": filename,
			"chunk":    i + 1,
		}
		if strings.TrimSpace(category) != "" {
			metadata["category"] = strings.TrimSpace(category)
		}

		chunk := ChunkData{
			Title:    fmt.Sprintf("%s-%02d", title, i+1),
			Source:   filename,
			Content:  paragraph,
			Metadata: metadata,
		}
		if _, err := s.upsertChunkData(&chunk); err != nil {
			return loadedCount, err
		}
		loadedCount++
	}
	if loadedCount > 0 {
		s.invalidateKnowledgeCaches()
	}
	return loadedCount, nil
}

func splitKnowledgeParagraphs(content string, maxRunes int) []string {
	blocks := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n\n")
	chunks := make([]string, 0)
	var current strings.Builder

	flush := func() {
		text := strings.TrimSpace(current.String())
		if text != "" {
			chunks = append(chunks, text)
		}
		current.Reset()
	}

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if current.Len() > 0 && len([]rune(current.String()+"\n\n"+block)) > maxRunes {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(block)
	}
	flush()

	if len(chunks) == 0 {
		return []string{content}
	}
	return chunks
}

func (s *RAGService) SaveUploadedFile(filename string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jsonl", ".json", ".md", ".markdown", ".txt":
	default:
		return "", fmt.Errorf("only .jsonl, .json, .md, .markdown and .txt files are supported")
	}
	if err := os.MkdirAll(s.uploadDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	savePath := filepath.Join(s.uploadDir, filepath.Base(filename))
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存文件失败: %v", err)
	}

	return savePath, nil
}

func (s *RAGService) DeleteKnowledge(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.invalidateKnowledgeCaches()
	return nil
}

func (s *RAGService) DeleteAllKnowledge() error {
	if err := s.repo.DeleteAll(); err != nil {
		return err
	}
	s.invalidateKnowledgeCaches()
	return nil
}

func (s *RAGService) ListKnowledge(page, pageSize int, keyword, category string) ([]model.KnowledgeChunk, int64, error) {
	return s.repo.List(page, pageSize, keyword, category)
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
	allChunks, err := s.getCachedKnowledge()
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

	scoredChunks := make([]scoredChunk, 0, len(allChunks))

	var queryVec []float64
	if !s.useBM25 {
		vec, err := s.getCachedEmbedding(query)
		if err == nil {
			queryVec = vec
		}
	}

	for _, chunk := range allChunks {
		var similarity float64

		if s.useBM25 || len(queryVec) == 0 {
			similarity = s.BM25Similarity(query, chunk.Content)
		} else {
			vec2, err := s.parseVector(chunk.Vector)
			if err != nil {
				similarity = s.BM25Similarity(query, chunk.Content)
			} else {
				similarity = s.CosineSimilarity(queryVec, vec2)
			}
		}

		if similarity >= MinSimilarityThreshold {
			scoredChunks = append(scoredChunks, scoredChunk{
				chunk:      chunk,
				similarity: similarity,
			})
		}
	}

	if len(scoredChunks) == 0 {
		return nil, nil
	}

	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].similarity > scoredChunks[j].similarity
	})

	result := make([]model.KnowledgeChunk, 0, topK)
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

	prompt := fmt.Sprintf(`你是灵山胜境景区的AI数字人导览员”小灵”，负责为游客提供专业、热情的导览服务。

【身份设定】
- 你是灵山胜境景区的官方数字人导览员
- 你熟悉灵山大佛、九龙灌浴、梵宫等所有景点
- 你的职责是帮助游客了解景区、规划行程、解答疑问

【回答策略】
1. 优先使用知识库资料中的内容回答
2. 不要编造资料中没有的信息，尤其是门票价格、开放时间、演出时间、路线安排、交通方式等
3. 如果资料中有多个相关信息，请整理成清晰、自然的中文回答
4. 如果用户问题适合分点回答，请使用简洁列表
5. 如果资料中只包含部分答案，请说明”根据当前资料，可以确认的是……”
6. 如果资料中完全没有相关答案，请回答”这个问题我暂时无法确认，建议您咨询景区服务中心”
7. 不要提到”RAG””向量数据库””知识片段”等技术词

【语言风格】
- 称呼游客为”您”，保持礼貌尊重
- 使用温暖、亲切的语气，像真人导游一样自然
- 适当使用”欢迎来到灵山胜境”、”祝您游览愉快”等礼貌用语
- 主动提供相关建议（如游览路线、最佳时间、注意事项等）
- 回答要简洁明了，突出重点

【知识库资料】
%s

【游客问题】
%s

请以灵山景区数字人导览员的身份，基于以上资料回答：`, context.String(), query)

	return prompt
}

func (s *RAGService) QueryWithRAG(query string) (string, error) {
	if cachedResp, ok := s.getCachedResponse(query); ok {
		fmt.Println("命中查询缓存")
		return cachedResp, nil
	}

	chunks, err := s.RetrieveRelevantKnowledge(query, TopK)
	if err != nil {
		return "", fmt.Errorf("检索相关知识失败: %v", err)
	}

	if len(chunks) == 0 {
		fmt.Println("未检索到相关知识，使用通用Chat模式")
		return s.QueryGeneralChat(query)
	}

	fmt.Println("使用RAG知识库回答")

	if s.chatAPIKey == "" {
		answer := s.generateAnswerFromChunks(query, chunks)
		s.setCachedResponse(query, answer)
		return answer, nil
	}

	prompt := s.BuildRAGPrompt(query, chunks)

	type OpenAIRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	type OpenAIError struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	}

	type OpenAIResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *OpenAIError `json:"error"`
	}

	req := OpenAIRequest{
		Model: s.chatModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: "你是灵山胜境景区的AI数字人导览员「小灵」，负责为游客提供专业、热情的导览服务。你熟悉灵山大佛、九龙灌浴、梵宫等所有景点。回答要热情友好、准确专业，像真人导游一样自然亲切。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("JSON序列化失败: %v\n", err)
		answer := s.generateAnswerFromChunks(query, chunks)
		s.setCachedResponse(query, answer)
		return answer, nil
	}

	apiURL := s.chatBaseURL + "/chat/completions"

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("创建HTTP请求失败: %v\n", err)
		answer := s.generateAnswerFromChunks(query, chunks)
		s.setCachedResponse(query, answer)
		return answer, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.chatAPIKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		fmt.Printf("调用DeepSeek API失败: %v\n", err)
		answer := s.generateAnswerFromChunks(query, chunks)
		s.setCachedResponse(query, answer)
		return answer, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("RAG API返回错误状态码: %d, 响应长度: %d bytes\n", resp.StatusCode, len(body))
		answer := s.generateAnswerFromChunks(query, chunks)
		s.setCachedResponse(query, answer)
		return answer, nil
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		fmt.Printf("读取响应体失败: %v\n", readErr)
		answer := s.generateAnswerFromChunks(query, chunks)
		s.setCachedResponse(query, answer)
		return answer, nil
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		fmt.Printf("解析RAG API响应失败: %v\n", err)
		answer := s.generateAnswerFromChunks(query, chunks)
		s.setCachedResponse(query, answer)
		return answer, nil
	}

	if openAIResp.Error != nil {
		fmt.Printf("RAG DeepSeek API错误 - Type: %s, Code: %s, Message: %s\n",
			openAIResp.Error.Type, openAIResp.Error.Code, openAIResp.Error.Message)
		answer := s.generateAnswerFromChunks(query, chunks)
		s.setCachedResponse(query, answer)
		return answer, nil
	}

	if len(openAIResp.Choices) > 0 && openAIResp.Choices[0].Message.Content != "" {
		answer := openAIResp.Choices[0].Message.Content
		s.setCachedResponse(query, answer)
		return answer, nil
	}

	answer := s.generateAnswerFromChunks(query, chunks)
	s.setCachedResponse(query, answer)
	return answer, nil
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
	if cachedResp, ok := s.getCachedResponse(query); ok {
		fmt.Println("命中查询缓存 (通用Chat)")
		return cachedResp, nil
	}

	if strings.TrimSpace(s.chatAPIKey) == "" {
		answer := "当前知识库没有检索到足够匹配的资料，并且 AI API Key 尚未配置。您可以先在管理后台补充相关知识，或配置 AI API Key 后再启用通用问答。"
		s.setCachedResponse(query, answer)
		return answer, nil
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("QueryGeneralChat panic: %v\n", r)
		}
	}()

	fmt.Printf("\n--- 调用QueryGeneralChat ---\n")
	fmt.Printf("General chat query length: %d\n", len([]rune(query)))
	fmt.Printf("API Base URL: %s\n", s.chatBaseURL)
	fmt.Printf("Model: %s\n", s.chatModel)

	type OpenAIRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	type OpenAIError struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	}

	type OpenAIResponse struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
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
		Error *OpenAIError `json:"error"`
	}

	userPrompt := fmt.Sprintf(`你是灵山胜境景区的AI数字人导览员「小灵」。

【身份设定】
- 你是灵山胜境景区的官方数字人导览员
- 你熟悉灵山大佛、九龙灌浴、梵宫等所有景点
- 你的职责是帮助游客了解景区、规划行程、解答疑问

【回答策略】
1. 首先判断问题类型：
   - 如果问题与灵山胜境、无锡旅游、佛教文化直接相关，但当前知识库没有资料，请明确说明："当前灵山胜境知识库中没有检索到相关资料，不过我可以为您提供一些一般性的参考信息..."
   - 如果问题与灵山胜境无关，而是询问其他景点，请礼貌说明："我主要负责灵山胜境的导览服务。关于其他景区，建议您咨询相关景区的导览服务。"
   - 如果是完全无关的问题，礼貌地引导回到景区话题

2. 回答灵山相关问题时：
   - 重点介绍灵山大佛（88米高，世界著名青铜立佛）
   - 推荐九龙灌浴表演（精彩的佛教文化演出）
   - 介绍灵山梵宫（佛教艺术殿堂）
   - 提供游览建议和注意事项
   - 对于实时性信息（票价、营业时间等），请说明"具体信息请以景区官方公告为准"

3. 语言风格：
   - 称呼游客为"您"，保持礼貌尊重
   - 使用温暖、亲切的语气
   - 适当使用"欢迎来到灵山胜境"、"祝您游览愉快"等礼貌用语
   - 像真人导游一样自然、专业
   - 不要提到"RAG""向量数据库""知识片段"等技术词

【游客问题】
%s

请以灵山景区数字人导览员的身份回答：`, query)

	req := OpenAIRequest{
		Model: s.chatModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: "你是灵山胜境景区的AI数字人导览员「小灵」，熟悉灵山大佛、九龙灌浴、梵宫等所有景点。回答问题时要热情友好、专业准确，像真人导游一样自然亲切。",
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
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("JSON序列化失败: %v", err)
	}

	fmt.Printf("请求体长度: %d bytes\n", len(reqBody))

	apiURL := s.chatBaseURL + "/chat/completions"
	fmt.Printf("完整API URL: %s\n", apiURL)

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("创建HTTP请求失败: %v\n", err)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.chatAPIKey)

	fmt.Println("开始调用DeepSeek API...")
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		fmt.Printf("调用DeepSeek API失败: %v\n", err)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("调用DeepSeek API失败: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("API响应状态码: %d\n", resp.StatusCode)

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		fmt.Printf("读取响应体失败: %v\n", readErr)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("读取响应体失败: %v", readErr)
	}

	fmt.Printf("响应体长度: %d bytes\n", len(body))

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("API返回错误状态码: %d\n", resp.StatusCode)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("API返回错误状态码: %d, 响应长度: %d bytes", resp.StatusCode, len(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		fmt.Printf("解析API响应失败: %v\n", err)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("解析API响应失败: %v", err)
	}

	if openAIResp.Error != nil {
		fmt.Printf("DeepSeek API错误 - Type: %s, Code: %s, Message: %s\n",
			openAIResp.Error.Type, openAIResp.Error.Code, openAIResp.Error.Message)
		return "抱歉，我现在无法回答这个问题。", fmt.Errorf("DeepSeek API错误: %s - %s",
			openAIResp.Error.Code, openAIResp.Error.Message)
	}

	var answer string
	if len(openAIResp.Choices) > 0 && openAIResp.Choices[0].Message.Content != "" {
		answer = openAIResp.Choices[0].Message.Content
	} else {
		fmt.Println("API返回空结果")
		return "抱歉，我无法生成合适的回答。", fmt.Errorf("API返回空结果")
	}

	s.setCachedResponse(query, answer)
	fmt.Printf("API返回回答长度: %d\n", len([]rune(answer)))
	return answer, nil
}

func (s *RAGService) queryWithoutKnowledge(query string) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("queryWithoutKnowledge panic: %v\n", r)
		}
	}()

	fmt.Printf("\n--- 调用通用知识API ---\n")
	fmt.Printf("Fallback chat query length: %d\n", len([]rune(query)))
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
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	userPrompt := fmt.Sprintf(`你是灵山胜境景区的AI数字人导览员「小灵」。

【身份设定】
- 你是灵山胜境景区的官方数字人导览员
- 你熟悉灵山大佛、九龙灌浴、梵宫等所有景点
- 你的职责是帮助游客了解景区、规划行程、解答疑问

【回答策略】
1. 首先判断问题类型：
   - 如果问题与灵山胜境、无锡旅游、佛教文化直接相关，但当前知识库没有资料，请明确说明："当前灵山胜境知识库中没有检索到相关资料，不过我可以为您提供一些一般性的参考信息..."
   - 如果问题与灵山胜境无关，而是询问其他景点，请礼貌说明："我主要负责灵山胜境的导览服务。关于其他景区，建议您咨询相关景区的导览服务。"
   - 如果是完全无关的问题，礼貌地引导回到景区话题

2. 回答灵山相关问题时：
   - 重点介绍灵山大佛（88米高，世界著名青铜立佛）
   - 推荐九龙灌浴表演（精彩的佛教文化演出）
   - 介绍灵山梵宫（佛教艺术殿堂）
   - 提供游览建议和注意事项
   - 对于实时性信息（票价、营业时间等），请说明"具体信息请以景区官方公告为准"

3. 语言风格：
   - 称呼游客为"您"，保持礼貌尊重
   - 使用温暖、亲切的语气
   - 适当使用"欢迎来到灵山胜境"、"祝您游览愉快"等礼貌用语
   - 像真人导游一样自然、专业
   - 不要提到"RAG""向量数据库""知识片段"等技术词

【游客问题】
%s

请以灵山景区数字人导览员的身份回答：`, query)

	req := DashScopeRequest{
		Model: s.chatModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: "你是灵山胜境景区的AI数字人导览员「小灵」，熟悉灵山大佛、九龙灌浴、梵宫等所有景点。回答问题时要热情友好、专业准确，像真人导游一样自然亲切。",
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
		return "根据当前资料无法确认", nil
	}

	fmt.Printf("请求体长度: %d bytes\n", len(reqBody))

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
		fmt.Printf("API返回错误状态码: %d, 响应长度: %d bytes\n", resp.StatusCode, len(body))
		return "根据当前资料无法确认", nil
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("API响应体长度: %d bytes\n", len(body))

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
	} else {
		fmt.Println("API返回空结果")
		return "根据当前资料无法确认", nil
	}

	fmt.Printf("API返回回答长度: %d\n", len([]rune(answer)))
	return "[通用知识回答] " + answer, nil
}

func (s *RAGService) GenerateTourRoute(query string) *TourRoute {
	defaultRoutes := []TourRoute{
		{
			Name:        "经典文化之旅",
			Description: "建议从景区入口出发，依次参观灵山大照壁、五明桥、佛足坛、五智门，最后到达灵山大佛。我会重点讲解佛教文化和历史故事。",
			Duration:    "约2.5小时",
			Steps: []TourRouteStep{
				{Number: 1, Name: "灵山胜境入口", Desc: "起点"},
				{Number: 2, Name: "灵山大照壁", Desc: "华夏第一壁"},
				{Number: 3, Name: "五明桥", Desc: "佛教智慧象征"},
				{Number: 4, Name: "佛足坛", Desc: "祈福朝圣"},
				{Number: 5, Name: "五智门", Desc: "核心景区门户"},
				{Number: 6, Name: "灵山大佛", Desc: "世界最高佛立像"},
			},
		},
		{
			Name:        "禅意体验之旅",
			Description: "深入体验灵山禅意文化，从灵山梵宫开始，途经五印坛城、曼飞龙塔，最后到达拈花湾禅意小镇。",
			Duration:    "约3小时",
			Steps: []TourRouteStep{
				{Number: 1, Name: "灵山梵宫", Desc: "东方卢浮宫"},
				{Number: 2, Name: "五印坛城", Desc: "藏传佛教文化"},
				{Number: 3, Name: "曼飞龙塔", Desc: "南传佛教建筑"},
				{Number: 4, Name: "拈花广场", Desc: "禅意开篇"},
				{Number: 5, Name: "梵天花海", Desc: "自然禅意"},
				{Number: 6, Name: "香月花街", Desc: "禅意商业街"},
			},
		},
		{
			Name:        "亲子欢乐之旅",
			Description: "适合家庭出游的轻松路线，从九龙灌浴开始，途经百子戏弥勒、祥符禅寺，最后到灵山胜境儿童乐园。",
			Duration:    "约2小时",
			Steps: []TourRouteStep{
				{Number: 1, Name: "九龙灌浴", Desc: "动态喷泉表演"},
				{Number: 2, Name: "百子戏弥勒", Desc: "亲子互动"},
				{Number: 3, Name: "祥符禅寺", Desc: "千年古刹"},
				{Number: 4, Name: "佛教文化博览馆", Desc: "科普教育"},
				{Number: 5, Name: "灵山胜境儿童乐园", Desc: "亲子游乐"},
			},
		},
		{
			Name:        "深度历史之旅",
			Description: "探索灵山千年历史，从无尽意斋开始，了解赵朴初先生与灵山的渊源，最后到达祥符禅寺感受千年禅意。",
			Duration:    "约1.5小时",
			Steps: []TourRouteStep{
				{Number: 1, Name: "无尽意斋", Desc: "赵朴初纪念馆"},
				{Number: 2, Name: "降魔浮雕", Desc: "佛教故事"},
				{Number: 3, Name: "阿育王柱", Desc: "佛教文化象征"},
				{Number: 4, Name: "祥符禅寺", Desc: "千年禅宗祖庭"},
			},
		},
	}

	queryLower := strings.ToLower(query)

	if strings.Contains(queryLower, "亲子") || strings.Contains(queryLower, "儿童") || strings.Contains(queryLower, "家庭") {
		return &defaultRoutes[2]
	} else if strings.Contains(queryLower, "历史") || strings.Contains(queryLower, "文化") || strings.Contains(queryLower, "千年") {
		return &defaultRoutes[3]
	} else if strings.Contains(queryLower, "禅意") || strings.Contains(queryLower, "体验") || strings.Contains(queryLower, "拈花湾") {
		return &defaultRoutes[1]
	}

	return &defaultRoutes[0]
}

func (s *RAGService) QueryWithRAGAndRoute(query string) (string, *TourRoute, error) {
	response, err := s.QueryWithRAG(query)
	if err != nil {
		return "", nil, err
	}

	route := s.GenerateTourRoute(query)
	return response, route, nil
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
