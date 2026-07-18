package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/scenic-guide/internal/config"
)

var ErrMultimodalDisabled = errors.New("multimodal service is disabled")

type MultimodalPart struct {
	Kind     string
	MIMEType string
	Data     []byte
}

type MultimodalResult struct {
	Text     string
	Model    string
	Modality string
}

type MultimodalClient struct {
	cfg        config.MultimodalConfig
	httpClient *http.Client
	guard      *modelGuard
}

func NewMultimodalClient(cfg *config.MultimodalConfig) (*MultimodalClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("multimodal config is nil")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	guard := newModelGuard("multimodal")
	guard.cfg.Timeout = timeout
	return &MultimodalClient{
		cfg: *cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		guard: guard,
	}, nil
}

func (c *MultimodalClient) Enabled() bool {
	return c != nil && c.cfg.Enabled
}

func (c *MultimodalClient) ModelHealth() ModelProviderHealth {
	if c == nil || !c.cfg.Enabled {
		return ModelProviderHealth{Provider: "multimodal", State: "disabled"}
	}
	return c.guard.health()
}

func (c *MultimodalClient) Chat(ctx context.Context, text string, parts []MultimodalPart) (MultimodalResult, error) {
	if c == nil || !c.cfg.Enabled {
		return MultimodalResult{}, ErrMultimodalDisabled
	}
	text = strings.TrimSpace(text)
	if text == "" && len(parts) == 0 {
		return MultimodalResult{}, fmt.Errorf("multimodal input is empty")
	}

	content := make([]multimodalContent, 0, len(parts)+1)
	if text != "" {
		content = append(content, multimodalContent{Type: "text", Text: text})
	}
	for _, part := range parts {
		item, err := buildMultimodalContent(part)
		if err != nil {
			return MultimodalResult{}, err
		}
		content = append(content, item)
	}

	body, err := json.Marshal(multimodalRequest{
		Model: c.cfg.Model,
		Messages: []multimodalMessage{{
			Role:    "user",
			Content: content,
		}},
	})
	if err != nil {
		return MultimodalResult{}, fmt.Errorf("marshal multimodal request: %w", err)
	}

	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	var responseBody []byte
	err = c.guard.run(ctx, func(callCtx context.Context) error {
		req, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<20))
			return &modelHTTPError{status: resp.StatusCode}
		}
		responseBody, err = io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		return err
	})
	if err != nil {
		return MultimodalResult{}, fmt.Errorf("call multimodal provider: %w", err)
	}

	var parsed multimodalResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return MultimodalResult{}, fmt.Errorf("parse multimodal response: %w", err)
	}
	if parsed.Error != nil {
		return MultimodalResult{}, fmt.Errorf("multimodal provider returned an error")
	}
	if len(parsed.Choices) == 0 {
		return MultimodalResult{}, fmt.Errorf("multimodal provider returned no choices")
	}
	answer, err := extractMultimodalText(parsed.Choices[0].Message.Content)
	if err != nil {
		return MultimodalResult{}, err
	}
	if strings.TrimSpace(answer) == "" {
		return MultimodalResult{}, fmt.Errorf("multimodal provider returned empty content")
	}

	return MultimodalResult{
		Text:     answer,
		Model:    c.cfg.Model,
		Modality: modalityForParts(parts),
	}, nil
}

type multimodalRequest struct {
	Model    string              `json:"model"`
	Messages []multimodalMessage `json:"messages"`
}

type multimodalMessage struct {
	Role    string              `json:"role"`
	Content []multimodalContent `json:"content"`
}

type multimodalContent struct {
	Type       string                `json:"type"`
	Text       string                `json:"text,omitempty"`
	ImageURL   *multimodalImageURL   `json:"image_url,omitempty"`
	InputAudio *multimodalInputAudio `json:"input_audio,omitempty"`
	VideoURL   *multimodalVideoURL   `json:"video_url,omitempty"`
}

type multimodalImageURL struct {
	URL string `json:"url"`
}

type multimodalInputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type multimodalVideoURL struct {
	URL string `json:"url"`
}

type multimodalResponse struct {
	Choices []struct {
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func buildMultimodalContent(part MultimodalPart) (multimodalContent, error) {
	if len(part.Data) == 0 {
		return multimodalContent{}, fmt.Errorf("multimodal part %q is empty", part.Kind)
	}
	if strings.TrimSpace(part.MIMEType) == "" {
		return multimodalContent{}, fmt.Errorf("multimodal part %q has no MIME type", part.Kind)
	}
	encoded := base64.StdEncoding.EncodeToString(part.Data)
	switch part.Kind {
	case "image":
		return multimodalContent{
			Type:     "image_url",
			ImageURL: &multimodalImageURL{URL: "data:" + part.MIMEType + ";base64," + encoded},
		}, nil
	case "audio":
		format := "webm"
		if strings.HasSuffix(strings.ToLower(part.MIMEType), "/wav") {
			format = "wav"
		} else if strings.HasSuffix(strings.ToLower(part.MIMEType), "/mpeg") {
			format = "mp3"
		} else if strings.HasSuffix(strings.ToLower(part.MIMEType), "/ogg") {
			format = "ogg"
		}
		return multimodalContent{
			Type:       "input_audio",
			InputAudio: &multimodalInputAudio{Data: encoded, Format: format},
		}, nil
	case "video":
		return multimodalContent{
			Type:     "video_url",
			VideoURL: &multimodalVideoURL{URL: "data:" + part.MIMEType + ";base64," + encoded},
		}, nil
	default:
		return multimodalContent{}, fmt.Errorf("unsupported multimodal part kind %q", part.Kind)
	}
}

func extractMultimodalText(raw json.RawMessage) (string, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}
	var contents []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &contents); err != nil {
		return "", fmt.Errorf("parse multimodal content: %w", err)
	}
	var builder strings.Builder
	for _, item := range contents {
		builder.WriteString(item.Text)
	}
	return builder.String(), nil
}

func modalityForParts(parts []MultimodalPart) string {
	if len(parts) == 0 {
		return "text"
	}
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		seen[part.Kind] = true
	}
	if len(seen) == 1 {
		for kind := range seen {
			return "text_" + kind
		}
	}
	return "mixed"
}
