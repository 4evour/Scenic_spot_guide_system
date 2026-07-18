package service

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/hashicorp/golang-lru/v2"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/repository"
	"golang.org/x/sync/singleflight"

	iconfig "github.com/scenic-guide/internal/config"
)

// newLRUWithOnError 创建一个容量受限的 LRU 缓存,构造失败时 panic(仅限启动期,表示 MaxCacheSize 非法)。
// 用 LRU 取代"满则全清空"的 map 策略,避免访问模式略超容量时持续 thrashing 导致命中率掉到 0。
func newLRUWithOnError[V any](name string, capacity int) *lru.Cache[string, V] {
	c, err := lru.New[string, V](capacity)
	if err != nil {
		panic(fmt.Sprintf("failed to create LRU cache %s: %v", name, err))
	}
	return c
}

const (
	MinSimilarityThreshold = 0.01
	TopK                   = 8
	CacheTTL               = 5 * time.Minute
	MaxCacheSize           = 1000
	MaxSessionIDLength     = 128
	MaxSessionHistorySize  = 1000
	MaxCachedTurns         = 10 // 内存缓存每会话最大轮数
	SessionHistoryTTL      = 30 * time.Minute
	SlowRequestThresholdMs = 5000
	EmbeddingCacheTTL      = 10 * time.Minute
)

type RetrievalMode string

const (
	RetrievalModeDefault        RetrievalMode = ""
	RetrievalModeBM25Local      RetrievalMode = "bm25-local"
	RetrievalModeEmbedding      RetrievalMode = "embedding"
	RetrievalModeHybridWeighted RetrievalMode = "hybrid-weighted"
	RetrievalModeRRFFusion      RetrievalMode = "rrf-fusion"
	RetrievalModeLightRerank    RetrievalMode = "light-rerank"
)

type RetrievalOptions struct {
	TopK                 int
	Mode                 RetrievalMode
	EmbeddingWeight      float64
	BM25Weight           float64
	RRFK                 float64
	SkipModelEnhancement bool
}

type retrievalScoredChunk struct {
	chunk      model.KnowledgeChunk
	similarity float64
}

type retrievalQueryExpansion struct {
	Original      string
	RetrievalText string
	AddedTerms    []string
}

type RAGTrace struct {
	TraceID        string      `json:"trace_id"`
	Provider       string      `json:"provider"`
	CacheHit       bool        `json:"cache_hit"`
	ChunkCount     int         `json:"chunk_count"`
	Sources        []RAGSource `json:"sources"`
	RetrievalMs    int64       `json:"retrieval_ms"`
	EmbeddingMs    int64       `json:"embedding_ms"`
	GenerationMs   int64       `json:"generation_ms"`
	TotalMs        int64       `json:"total_ms"`
	RetrievalMode  string      `json:"retrieval_mode"`
	RewrittenQuery string      `json:"rewritten_query,omitempty"`
	SlowRequest    bool        `json:"slow_request"`
	Confidence     float64     `json:"confidence"`
	ShouldAbstain  bool        `json:"should_abstain"`
}

type RAGSource struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	Source            string `json:"source"`
	KnowledgeCategory string `json:"knowledge_category,omitempty"`
	SpotID            uint   `json:"spot_id,omitempty"`
	SpotCategory      string `json:"spot_category,omitempty"`
	Preview           string `json:"preview,omitempty"`
}

// calculateAnswerEvidence derives a conservative evidence signal from local retrieval.
// It is intentionally independent from any model-reported confidence value.
func calculateAnswerEvidence(query string, sources []RAGSource) (float64, bool) {
	if len(sources) == 0 {
		return 0, true
	}

	maxRelevance := 0.0
	supportingSources := 0
	for _, source := range sources {
		relevance := sourceEvidenceRelevance(query, source)
		if relevance > maxRelevance {
			maxRelevance = relevance
		}
		if relevance >= 0.24 {
			supportingSources++
		}
	}

	if isBoundaryIntent(query) {
		return minFloat(0.2+maxRelevance*0.35, 0.45), true
	}
	if maxRelevance < 0.24 {
		return minFloat(0.15+maxRelevance, 0.45), true
	}

	confidence := 0.5 + maxRelevance*0.35 + float64(min(supportingSources-1, 2))*0.05
	return minFloat(confidence, 0.9), false
}

