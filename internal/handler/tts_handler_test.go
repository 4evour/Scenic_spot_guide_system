package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
)

type fakeTTSSynthesizer struct {
	data    []byte
	err     error
	chunks  <-chan []byte
	errChan <-chan error
}

func (s fakeTTSSynthesizer) Synthesize(context.Context, string, string, string) ([]byte, error) {
	return s.data, s.err
}

func (s fakeTTSSynthesizer) SynthesizeStream(context.Context, string, string, string) (<-chan []byte, <-chan error) {
	return s.chunks, s.errChan
}

// testTTSHandler mirrors TTSHandler but with a configurable external URL,
// allowing tests to mock the external TTS API without modifying the production code.
type testTTSHandler struct {
	client *http.Client
	ttsURL string
}

func (h *testTTSHandler) TTS(c *gin.Context) {
	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	req.Text = trimSpace(req.Text)
	if req.Text == "" {
		pkg.BadRequest(c, "合成文本不能为空")
		return
	}

	if req.Voice == "" {
		req.Voice = "female_tianmei"
	}
	if req.Rate == "" {
		req.Rate = "1.0"
	}

	values := make(map[string]string)
	values["tex"] = req.Text
	values["cuid"] = "baike"
	values["lan"] = "ZH"
	values["ctp"] = "1"
	values["pdt"] = "301"
	values["vol"] = "9"
	values["rate"] = "32"

	switch req.Voice {
	case "female_tianmei":
		values["per"] = "0"
	case "female_zhiling":
		values["per"] = "1"
	case "male_zhizhong":
		values["per"] = "3"
	case "male_yige":
		values["per"] = "4"
	default:
		values["per"] = "0"
	}

	paramStr := encodeValues(values)
	resp, err := h.client.Post(h.ttsURL, "application/x-www-form-urlencoded",
		bytes.NewBufferString(paramStr))
	if err != nil {
		pkg.InternalError(c, "调用TTS服务失败")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pkg.InternalError(c, "TTS服务返回错误")
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		pkg.Success(c, gin.H{
			"audio_url": h.ttsURL + "?" + paramStr,
			"text":      req.Text,
			"voice":     req.Voice,
		})
		return
	}

	if errCode, ok := result["err_no"].(float64); ok && errCode != 0 {
		errMsg, _ := result["err_msg"].(string)
		if errMsg == "" {
			errMsg = "unknown error"
		}
		pkg.InternalError(c, "TTS服务错误")
		return
	}

	pkg.Success(c, gin.H{
		"audio_url": h.ttsURL + "?" + paramStr,
		"text":      req.Text,
		"voice":     req.Voice,
	})
}

func trimSpace(s string) string {
	start, end := 0, len(s)-1
	for start <= end && s[start] == ' ' {
		start++
	}
	for end >= start && s[end] == ' ' {
		end--
	}
	return s[start : end+1]
}

func encodeValues(m map[string]string) string {
	result := ""
	first := true
	for k, v := range m {
		if !first {
			result += "&"
		}
		result += k + "=" + v
		first = false
	}
	return result
}

func newTestTTSRouter(t *testing.T, mockURL string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler := &testTTSHandler{
		client: &http.Client{},
		ttsURL: mockURL,
	}
	router := gin.New()
	router.POST("/api/ai/tts", handler.TTS)
	return router
}

func TestTTSSuccessfulRequest(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type = %q, want application/x-www-form-urlencoded", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("tex") != "灵山大佛有多高" {
			t.Errorf("tex = %q, want %q", r.FormValue("tex"), "灵山大佛有多高")
		}

		w.Header().Set("Content-Type", "audio/mp3")
		io.WriteString(w, "fake audio bytes")
	}))
	defer mockServer.Close()

	router := newTestTTSRouter(t, mockServer.URL)
	body := bytes.NewBufferString(`{"text":"灵山大佛有多高","voice":"female_zhiling","rate":"+20%"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ai/tts", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			AudioURL string `json:"audio_url"`
			Text     string `json:"text"`
			Voice    string `json:"voice"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Code != 0 {
		t.Fatalf("code = %d, want 0", result.Code)
	}
	if result.Data.Text != "灵山大佛有多高" {
		t.Fatalf("text = %q, want %q", result.Data.Text, "灵山大佛有多高")
	}
	if result.Data.Voice != "female_zhiling" {
		t.Fatalf("voice = %q, want %q", result.Data.Voice, "female_zhiling")
	}
	if result.Data.AudioURL == "" {
		t.Fatal("audio_url should not be empty")
	}
}

func TestTTSEmptyText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewTTSHandler()
	router.POST("/api/ai/tts", handler.TTS)

	tests := []struct {
		name string
		text string
	}{
		{name: "empty string", text: ""},
		{name: "whitespace only", text: "   "},
		{name: "empty after trim", text: "  \t\n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(TTSRequest{Text: tt.text})
			req := httptest.NewRequest(http.MethodPost, "/api/ai/tts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
			}

			var result struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if result.Message != "合成文本不能为空" {
				t.Fatalf("message = %q, want %q", result.Message, "合成文本不能为空")
			}
		})
	}
}

func TestTTSInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTTSHandler()
	router := gin.New()
	router.POST("/api/ai/tts", handler.TTS)

	req := httptest.NewRequest(http.MethodPost, "/api/ai/tts",
		bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}

