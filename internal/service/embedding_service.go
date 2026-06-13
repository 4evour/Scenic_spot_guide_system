package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/scenic-guide/internal/config"
)

type EmbeddingProvider interface {
	GenerateEmbedding(text string) ([]float64, error)
	Name() string
	IsAvailable() bool
}

type QwenEmbeddingProvider struct {
	apiKey    string
	model     string
	baseURL   string
	available bool
	client    *http.Client
}

func NewQwenEmbeddingProvider(cfg *config.EmbeddingConfig) *QwenEmbeddingProvider {
	return &QwenEmbeddingProvider{
		apiKey:    cfg.APIKey,
		model:     cfg.Model,
		baseURL:   cfg.BaseURL,
		available: cfg.APIKey != "",
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConnsPerHost: 10,
			},
		},
	}
}

func (p *QwenEmbeddingProvider) Name() string {
	if strings.TrimSpace(p.model) != "" {
		return p.model
	}
	return "qwen-embedding"
}

func (p *QwenEmbeddingProvider) IsAvailable() bool {
	return p.available
}

func (p *QwenEmbeddingProvider) GenerateEmbedding(text string) ([]float64, error) {
	if !p.available {
		return nil, fmt.Errorf("Qwen embedding API is not available")
	}

	type Request struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}

	type Response struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	reqBody := Request{
		Model: p.model,
		Input: []string{text},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	apiURL := p.baseURL + "/embeddings"

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用embedding API失败: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := readLimitedBody(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", readErr)
	}

	var embResp Response
	if err := json.Unmarshal(bodyBytes, &embResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if embResp.Error.Message != "" {
		return nil, fmt.Errorf("embedding API错误: %s", embResp.Error.Message)
	}

	if len(embResp.Data) == 0 || len(embResp.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("未获取到embedding向量")
	}

	return embResp.Data[0].Embedding, nil
}

type BM25FallbackProvider struct {
	available bool

	// Corpus-level statistics for proper BM25 scoring
	docFreq    map[string]int     // token -> number of documents containing it
	docLen     map[string]float64 // docID -> document length (token count)
	avgDocLen  float64            // average document length across corpus
	totalDocs  int                // total number of documents
	bm25K1     float64            // term frequency saturation parameter (default 1.5)
	bm25B      float64            // length normalization parameter (default 0.75)
}

func NewBM25FallbackProvider() *BM25FallbackProvider {
	return &BM25FallbackProvider{
		available: true,
		docFreq:   make(map[string]int),
		docLen:    make(map[string]float64),
		bm25K1:    1.5,
		bm25B:     0.75,
	}
}

func (p *BM25FallbackProvider) Name() string {
	return "bm25-fallback"
}

func (p *BM25FallbackProvider) IsAvailable() bool {
	return p.available
}

type BM25Document struct {
	ID      string
	Content string
	Tokens  []string
}

func (p *BM25FallbackProvider) Tokenize(text string) []string {
	text = strings.ToLower(text)

	var tokens []string

	// 添加n-gram分词（2-gram和3-gram）来处理中文
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		// 2-gram
		if i+1 < len(runes) {
			tokens = append(tokens, string(runes[i:i+2]))
		}
		// 3-gram
		if i+2 < len(runes) {
			tokens = append(tokens, string(runes[i:i+3]))
		}
	}

	// 添加单个词
	stopWords := map[string]bool{
		"的": true, "了": true, "和": true, "是": true, "就": true, "都": true,
		"而": true, "及": true, "与": true, "着": true, "或": true, "一个": true,
		"没有": true, "我们": true, "你们": true, "他们": true, "什么": true,
		"怎么": true, "为什么": true, "因为": true, "所以": true, "但是": true,
		"在": true, "有": true, "这": true, "那": true, "到": true, "去": true,
		"来": true, "上": true, "下": true, "里": true, "中": true, "内": true,
		"外": true, "前": true, "后": true, "左": true, "右": true, "东": true,
		"西": true, "南": true, "北": true, "吗": true, "呢": true, "吧": true,
		"啊": true, "呀": true, "哦": true, "嗯": true, "哎": true, "哈": true,
	}

	re := regexp.MustCompile(`[\p{L}\p{N}]+`)
	matches := re.FindAllString(text, -1)
	for _, token := range matches {
		if !stopWords[token] && len(token) > 0 {
			tokens = append(tokens, token)
		}
	}

	return tokens
}

