package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"

	"github.com/scenic-guide/internal/config"
)

type EmbeddingProvider interface {
	GenerateEmbedding(text string) ([]float64, error)
	Name() string
	IsAvailable() bool
}

type QwenEmbeddingProvider struct {
	apiKey string
	model  string
	baseURL string
	available bool
}

func NewQwenEmbeddingProvider(cfg *config.EmbeddingConfig) *QwenEmbeddingProvider {
	return &QwenEmbeddingProvider{
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		baseURL:  cfg.BaseURL,
		available: cfg.APIKey != "",
	}
}

func (p *QwenEmbeddingProvider) Name() string {
	return "qwen-embedding-v4"
}

func (p *QwenEmbeddingProvider) IsAvailable() bool {
	return p.available
}

func (p *QwenEmbeddingProvider) GenerateEmbedding(text string) ([]float64, error) {
	if !p.available {
		return nil, fmt.Errorf("Qwen embedding API is not available")
	}

	type Request struct {
		Model string `json:"model"`
		Input string `json:"input"`
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
		Input: text,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	req, err := http.NewRequest("POST", p.baseURL+"/embeddings", bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用embedding API失败: %v", err)
	}
	defer resp.Body.Close()

	var embResp Response
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(embResp.Error.Message) > 0 {
		return nil, fmt.Errorf("embedding API错误: %s", embResp.Error.Message)
	}

	if len(embResp.Data) == 0 || len(embResp.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("未获取到embedding向量")
	}

	return embResp.Data[0].Embedding, nil
}

type BM25FallbackProvider struct {
	available bool
}

func NewBM25FallbackProvider() *BM25FallbackProvider {
	return &BM25FallbackProvider{
		available: true,
	}
}

func (p *BM25FallbackProvider) Name() string {
	return "bm25-fallback"
}

func (p *BM25FallbackProvider) IsAvailable() bool {
	return p.available
}

type BM25Document struct {
	ID     string
	Content string
	Tokens []string
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
	querySet := make(map[string]int)
	for _, t := range queryTokens {
		querySet[t]++
	}

	docSet := make(map[string]int)
	for _, t := range docTokens {
		docSet[t]++
	}

	score := 0.0
	for token, qf := range querySet {
		if df, ok := docSet[token]; ok {
			// 对较长的token给予更高权重
			tokenLen := float64(len(token))
			weight := 1.0
			if tokenLen >= 3 {
				weight = 2.0
			} else if tokenLen == 2 {
				weight = 1.5
			}
			score += float64(qf*df) * weight
		}
	}

	// 使用更稳定的归一化
	if len(docTokens) == 0 {
		return 0
	}
	
	return score / (1.0 + math.Log10(float64(len(docTokens)+1)))
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
