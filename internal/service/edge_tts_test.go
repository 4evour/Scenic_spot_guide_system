package service

import (
	"encoding/binary"
	"net/url"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestGenerateEdgeGECToken(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	const want = "A655E5F5FD06CCAD2AA66D046E46B8C3CE948F2077834022C50F33B429FA8BAC"
	if got := generateEdgeGECToken(now); got != want {
		t.Fatalf("generateEdgeGECToken() = %q, want %q", got, want)
	}
}

func TestBuildEdgeWSSURLIncludesCurrentProtocolParameters(t *testing.T) {
	endpoint, err := url.Parse(buildEdgeWSSURL(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC), "connectionid"))
	if err != nil {
		t.Fatalf("parse websocket url: %v", err)
	}
	query := endpoint.Query()
	if query.Get("TrustedClientToken") != edgeTrustedClientToken {
		t.Fatalf("TrustedClientToken = %q", query.Get("TrustedClientToken"))
	}
	if query.Get("ConnectionId") != "connectionid" {
		t.Fatalf("ConnectionId = %q", query.Get("ConnectionId"))
	}
	if query.Get("Sec-MS-GEC-Version") != "1-"+edgeChromiumVersion {
		t.Fatalf("Sec-MS-GEC-Version = %q", query.Get("Sec-MS-GEC-Version"))
	}
	if len(query.Get("Sec-MS-GEC")) != 64 {
		t.Fatalf("Sec-MS-GEC length = %d, want 64", len(query.Get("Sec-MS-GEC")))
	}
}

func TestNewEdgeMUIDIsUppercaseHex(t *testing.T) {
	muid := newEdgeMUID()
	if len(muid) != 32 || strings.ToUpper(muid) != muid || strings.Contains(muid, "-") {
		t.Fatalf("invalid MUID %q", muid)
	}
}

func TestExtractAudioPayloadUsesBinaryHeaderLength(t *testing.T) {
	header := []byte("X-RequestId:test\r\nContent-Type:audio/mpeg\r\nPath:audio\r\n")
	frame := make([]byte, 2+len(header)+5)
	binary.BigEndian.PutUint16(frame[:2], uint16(len(header)))
	copy(frame[2:], header)
	copy(frame[2+len(header):], []byte("audio"))

	audio, err := extractAudioPayload(frame)
	if err != nil {
		t.Fatalf("extractAudioPayload() error = %v", err)
	}
	if got := string(audio); got != "audio" {
		t.Fatalf("extractAudioPayload() = %q, want audio", got)
	}
}

func TestExtractAudioPayloadRejectsInvalidBinaryFrames(t *testing.T) {
	validHeader := []byte("X-RequestId:test\r\nContent-Type:audio/mpeg\r\nPath:audio\r\n")
	validFrame := make([]byte, 2+len(validHeader)+5)
	binary.BigEndian.PutUint16(validFrame[:2], uint16(len(validHeader)))
	copy(validFrame[2:], validHeader)
	copy(validFrame[2+len(validHeader):], []byte("audio"))

	tests := []struct {
		name  string
		frame []byte
	}{
		{name: "missing length header", frame: []byte("not a binary audio frame")},
		{name: "invalid declared header length", frame: []byte{0, 20, 'x'}},
		{name: "missing path audio", frame: binaryFrame("X-RequestId:test\r\nContent-Type:audio/mpeg\r\n", []byte("audio"))},
		{name: "missing header delimiter", frame: binaryFrame("Path:audio", []byte("audio"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audio, err := extractAudioPayload(tt.frame)
			if err == nil {
				t.Fatalf("extractAudioPayload() audio = %q, want error", audio)
			}
		})
	}

	if _, err := extractAudioPayload(validFrame); err != nil {
		t.Fatalf("valid frame rejected: %v", err)
	}
}

func TestSplitEdgeTTSTextKeepsLongUTF8TextWithinSafeLimit(t *testing.T) {
	text := strings.Repeat("灵山胜境欢迎您，&请注意安全。", 1000)
	chunks := splitEdgeTTSText(text, edgeMaxTextBytes)
	if len(chunks) < 2 {
		t.Fatalf("chunk count = %d, want multiple chunks", len(chunks))
	}
	if got := strings.Join(chunks, ""); got != text {
		t.Fatalf("joined chunks differ from original text")
	}
	for i, chunk := range chunks {
		if !utf8.ValidString(chunk) {
			t.Fatalf("chunk %d is not valid UTF-8", i)
		}
		if got := len(xmlEscape(chunk)); got > edgeMaxTextBytes {
			t.Fatalf("chunk %d escaped byte length = %d, limit = %d", i, got, edgeMaxTextBytes)
		}
	}
}

func binaryFrame(header string, payload []byte) []byte {
	frame := make([]byte, 2+len(header)+len(payload))
	binary.BigEndian.PutUint16(frame[:2], uint16(len(header)))
	copy(frame[2:], header)
	copy(frame[2+len(header):], payload)
	return frame
}
