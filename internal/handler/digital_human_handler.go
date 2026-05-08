package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

type DigitalHumanHandler struct {
	ragService          *service.RAGService
	routeService        service.TourRouteService
	visitorQueryService service.VisitorQueryService
}

func NewDigitalHumanHandler(
	ragService *service.RAGService,
	routeService service.TourRouteService,
	visitorQueryService service.VisitorQueryService,
) *DigitalHumanHandler {
	return &DigitalHumanHandler{
		ragService:          ragService,
		routeService:        routeService,
		visitorQueryService: visitorQueryService,
	}
}

type SessionCreateRequest struct {
	UserID      string   `json:"user_id,omitempty"`
	Scene       string   `json:"scene,omitempty"`
	Location    string   `json:"location,omitempty"`
	Preferences []string `json:"preferences,omitempty"`
}

type SessionCreateResponse struct {
	SessionID string `json:"session_id"`
	ExpiresAt string `json:"expires_at"`
}

func (h *DigitalHumanHandler) CreateSession(c *gin.Context) {
	var req SessionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(30 * time.Minute).Format(time.RFC3339)

	fmt.Printf("[数字人] 创建会话: session_id=%s, user_id=%s, scene=%s\n",
		sessionID, req.UserID, req.Scene)

	pkg.Success(c, SessionCreateResponse{
		SessionID: sessionID,
		ExpiresAt: expiresAt,
	})
}

type ChatTextRequest struct {
	SessionID   string   `json:"session_id"`
	UserID      string   `json:"user_id,omitempty"`
	InputText   string   `json:"input_text"`
	Scene       string   `json:"scene,omitempty"`
	Location    string   `json:"location,omitempty"`
	Preferences []string `json:"preferences,omitempty"`
}

type ChatTextResponse struct {
	AnswerText   string                 `json:"answer_text"`
	Emotion      string                 `json:"emotion"`
	RoutePayload *RoutePayload          `json:"route_payload,omitempty"`
	TraceID      string                 `json:"trace_id"`
}

type RoutePayload struct {
	RouteID string   `json:"route_id"`
	Stops   []string `json:"stops"`
}

func (h *DigitalHumanHandler) ChatText(c *gin.Context) {
	var req ChatTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.InputText == "" {
		pkg.BadRequest(c, "输入文本不能为空")
		return
	}

	traceID := uuid.New().String()
	fmt.Printf("[数字人] 文本聊天: session_id=%s, trace_id=%s, input=%s\n",
		req.SessionID, traceID, req.InputText)

	startTime := time.Now()

	var answer string
	var emotion string

	if h.ragService != nil {
		response, _, err := h.ragService.QueryWithRAGAndRoute(req.InputText)
		elapsed := time.Since(startTime).Milliseconds()
		if err != nil {
			fmt.Printf("[数字人] RAG错误: %v\n", err)
			answer = "抱歉，我暂时无法回答这个问题。"
			emotion = "sadness"
		} else {
			answer = response
			emotion = h.detectEmotion(answer)

			// 记录交互日志
			if pkg.StatsService != nil {
				pkg.StatsService.RecordInteraction(service.InteractionRecord{
					SessionID:      req.SessionID,
					Query:          req.InputText,
					Response:       answer,
					Emotion:        emotion,
					ResponseTimeMs: elapsed,
					Category:       service.DetectCategory(req.InputText),
					Source:         "digital_human",
				})
			}
		}
	} else {
		answer = "抱歉，智能服务暂不可用。"
		emotion = "sadness"
	}

	// 将表情标签嵌入到文本开头，让Live2D能够识别
	answerWithEmotion := fmt.Sprintf("[%s] %s", emotion, answer)

	responseData := ChatTextResponse{
		AnswerText: answerWithEmotion,
		Emotion:    emotion,
		TraceID:    traceID,
	}

	matchedRoute := h.matchRoute(req.InputText)
	if matchedRoute != nil {
		responseData.RoutePayload = &RoutePayload{
			RouteID: matchedRoute.Name,
			Stops:   h.extractStops(matchedRoute),
		}
	}

	pkg.Success(c, responseData)
}

