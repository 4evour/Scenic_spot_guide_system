package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCreateHTTPClientConfiguresBoundedConnectionAndIdleReuse(t *testing.T) {
	client := createHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.DialContext == nil {
		t.Fatal("DialContext must be configured with a bounded dialer")
	}
	if defaultChatDialTimeout != 2*time.Second {
		t.Fatalf("dial timeout = %v, want 2s", defaultChatDialTimeout)
	}
	if transport.TLSHandshakeTimeout != 3*time.Second {
		t.Fatalf("TLS handshake timeout = %v, want 3s", transport.TLSHandshakeTimeout)
	}
	if transport.IdleConnTimeout != 10*time.Minute {
		t.Fatalf("idle connection timeout = %v, want 10m", transport.IdleConnTimeout)
	}
}

func TestWarmChatConnectionRequestsModelsWithAuthorization(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/models" {
			t.Errorf("path = %q, want /models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	rag := NewRAGService(nil, "test-key", "test-model", server.URL+"/", nil, nil)
	if err := rag.WarmChatConnection(context.Background()); err != nil {
		t.Fatalf("WarmChatConnection returned error: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("warm-up calls = %d, want 1", got)
	}
}

func TestWarmChatConnectionLeavesConnectionReusable(t *testing.T) {
	var mu sync.Mutex
	remoteAddresses := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		remoteAddresses = append(remoteAddresses, r.RemoteAddr)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/models":
			_, _ = w.Write([]byte(`{"data":[]}`))
		case "/chat/completions":
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ready"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	rag := NewRAGService(nil, "test-key", "test-model", server.URL, nil, nil)
	if err := rag.WarmChatConnection(context.Background()); err != nil {
		t.Fatalf("WarmChatConnection returned error: %v", err)
	}
	if _, err := rag.callChatCompletion(context.Background(), []byte(`{"model":"test-model","messages":[]}`)); err != nil {
		t.Fatalf("callChatCompletion returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(remoteAddresses) != 2 {
		t.Fatalf("request count = %d, want 2", len(remoteAddresses))
	}
	if remoteAddresses[0] != remoteAddresses[1] {
		t.Fatalf("warm-up connection %q was not reused by chat request %q", remoteAddresses[0], remoteAddresses[1])
	}
}

func TestQueryWithRAGTraceFallsBackWhenChatRequestBudgetExpires(t *testing.T) {
	rag := newTestRAGService(t)
	seedChatResilienceKnowledge(t, rag)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
			return
		case <-time.After(time.Second):
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"late answer"}}]}`))
		}
	}))
	defer server.Close()

	configureResilienceTestChat(rag, server.URL)
	rag.chatRequestTimeout = 40 * time.Millisecond

	started := time.Now()
	answer, trace, err := rag.QueryWithRAGTrace(context.Background(), "灵山大佛有多高？", "zh-CN")
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("QueryWithRAGTrace returned error: %v", err)
	}
	if trace.Provider != "local-rag-fallback" {
		t.Fatalf("provider = %q, want local-rag-fallback", trace.Provider)
	}
	if strings.TrimSpace(answer) == "" {
		t.Fatal("fallback answer must not be empty")
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("fallback took %v, want bounded well below the upstream delay", elapsed)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("model calls = %d, want one call within the shared request budget", got)
	}
}

func TestQueryWithRAGStreamingFallsBackWhenFirstTokenBudgetExpires(t *testing.T) {
	rag := newTestRAGService(t)
	seedChatResilienceKnowledge(t, rag)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(time.Second):
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"late token\"}}]}\n\n"))
		}
	}))
	defer server.Close()

	configureResilienceTestChat(rag, server.URL)
	rag.streamFirstTokenTimeout = 40 * time.Millisecond

	var streamed strings.Builder
	started := time.Now()
	answer, _, trace, err := rag.QueryWithRAGStreaming(
		context.Background(),
		"",
		"灵山大佛有多高？",
		"zh-CN",
		func(token string) { streamed.WriteString(token) },
	)
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("QueryWithRAGStreaming returned error: %v", err)
	}
	if trace.Provider != "local-rag-fallback" {
		t.Fatalf("provider = %q, want local-rag-fallback", trace.Provider)
	}
	if answer == "" || streamed.String() != answer {
		t.Fatalf("fallback answer = %q, streamed = %q", answer, streamed.String())
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("streaming fallback took %v, want bounded well below the upstream delay", elapsed)
	}
}

func TestCallLLMStreamingStopsFirstTokenTimerAfterFirstToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("streaming response does not implement http.Flusher")
			return
		}

		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"第一段\"}}]}\n\n"))
		flusher.Flush()
		time.Sleep(120 * time.Millisecond)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"第二段\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	rag := NewRAGService(nil, "test-key", "test-model", server.URL, nil, nil)
	rag.streamFirstTokenTimeout = 40 * time.Millisecond

	var streamed strings.Builder
	answer, err := rag.CallLLMStreaming(
		context.Background(),
		"system prompt",
		"user prompt",
		func(token string) { streamed.WriteString(token) },
	)
	if err != nil {
		t.Fatalf("CallLLMStreaming returned error after the first token: %v", err)
	}
	if answer != "第一段第二段" {
		t.Fatalf("answer = %q, want both streamed segments", answer)
	}
	if streamed.String() != answer {
		t.Fatalf("streamed = %q, answer = %q", streamed.String(), answer)
	}
}

func seedChatResilienceKnowledge(t *testing.T, rag *RAGService) {
	t.Helper()
	if _, err := rag.LoadKnowledgeJSON([]byte(`[
		{"id":"buddha-height","title":"灵山大佛高度","source":"official","content":"灵山大佛通高88米，主体高79米。"}
	]`)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}
}

func configureResilienceTestChat(rag *RAGService, baseURL string) {
	rag.chatAPIKey = "test-key"
	rag.chatModel = "test-model"
	rag.chatBaseURL = baseURL
	rag.chatGuard.cfg.RetryBackoff = time.Millisecond
}
