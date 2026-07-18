package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// EdgeTTSService provides TTS via Microsoft Edge's free public TTS API.
// It connects to the Edge speech service over WebSocket and streams audio back.
type EdgeTTSService struct {
	dialer  websocket.Dialer
	timeout time.Duration
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
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	// Edge TTS intermittently closes IPv6 handshakes in environments where the
	// host resolves to both address families. Keep the external connection on
	// IPv4 so a transient IPv6 route cannot turn into a synthetic 500 response.
	dialer.NetDialContext = func(ctx context.Context, _ string, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp4", address)
	}
	return &EdgeTTSService{
		dialer:  dialer,
		timeout: timeout,
	}
}

const (
	edgeWSSBaseURL         = "wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1"
	edgeTrustedClientToken = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"
	edgeChromiumVersion    = "143.0.3650.75"
	edgeOrigin             = "chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold"
	edgeUserAgent          = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 Edg/143.0.0.0"
	edgeWindowsEpoch       = int64(11644473600)
	edgeMaxTextBytes       = 4096
)

// ErrEdgeTTSNoAudio indicates that Edge completed a synthesis session without an audio frame.
var ErrEdgeTTSNoAudio = errors.New("edge tts: no audio received")

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

// ResolveVoice returns the full voice name for a friendly key.
// 仅接受 edgeVoices 白名单内的 key,未命中时回退到默认声音(女声-晓晓)。
// 此前这里有一个"key 含 Neural 则原样返回"的透传分支,会被攻击者用于 SSML 注入,
// 现已移除——voice 必须来自固定白名单。
func ResolveVoice(key string) string {
	if v, ok := edgeVoices[key]; ok {
		return v
	}
	return edgeVoices["female_xiaoxiao"]
}

// validRatePattern 匹配合法的 SSML prosody rate 值,如 "+0%"、"-10%"、"+150%"。
// rate 会被直接拼入 <prosody rate='%s'>,必须严格校验以防止 SSML 注入。
var validRatePattern = regexp.MustCompile(`^[+-]\d{1,3}%$`)

// validateRate 校验 rate 格式,非法时返回默认值 "+0%"。
func validateRate(rate string) string {
	if validRatePattern.MatchString(rate) {
		return rate
	}
	return "+0%"
}

// Synthesize generates audio for the given text, returning the complete MP3 data.
// This is the non-streaming variant. Prefer SynthesizeStream for real-time use.
func (s *EdgeTTSService) Synthesize(ctx context.Context, text, voice, rate string) ([]byte, error) {
	chunks, errCh := s.SynthesizeStream(ctx, text, voice, rate)

	var result []byte
	for chunks != nil || errCh != nil {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				chunks = nil
				continue
			}
			result = append(result, chunk...)
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err != nil {
				return nil, err
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if len(result) == 0 {
		return nil, ErrEdgeTTSNoAudio
	}
	return result, nil
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
	voice = ResolveVoice(voice)
	if rate == "" {
		rate = "+0%"
	}
	// 严格校验 rate 格式,防止 SSML 注入(rate 会直接拼入 <prosody rate='%s'>)。
	rate = validateRate(rate)

	segments := splitEdgeTTSText(text, edgeMaxTextBytes)
	for i, segment := range segments {
		segmentCtx, cancel := context.WithTimeout(ctx, s.timeout)
		err := s.streamSegment(segmentCtx, segment, voice, rate, chunkCh)
		cancel()
		if err != nil {
			return fmt.Errorf("edge tts: segment %d/%d: %w", i+1, len(segments), err)
		}
	}
	return nil
}

func (s *EdgeTTSService) streamSegment(ctx context.Context, text, voice, rate string, chunkCh chan<- []byte) error {
	ssml := buildSSML(text, voice, rate)

	// Connect
	header := http.Header{}
	header.Set("Pragma", "no-cache")
	header.Set("Cache-Control", "no-cache")
	header.Set("Origin", edgeOrigin)
	header.Set("User-Agent", edgeUserAgent)
	header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	header.Set("Accept-Language", "en-US,en;q=0.9")
	header.Set("Cookie", "muid="+newEdgeMUID()+";")

	connectionID := strings.ReplaceAll(uuid.NewString(), "-", "")
	conn, _, err := s.dialer.DialContext(ctx, buildEdgeWSSURL(time.Now().UTC(), connectionID), header)
	if err != nil {
		return fmt.Errorf("edge tts: dial failed: %w", err)
	}
	defer conn.Close()

	readResult := make(chan error, 1)
	go func() {
		readResult <- readEdgeTTSMessages(ctx, conn, chunkCh)
	}()

	// Send config message
	configMsg := buildConfigMessage()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(configMsg)); err != nil {
		return fmt.Errorf("edge tts: send config: %w", err)
	}

	// Send SSML message
	ssmlMsg := buildSSMLMessage(uuid.NewString(), ssml)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(ssmlMsg)); err != nil {
		return fmt.Errorf("edge tts: send ssml: %w", err)
	}

	// Wait for completion or error
	select {
	case err := <-readResult:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func readEdgeTTSMessages(ctx context.Context, conn *websocket.Conn, chunkCh chan<- []byte) error {
	audioReceived := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgType, reader, err := conn.NextReader()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) && audioReceived {
				return nil
			}
			return fmt.Errorf("edge tts: read error: %w", err)
		}

		switch msgType {
		case websocket.BinaryMessage:
			data, readErr := io.ReadAll(reader)
			if readErr != nil && readErr != io.EOF {
				return fmt.Errorf("edge tts: read binary: %w", readErr)
			}
			audio, extractErr := extractAudioPayload(data)
			if extractErr != nil {
				return extractErr
			}
			if len(audio) == 0 {
				continue
			}
			audioReceived = true
			select {
			case chunkCh <- audio:
			case <-ctx.Done():
				return ctx.Err()
			}

		case websocket.TextMessage:
			textData, readErr := io.ReadAll(reader)
			if readErr != nil && readErr != io.EOF {
				return fmt.Errorf("edge tts: read text: %w", readErr)
			}
			if strings.Contains(string(textData), "Path:turn.end") || strings.Contains(string(textData), "Path:turn.finish") {
				if !audioReceived {
					return ErrEdgeTTSNoAudio
				}
				return nil
			}
		}
	}
}

