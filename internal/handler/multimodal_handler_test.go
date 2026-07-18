package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/service"
)

func newMultimodalTestClient(t *testing.T, response string) *service.MultimodalClient {
	t.Helper()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + response + `"}}]}`))
	}))
	t.Cleanup(provider.Close)

	client, err := service.NewMultimodalClient(&config.MultimodalConfig{
		Enabled: true, Provider: "qwen", Model: "qwen3.5-omni-plus", BaseURL: provider.URL, APIKey: "test-key", TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("new multimodal client: %v", err)
	}
	return client
}

func makeMultimodalMultipart(t *testing.T, message string, image []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if message != "" {
		if err := writer.WriteField("message", message); err != nil {
			t.Fatalf("write message: %v", err)
		}
	}
	if image != nil {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", `form-data; name="image"; filename="spot.png"`)
		header.Set("Content-Type", "image/png")
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatalf("create image part: %v", err)
		}
		if _, err := part.Write(image); err != nil {
			t.Fatalf("write image: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}

func TestMultimodalChatAcceptsTextAndImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAIHandler(nil, nil)
	handler.SetMultimodalClient(newMultimodalTestClient(t, "这是灵山大佛。"))
	router := gin.New()
	router.POST("/api/v1/ai/multimodal/chat", handler.MultimodalChat)

	body, contentType := makeMultimodalMultipart(t, "这是什么景点？", []byte{
		0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/multimodal/chat", body)
	req.Header.Set("Content-Type", contentType)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Response string `json:"response"`
			Modality string `json:"modality"`
			Degraded bool   `json:"degraded"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Code != 0 || envelope.Data.Response != "这是灵山大佛。" || envelope.Data.Modality != "text_image" || envelope.Data.Degraded {
		t.Fatalf("unexpected response: %+v", envelope)
	}
}

func TestMultimodalChatRejectsMismatchedMediaSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAIHandler(nil, nil)
	handler.SetMultimodalClient(newMultimodalTestClient(t, "should not be called"))
	router := gin.New()
	router.POST("/api/v1/ai/multimodal/chat", handler.MultimodalChat)

	body, contentType := makeMultimodalMultipart(t, "检查图片", []byte("not an image"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/multimodal/chat", body)
	req.Header.Set("Content-Type", contentType)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
}

func TestMultimodalChatReturnsUnavailableWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, err := service.NewMultimodalClient(&config.MultimodalConfig{})
	if err != nil {
		t.Fatalf("new disabled client: %v", err)
	}
	handler := NewAIHandler(nil, nil)
	handler.SetMultimodalClient(client)
	router := gin.New()
	router.POST("/api/v1/ai/multimodal/chat", handler.MultimodalChat)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/multimodal/chat", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
}

func TestMultimodalHandlerContextCancellationIsPassedThrough(t *testing.T) {
	client := newMultimodalTestClient(t, "unused")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Chat(ctx, "hello", nil)
	if err == nil {
		t.Fatal("expected canceled context to fail")
	}
}