func calculateChunkEvidence(query string, chunks []model.KnowledgeChunk) (float64, bool) {
	if len(chunks) == 0 {
		return 0, true
	}

	sources := make([]RAGSource, 0, len(chunks))
	for _, chunk := range chunks {
		sources = append(sources, RAGSource{
			ID:                chunk.ID,
			Title:             strings.TrimSpace(chunk.Title),
			Source:            strings.TrimSpace(chunk.Source),
			KnowledgeCategory: strings.TrimSpace(chunk.KnowledgeCategory),
			SpotID:            chunk.SpotID,
			SpotCategory:      strings.TrimSpace(chunk.SpotCategory),
			Preview:           chunk.Content,
		})
	}
	return calculateAnswerEvidence(query, sources)
}

func sourceEvidenceRelevance(query string, source RAGSource) float64 {
	queryBigrams := meaningfulEvidenceBigrams(query)
	if len(queryBigrams) == 0 {
		return 0
	}

	evidenceText := strings.Join([]string{
		source.Title,
		source.Preview,
		source.KnowledgeCategory,
		source.SpotCategory,
	}, " ")
	evidenceBigrams := meaningfulEvidenceBigrams(evidenceText)
	matched := 0
	for term := range queryBigrams {
		if _, ok := evidenceBigrams[term]; ok {
			matched++
		}
	}
	if matched == 0 {
		return 0
	}

	relevance := float64(matched) / float64(len(queryBigrams))
	if evidenceIntentMatches(query, evidenceText) {
		relevance += 0.1
	}
	if containsAny(strings.ToLower(source.Source), []string{"official", "官方"}) {
		relevance += 0.03
	}
	return minFloat(relevance, 1)
}

func meaningfulEvidenceBigrams(text string) map[string]struct{} {
	normalized := []rune(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, text))
	result := make(map[string]struct{})
	ignored := map[string]struct{}{
		"什么": {}, "怎么": {}, "多少": {}, "哪里": {}, "是否": {}, "可以": {},
		"介绍": {}, "一下": {}, "这个": {}, "那个": {}, "有多": {}, "多高": {},
		"景区": {}, "景点": {}, "请问": {}, "的是": {}, "哪类": {}, "哪个": {},
		"如何": {},
	}
	for i := 0; i+1 < len(normalized); i++ {
		term := string(normalized[i : i+2])
		if _, skip := ignored[term]; !skip {
			result[term] = struct{}{}
		}
	}
	return result
}

func evidenceIntentMatches(query, evidence string) bool {
	switch {
	case containsAny(query, []string{"多高", "高度"}):
		return containsAny(evidence, []string{"通高", "高度", "米"})
	case containsAny(query, []string{"哪里", "在哪", "地址", "位置"}):
		return containsAny(evidence, []string{"位于", "地址", "位置", "地处"})
	case containsAny(query, []string{"路线", "怎么走", "怎么玩", "先看", "半天"}):
		return containsAny(evidence, []string{"路线", "游览", "步行", "先后", "行程"})
	case containsAny(query, []string{"消费", "占比", "客单价", "购物", "餐饮"}):
		return containsAny(evidence, []string{"消费", "占比", "客单价", "购物", "餐饮"})
	default:
		return false
	}
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

type CacheEntry struct {
	Response          string
	Sources           []RAGSource
	Confidence        float64
	ShouldAbstain     bool
	EvidenceEvaluated bool
	ExpireTime        time.Time
}

type embeddingCacheEntry struct {
	Vector     []float64
	ExpireTime time.Time
}

type TourRouteStep struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
}

