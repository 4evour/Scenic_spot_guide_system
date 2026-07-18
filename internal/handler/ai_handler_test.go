package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/service"
)

func TestModelHealthReportsConfiguredProviderState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rag := service.NewRAGService(nil, "", "", "", nil, nil)
	handler := NewAIHandler(rag, nil)
	router := gin.New()
	router.GET("/ai/model-health", handler.ModelHealth)

	req := httptest.NewRequest(http.MethodGet, "/ai/model-health", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Status    string                        `json:"status"`
		Providers []service.ModelProviderHealth `json:"providers"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != "disabled" || len(payload.Providers) != 1 {
		t.Fatalf("unexpected health payload: %+v", payload)
	}
	if payload.Providers[0].Provider != "chat" || payload.Providers[0].State != "disabled" {
		t.Fatalf("unexpected provider state: %+v", payload.Providers[0])
	}
}

func TestAIChatUsesValidatedVoiceFeatures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAIHandler(newProxyTestRAGService(t), nil)
	router := gin.New()
	router.POST("/api/v1/ai/chat", handler.Chat)

	body := bytes.NewBufferString(`{
		"message":"这个景色很特别",
		"voice_features":{
			"duration_ms":1800,"sample_count":20,"rms_mean":0.08,"rms_peak":0.22,
			"rms_variation":0.05,"pause_ratio":0.1,"pitch_mean_hz":210,
			"pitch_variation_hz":60,"speech_rate_chars_per_second":7,"repetition_ratio":0
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Data struct {
			EmotionCategory    string  `json:"emotion"`
			EmotionModality    string  `json:"emotion_modality"`
			AcousticConfidence float64 `json:"acoustic_confidence"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.EmotionCategory != string(service.VisitorEmotionExcitement) ||
		payload.Data.EmotionModality != "text+acoustic" || payload.Data.AcousticConfidence <= 0 {
		t.Fatalf("unexpected voice emotion payload: %+v", payload.Data)
	}
}
