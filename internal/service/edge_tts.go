package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// EdgeTTSService provides TTS via Microsoft Edge's free public TTS API.
// It connects to the Edge speech service over WebSocket and streams audio back.
type EdgeTTSService struct {
	dialer   websocket.Dialer
	timeout  time.Duration
}

// EdgeTTSConfig holds configuration for the Edge TTS service.
type EdgeTTSConfig struct {
	Voice   string // e.g. "zh-CN-XiaoxiaoNeural"
	Rate    string // e.g. "+0%", "-10%", "+20%"
	Timeout time.Duration
}

// DefaultEdgeTTSConfig returns sensible defaults.
func DefaultEdgeTTSConfig() EdgeTTSConfig {
	return EdgeTTSConfig{
		Voice:   "zh-CN-XiaoxiaoNeural",
		Rate:    "+0%",
		Timeout: 30 * time.Second,
	}
}

// NewEdgeTTSService creates a new Edge TTS service.
func NewEdgeTTSService(timeout time.Duration) *EdgeTTSService {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &EdgeTTSService{
		dialer: websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
		timeout: timeout,
	}
}

const (
	edgeWSSURL   = "wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1?TrustedClientToken=6A5AA1D4EAFF4E9FB37E23D68491C6F4"
	edgeOrigin   = "chrome-extension://jdkknkkbebbapilgoeccciglkfbmbnfm"
)

// edgeVoices maps friendly names to Microsoft Speech Service voice names.
var edgeVoices = map[string]string{
	"female_xiaoxiao": "zh-CN-XiaoxiaoNeural",
	"female_xiaoyi":   "zh-CN-XiaoyiNeural",
	"female_yunxi":    "zh-CN-YunxiNeural",
	"male_yunyang":    "zh-CN-YunyangNeural",
	"male_yunjian":    "zh-CN-YunjianNeural",
	"female_xiaobei":  "zh-CN-liaoning-XiaobeiNeural",
	"female_yunxia":   "zh-CN-shaanxi-YunxiaNeural",
}

// ResolveVoice returns the full voice name for a friendly key, falling back to the input.
func ResolveVoice(key string) string {
	if v, ok := edgeVoices[key]; ok {
		return v
	}
	// If it already looks like a full voice name (contains "Neural"), use as-is.
	if strings.Contains(key, "Neural") {
		return key
	}
	return edgeVoices["female_xiaoxiao"]
}

// Synthesize generates audio for the given text, returning the complete MP3 data.
// This is the non-streaming variant. Prefer SynthesizeStream for real-time use.
func (s *EdgeTTSService) Synthesize(ctx context.Context, text, voice, rate string) ([]byte, error) {
	chunks, errCh := s.SynthesizeStream(ctx, text, voice, rate)

	var result []byte
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				return result, nil
			}
			result = append(result, chunk...)
		case err := <-errCh:
			if err != nil {
				return nil, err
			}
			return result, nil
		case <-ctx.Done():
			return result, ctx.Err()
		}
	}
}

// SynthesizeStream streams audio chunks from Edge TTS.
// Returns a channel of audio/mpeg binary chunks and an error channel.
// The chunk channel is closed when synthesis completes.
func (s *EdgeTTSService) SynthesizeStream(ctx context.Context, text, voice, rate string) (<-chan []byte, <-chan error) {
	chunkCh := make(chan []byte, 8)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		if err := s.streamInternal(ctx, text, voice, rate, chunkCh); err != nil {
			errCh <- err
		}
	}()

	return chunkCh, errCh
}

func (s *EdgeTTSService) streamInternal(ctx context.Context, text, voice, rate string, chunkCh chan<- []byte) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	voice = ResolveVoice(voice)
	if rate == "" {
		rate = "+0%"
	}

	// Build SSML
	ssml := buildSSML(text, voice, rate)

	// Connect
	header := http.Header{}
	header.Set("Origin", edgeOrigin)
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	conn, _, err := s.dialer.DialContext(ctx, edgeWSSURL, header)
	if err != nil {
		return fmt.Errorf("edge tts: dial failed: %w", err)
	}
	defer conn.Close()

	done := make(chan struct{})
	errChan := make(chan error, 1)

	// Read goroutine
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msgType, reader, err := conn.NextReader()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					errChan <- fmt.Errorf("edge tts: read error: %w", err)
				}
				return
			}

			switch msgType {
			case websocket.BinaryMessage:
				data, readErr := io.ReadAll(reader)
				if readErr != nil && readErr != io.EOF {
					errChan <- fmt.Errorf("edge tts: read binary: %w", readErr)
					return
				}
				// Each binary frame contains header prefix + audio data.
				// Headers end with \r\n\r\n; audio follows.
				audio := extractAudioPayload(data)
				if len(audio) > 0 {
					select {
					case chunkCh <- audio:
					case <-ctx.Done():
						return
					}
				}

			case websocket.TextMessage:
				textData, readErr := io.ReadAll(reader)
				if readErr != nil && readErr != io.EOF {
					slog.Debug("edge tts: text message read error", "error", readErr)
				}
				// Check for error in turn.finish
				if strings.Contains(string(textData), "Path:turn.finish") {
					return
				}
			}
		}
	}()

	// Send config message
	requestID := uuid.New().String()
	configMsg := buildConfigMessage(requestID)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(configMsg)); err != nil {
		return fmt.Errorf("edge tts: send config: %w", err)
	}

	// Send SSML message
	ssmlMsg := buildSSMLMessage(requestID, ssml)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(ssmlMsg)); err != nil {
		return fmt.Errorf("edge tts: send ssml: %w", err)
	}

	// Wait for completion or error
	select {
	case <-done:
		return nil
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// extractAudioPayload strips the HTTP-style headers from a binary WebSocket frame,
// returning just the raw audio data. Edge TTS prepends headers like:
//
//	X-RequestId:...\r\nContent-Type:audio/mpeg\r\nPath:audio\r\n\r\n<binary audio>
func extractAudioPayload(frame []byte) []byte {
	// Header section ends at \r\n\r\n
	const delim = "\r\n\r\n"
	idx := 0
	// Search for the delimiter in the binary data
	for i := 0; i <= len(frame)-len(delim); i++ {
		if string(frame[i:i+len(delim)]) == delim {
			idx = i + len(delim)
			break
		}
	}
	if idx >= len(frame) {
		return frame
	}
	return frame[idx:]
}

func buildSSML(text, voice, rate string) string {
	// Escape XML special chars in text
	text = xmlEscape(text)
	return fmt.Sprintf(
		`<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xmlns:mstts='http://www.w3.org/2001/mstts' xml:lang='zh-CN'><voice name='%s'><prosody rate='%s'>%s</prosody></voice></speak>`,
		voice, rate, text,
	)
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func buildConfigMessage(requestID string) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(
		"X-RequestId:%s\r\nContent-Type:application/json; charset=utf-8\r\nX-Timestamp:%s\r\nPath:speech.config\r\n\r\n"+
			`{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":false,"wordBoundaryEnabled":false},"outputFormat":"audio-24khz-48kbitrate-mono-mp3"}}}}`,
		requestID, timestamp,
	)
}

func buildSSMLMessage(requestID, ssml string) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(
		"X-RequestId:%s\r\nContent-Type:application/ssml+xml\r\nX-Timestamp:%s\r\nPath:ssml\r\n\r\n%s",
		requestID, timestamp, ssml,
	)
}