type TourRoute struct {
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Steps            []TourRouteStep `json:"steps"`
	Duration         string          `json:"duration"`
	RouteType        string          `json:"route_type,omitempty"`
	Source           string          `json:"source,omitempty"`
	SourceURL        string          `json:"source_url,omitempty"`
	Confidence       float64         `json:"confidence,omitempty"`
	OfficialVerified bool            `json:"official_verified"`
}

func buildRAGSources(chunks []model.KnowledgeChunk, limit int) []RAGSource {
	if limit <= 0 || len(chunks) == 0 {
		return nil
	}
	if len(chunks) < limit {
		limit = len(chunks)
	}
	sources := make([]RAGSource, 0, limit)
	for _, chunk := range chunks {
		if len(sources) >= limit {
			break
		}
		sources = append(sources, RAGSource{
			ID:                chunk.ID,
			Title:             strings.TrimSpace(chunk.Title),
			Source:            strings.TrimSpace(chunk.Source),
			KnowledgeCategory: strings.TrimSpace(chunk.KnowledgeCategory),
			SpotID:            chunk.SpotID,
			SpotCategory:      strings.TrimSpace(chunk.SpotCategory),
			Preview:           buildRAGSourcePreview(chunk.Content, 96),
		})
	}
	return sources
}

func buildRAGSourcePreview(content string, limit int) string {
	preview := strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	if preview == "" || limit <= 0 {
		return preview
	}
	runes := []rune(preview)
	if len(runes) <= limit {
		return preview
	}
	return string(runes[:limit]) + "..."
}

// isScenicRelatedQuestion 判断问题是否与当前景区相关（配置化，支持任意景区）
func (s *RAGService) isScenicRelatedQuestion(query string) bool {
	if s.profile == nil {
		return false
	}
	queryLower := strings.ToLower(query)
	for _, keyword := range s.profile.Keywords.RelatedKeywords {
		if strings.Contains(queryLower, strings.ToLower(keyword)) {
			return true
		}
	}
	for _, pattern := range s.profile.Keywords.RelatedPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}
	return false
}

type RAGService struct {
	repo               *repository.KnowledgeRepository
	chatAPIKey         string
	chatModel          string
	chatBaseURL        string
	embedding          EmbeddingProvider
	bm25               *BM25FallbackProvider
	useBM25            bool
	uploadDir          string
	httpClient         *http.Client
	chatGuard          *modelGuard
	modelRequests      singleflight.Group
	queryCache         *lru.Cache[string, CacheEntry]
	embeddingCache     *lru.Cache[string, embeddingCacheEntry]
	knowledgeCache     []model.KnowledgeChunk
	tokenCache         map[string][]string
	tokenIndex         map[string][]string
	chunkByID          map[string]model.KnowledgeChunk
	sessionHistory     map[string][]sessionTurn
	cacheMutex         sync.RWMutex
	lastCacheTime      time.Time
	profile            *iconfig.ScenicProfile // 景区配置
	chatSessionService *ChatSessionService    // 会话持久化服务
}

type sessionTurn struct {
	Query    string
	Answer   string
	Topic    string
	Intent   string
	Boundary bool
	Updated  time.Time
}

type conversationContext struct {
	Topic    string
	Intent   string
	Boundary bool
}

func NewRAGService(repo *repository.KnowledgeRepository, chatAPIKey, chatModel, chatBaseURL string, embeddingProvider EmbeddingProvider, profile *iconfig.ScenicProfile) *RAGService {
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
		chatGuard:      newModelGuard("chat"),
		queryCache:     newLRUWithOnError[CacheEntry]("queryCache", MaxCacheSize),
		embeddingCache: newLRUWithOnError[embeddingCacheEntry]("embeddingCache", MaxCacheSize),
		knowledgeCache: nil,
		tokenCache:     make(map[string][]string),
		tokenIndex:     make(map[string][]string),
		chunkByID:      make(map[string]model.KnowledgeChunk),
		profile:        profile,
		sessionHistory: make(map[string][]sessionTurn),
		lastCacheTime:  time.Now(),
	}
}

