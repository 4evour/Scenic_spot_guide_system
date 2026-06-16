package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

var registerProxyTestDriver sync.Once

func newProxyTestRAGService(t *testing.T) *service.RAGService {
	t.Helper()

	const driverName = "modernc-proxy-test"
	registerProxyTestDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	rag := service.NewRAGService(repository.NewKnowledgeRepository(db), "", "", "", nil, nil)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"test","content":"灵山大佛高88米，主体高79米，莲花瓣高9米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}
	return rag
}

func newProxyTestRAGServiceWithLLM(t *testing.T, llmBaseURL string) *service.RAGService {
	t.Helper()

	const driverName = "modernc-proxy-test"
	registerProxyTestDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	rag := service.NewRAGService(repository.NewKnowledgeRepository(db), "configured-key", "test-model", llmBaseURL, nil, nil)
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"test","content":"灵山大佛高88米，主体高79米，莲花瓣高9米。"},
		{"id":"buddha-meta","title":"灵山大佛问答素材","source":"test","content":"游客常问灵山大佛多高、是不是景区最代表性的景点。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}
	return rag
}

func TestOpenAIProxyChatCompletionsNonStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/v1/chat/completions", NewOpenAIProxyHandler(newProxyTestRAGService(t), nil).ChatCompletions)

	body := bytes.NewBufferString(`{"model":"test-model","messages":[{"role":"user","content":"灵山大佛有多高？"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload ChatCompletionResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Choices) != 1 {
		t.Fatalf("choices length = %d, want 1", len(payload.Choices))
	}
	content := payload.Choices[0].Message.Content
	if !strings.Contains(content, "88米") {
		t.Fatalf("content %q does not contain expected knowledge", content)
	}
}

func TestOpenAIProxyChatCompletionsStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/v1/chat/completions", NewOpenAIProxyHandler(newProxyTestRAGService(t), nil).ChatCompletions)

	body := bytes.NewBufferString(`{"model":"test-model","stream":true,"messages":[{"role":"user","content":[{"type":"text","text":"灵山大佛有多高？"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if contentType := resp.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("content-type = %q, want event-stream", contentType)
	}
	bodyText := resp.Body.String()
	if !strings.Contains(bodyText, "data: ") || !strings.Contains(bodyText, "[DONE]") {
		t.Fatalf("stream body missing SSE markers: %s", bodyText)
	}
}

func TestOpenAIProxyChatCompletionsStreamUsesRAGStreamingLLM(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var sawStreamRequest bool
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected LLM path: %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode LLM request: %v", err)
		}
		if payload["stream"] == true {
			sawStreamRequest = true
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"灵山大佛高88米\"}}]}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"1,2"}}]}`)
	}))
	defer llm.Close()

	router := gin.New()
	router.POST("/v1/chat/completions", NewOpenAIProxyHandler(newProxyTestRAGServiceWithLLM(t, llm.URL), nil).ChatCompletions)

	body := bytes.NewBufferString(`{"model":"test-model","stream":true,"messages":[{"role":"user","content":"灵山大佛有多高？"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if !sawStreamRequest {
		t.Fatal("expected upstream LLM stream request")
	}
	bodyText := resp.Body.String()
	if !strings.Contains(bodyText, "灵山大佛高88米") {
		t.Fatalf("stream body missing upstream token: %s", bodyText)
	}
	if !strings.Contains(bodyText, "[DONE]") {
		t.Fatalf("stream body missing done marker: %s", bodyText)
	}
}

func TestOpenAIProxyChatCompletionsStreamReturnsErrorWhenLLMFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	defer llm.Close()

	router := gin.New()
	router.POST("/v1/chat/completions", NewOpenAIProxyHandler(newProxyTestRAGServiceWithLLM(t, llm.URL), nil).ChatCompletions)

	body := bytes.NewBufferString(`{"model":"test-model","stream":true,"messages":[{"role":"user","content":"灵山大佛有什么特色？"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	bodyText := resp.Body.String()
	if !strings.Contains(bodyText, `"error"`) {
		t.Fatalf("stream failure should return explicit SSE error: %s", bodyText)
	}
	if strings.Contains(bodyText, "游客常问") || strings.Contains(bodyText, "问答素材") {
		t.Fatalf("stream failure should not expose knowledge meta as answer: %s", bodyText)
	}
}

func TestOpenAIProxyChatCompletionsUsesSessionContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewOpenAIProxyHandler(newProxyTestRAGService(t), nil)
	router.POST("/v1/chat/completions", handler.ChatCompletions)

	firstBody := bytes.NewBufferString(`{"model":"test-model","session_id":"s1","messages":[{"role":"user","content":"灵山大佛是什么？"}]}`)
	firstReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", firstBody)
	firstReq.Header.Set("Content-Type", "application/json")
	firstResp := httptest.NewRecorder()
	router.ServeHTTP(firstResp, firstReq)
	if firstResp.Code != http.StatusOK {
		t.Fatalf("first status = %d, body=%s", firstResp.Code, firstResp.Body.String())
	}

	secondBody := bytes.NewBufferString(`{"model":"test-model","session_id":"s1","messages":[{"role":"user","content":"它有多高？"}]}`)
	secondReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", secondBody)
	secondReq.Header.Set("Content-Type", "application/json")
	secondResp := httptest.NewRecorder()
	router.ServeHTTP(secondResp, secondReq)

	if secondResp.Code != http.StatusOK {
		t.Fatalf("second status = %d, body=%s", secondResp.Code, secondResp.Body.String())
	}

	var payload ChatCompletionResponse
	if err := json.Unmarshal(secondResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Choices) != 1 {
		t.Fatalf("choices length = %d, want 1", len(payload.Choices))
	}
	content := payload.Choices[0].Message.Content
	if !strings.Contains(content, "88米") {
		t.Fatalf("follow-up content %q does not contain expected contextual answer", content)
	}
}

func TestAIChatResponseDoesNotExposeInternalRewriteContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAIHandler(newProxyTestRAGService(t), nil)
	router.POST("/api/v1/ai/chat", handler.Chat)

	body := bytes.NewBufferString(`{"session_id":"s1","message":"灵山大佛有多高？"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if strings.Contains(resp.Body.String(), "rewritten_query") || strings.Contains(resp.Body.String(), "context_topic") {
		t.Fatalf("response leaked internal rewrite context: %s", resp.Body.String())
	}

	var payload struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	for _, want := range []string{"response", "trace_id"} {
		if _, ok := payload.Data[want]; !ok {
			t.Fatalf("response data missing %q: %s", want, resp.Body.String())
		}
	}
}