type VoiceTranscriptRequest struct {
	SessionID   string  `json:"session_id"`
	UserID      string  `json:"user_id,omitempty"`
	Transcript  string  `json:"transcript"`
	Scene       string  `json:"scene,omitempty"`
	Location    string  `json:"location,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
}

type VoiceTranscriptResponse struct {
	AnswerText   string                 `json:"answer_text"`
	Emotion      string                 `json:"emotion"`
	RoutePayload *RoutePayload          `json:"route_payload,omitempty"`
	TraceID      string                 `json:"trace_id"`
}

func (h *DigitalHumanHandler) ChatVoiceTranscript(c *gin.Context) {
	var req VoiceTranscriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.Transcript == "" {
		pkg.BadRequest(c, "语音识别结果不能为空")
		return
	}

	traceID := uuid.New().String()
	fmt.Printf("[数字人] 语音聊天: session_id=%s, trace_id=%s, transcript=%s, confidence=%.2f\n",
		req.SessionID, traceID, req.Transcript, req.Confidence)

	startTime := time.Now()

	var answer string
	var emotion string

	if h.ragService != nil {
		response, _, err := h.ragService.QueryWithRAGAndRoute(req.Transcript)
		elapsed := time.Since(startTime).Milliseconds()
		if err != nil {
			fmt.Printf("[数字人] RAG错误: %v\n", err)
			answer = "抱歉，我暂时无法回答这个问题。"
			emotion = "sadness"
		} else {
			answer = response
			emotion = h.detectEmotion(answer)

			// 记录交互日志
			if pkg.StatsService != nil {
				pkg.StatsService.RecordInteraction(service.InteractionRecord{
					SessionID:      req.SessionID,
					Query:          req.Transcript,
					Response:       answer,
					Emotion:        emotion,
					ResponseTimeMs: elapsed,
					Category:       service.DetectCategory(req.Transcript),
					Source:         "voice",
				})
			}
		}
	} else {
		answer = "抱歉，智能服务暂不可用。"
		emotion = "sadness"
	}

	// 将表情标签嵌入到文本开头，让Live2D能够识别
	answerWithEmotion := fmt.Sprintf("[%s] %s", emotion, answer)

	responseData := VoiceTranscriptResponse{
		AnswerText: answerWithEmotion,
		Emotion:    emotion,
		TraceID:    traceID,
	}

	matchedRoute := h.matchRoute(req.Transcript)
	if matchedRoute != nil {
		responseData.RoutePayload = &RoutePayload{
			RouteID: matchedRoute.Name,
			Stops:   h.extractStops(matchedRoute),
		}
	}

	pkg.Success(c, responseData)
}

type FeedbackRequest struct {
	SessionID    string `json:"session_id"`
	TraceID      string `json:"trace_id"`
	QuestionType string `json:"question_type,omitempty"`
	ResponseTime int    `json:"response_time_ms,omitempty"`
	Rating       int    `json:"rating,omitempty"`
	Comment      string `json:"comment,omitempty"`
}

func (h *DigitalHumanHandler) SubmitFeedback(c *gin.Context) {
	var req FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	fmt.Printf("[数字人] 反馈上报: session_id=%s, trace_id=%s, rating=%d, comment=%s\n",
		req.SessionID, req.TraceID, req.Rating, req.Comment)

	if h.visitorQueryService != nil {
		query := &model.VisitorQuery{
			Query:      req.QuestionType,
			Response:   req.Comment,
			IsAnswered: true,
		}
		err := h.visitorQueryService.CreateQuery(query)
		if err != nil {
			fmt.Printf("[数字人] 反馈保存失败: %v\n", err)
		}
	}

	pkg.Success(c, gin.H{"message": "反馈已接收"})
}

func (h *DigitalHumanHandler) Health(c *gin.Context) {
	status := "healthy"
	ragStatus := "not_available"

	if h.ragService != nil {
		ragStatus = "available"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        status,
		"rag_service":   ragStatus,
		"route_service": "available",
		"timestamp":     time.Now().Format(time.RFC3339),
	})
}

func (h *DigitalHumanHandler) detectEmotion(text string) string {
	textLower := strings.ToLower(text)

	// 返回Live2D可识别的表情标签格式 [emotion]
	// 根据model_dict.json中的emotionMap映射：
	// neutral:0, anger:2, disgust:2, fear:1, joy:3, smirk:3, sadness:1, surprise:3

	if strings.Contains(textLower, "抱歉") || strings.Contains(textLower, "对不起") || strings.Contains(textLower, "无法") {
		return "sadness"  // 道歉/遗憾 -> sadness
	}
	if strings.Contains(textLower, "欢迎") || strings.Contains(textLower, "您好") || strings.Contains(textLower, "很高兴") {
		return "joy"  // 欢迎/高兴 -> joy
	}
	if strings.Contains(textLower, "推荐") || strings.Contains(textLower, "建议") || strings.Contains(textLower, "最佳") {
		return "joy"  // 推荐/建议 -> joy
	}
	if strings.Contains(textLower, "注意") || strings.Contains(textLower, "提醒") || strings.Contains(textLower, "警告") {
		return "surprise"  // 提醒/警告 -> surprise
	}
	if strings.Contains(textLower, "不好") || strings.Contains(textLower, "糟糕") || strings.Contains(textLower, "问题") {
		return "fear"  // 担心/问题 -> fear
	}
	return "neutral"  // 默认中性表情
}

func (h *DigitalHumanHandler) matchRoute(text string) *model.TourRoute {
	if h.routeService == nil {
		return nil
	}

	textLower := strings.ToLower(text)

	if strings.Contains(textLower, "亲子") || strings.Contains(textLower, "儿童") || strings.Contains(textLower, "家庭") {
		routes, _ := h.routeService.GetRoutesByDifficulty("easy")
		if len(routes) > 0 {
			return &routes[0]
		}
	}
	if strings.Contains(textLower, "历史") || strings.Contains(textLower, "文化") {
		routes, _ := h.routeService.GetAllRoutes()
		for _, r := range routes {
			nameLower := strings.ToLower(r.Name)
			if strings.Contains(nameLower, "历史") || strings.Contains(nameLower, "文化") {
				return &r
			}
		}
	}
	if strings.Contains(textLower, "自然") || strings.Contains(textLower, "风景") {
		routes, _ := h.routeService.GetAllRoutes()
		for _, r := range routes {
			nameLower := strings.ToLower(r.Name)
			if strings.Contains(nameLower, "自然") {
				return &r
			}
		}
	}
	return nil
}

func (h *DigitalHumanHandler) extractStops(route *model.TourRoute) []string {
	if route == nil || route.Spots == "" {
		return nil
	}
	return strings.Split(route.Spots, ",")
}

func (h *DigitalHumanHandler) Routes(r *gin.RouterGroup) {
	dh := r.Group("/dh")
	{
		dh.POST("/session/create", h.CreateSession)
		dh.POST("/chat/text", h.ChatText)
		dh.POST("/chat/voice-transcript", h.ChatVoiceTranscript)
		dh.POST("/feedback", h.SubmitFeedback)
		dh.GET("/health", h.Health)
	}
}