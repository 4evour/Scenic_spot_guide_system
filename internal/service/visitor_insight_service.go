package service

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"gorm.io/gorm"
)

type VisitorInsightService struct {
	db            *gorm.DB
	ragService    *RAGService
	sessionRepo   repository.ChatSessionRepository
	messageRepo   repository.ChatMessageRepository
	knowledgeRepo *repository.KnowledgeRepository
}

type KnowledgeCandidateApprovalInput struct {
	Title             string `json:"title"`
	Content           string `json:"content"`
	KnowledgeCategory string `json:"knowledge_category"`
	SpotID            uint   `json:"spot_id"`
	SpotCategory      string `json:"spot_category"`
}

type insightLLMResult struct {
	Summary           string   `json:"summary"`
	SatisfactionScore int      `json:"satisfaction_score"`
	NegativeReasons   []string `json:"negative_reasons"`
	AttentionPoints   []string `json:"attention_points"`
	Candidates        []struct {
		Title             string `json:"title"`
		Content           string `json:"content"`
		KnowledgeCategory string `json:"knowledge_category"`
		SpotID            uint   `json:"spot_id"`
		SpotCategory      string `json:"spot_category"`
	} `json:"candidates"`
}

func NewVisitorInsightService(db *gorm.DB, ragService *RAGService) *VisitorInsightService {
	return &VisitorInsightService{
		db:            db,
		ragService:    ragService,
		sessionRepo:   repository.NewChatSessionRepository(db),
		messageRepo:   repository.NewChatMessageRepository(db),
		knowledgeRepo: repository.NewKnowledgeRepository(db),
	}
}

func (s *VisitorInsightService) SaveFeedback(feedback *model.UserFeedback) error {
	if feedback == nil {
		return fmt.Errorf("feedback cannot be nil")
	}
	if feedback.Rating < 0 {
		feedback.Rating = 0
	}
	if feedback.Rating > 5 {
		feedback.Rating = 5
	}
	return s.db.Create(feedback).Error
}

