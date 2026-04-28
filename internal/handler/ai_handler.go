package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
)

type AIHandler struct{}

func NewAIHandler() *AIHandler {
	return &AIHandler{}
}

type ChatRequest struct {
	Message string `json:"message"`
}

type DeepSeekRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type DeepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (h *AIHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.Message == "" {
		pkg.BadRequest(c, "消息内容不能为空")
		return
	}

	deepSeekReq := DeepSeekRequest{
		Model: "deepseek-chat",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: "你是一位专业的景区导游AI助手，名字叫云岚。你需要用友好、专业的语气回答游客的问题，包括景点介绍、历史背景、路线规划、服务信息等。请用中文回答，保持亲切自然。",
			},
			{
				Role:    "user",
				Content: req.Message,
			},
		},
	}

	reqBody, err := json.Marshal(deepSeekReq)
	if err != nil {
		pkg.InternalError(c, "请求序列化失败")
		return
	}

	apiKey := "<DEEPSEEK_API_KEY>"

	httpReq, err := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		pkg.InternalError(c, "创建请求失败")
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		pkg.InternalError(c, "调用AI服务失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pkg.InternalError(c, "AI服务返回错误: "+resp.Status)
		return
	}

	var deepSeekResp DeepSeekResponse
	if err := json.NewDecoder(resp.Body).Decode(&deepSeekResp); err != nil {
		pkg.InternalError(c, "解析响应失败")
		return
	}

	if len(deepSeekResp.Choices) == 0 {
		pkg.InternalError(c, "未获取到响应")
		return
	}

	pkg.Success(c, gin.H{
		"response": deepSeekResp.Choices[0].Message.Content,
	})
}

func (h *AIHandler) Routes(r *gin.RouterGroup) {
	r.POST("/ai/chat", h.Chat)
}