func (s *RAGService) ModelHealth() []ModelProviderHealth {
	if s == nil {
		return nil
	}
	providers := make([]ModelProviderHealth, 0, 2)
	if !s.HasConfiguredLLM() {
		providers = append(providers, ModelProviderHealth{Provider: "chat", State: "disabled"})
	} else if s.chatGuard != nil {
		providers = append(providers, s.chatGuard.health())
	}
	if reporter, ok := s.embedding.(modelHealthReporter); ok {
		providers = append(providers, reporter.ModelHealth())
	}
	return providers
}

type modelHealthReporter interface {
	ModelHealth() ModelProviderHealth
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

	total, countErr := s.repo.Count()
	if countErr != nil {
		return nil, countErr
	}
	if total == 0 {
		s.knowledgeCache = []model.KnowledgeChunk{}
		return s.knowledgeCache, nil
	}

	if total <= 2000 {
		allChunks, err := s.repo.GetAll()
		if err != nil {
			return nil, err
		}
		s.knowledgeCache = allChunks
		s.rebuildBM25IndexLocked(allChunks)
		s.lastCacheTime = now
		return allChunks, nil
	}

	const batchSize = 1000
	allChunks := make([]model.KnowledgeChunk, 0, total)
	for page := 1; len(allChunks) < int(total); page++ {
		chunks, _, err := s.repo.List(page, batchSize, "", "")
		if err != nil {
			return nil, err
		}
		allChunks = append(allChunks, chunks...)
		if len(chunks) < batchSize {
			break
		}
	}

	s.knowledgeCache = allChunks
	s.rebuildBM25IndexLocked(allChunks)
	s.lastCacheTime = now
	return allChunks, nil
}

func (s *RAGService) getCachedEmbedding(text string) ([]float64, error) {
	s.cacheMutex.RLock()
	if entry, ok := s.embeddingCache.Get(text); ok && time.Now().Before(entry.ExpireTime) {
		s.cacheMutex.RUnlock()
		return entry.Vector, nil
	}
	s.cacheMutex.RUnlock()

	vec, err := s.embedding.GenerateEmbedding(text)
	if err != nil {
		return nil, err
	}

	s.cacheMutex.Lock()
	// LRU 容量到上限时自动驱逐最久未访问的条目,无需手动全清空。
	s.embeddingCache.Add(text, embeddingCacheEntry{Vector: vec, ExpireTime: time.Now().Add(EmbeddingCacheTTL)})
	s.cacheMutex.Unlock()

	return vec, nil
}

func (s *RAGService) getCachedResponse(query string) (CacheEntry, bool) {
	s.cacheMutex.RLock()
	entry, ok := s.queryCache.Get(query)
	s.cacheMutex.RUnlock()

	if !ok {
		return CacheEntry{}, false
	}

	if time.Now().After(entry.ExpireTime) {
		s.cacheMutex.Lock()
		s.queryCache.Remove(query)
		s.cacheMutex.Unlock()
		return CacheEntry{}, false
	}

	return entry, true
}

func (s *RAGService) setCachedResponse(query, response string, sources []RAGSource) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// LRU 容量到上限时自动驱逐最久未访问的条目,无需手动全清空。
	s.queryCache.Add(query, CacheEntry{
		Response:   response,
		Sources:    sources,
		ExpireTime: time.Now().Add(CacheTTL),
	})
}

func (s *RAGService) setCachedRAGResponse(query, response string, sources []RAGSource, confidence float64, shouldAbstain bool) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.queryCache.Add(query, CacheEntry{
		Response:          response,
		Sources:           sources,
		Confidence:        confidence,
		ShouldAbstain:     shouldAbstain,
		EvidenceEvaluated: true,
		ExpireTime:        time.Now().Add(CacheTTL),
	})
}