func buildEdgeWSSURL(now time.Time, connectionID string) string {
	query := url.Values{}
	query.Set("TrustedClientToken", edgeTrustedClientToken)
	query.Set("ConnectionId", connectionID)
	query.Set("Sec-MS-GEC", generateEdgeGECToken(now))
	query.Set("Sec-MS-GEC-Version", "1-"+edgeChromiumVersion)
	return edgeWSSBaseURL + "?" + query.Encode()
}

func generateEdgeGECToken(now time.Time) string {
	seconds := now.UTC().Unix()
	seconds -= seconds % 300
	ticks := (seconds + edgeWindowsEpoch) * 10_000_000
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d%s", ticks, edgeTrustedClientToken)))
	return strings.ToUpper(hex.EncodeToString(digest[:]))
}

func newEdgeMUID() string {
	return strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))
}

func edgeTimestamp(now time.Time) string {
	return now.UTC().Format("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)")
}

// extractAudioPayload validates and strips the length-prefixed headers from a
// binary WebSocket frame. Edge TTS prepends headers like:
//
//	<2-byte header length>X-RequestId:...\r\nContent-Type:audio/mpeg\r\nPath:audio\r\n<audio>
func extractAudioPayload(frame []byte) ([]byte, error) {
	if len(frame) < 2 {
		return nil, errors.New("edge tts: invalid audio frame: missing header length")
	}

	headerLength := int(binary.BigEndian.Uint16(frame[:2]))
	if headerLength <= 0 || headerLength > len(frame)-2 {
		return nil, errors.New("edge tts: invalid audio frame: invalid header length")
	}

	header := frame[2 : 2+headerLength]
	if !bytes.HasSuffix(header, []byte("\r\n")) {
		return nil, errors.New("edge tts: invalid audio frame: missing header delimiter")
	}

	pathAudio := false
	for _, line := range bytes.Split(header[:len(header)-2], []byte("\r\n")) {
		key, value, ok := bytes.Cut(line, []byte(":"))
		if !ok || len(key) == 0 {
			return nil, errors.New("edge tts: invalid audio frame: malformed header")
		}
		if bytes.EqualFold(key, []byte("Path")) && bytes.Equal(bytes.TrimSpace(value), []byte("audio")) {
			pathAudio = true
		}
	}
	if !pathAudio {
		return nil, errors.New("edge tts: invalid audio frame: Path is not audio")
	}

	return frame[2+headerLength:], nil
}

func splitEdgeTTSText(text string, maxEscapedBytes int) []string {
	if text == "" || maxEscapedBytes <= 0 {
		return []string{text}
	}

	chunks := make([]string, 0, len(text)/maxEscapedBytes+1)
	for start := 0; start < len(text); {
		cut := len(text)
		lastBoundary := -1
		escapedBytes := 0

		for pos := start; pos < len(text); {
			r, size := utf8.DecodeRuneInString(text[pos:])
			if escapedBytes+escapedRuneBytes(r) > maxEscapedBytes {
				cut = pos
				if lastBoundary > start {
					cut = lastBoundary
				}
				if cut == start {
					cut = pos + size
				}
				break
			}

			escapedBytes += escapedRuneBytes(r)
			pos += size
			if isEdgeTextBoundary(r) {
				lastBoundary = pos
			}
		}

		chunks = append(chunks, text[start:cut])
		start = cut
	}
	return chunks
}

func escapedRuneBytes(r rune) int {
	switch r {
	case '&':
		return len("&amp;")
	case '<':
		return len("&lt;")
	case '>':
		return len("&gt;")
	case '"':
		return len("&quot;")
	case '\'':
		return len("&apos;")
	default:
		return utf8.RuneLen(r)
	}
}

func isEdgeTextBoundary(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case '。', '！', '？', '；', '.', '!', '?', ';':
		return true
	default:
		return false
	}
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

func buildConfigMessage() string {
	timestamp := edgeTimestamp(time.Now())
	return fmt.Sprintf(
		"X-Timestamp:%s\r\nContent-Type:application/json; charset=utf-8\r\nPath:speech.config\r\n\r\n"+
			`{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":false,"wordBoundaryEnabled":false},"outputFormat":"audio-24khz-48kbitrate-mono-mp3"}}}}`,
		timestamp,
	)
}

func buildSSMLMessage(requestID, ssml string) string {
	timestamp := edgeTimestamp(time.Now())
	return fmt.Sprintf(
		"X-RequestId:%s\r\nContent-Type:application/ssml+xml\r\nX-Timestamp:%sZ\r\nPath:ssml\r\n\r\n%s",
		requestID, timestamp, ssml,
	)
}
