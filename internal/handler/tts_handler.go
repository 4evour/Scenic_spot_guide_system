package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

// TTSHandler handles Text-to-Speech requests using Microsoft Edge TTS.
type TTSHandler struct {
	edgeTTS *service.EdgeTTSService
	timeout time.Duration
}

// NewTTSHandler creates a TTS handler with Edge TTS as the backend.
func NewTTSHandler() *TTSHandler {
	timeout := 30 * time.Second
	return &TTSHandler{
		edgeTTS: service.NewEdgeTTSService(timeout),
		timeout: timeout,
	}
}

// TTSRequest is the request body for TTS endpoints.
type TTSRequest struct {
	Text  string `json:"text"`
	Voice string `json:"voice"`
	Rate  string `json:"rate"`
}

// TTS handles non-streaming TTS: returns the complete audio as a single response.
// POST /api/v1/ai/tts
func (h *TTSHandler) TTS(c *gin.Context) {
	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		pkg.BadRequest(c, pkg.T(c, "msg_tts_empty_text"))
		return
	}

	if req.Voice == "" {
		req.Voice = "female_xiaoxiao"
	}
	if req.Rate == "" {
		req.Rate = "+0%"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	data, err := h.edgeTTS.Synthesize(ctx, req.Text, req.Voice, req.Rate)
	if err != nil {
		slog.Error("Edge TTS 合成失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_tts_failed"))
		return
	}

	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Disposition", "inline; filename=tts.mp3")
	c.Data(http.StatusOK, "audio/mpeg", data)
}

// TTSStream handles streaming TTS: audio chunks are written as they arrive.
// POST /api/v1/ai/tts/stream
func (h *TTSHandler) TTSStream(c *gin.Context) {
	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		pkg.BadRequest(c, pkg.T(c, "msg_tts_empty_text"))
		return
	}

	if req.Voice == "" {
		req.Voice = "female_xiaoxiao"
	}
	if req.Rate == "" {
		req.Rate = "+0%"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	chunks, errCh := h.edgeTTS.SynthesizeStream(ctx, req.Text, req.Voice, req.Rate)

	c.Header("Content-Type", "audio/mpeg")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		slog.Error("streaming not supported by http response writer")
		pkg.InternalError(c, pkg.T(c, "msg_streaming_unsupported"))
		return
	}

	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				// Channel closed: synthesis complete
				return
			}
			if _, err := c.Writer.Write(chunk); err != nil {
				slog.Debug("tts stream: client disconnected", "error", err)
				return
			}
			flusher.Flush()

		case err := <-errCh:
			if err != nil {
				slog.Error("Edge TTS 流式合成失败", "error", err)
			}
			return

		case <-ctx.Done():
			slog.Debug("tts stream: context done", "error", ctx.Err())
			return
		}
	}
}

// Routes registers TTS endpoints.
func (h *TTSHandler) Routes(r *gin.RouterGroup) {
	r.POST("/ai/tts", h.TTS)
	r.POST("/ai/tts/stream", h.TTSStream)
}