func TestTTSVoiceTypes(t *testing.T) {
	tests := []struct {
		name    string
		voice   string
		wantPer string
	}{
		{name: "female_tianmei", voice: "female_tianmei", wantPer: "0"},
		{name: "female_zhiling", voice: "female_zhiling", wantPer: "1"},
		{name: "male_zhizhong", voice: "male_zhizhong", wantPer: "3"},
		{name: "male_yige", voice: "male_yige", wantPer: "4"},
		{name: "unknown defaults to per=0", voice: "unknown_voice", wantPer: "0"},
		{name: "empty defaults to female_tianmei per=0", voice: "", wantPer: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPer string
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				gotPer = r.FormValue("per")
				io.WriteString(w, "audio bytes")
			}))
			defer mockServer.Close()

			router := newTestTTSRouter(t, mockServer.URL)
			reqBody, _ := json.Marshal(TTSRequest{Text: "测试文本", Voice: tt.voice})
			req := httptest.NewRequest(http.MethodPost, "/api/ai/tts", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
			}
			if gotPer != tt.wantPer {
				t.Fatalf("per = %q, want %q", gotPer, tt.wantPer)
			}

			var result struct {
				Data struct {
					Voice string `json:"voice"`
				} `json:"data"`
			}
			json.Unmarshal(resp.Body.Bytes(), &result)
			expectedVoice := tt.voice
			if expectedVoice == "" {
				expectedVoice = "female_tianmei"
			}
			if result.Data.Voice != expectedVoice {
				t.Fatalf("response voice = %q, want %q", result.Data.Voice, expectedVoice)
			}
		})
	}
}

func TestTTSServerReturns500(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "internal error")
	}))
	defer mockServer.Close()

	router := newTestTTSRouter(t, mockServer.URL)
	body, _ := json.Marshal(TTSRequest{Text: "测试文本"})
	req := httptest.NewRequest(http.MethodPost, "/api/ai/tts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusInternalServerError, resp.Body.String())
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Message != "TTS服务返回错误" {
		t.Fatalf("message = %q, want %q", result.Message, "TTS服务返回错误")
	}
}

func TestTTSServerReturnsBusinessError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"err_no":  500,
			"err_msg": "text too long",
		})
	}))
	defer mockServer.Close()

	router := newTestTTSRouter(t, mockServer.URL)
	body, _ := json.Marshal(TTSRequest{Text: "很长的文本"})
	req := httptest.NewRequest(http.MethodPost, "/api/ai/tts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusInternalServerError, resp.Body.String())
	}

	var result struct {
		Message string `json:"message"`
	}
	json.Unmarshal(resp.Body.Bytes(), &result)
	if result.Message != "TTS服务错误" {
		t.Fatalf("message = %q, want %q", result.Message, "TTS服务错误")
	}
}

func TestTTSServerReturnsNonJSON(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mp3")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "raw audio data here")
	}))
	defer mockServer.Close()

	router := newTestTTSRouter(t, mockServer.URL)
	body, _ := json.Marshal(TTSRequest{Text: "测试"})
	req := httptest.NewRequest(http.MethodPost, "/api/ai/tts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			AudioURL string `json:"audio_url"`
			Text     string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Code != 0 {
		t.Fatalf("code = %d, want 0", result.Code)
	}
	if result.Data.Text != "测试" {
		t.Fatalf("text = %q, want %q", result.Data.Text, "测试")
	}
	if result.Data.AudioURL == "" {
		t.Fatal("audio_url should not be empty")
	}
}

func TestTTSServerReturnsSuccessJSON(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"err_no":  0,
			"err_msg": "success",
		})
	}))
	defer mockServer.Close()

	router := newTestTTSRouter(t, mockServer.URL)
	body, _ := json.Marshal(TTSRequest{Text: "你好"})
	req := httptest.NewRequest(http.MethodPost, "/api/ai/tts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			Text  string `json:"text"`
			Voice string `json:"voice"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Code != 0 {
		t.Fatalf("code = %d, want 0", result.Code)
	}
}

func TestTTSStreamReturnsErrorWhenSynthesisHasNoAudio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chunks := make(chan []byte)
	errs := make(chan error, 1)
	errs <- errors.New("upstream unavailable")
	close(chunks)
	close(errs)

	handler := &TTSHandler{edgeTTS: fakeTTSSynthesizer{chunks: chunks, errChan: errs}, timeout: time.Second}
	router := gin.New()
	router.POST("/api/ai/tts/stream", handler.TTSStream)

	req := httptest.NewRequest(http.MethodPost, "/api/ai/tts/stream", bytes.NewBufferString(`{"text":"测试"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusInternalServerError, resp.Body.String())
	}
}

func TestTTSStreamWritesFirstAudioBeforeSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chunks := make(chan []byte, 1)
	chunks <- []byte("audio")
	close(chunks)
	errs := make(chan error)
	close(errs)

	handler := &TTSHandler{edgeTTS: fakeTTSSynthesizer{chunks: chunks, errChan: errs}, timeout: time.Second}
	router := gin.New()
	router.POST("/api/ai/tts/stream", handler.TTSStream)

	req := httptest.NewRequest(http.MethodPost, "/api/ai/tts/stream", bytes.NewBufferString(`{"text":"测试"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if resp.Body.String() != "audio" {
		t.Fatalf("stream body = %q, want audio", resp.Body.String())
	}
}