func (s *VisitorInsightService) AnalyzeSession(sessionID string) (*model.VisitorInsightAnalysis, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id 不能为空")
	}
	if s.ragService == nil || !s.ragService.HasConfiguredLLM() {
		return nil, fmt.Errorf("AI API Key 未配置，无法执行满意度分析")
	}
	session, err := s.sessionRepo.FindBySessionID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("会话不存在: %w", err)
	}
	messages, err := s.messageRepo.GetRecentBySession(session.ID, 80)
	if err != nil {
		return nil, fmt.Errorf("读取会话消息失败: %w", err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("会话暂无可分析消息")
	}

	prompt := buildInsightPrompt(messages)
	raw, err := s.ragService.CallLLM(
		"你是景区数字人运营分析助手。只输出严格 JSON，不要输出 Markdown。",
		prompt,
	)
	if err != nil {
		return nil, err
	}
	result, err := parseInsightLLMResult(raw)
	if err != nil {
		return nil, err
	}
	negativeJSON, _ := json.Marshal(result.NegativeReasons)
	attentionJSON, _ := json.Marshal(result.AttentionPoints)
	analysis := &model.VisitorInsightAnalysis{
		UserID:            session.UserID,
		SessionID:         sessionID,
		Summary:           strings.TrimSpace(result.Summary),
		SatisfactionScore: clampScore(result.SatisfactionScore),
		NegativeReasons:   string(negativeJSON),
		AttentionPoints:   string(attentionJSON),
		RawResult:         raw,
		Status:            "completed",
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(analysis).Error; err != nil {
			return err
		}
		for _, item := range result.Candidates {
			title := strings.TrimSpace(item.Title)
			content := strings.TrimSpace(item.Content)
			if title == "" || content == "" {
				continue
			}
			candidate := &model.KnowledgeCandidate{
				AnalysisID:        analysis.ID,
				SessionID:         sessionID,
				Title:             title,
				Content:           content,
				Source:            "chat-insight",
				KnowledgeCategory: strings.TrimSpace(item.KnowledgeCategory),
				SpotID:            item.SpotID,
				SpotCategory:      strings.TrimSpace(item.SpotCategory),
				Status:            "pending",
			}
			if err := tx.Create(candidate).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return analysis, nil
}

func (s *VisitorInsightService) ListAnalyses(page, pageSize int) ([]model.VisitorInsightAnalysis, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := s.db.Model(&model.VisitorInsightAnalysis{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VisitorInsightAnalysis
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *VisitorInsightService) ListCandidates(status string, page, pageSize int) ([]model.KnowledgeCandidate, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := s.db.Model(&model.KnowledgeCandidate{})
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.KnowledgeCandidate
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *VisitorInsightService) ApproveCandidate(id uint, input KnowledgeCandidateApprovalInput) (*model.KnowledgeChunk, error) {
	var candidate model.KnowledgeCandidate
	if err := s.db.First(&candidate, id).Error; err != nil {
		return nil, err
	}
	if candidate.Status == "approved" {
		return nil, fmt.Errorf("候选知识已入库")
	}
	if strings.TrimSpace(input.Title) != "" {
		candidate.Title = strings.TrimSpace(input.Title)
	}
	if strings.TrimSpace(input.Content) != "" {
		candidate.Content = strings.TrimSpace(input.Content)
	}
	if strings.TrimSpace(input.KnowledgeCategory) != "" {
		candidate.KnowledgeCategory = strings.TrimSpace(input.KnowledgeCategory)
	}
	if input.SpotID > 0 {
		candidate.SpotID = input.SpotID
	}
	if strings.TrimSpace(input.SpotCategory) != "" {
		candidate.SpotCategory = strings.TrimSpace(input.SpotCategory)
	}
	metadata := map[string]interface{}{
		"knowledge_category": candidate.KnowledgeCategory,
		"category":           candidate.KnowledgeCategory,
		"spot_id":            candidate.SpotID,
		"spot_category":      candidate.SpotCategory,
		"source":             "chat-insight",
	}
	var created *model.KnowledgeChunk
	err := s.db.Transaction(func(tx *gorm.DB) error {
		chunk, err := s.ragService.CreateKnowledge(KnowledgeUpsertInput{
			ID:                knowledgeIDFromCandidate(candidate),
			Title:             candidate.Title,
			Source:            candidate.Source,
			Content:           candidate.Content,
			KnowledgeCategory: candidate.KnowledgeCategory,
			SpotID:            candidate.SpotID,
			SpotCategory:      candidate.SpotCategory,
			Metadata:          metadata,
		})
		if err != nil {
			return err
		}
		created = chunk
		now := time.Now()
		candidate.Status = "approved"
		candidate.ApprovedAt = &now
		return tx.Save(&candidate).Error
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *VisitorInsightService) RejectCandidate(id uint, reason string) error {
	return s.db.Model(&model.KnowledgeCandidate{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        "rejected",
		"reject_reason": strings.TrimSpace(reason),
	}).Error
}

func buildInsightPrompt(messages []model.ChatMessage) string {
	var b strings.Builder
	b.WriteString("请分析以下游客与景区数字人的聊天记录，输出 JSON：")
	b.WriteString(`{"summary":"","satisfaction_score":0,"negative_reasons":[],"attention_points":[],"candidates":[{"title":"","content":"","knowledge_category":"","spot_id":0,"spot_category":""}]}`)
	b.WriteString("\n评分范围 0-100；candidates 只放适合沉淀到知识库的事实性内容，不能编造。\n\n")
	for _, msg := range messages {
		role := msg.Role
		if role == "" {
			role = "unknown"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(sanitizeInsightText(msg.Content))
		b.WriteString("\n")
	}
	return b.String()
}

var (
	phonePattern = regexp.MustCompile(`1[3-9]\d{9}`)
	emailPattern = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	idPattern    = regexp.MustCompile(`\d{17}[\dXx]`)
)

func sanitizeInsightText(text string) string {
	text = phonePattern.ReplaceAllString(text, "[手机号]")
	text = emailPattern.ReplaceAllString(text, "[邮箱]")
	text = idPattern.ReplaceAllString(text, "[身份证]")
	return text
}

func parseInsightLLMResult(raw string) (insightLLMResult, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end >= start {
		raw = raw[start : end+1]
	}
	var result insightLLMResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return result, fmt.Errorf("AI 分析 JSON 解析失败: %w", err)
	}
	result.SatisfactionScore = clampScore(result.SatisfactionScore)
	return result, nil
}

func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func knowledgeIDFromCandidate(candidate model.KnowledgeCandidate) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%d\n%s\n%s", candidate.ID, candidate.Title, candidate.Content)))
	return fmt.Sprintf("chat-candidate-%x", sum[:8])
}
