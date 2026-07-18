package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestBuildLocationAwareQueryOnlyAcceptsPreciseNearbyContext(t *testing.T) {
	location := &ChatLocationContext{SpotName: "灵山大佛", DistanceMeters: 42, AccuracyMeters: 8}
	query := buildLocationAwareQuery("我现在在哪个景点附近？", location)
	if query == "我现在在哪个景点附近？" || !strings.Contains(query, "灵山大佛") {
		t.Fatalf("location-aware query = %q", query)
	}

	for _, invalid := range []*ChatLocationContext{
		{SpotName: "灵山大佛", DistanceMeters: 301, AccuracyMeters: 8},
		{SpotName: "灵山大佛", DistanceMeters: 42, AccuracyMeters: 10.1},
	} {
		if got := buildLocationAwareQuery("我现在在哪个景点附近？", invalid); got != "我现在在哪个景点附近？" {
			t.Fatalf("invalid location context changed query: %q", got)
		}
	}
	if got := buildLocationAwareQuery("灵山大佛有多高？", location); got != "灵山大佛有多高？" {
		t.Fatalf("non-location query changed: %q", got)
	}
}

func TestBuildDirectLocationAnswerOnlyUsesValidatedContext(t *testing.T) {
	answer, ok := buildDirectLocationAnswer("我现在在哪个景点附近？", &ChatLocationContext{
		SpotName: "灵山大佛", DistanceMeters: 42, AccuracyMeters: 8,
	})
	if !ok || !strings.Contains(answer, "灵山大佛") || !strings.Contains(answer, "42") {
		t.Fatalf("direct location answer = %q, ok=%v", answer, ok)
	}
	if _, ok := buildDirectLocationAnswer("灵山大佛有多高？", &ChatLocationContext{
		SpotName: "灵山大佛", DistanceMeters: 42, AccuracyMeters: 8,
	}); ok {
		t.Fatal("fact query must not use direct location answer")
	}
	if _, ok := buildDirectLocationAnswer("我现在在哪个景点附近？", &ChatLocationContext{
		SpotName: "灵山大佛", DistanceMeters: 42, AccuracyMeters: 11,
	}); ok {
		t.Fatal("inaccurate location context must not use direct location answer")
	}
}

func TestEnsureRouteDetailsInAnswerAddsMissingStepsWithoutDuplication(t *testing.T) {
	route := &service.TourRoute{Steps: []service.TourRouteStep{
		{Number: 1, Name: "灵山大照壁"},
		{Number: 2, Name: "佛足坛"},
		{Number: 3, Name: "杏坛广场"},
	}}
	answer := ensureRouteDetailsInAnswer("观光车路线有哪些站点？", "观光车站点以现场公告为准。", route)
	if !strings.Contains(answer, "灵山大照壁 > 佛足坛 > 杏坛广场") {
		t.Fatalf("route details were not appended: %q", answer)
	}
	complete := "路线依次经过灵山大照壁、佛足坛和杏坛广场。"
	if got := ensureRouteDetailsInAnswer("观光车路线有哪些站点？", complete, route); got != complete {
		t.Fatalf("complete route answer was duplicated: %q", got)
	}
	if got := ensureRouteDetailsInAnswer("灵山大佛有多高？", "高88米。", route); got != "高88米。" {
		t.Fatalf("non-route answer changed: %q", got)
	}
}