type ChunkData struct {
	ID                string                 `json:"id"`
	Content           string                 `json:"content"`
	Source            string                 `json:"source"`
	Title             string                 `json:"title"`
	KnowledgeCategory string                 `json:"knowledge_category"`
	SpotID            uint                   `json:"spot_id"`
	SpotCategory      string                 `json:"spot_category"`
	Metadata          map[string]interface{} `json:"metadata"`
}

type KnowledgeUpsertInput struct {
	ID                string                 `json:"id"`
	Title             string                 `json:"title"`
	Source            string                 `json:"source"`
	Content           string                 `json:"content"`
	KnowledgeCategory string                 `json:"knowledge_category"`
	SpotID            uint                   `json:"spot_id"`
	SpotCategory      string                 `json:"spot_category"`
	Metadata          map[string]interface{} `json:"metadata"`
}

func (s *RAGService) invalidateKnowledgeCaches() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.queryCache.Purge()
	s.knowledgeCache = nil
	s.tokenCache = make(map[string][]string)
	s.tokenIndex = make(map[string][]string)
	s.chunkByID = make(map[string]model.KnowledgeChunk)
	s.lastCacheTime = time.Time{}
}

func normalizeKnowledgeChunk(chunk *ChunkData) {
	chunk.ID = strings.TrimSpace(chunk.ID)
	chunk.Title = strings.TrimSpace(chunk.Title)
	chunk.Source = strings.TrimSpace(chunk.Source)
	chunk.Content = strings.TrimSpace(chunk.Content)
	chunk.KnowledgeCategory = strings.TrimSpace(chunk.KnowledgeCategory)
	chunk.SpotCategory = strings.TrimSpace(chunk.SpotCategory)

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
	if chunk.KnowledgeCategory == "" {
		chunk.KnowledgeCategory = firstStringMetadata(chunk.Metadata, "knowledge_category", "category", "type")
	}
	if chunk.SpotCategory == "" {
		chunk.SpotCategory = firstStringMetadata(chunk.Metadata, "spot_category")
	}
	if chunk.SpotID == 0 {
		chunk.SpotID = uintFromMetadata(chunk.Metadata, "spot_id")
	}
	if chunk.KnowledgeCategory != "" {
		chunk.Metadata["knowledge_category"] = chunk.KnowledgeCategory
		chunk.Metadata["category"] = chunk.KnowledgeCategory
	}
	if chunk.SpotCategory != "" {
		chunk.Metadata["spot_category"] = chunk.SpotCategory
	}
	if chunk.SpotID > 0 {
		chunk.Metadata["spot_id"] = chunk.SpotID
	}
}

func firstStringMetadata(metadata map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := metadata[key]; ok {
			if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func uintFromMetadata(metadata map[string]interface{}, key string) uint {
	val, ok := metadata[key]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case float64:
		if v > 0 {
			return uint(v)
		}
	case int:
		if v > 0 {
			return uint(v)
		}
	case string:
		var parsed uint64
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
			return uint(parsed)
		}
	}
	return 0
}

func (s *RAGService) getRelatedKeywords() []string {
	if s.profile == nil {
		return nil
	}
	return s.profile.Keywords.RelatedKeywords
}

// getTopicEntities 从 profile 获取景点实体列表，nil-safe
func (s *RAGService) getTopicEntities() []string {
	if s.profile == nil {
		return nil
	}
	return s.profile.Keywords.TopicEntities
}

// getConditionalBoosts 从 profile 获取条件加分规则，nil-safe
func (s *RAGService) getConditionalBoosts() []iconfig.ConditionalBoost {
	if s.profile == nil {
		return nil
	}
	return s.profile.Keywords.ConditionalBoosts
}

