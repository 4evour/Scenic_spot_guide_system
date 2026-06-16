package pkg

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestWSProxyForwardsBackendHandshakeHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backend := newFakeWebSocketBackend(t)
	defer backend.Close()

	router := gin.New()
	router.GET("/vtuber-ws/*path", WSProxyHandler("http://"+backend.Addr().String()))
	server := httptest.NewServer(router)
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	req := strings.Join([]string{
		"GET /vtuber-ws/client-ws HTTP/1.1",
		"Host: " + addr,
		"Upgrade: websocket",
		"Connection: Upgrade",
		"Sec-WebSocket-Key: test-key",
		"Sec-WebSocket-Version: 13",
		"",
		"",
	}, "\r\n")
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read proxy handshake response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", resp.StatusCode)
	}
	if got := resp.Header.Get("Sec-WebSocket-Accept"); got != "backend-accept" {
		t.Fatalf("Sec-WebSocket-Accept = %q, want backend-accept", got)
	}
}

func TestForwardOpaquePreservesExtendedPayloadLengthFrame(t *testing.T) {
	payload := bytes.Repeat([]byte("a"), 130)
	srcReader, srcWriter := net.Pipe()
	dstReader, dstWriter := net.Pipe()
	defer srcReader.Close()
	defer srcWriter.Close()
	defer dstReader.Close()
	defer dstWriter.Close()

	hdr := []byte{0x81, 126}

	errCh := make(chan error, 1)
	go func() {
		errCh <- forwardOpaque(srcReader, dstWriter, hdr, uint64(len(payload)), 0)
		dstWriter.Close()
	}()
	go func() {
		_, _ = srcWriter.Write(payload)
		srcWriter.Close()
	}()

	wantLen := 2 + 2 + len(payload)
	got := make([]byte, wantLen)
	if _, err := io.ReadFull(dstReader, got); err != nil {
		t.Fatalf("read forwarded frame: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("forward opaque frame: %v", err)
	}

	if got[0] != 0x81 || got[1] != 126 || got[2] != 0 || got[3] != byte(len(payload)) {
		t.Fatalf("forwarded extended length header = %v, want [129 126 0 130]", got[:4])
	}
	if !bytes.Equal(got[4:], payload) {
		t.Fatalf("forwarded payload was not preserved")
	}
}

func newFakeWebSocketBackend(t *testing.T) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen backend: %v", err)
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if line == "\r\n" {
				break
			}
		}

		_, _ = fmt.Fprint(conn, strings.Join([]string{
			"HTTP/1.1 101 Switching Protocols",
			"Upgrade: websocket",
			"Connection: Upgrade",
			"Sec-WebSocket-Accept: backend-accept",
			"",
			"",
		}, "\r\n"))

		_, _ = reader.Peek(1)
	}()

	return listener
}
