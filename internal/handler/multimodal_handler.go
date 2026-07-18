package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

const (
	maxMultimodalImageSize = 10 << 20
	maxMultimodalAudioSize = 20 << 20
	maxMultimodalVideoSize = 50 << 20
	maxMultimodalPartCount = 3
)

type multimodalUploadSpec struct {
	field    string
	kind     string
	maxBytes int64
	allowed  map[string]bool
}

var multimodalUploadSpecs = []multimodalUploadSpec{
	{field: "image", kind: "image", maxBytes: maxMultimodalImageSize, allowed: map[string]bool{
		"image/jpeg": true, "image/png": true, "image/webp": true,
	}},
	{field: "audio", kind: "audio", maxBytes: maxMultimodalAudioSize, allowed: map[string]bool{
		"audio/wav": true, "audio/mpeg": true, "audio/ogg": true, "audio/webm": true,
	}},
	{field: "video", kind: "video", maxBytes: maxMultimodalVideoSize, allowed: map[string]bool{
		"video/mp4": true, "video/webm": true,
	}},
}

func (h *AIHandler) MultimodalChat(c *gin.Context) {
	if h.multimodalClient == nil || !h.multimodalClient.Enabled() {
		c.JSON(http.StatusServiceUnavailable, pkg.Response{
			Code:    http.StatusServiceUnavailable,
			Message: pkg.T(c, "msg_multimodal_unavailable"),
		})
		return
	}

	if err := c.Request.ParseMultipartForm(maxMultimodalBodySize); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	message := strings.TrimSpace(c.PostForm("message"))
	if len([]rune(message)) > 4000 {
		pkg.BadRequest(c, pkg.T(c, "msg_multimodal_message_too_long"))
		return
	}

	parts := make([]service.MultimodalPart, 0, len(multimodalUploadSpecs))
	for _, spec := range multimodalUploadSpecs {
		part, present, err := readMultimodalUpload(c, spec)
		if err != nil {
			pkg.BadRequest(c, err.Error())
			return
		}
		if present {
			parts = append(parts, part)
		}
	}
	if message == "" && len(parts) == 0 {
		pkg.BadRequest(c, pkg.T(c, "msg_multimodal_input_empty"))
		return
	}
	if len(parts) > maxMultimodalPartCount {
		pkg.BadRequest(c, pkg.T(c, "msg_multimodal_too_many_files"))
		return
	}

	traceID := uuid.NewString()
	start := time.Now()
	result, err := h.multimodalClient.Chat(c.Request.Context(), message, parts)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		slog.Error("多模态请求失败", "trace_id", traceID, "elapsed_ms", elapsed, "part_count", len(parts), "error", err)
		if message != "" && h.ragService != nil {
			answer, _, trace, fallbackErr := h.ragService.QueryWithRAGAndRouteTraceInSession(
				c.Request.Context(),
				strings.TrimSpace(c.PostForm("session_id")),
				message,
				"zh-CN",
			)
			if fallbackErr == nil {
				pkg.Success(c, gin.H{
					"response":       answer,
					"trace_id":       trace.TraceID,
					"sources":        trace.Sources,
					"confidence":     trace.Confidence,
					"should_abstain": trace.ShouldAbstain,
					"degraded":       true,
					"elapsed_ms":     elapsed,
				})
				return
			}
			slog.Warn("多模态文字降级失败", "trace_id", traceID, "error", fallbackErr)
		}
		c.JSON(http.StatusBadGateway, pkg.Response{
			Code:    http.StatusBadGateway,
			Message: pkg.T(c, "msg_multimodal_failed"),
		})
		return
	}

	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID, _ = uid.(uint)
	}
	if h.statsService != nil {
		h.statsService.RecordInteraction(service.InteractionRecord{
			UserID:         userID,
			SessionID:      strings.TrimSpace(c.PostForm("session_id")),
			Query:          message,
			Response:       result.Text,
			ResponseTimeMs: elapsed,
			Category:       "multimodal",
			Source:         "multimodal",
		})
	}

	pkg.Success(c, gin.H{
		"response":   result.Text,
		"model":      result.Model,
		"modality":   result.Modality,
		"trace_id":   traceID,
		"degraded":   false,
		"elapsed_ms": elapsed,
	})
}

func readMultimodalUpload(c *gin.Context, spec multimodalUploadSpec) (service.MultimodalPart, bool, error) {
	header, err := c.FormFile(spec.field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return service.MultimodalPart{}, false, nil
		}
		return service.MultimodalPart{}, false, fmt.Errorf("%s upload failed", spec.field)
	}
	if header.Size <= 0 || header.Size > spec.maxBytes {
		return service.MultimodalPart{}, false, fmt.Errorf("%s file is empty or too large", spec.field)
	}
	declaredType := strings.ToLower(strings.TrimSpace(strings.Split(header.Header.Get("Content-Type"), ";")[0]))
	if !spec.allowed[declaredType] {
		return service.MultimodalPart{}, false, fmt.Errorf("%s file type is not allowed", spec.field)
	}

	file, err := header.Open()
	if err != nil {
		return service.MultimodalPart{}, false, fmt.Errorf("open %s file failed", spec.field)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, spec.maxBytes+1))
	if err != nil || int64(len(data)) > spec.maxBytes {
		return service.MultimodalPart{}, false, fmt.Errorf("read %s file failed", spec.field)
	}
	if !hasRecognizedMediaSignature(spec.kind, declaredType, data) {
		return service.MultimodalPart{}, false, fmt.Errorf("%s file content does not match its type", spec.field)
	}
	return service.MultimodalPart{Kind: spec.kind, MIMEType: declaredType, Data: data}, true, nil
}

func hasRecognizedMediaSignature(kind, mimeType string, data []byte) bool {
	if len(data) < 4 {
		return false
	}
	switch kind {
	case "image":
		detected := http.DetectContentType(data)
		return strings.HasPrefix(detected, "image/")
	case "audio":
		return (mimeType == "audio/wav" && len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE") ||
			(mimeType == "audio/ogg" && string(data[:4]) == "OggS") ||
			(mimeType == "audio/webm" && data[0] == 0x1a && data[1] == 0x45 && data[2] == 0xdf && data[3] == 0xa3) ||
			(mimeType == "audio/mpeg" && (string(data[:3]) == "ID3" || (data[0] == 0xff && data[1]&0xe0 == 0xe0)))
	case "video":
		return (mimeType == "video/mp4" && len(data) >= 8 && string(data[4:8]) == "ftyp") ||
			(mimeType == "video/webm" && data[0] == 0x1a && data[1] == 0x45 && data[2] == 0xdf && data[3] == 0xa3)
	default:
		return false
	}
}