// getSystemPromptOrDefault 获取系统 prompt，优先使用 profile 配置
func (s *RAGService) getSystemPromptOrDefault(lang string) string {
	base := "你是一位专业的景区数字人导览员，负责为游客提供导览服务。回答要热情友好、准确专业。\n\n" +
		"在回答时，请在文本适当位置插入情感标签，用于驱动虚拟形象的面部表情。可用标签：[joy]、[sadness]、[surprise]、[anger]、[fear]、[disgust]、[neutral]。\n" +
		"例如：\"[joy]欢迎来到灵山景区！灵山大佛高88米，非常壮观。[surprise]值得一提的是，这里还有一项吉尼斯世界纪录。\"\n" +
		"标签放置原则：\n" +
		"- [joy] 放在欢迎、赞美、推荐等热情语句前\n" +
		"- [sadness] 放在道歉、遗憾、无法满足要求处\n" +
		"- [surprise] 放在提醒、亮点介绍、有趣事实前\n" +
		"- [anger] 很少使用，仅在回应用户明显不满时\n" +
		"- [fear] 放在安全提醒、注意事项前\n" +
		"- [disgust] 几乎不使用\n" +
		"- [neutral] 放在纯事实陈述前\n"
	if s.profile != nil && s.profile.Prompts.SystemRole != "" {
		base = s.profile.GetSystemPrompt()
	}
	// 追加语言指令：强制 LLM 用指定语言回答
	if lang == "en-US" {
		base += "\n\nYou MUST respond in English. Use only English in your answer."
	} else if lang != "" && lang != "zh-CN" {
		base += fmt.Sprintf("\n\nYou MUST respond in %s. Use only that language in your answer.", lang)
	}
	return base
}

func (s *RAGService) GenerateTourRoute(query string) *TourRoute {
	routes := s.getTourRoutes()
	if len(routes) == 0 {
		return nil
	}

	queryLower := strings.ToLower(query)

	if strings.Contains(queryLower, "观光车") || strings.Contains(queryLower, "摆渡车") {
		for _, r := range routes {
			if r.RouteType == "sightseeing_bus" {
				return &r
			}
		}
	} else if strings.Contains(queryLower, "错峰") || strings.Contains(queryLower, "避开人流") || strings.Contains(queryLower, "人多") {
		for _, r := range routes {
			if r.RouteType == "off_peak_walking" {
				return &r
			}
		}
	} else if strings.Contains(queryLower, "官方") || strings.Contains(queryLower, "完整路线") || strings.Contains(queryLower, "第一次") {
		for _, r := range routes {
			if r.RouteType == "reference_walking" {
				return &r
			}
		}
	} else if strings.Contains(queryLower, "亲子") || strings.Contains(queryLower, "儿童") || strings.Contains(queryLower, "家庭") {
		for _, r := range routes {
			if strings.Contains(r.Name, "亲子") {
				return &r
			}
		}
	} else if strings.Contains(queryLower, "历史") || strings.Contains(queryLower, "千年") {
		for _, r := range routes {
			if strings.Contains(r.Name, "深度") || strings.Contains(r.Name, "历史") {
				return &r
			}
		}
	} else if strings.Contains(queryLower, "禅意") || strings.Contains(queryLower, "体验") || strings.Contains(queryLower, "拈花湾") {
		for _, r := range routes {
			if strings.Contains(r.Name, "禅意") {
				return &r
			}
		}
	} else if strings.Contains(queryLower, "文化") {
		for _, r := range routes {
			if strings.Contains(r.Name, "经典") || strings.Contains(r.Name, "文化") {
				return &r
			}
		}
	}

	// 默认返回第一条路线
	return &routes[0]
}

// getTourRoutes 从景区配置加载路线，转换为 API 格式
func (s *RAGService) getTourRoutes() []TourRoute {
	if s.profile == nil || len(s.profile.Routes) == 0 {
		return nil
	}
	routes := make([]TourRoute, 0, len(s.profile.Routes))
	for _, rc := range s.profile.Routes {
		routes = append(routes, routeConfigToTourRoute(rc))
	}
	return routes
}

