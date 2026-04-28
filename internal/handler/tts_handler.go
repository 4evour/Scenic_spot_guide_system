package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
)

type TTSHandler struct{}

func NewTTSHandler() *TTSHandler {
	return &TTSHandler{}
}

type TTSRequest struct {
	Text   string `json:"text"`
	Voice  string `json:"voice"`
	Speed  string `json:"speed"`
}

func (h *TTSHandler) TTS(c *gin.Context) {
	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

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

	paramStr := strings.Builder{}
	paramStr.WriteString("tex=")
	paramStr.WriteString(req.Text)
	paramStr.WriteString("&cuid=baike&lan=ZH&ctp=1&pdt=301&vol=9&rate=32&per=")

	switch req.Voice {
	case "female_tianmei":
		paramStr.WriteString("0")
	case "female_zhiling":
		paramStr.WriteString("1")
	case "male_zhizhong":
		paramStr.WriteString("3")
	case "male_yige":
		paramStr.WriteString("4")
	default:
		paramStr.WriteString("0")
	}

	ttsReq, err := http.NewRequest("POST", ttsURL, strings.NewReader(paramStr.String()))
	if err != nil {
		pkg.InternalError(c, "创建TTS请求失败")
		return
	}

	ttsReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(ttsReq)
	if err != nil {
		pkg.InternalError(c, "调用TTS服务失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pkg.InternalError(c, "TTS服务返回错误: "+resp.Status)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		pkg.Success(c, gin.H{
			"audio_url": ttsURL + "?" + paramStr.String(),
			"text":      req.Text,
			"voice":     req.Voice,
		})
		return
	}

	if errCode, ok := result["err_no"].(float64); ok && errCode != 0 {
		pkg.InternalError(c, "TTS服务错误: "+result["err_msg"].(string))
		return
	}

	pkg.Success(c, gin.H{
		"audio_url": ttsURL + "?" + paramStr.String(),
		"text":      req.Text,
		"voice":     req.Voice,
	})
}

func (h *TTSHandler) Routes(r *gin.RouterGroup) {
	r.POST("/ai/tts", h.TTS)
}
