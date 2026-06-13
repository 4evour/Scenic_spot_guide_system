package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/service"
)

// OpenAIProxyHandler 提供 OpenAI 兼容的 /v1/chat/completions 端点
// 让 Open-LLM-VTuber 可以直接调用 Go 后端的 RAG 服务
type OpenAIProxyHandler struct {
	ragService   *service.RAGService
	statsService *service.StatsService
}

func NewOpenAIProxyHandler(ragService *service.RAGService, statsService *service.StatsService) *OpenAIProxyHandler {
	return &OpenAIProxyHandler{ragService: ragService, statsService: statsService}
}

// ==================== 请求格式 ====================

// ChatCompletionRequest OpenAI 兼容请求
// Content 使用 json.RawMessage 以同时支持字符串和多模态数组格式
type ChatCompletionRequest struct {
	Model     string `json:"model"`
	SessionID string `json:"session_id,omitempty"`
	Messages  []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"` // 兼容 string 和 array 两种格式
		Name    string          `json:"name,omitempty"`
	} `json:"messages"`
	Stream bool `json:"stream"`
}

// ContentPart 多模态内容的一部分
type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

// extractMessageContent 从 Content 字段提取纯文本
// 支持两种格式:
//   - 字符串: "hello"
//   - 数组: [{"type":"text","text":"hello"},{"type":"image_url","image_url":{"url":"..."}}]
func extractMessageContent(raw json.RawMessage) string {
	// 尝试解析为字符串
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	// 尝试解析为多模态数组
	var parts []ContentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		var texts []string
		for _, p := range parts {
			if p.Type == "text" && p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
		return strings.Join(texts, " ")
	}

	return ""
}

// ==================== 响应格式 ====================

// ChatCompletionResponse 非流式响应
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ChatCompletionChunk SSE 流式响应的 chunk 结构
type ChatCompletionChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// ==================== 主处理函数 ====================

func (h *OpenAIProxyHandler) ChatCompletions(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20) // 1MB
	// 先用 json.RawMessage 读取原始请求体进行灵活解析
	rawBody, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read request body"})
		return
	}

	var req ChatCompletionRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		slog.Warn("OpenAI 兼容请求 JSON 解析失败", "error", err, "body_len", len(rawBody))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 取最后一条 user 消息作为查询（兼容多模态格式的 content）
	var query string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			query = extractMessageContent(req.Messages[i].Content)
			if query != "" {
				break
			}
		}
	}
	if query == "" {
		slog.Warn("OpenAI 兼容请求缺少 user 消息", "messages_count", len(req.Messages))
		c.JSON(http.StatusBadRequest, gin.H{"error": "no user message found"})
		return
	}

	slog.Info("收到 OpenAI 兼容请求", "query_len", len([]rune(query)), "stream", req.Stream)

	if h.ragService == nil {
		if req.Stream {
			h.writeStreamError(c, "RAG service not available")
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RAG service not available"})
		}
		return
	}

	// 使用 RAG 服务生成回答
	startTime := time.Now()
	lang := c.GetString("lang")
		response, _, trace, err := h.ragService.QueryWithRAGAndRouteTraceInSession(req.SessionID, query, lang)
	elapsed := time.Since(startTime).Milliseconds()
	if err != nil {
		slog.Error("OpenAI 兼容请求 RAG 查询失败", "error", err, "trace_id", trace.TraceID, "elapsed_ms", elapsed)
		if req.Stream {
			h.writeStreamError(c, "RAG query failed")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "RAG query failed"})
		}
		return
	}

	// 检测情绪并嵌入表情标签，让 Live2D 数字人展示表情
	emotion := detectEmotion(response)
	responseWithEmotion := fmt.Sprintf("[%s] %s", emotion, response)

	// 记录交互日志
	if h.statsService != nil {
		h.statsService.RecordInteraction(service.InteractionRecord{
			SessionID:      req.SessionID,
			Query:          query,
			Response:       response,
			Emotion:        emotion,
			ResponseTimeMs: elapsed,
			Category:       service.DetectCategory(query),
			Source:         "digital_human",
		})
	}

	slog.Info("OpenAI 兼容请求回答完成",
		"trace_id", trace.TraceID,
		"emotion", emotion,
		"response_len", len([]rune(response)),
		"retrieval_ms", trace.RetrievalMs,
		"generation_ms", trace.GenerationMs,
		"total_ms", trace.TotalMs,
		"elapsed_ms", elapsed,
	)

	if req.Stream {
		h.writeStreamResponse(c, req.Model, responseWithEmotion)
	} else {
		h.writeNonStreamResponse(c, req.Model, responseWithEmotion)
	}
}

// ==================== 非流式响应 ====================

func (h *OpenAIProxyHandler) writeNonStreamResponse(c *gin.Context, model, content string) {
	now := time.Now().Unix()
	resp := ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", now),
		Object:  "chat.completion",
		Created: now,
		Model:   model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{0, 0, 0},
	}

	c.JSON(http.StatusOK, resp)
}

// ==================== SSE 流式响应 ====================

// writeStreamResponse 返回 SSE 流式响应 (Open-LLM-VTuber 需要 stream=true)
func (h *OpenAIProxyHandler) writeStreamResponse(c *gin.Context, model, content string) {
	now := time.Now().Unix()
	chatID := fmt.Sprintf("chatcmpl-%d", now)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.Status(http.StatusOK)

	writer := c.Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		slog.Error("OpenAI 兼容流式响应失败，ResponseWriter 不支持 Flusher")
		return
	}

	// 辅助函数：发送一个 SSE data 行
	sendChunk := func(chunk ChatCompletionChunk) {
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(writer, "data: %s\n\n", string(data))
		flusher.Flush()
	}

	// 1. 发送角色 chunk
	sendChunk(ChatCompletionChunk{
		ID:      chatID,
		Object:  "chat.completion.chunk",
		Created: now,
		Model:   model,
		Choices: []struct {
			Index int `json:"index"`
			Delta struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Delta: struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				}{
					Role: "assistant",
				},
				FinishReason: nil,
			},
		},
	})

	// 2. 将内容按字符分批发送
	runes := []rune(content)
	chunkSize := 12
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		var finishReason *string
		if end >= len(runes) {
			reason := "stop"
			finishReason = &reason
		}

		sendChunk(ChatCompletionChunk{
			ID:      chatID,
			Object:  "chat.completion.chunk",
			Created: now,
			Model:   model,
			Choices: []struct {
				Index int `json:"index"`
				Delta struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Delta: struct {
						Role    string `json:"role,omitempty"`
						Content string `json:"content,omitempty"`
					}{
						Content: string(runes[i:end]),
					},
					FinishReason: finishReason,
				},
			},
		})

	}

	// 3. 发送 [DONE] 标记
	fmt.Fprintf(writer, "data: [DONE]\n\n")
	flusher.Flush()

	slog.Info("OpenAI 兼容 SSE 流式响应完成", "chars", len(runes))
}

// writeStreamError 以 SSE 格式返回错误
func (h *OpenAIProxyHandler) writeStreamError(c *gin.Context, errMsg string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Status(http.StatusOK)

	writer := c.Writer
	flusher, ok := writer.(http.Flusher)
	if ok {
		errPayload, _ := json.Marshal(gin.H{"error": errMsg})
		fmt.Fprintf(writer, "data: %s\n\n", string(errPayload))
		flusher.Flush()
		fmt.Fprintf(writer, "data: [DONE]\n\n")
		flusher.Flush()
	}
}

// ==================== 工具函数 ====================