// routeConfigToTourRoute 将配置中的路线转换为 API 响应格式
func routeConfigToTourRoute(rc iconfig.RouteConfig) TourRoute {
	spots := strings.Split(rc.Spots, ",")
	steps := make([]TourRouteStep, len(spots))
	for i, spot := range spots {
		steps[i] = TourRouteStep{
			Number: i + 1,
			Name:   strings.TrimSpace(spot),
		}
	}

	duration := fmt.Sprintf("约%d分钟", rc.Duration)
	if rc.Duration >= 60 {
		hours := rc.Duration / 60
		mins := rc.Duration % 60
		if mins == 0 {
			duration = fmt.Sprintf("约%d小时", hours)
		} else {
			duration = fmt.Sprintf("约%d小时%d分钟", hours, mins)
		}
	}

	return TourRoute{
		Name:             rc.Name,
		Description:      rc.Description,
		Steps:            steps,
		Duration:         duration,
		RouteType:        rc.RouteType,
		Source:           rc.Source,
		SourceURL:        rc.SourceURL,
		Confidence:       rc.Confidence,
		OfficialVerified: rc.OfficialVerified,
	}
}

func (s *RAGService) QueryWithRAGAndRoute(ctx context.Context, query, lang string) (string, *TourRoute, error) {
	response, err := s.QueryWithRAG(ctx, query)
	if err != nil {
		return "", nil, err
	}

	route := s.GenerateTourRoute(query)
	return response, route, nil
}

func (s *RAGService) QueryWithRAGAndRouteInSession(ctx context.Context, sessionID, query, lang string) (string, *TourRoute, error) {
	response, _, err := s.QueryWithRAGTraceInSession(ctx, sessionID, query, lang)
	if err != nil {
		return "", nil, err
	}

	route := s.GenerateTourRoute(query)
	return response, route, nil
}

func (s *RAGService) QueryWithRAGAndRouteTraceInSession(ctx context.Context, sessionID, query, lang string) (string, *TourRoute, RAGTrace, error) {
	response, trace, err := s.QueryWithRAGTraceInSession(ctx, sessionID, query, lang)
	if err != nil {
		return "", nil, trace, err
	}

	route := s.GenerateTourRoute(query)
	return response, route, trace, nil
}

