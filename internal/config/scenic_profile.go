package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ScenicProfile 景区配置文件结构
type ScenicProfile struct {
	ID           string         `yaml:"id"`
	Name         string         `yaml:"name"`
	ShortName    string         `yaml:"short_name"`
	Description  string         `yaml:"description"`
	Hotline      string         `yaml:"hotline"`
	Slogan       string         `yaml:"slogan"`

	DigitalHuman DHProfile      `yaml:"digital_human"`
	Knowledge    KnowledgePaths `yaml:"knowledge"`
	Keywords     KeywordConfig  `yaml:"keywords"`
	Prompts      PromptConfig   `yaml:"prompts"`
	Routes       []RouteConfig  `yaml:"routes"`
	Map          MapConfig      `yaml:"map"`
}

type DHProfile struct {
	Name              string            `yaml:"name"`
	Greeting          string            `yaml:"greeting"`
	Personality       string            `yaml:"personality"`
	Live2DModel       string            `yaml:"live2d_model"`
	Live2DExpressions map[string]string `yaml:"live2d_expressions"`
	TTSVoice          string            `yaml:"tts_voice"`
	TTSSpeed          float64           `yaml:"tts_speed"`
	FallbackAvatar    string            `yaml:"fallback_avatar"`
}

type KnowledgePaths struct {
	ChunksFile    string `yaml:"chunks_file"`
	EvalQAFile    string `yaml:"eval_qa_file"`
	RealChunksFile string `yaml:"real_chunks_file"`
	RealEvalFile  string `yaml:"real_eval_file"`
}

type KeywordConfig struct {
	Spots             []string           `yaml:"spots"`
	SpotAliases       map[string]string  `yaml:"spot_aliases"`
	RelatedKeywords   []string           `yaml:"related_keywords"`
	RelatedPatterns   []string           `yaml:"related_patterns"`
	TopicEntities     []string           `yaml:"topic_entities"`
	QueryExpansion    []ExpansionRule    `yaml:"query_expansion"`
	IntentBoosts      []IntentBoostRule  `yaml:"intent_boosts"`
	ConditionalBoosts []ConditionalBoost `yaml:"conditional_boosts"`
}

type ConditionalBoost struct {
	QueryTerms   []string `yaml:"query_terms"`
	ContentTerms []string `yaml:"content_terms"`
	Boost        float64  `yaml:"boost"`
}

type ExpansionRule struct {
	Trigger []string `yaml:"trigger"`
	Expand  string   `yaml:"expand"`
}

type IntentBoostRule struct {
	QueryContains []string `yaml:"query_contains"`
	ChunkContains []string `yaml:"chunk_contains"`
	Topic         string   `yaml:"topic"`
	Boost         float64  `yaml:"boost"`
}

type PromptConfig struct {
	SystemRole        string            `yaml:"system_role"`
	ChatInstructions  string            `yaml:"chat_instructions"`
	GeneralChat       string            `yaml:"general_chat"`
	RAGPrompt         string            `yaml:"rag_prompt"`
	GeneralChatPrompt string            `yaml:"general_chat_prompt"`
	NoKnowledgePrompt string            `yaml:"no_knowledge_prompt"`
	FollowUpRewrite   map[string]string `yaml:"follow_up_rewrite"`
	FallbackAnswers   map[string]string `yaml:"fallback_answers"`
}

type RouteConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Spots       string `yaml:"spots"`
	Duration    int    `yaml:"duration"`
	Difficulty  string `yaml:"difficulty"`
	Rating      float64 `yaml:"rating"`
}

type MapConfig struct {
	Provider string      `yaml:"provider"`
	APIKey   string      `yaml:"api_key"`
	Center   [2]float64  `yaml:"center"`
	Zoom     int         `yaml:"zoom"`
	Style    string      `yaml:"style"`
}

// LoadScenicProfile 从 configs/scenic_profiles/ 加载景区配置
func LoadScenicProfile(scenicID string) (*ScenicProfile, error) {
	dir := filepath.Join("configs", "scenic_profiles")
	path := filepath.Join(dir, scenicID+".yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("加载景区配置失败 %s: %w", path, err)
	}

	// 仅替换白名单中的景区相关环境变量，避免泄露敏感信息
	content := os.Expand(string(data), func(key string) string {
		if strings.HasPrefix(key, "SCENIC_") {
			return os.Getenv(key)
		}
		return ""
	})

	var profile ScenicProfile
	if err := yaml.Unmarshal([]byte(content), &profile); err != nil {
		return nil, fmt.Errorf("解析景区配置失败 %s: %w", path, err)
	}

	if profile.ID == "" {
		profile.ID = scenicID
	}
	return &profile, nil
}

// ListScenicProfiles 列出所有可用景区
func ListScenicProfiles() ([]string, error) {
	dir := filepath.Join("configs", "scenic_profiles")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			ids = append(ids, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	return ids, nil
}

// RenderPrompt 将 prompt 模板中的变量替换为实际值
func (p *ScenicProfile) RenderPrompt(template string) string {
	result := template
	result = strings.ReplaceAll(result, "{scenic.name}", p.Name)
	result = strings.ReplaceAll(result, "{scenic.short_name}", p.ShortName)
	result = strings.ReplaceAll(result, "{scenic.description}", p.Description)
	result = strings.ReplaceAll(result, "{digital_human.name}", p.DigitalHuman.Name)
	result = strings.ReplaceAll(result, "{digital_human.personality}", p.DigitalHuman.Personality)
	result = strings.ReplaceAll(result, "{spots_list}", strings.Join(p.Keywords.Spots, "、"))
	return result
}

// GetSystemPrompt 获取渲染后的系统 prompt
func (p *ScenicProfile) GetSystemPrompt() string {
	return p.RenderPrompt(p.Prompts.SystemRole)
}

// GetChatInstructions 获取渲染后的回答指引
func (p *ScenicProfile) GetChatInstructions() string {
	return p.RenderPrompt(p.Prompts.ChatInstructions)
}

// GetGeneralChatPrefix 获取渲染后的通用对话前缀
func (p *ScenicProfile) GetGeneralChatPrefix() string {
	return p.RenderPrompt(p.Prompts.GeneralChat)
}

// GetRAGPrompt 获取渲染后的 RAG Prompt 模板
func (p *ScenicProfile) GetRAGPrompt() string {
	return p.Prompts.RAGPrompt
}

// GetGeneralChatPrompt 获取渲染后的通用 Chat Prompt
func (p *ScenicProfile) GetGeneralChatPrompt() string {
	return p.Prompts.GeneralChatPrompt
}

// GetNoKnowledgePrompt 获取渲染后的无知识兜底 Prompt
// 若未配置则 fallback 到 GeneralChatPrompt
func (p *ScenicProfile) GetNoKnowledgePrompt() string {
	if p.Prompts.NoKnowledgePrompt != "" {
		return p.Prompts.NoKnowledgePrompt
	}
	return p.Prompts.GeneralChatPrompt
}

// GetFallbackAnswer 根据问题关键词查找兜底答案
func (p *ScenicProfile) GetFallbackAnswer(query string) (string, bool) {
	if p.Prompts.FallbackAnswers == nil {
		return "", false
	}
	for key, answer := range p.Prompts.FallbackAnswers {
		if strings.Contains(query, key) {
			return p.RenderPrompt(answer), true
		}
	}
	return "", false
}
