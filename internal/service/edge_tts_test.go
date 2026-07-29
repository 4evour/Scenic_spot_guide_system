package service

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
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

func TestNewEdgeTTSServiceUsesIPv4Dialer(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp4: %v", err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- conn
	}()

	service := NewEdgeTTSService(time.Second)
	conn, err := service.dialer.NetDialContext(context.Background(), "tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial through Edge TTS dialer: %v", err)
	}
	defer conn.Close()

	select {
	case peer := <-accepted:
		defer peer.Close()
		addr, ok := peer.RemoteAddr().(*net.TCPAddr)
		if !ok || addr.IP.To4() == nil {
			t.Fatalf("peer address = %v, want IPv4", peer.RemoteAddr())
		}
	case err := <-acceptErr:
		t.Fatalf("accept: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for IPv4 connection")
	}
}

func TestNewEdgeTTSServiceBoundsHandshakeSilence(t *testing.T) {
	service := NewEdgeTTSService(30 * time.Second)
	if service.dialer.HandshakeTimeout > 4*time.Second {
		t.Fatalf("HandshakeTimeout = %v, want <= 4s", service.dialer.HandshakeTimeout)
	}
}

func TestRaceEdgeIPv4DialUsesFirstHealthyAddress(t *testing.T) {
	ips := []net.IP{net.ParseIP("192.0.2.10"), net.ParseIP("192.0.2.20")}
	started := time.Now()
	conn, err := raceEdgeIPv4Dial(
		context.Background(),
		ips,
		"443",
		func(ctx context.Context, _, address string) (net.Conn, error) {
			if strings.Contains(address, "192.0.2.10") {
				select {
				case <-time.After(200 * time.Millisecond):
					return nil, errors.New("blackholed address")
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			client, peer := net.Pipe()
			go func() {
				<-ctx.Done()
				peer.Close()
			}()
			return client, nil
		},
	)
	if err != nil {
		t.Fatalf("raceEdgeIPv4Dial() error = %v", err)
	}
	defer conn.Close()
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("raceEdgeIPv4Dial() took %v, want healthy address without waiting for blackhole", elapsed)
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