// QueryWithRAGStreaming performs RAG retrieval then streams the LLM response via onToken callback.
// Returns the full answer, tour route, trace, and error.
func (s *RAGService) QueryWithRAGStreaming(ctx context.Context, sessionID, query, lang string, onToken func(string)) (string, *TourRoute, RAGTrace, error) {
	sessionID = normalizeSessionID(sessionID)
	totalStart := time.Now()
	defer func() {
		pkg.RecordRAGQueryDuration(time.Since(totalStart).Seconds())
	}()

	// 1. 追问改写
	retrievalQuery := query
	if sessionID != "" {
		s.cacheMutex.RLock()
		_, hasHistory := s.sessionHistory[sessionID]
		s.cacheMutex.RUnlock()
		if !hasHistory {
			s.LoadSessionHistoryFromDB(sessionID)
		}
		retrievalQuery = s.RewriteFollowUpQuery(sessionID, query)
	}

	sessionContext := ""
	if sessionID != "" {
		sessionContext = s.buildSessionContextText(sessionID, query)
	}

	// 2. 检索 + prompt 构建（复用现有逻辑）
	trace := RAGTrace{
		TraceID:       fmt.Sprintf("rag-%d", time.Now().UnixNano()),
		Provider:      map[bool]string{true: "bm25-local", false: "embedding"}[s.useBM25],
		RetrievalMode: string(normalizeRetrievalMode(RetrievalModeDefault, s.embedding != nil && s.embedding.IsAvailable(), s.useBM25)),
	}
	if retrievalQuery != query {
		trace.RewrittenQuery = retrievalQuery
	}
	if casualAnswer, ok := s.buildCasualAnswer(query, lang); ok {
		trace.Provider = "local-conversation"
		trace.RetrievalMode = "conversation"
		trace.Confidence = 0.95
		trace.ShouldAbstain = false
		generationStart := time.Now()
		answer := casualAnswer
		trace.GenerationMs = time.Since(generationStart).Milliseconds()
		trace.TotalMs = time.Since(totalStart).Milliseconds()
		trace.SlowRequest = trace.TotalMs > SlowRequestThresholdMs
		if onToken != nil {
			onToken(answer)
		}
		return answer, nil, trace, nil
	}
	retrievalStart := time.Now()
	chunks, err := s.RetrieveRelevantKnowledge(retrievalQuery, TopK)
	trace.RetrievalMs = time.Since(retrievalStart).Milliseconds()
	trace.ChunkCount = len(chunks)
	trace.Sources = buildRAGSources(chunks, 3)
	trace.Confidence, trace.ShouldAbstain = calculateChunkEvidence(retrievalQuery+" "+query, chunks)
	if err != nil {
		trace.TotalMs = time.Since(totalStart).Milliseconds()
		return "", nil, trace, fmt.Errorf("检索相关知识失败: %v", err)
	}

	var answer string

	if len(chunks) == 0 || (trace.ShouldAbstain && !isBoundaryIntent(query)) {
		// 无知识命中时拒绝无依据生成，避免通用模型编造景区事实。
		genStart := time.Now()
		answer = addEmotionCare(query, s.buildNoEvidenceAnswer(lang))
		if onToken != nil {
			onToken(answer)
		}
		trace.GenerationMs = time.Since(genStart).Milliseconds()
	} else if s.chatAPIKey == "" {
		// 无 API key，使用本地生成
		answer = s.generateAnswerFromChunksWithContext(query, chunks, sessionContext)
		if onToken != nil {
			onToken(answer)
		}
	} else {
		// 有知识 + 有 API key，流式 LLM
		prompt := s.BuildRAGPromptWithContext(query, chunks, sessionContext)
		systemPrompt := s.getSystemPromptOrDefault(lang)
		genStart := time.Now()
		answer, err = s.CallLLMStreaming(ctx, systemPrompt, prompt, onToken)
		trace.GenerationMs = time.Since(genStart).Milliseconds()
		if err != nil && strings.TrimSpace(answer) == "" {
			slog.Warn("流式 Chat 调用失败，使用本地知识库降级", "error", err, "query_len", len([]rune(query)))
			fallbackStart := time.Now()
			answer = s.generateAnswerFromChunksWithContext(query, chunks, sessionContext)
			trace.GenerationMs += time.Since(fallbackStart).Milliseconds()
			trace.Provider = "local-rag-fallback"
			if onToken != nil {
				onToken(answer)
			}
			err = nil
		}
	}

	if err != nil {
		trace.TotalMs = time.Since(totalStart).Milliseconds()
		return "", nil, trace, err
	}

	trace.TotalMs = time.Since(totalStart).Milliseconds()
	trace.SlowRequest = trace.TotalMs > SlowRequestThresholdMs

	route := s.GenerateTourRoute(query)
	return answer, route, trace, nil
}

func (s *RAGService) RunEvaluation(evalFile string) error {
	report, err := s.EvaluateFile(evalFile)
	if err != nil {
		return err
	}

	fmt.Println("\n========== RAG系统评估测试 ==========")
	for i, result := range report.Results {
		status := "✅"
		if !result.Passed {
			status = "⚠️ "
		}

		fmt.Printf("[%d/%d] %s %s\n", i+1, report.Total, status, result.Question)
		if result.Error != "" {
			fmt.Printf("  错误: %s\n", result.Error)
		}
		if len(result.MissingKeywords) > 0 {
			fmt.Printf("  缺失关键词: %v\n", result.MissingKeywords)
		}
		if result.ResponsePreview != "" {
			fmt.Printf("  回答: %s\n\n", result.ResponsePreview)
		}
	}

	fmt.Printf("========== 测试完成: %d/%d 通过，关键词平均覆盖率 %.1f%% ==========\n",
		report.Passed,
		report.Total,
		report.AverageKeywordCoverage*100,
	)
	return nil
}
