package service

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

var registerVisitorInsightDriver sync.Once

func newVisitorInsightDB(t *testing.T) *gorm.DB {
	t.Helper()
	const driverName = "modernc-visitor-insight-test"
	registerVisitorInsightDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func TestAnalyzeSessionRequiresConfiguredLLM(t *testing.T) {
	db := newVisitorInsightDB(t)
	rag := NewRAGService(repository.NewKnowledgeRepository(db), "", "", "", nil, nil)
	svc := NewVisitorInsightService(db, rag)

	if _, err := svc.AnalyzeSession("missing"); err == nil {
		t.Fatal("expected missing AI configuration error")
	}
}

func TestAnalyzeSessionSendsSanitizedMessagesToLLM(t *testing.T) {
	db := newVisitorInsightDB(t)
	sessionRepo := repository.NewChatSessionRepository(db)
	messageRepo := repository.NewChatMessageRepository(db)
	sessionSvc := NewChatSessionService(sessionRepo, messageRepo)
	if err := sessionSvc.AddMessages("s1", 2, "我的手机号是13812345678，邮箱a@test.com，门票贵吗？", "可以查看官方票务。", "neutral", 1200); err != nil {
		t.Fatalf("seed messages: %v", err)
	}

	var captured string
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		for _, msg := range req.Messages {
			captured += msg.Content
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"summary\":\"游客关注票务价格\",\"satisfaction_score\":62,\"negative_reasons\":[\"价格疑虑\"],\"attention_points\":[\"票务咨询\"],\"candidates\":[{\"title\":\"门票咨询\",\"content\":\"游客常问门票价格与优惠政策。\",\"knowledge_category\":\"游客 FAQ\",\"spot_category\":\"服务设施\"}]}"}}]}`))
	}))
	defer llm.Close()

	rag := NewRAGService(repository.NewKnowledgeRepository(db), "key", "test-model", llm.URL, nil, nil)
	svc := NewVisitorInsightService(db, rag)
	analysis, err := svc.AnalyzeSession("s1")
	if err != nil {
		t.Fatalf("AnalyzeSession: %v", err)
	}
	if strings.Contains(captured, "13812345678") || strings.Contains(captured, "a@test.com") {
		t.Fatalf("PII was not sanitized before LLM call: %s", captured)
	}
	if analysis.SatisfactionScore != 62 {
		t.Fatalf("score = %d, want 62", analysis.SatisfactionScore)
	}
	candidates, _, err := svc.ListCandidates("pending", 1, 20)
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Title != "门票咨询" {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
}

func TestApproveKnowledgeCandidateCreatesKnowledgeChunk(t *testing.T) {
	db := newVisitorInsightDB(t)
	rag := NewRAGService(repository.NewKnowledgeRepository(db), "key", "test-model", "http://example.invalid", nil, nil)
	svc := NewVisitorInsightService(db, rag)
	candidate := &model.KnowledgeCandidate{
		Title:             "老人服务",
		Content:           "游客常问轮椅租借和无障碍路线。",
		KnowledgeCategory: "游客 FAQ",
		SpotID:            3,
		SpotCategory:      "服务设施",
		Status:            "pending",
	}
	if err := db.Create(candidate).Error; err != nil {
		t.Fatalf("seed candidate: %v", err)
	}

	created, err := svc.ApproveCandidate(candidate.ID, KnowledgeCandidateApprovalInput{})
	if err != nil {
		t.Fatalf("ApproveCandidate: %v", err)
	}
	if created.KnowledgeCategory != "游客 FAQ" || created.SpotID != 3 {
		t.Fatalf("knowledge fields not copied: %+v", created)
	}
	var updated model.KnowledgeCandidate
	if err := db.First(&updated, candidate.ID).Error; err != nil {
		t.Fatalf("load candidate: %v", err)
	}
	if updated.Status != "approved" {
		t.Fatalf("status = %q, want approved", updated.Status)
	}
}
