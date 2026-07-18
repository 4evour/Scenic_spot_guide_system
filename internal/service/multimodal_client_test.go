package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scenic-guide/internal/config"
)

func TestMultimodalClientChatBuildsOpenAICompatibleRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header missing")
		}
		var payload struct {
			Model    string `json:"model"`
			Messages []struct {
				Content []map[string]any `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != "qwen3.5-omni-plus" || len(payload.Messages) != 1 || len(payload.Messages[0].Content) != 2 {
			t.Fatalf("unexpected payload: %+v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"图片中的景点是灵山大佛。"}}]}`))
	}))
	defer server.Close()

	client, err := NewMultimodalClient(&config.MultimodalConfig{
		Enabled:        true,
		Provider:       "qwen",
		Model:          "qwen3.5-omni-plus",
		BaseURL:        server.URL,
		APIKey:         "test-key",
		TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	result, err := client.Chat(context.Background(), "这是什么景点？", []MultimodalPart{{
		Kind: "image", MIMEType: "image/png", Data: []byte("image"),
	}})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if result.Text != "图片中的景点是灵山大佛。" || result.Modality != "text_image" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestMultimodalClientRetriesTransientProviderFailure(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"已恢复"}}]}`))
	}))
	defer server.Close()

	client, err := NewMultimodalClient(&config.MultimodalConfig{
		Enabled: true, Provider: "qwen", Model: "test", BaseURL: server.URL, APIKey: "test-key", TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.guard.cfg.RetryBackoff = time.Millisecond
	result, err := client.Chat(context.Background(), "hello", nil)
	if err != nil || result.Text != "已恢复" || calls.Load() != 2 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, calls.Load())
	}
}

func TestMultimodalClientDoesNotCallProviderWhenDisabled(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	client, err := NewMultimodalClient(&config.MultimodalConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = client.Chat(context.Background(), "hello", nil)
	if err != ErrMultimodalDisabled {
		t.Fatalf("error = %v, want ErrMultimodalDisabled", err)
	}
	if called {
		t.Fatal("disabled client called provider")
	}
}

func TestMultimodalClientHidesUpstreamResponseBodyOnHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "secret provider details", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewMultimodalClient(&config.MultimodalConfig{
		Enabled: true, Provider: "qwen", Model: "test", BaseURL: server.URL, APIKey: "test-key", TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = client.Chat(context.Background(), "hello", nil)
	if err == nil || !strings.Contains(err.Error(), "status 401") || strings.Contains(err.Error(), "secret provider details") {
		t.Fatalf("unexpected upstream error: %v", err)
	}
}