func (p *BM25FallbackProvider) CalculateScore(queryTokens, docTokens []string) float64 {
	if len(queryTokens) == 0 || len(docTokens) == 0 {
		return 0
	}

	querySet := make(map[string]int)
	for _, t := range queryTokens {
		querySet[t]++
	}

	docSet := make(map[string]int)
	for _, t := range docTokens {
		docSet[t]++
	}

	// Without corpus stats, use simplified scoring
	if p.totalDocs == 0 {
		score := 0.0
		for token, qf := range querySet {
			if df, ok := docSet[token]; ok {
				score += float64(qf * df)
			}
		}
		return score / float64(len(docTokens)+1)
	}

	// Standard BM25 scoring with IDF and document length normalization
	dl := float64(len(docTokens))
	normFactor := 1.0 - p.bm25B + p.bm25B*(dl/p.avgDocLen)

	score := 0.0
	for token, qf := range querySet {
		df, hasDF := p.docFreq[token]
		if !hasDF || df == 0 {
			continue
		}
		_, inDoc := docSet[token]
		if !inDoc {
			continue
		}

		// IDF: Robertson-Sparck Jones weight with +0.5 smoothing
		idf := math.Log((float64(p.totalDocs)-float64(df)+0.5)/(float64(df)+0.5) + 1.0)

		// TF with saturation and length normalization
		tf := float64(qf * docSet[token])
		tfNorm := (tf * (p.bm25K1 + 1)) / (tf + p.bm25K1*normFactor)

		score += idf * tfNorm
	}
	return score
}


// UpdateCorpusStats recomputes corpus-level statistics (IDF, avg doc length)
// from the token index and chunk data. Must be called after the knowledge cache changes.
func (p *BM25FallbackProvider) UpdateCorpusStats(tokenIndex map[string][]string, chunkTokens map[string][]string) {
	docFreq := make(map[string]int, len(tokenIndex))
	for token, chunkIDs := range tokenIndex {
		docFreq[token] = len(chunkIDs)
	}

	docLen := make(map[string]float64, len(chunkTokens))
	totalLen := 0.0
	for id, tokens := range chunkTokens {
		dl := float64(len(tokens))
		docLen[id] = dl
		totalLen += dl
	}
	avgDocLen := 0.0
	if len(chunkTokens) > 0 {
		avgDocLen = totalLen / float64(len(chunkTokens))
	}

	p.docFreq = docFreq
	p.docLen = docLen
	p.avgDocLen = avgDocLen
	p.totalDocs = len(chunkTokens)
}

// GetChunkTokens returns cached tokens for a chunk, or tokenizes on demand.
func (p *BM25FallbackProvider) GetChunkTokens(chunkID string, title, content string, cache map[string][]string) []string {
	if tokens, ok := cache[chunkID]; ok {
		return tokens
	}
	tokens := p.Tokenize(title + "\n" + content)
	cache[chunkID] = tokens
	return tokens
}

func (p *BM25FallbackProvider) GenerateEmbedding(text string) ([]float64, error) {
	tokens := p.Tokenize(text)
	scores := make(map[string]float64)
	for _, token := range tokens {
		scores[token]++
	}

	vector := make([]float64, 0, len(scores))
	for _, score := range scores {
		vector = append(vector, score)
	}

	return vector, nil
}

func (p *BM25FallbackProvider) CalculateSimilarity(queryTokens, docTokens []string) float64 {
	return p.CalculateScore(queryTokens, docTokens)
}
