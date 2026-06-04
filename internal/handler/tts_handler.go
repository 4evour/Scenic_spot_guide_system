package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
)

type TTSHandler struct {
	client *http.Client
}

func NewTTSHandler() *TTSHandler {
	return &TTSHandler{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

type TTSRequest struct {
	Text  string `json:"text"`
	Voice string `json:"voice"`
	Speed string `json:"speed"`
}

func (h *TTSHandler) TTS(c *gin.Context) {
	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		pkg.BadRequest(c, "文本内容不能为空")
		return
	}

	if req.Voice == "" {
		req.Voice = "female_tianmei"
	}
	if req.Speed == "" {
		req.Speed = "1.0"
	}

	ttsURL := "https://tts.baidu.com/text2audio"

	values := url.Values{}
	values.Set("tex", strings.TrimSpace(req.Text))
	values.Set("cuid", "baike")
	values.Set("lan", "ZH")
	values.Set("ctp", "1")
	values.Set("pdt", "301")
	values.Set("vol", "9")
	values.Set("rate", "32")

	switch req.Voice {
	case "female_tianmei":
		values.Set("per", "0")
	case "female_zhiling":
		values.Set("per", "1")
	case "male_zhizhong":
		values.Set("per", "3")
	case "male_yige":
		values.Set("per", "4")
	default:
		values.Set("per", "0")
	}
	paramStr := values.Encode()

	ttsReq, err := http.NewRequest("POST", ttsURL, strings.NewReader(paramStr))
	if err != nil {
		pkg.InternalError(c, "创建TTS请求失败")
		return
	}

	ttsReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.client.Do(ttsReq)
	if err != nil {
		slog.Error("调用TTS服务失败", "error", err)
		pkg.InternalError(c, "调用TTS服务失败")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("TTS服务返回错误", "status", resp.StatusCode)
		pkg.InternalError(c, "TTS服务返回错误")
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		pkg.Success(c, gin.H{
			"audio_url": ttsURL + "?" + paramStr,
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
		slog.Error("TTS服务业务错误", "err_no", errCode, "err_msg", errMsg)
		pkg.InternalError(c, "TTS服务错误")
		return
	}

	pkg.Success(c, gin.H{
		"audio_url": ttsURL + "?" + paramStr,
		"text":      req.Text,
		"voice":     req.Voice,
	})
}

func (h *TTSHandler) Routes(r *gin.RouterGroup) {
	r.POST("/ai/tts", pkg.RateLimitMiddleware(30, time.Minute), h.TTS)
}
