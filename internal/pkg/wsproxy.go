package pkg

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var wsActiveConns = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "ws_active_connections",
	Help: "Number of active WebSocket proxy connections",
})

func init() {
	prometheus.MustRegister(wsActiveConns)
}

const (
	wsOpContinuation = 0x0
	wsOpText         = 0x1
	wsOpBinary       = 0x2
	wsOpClose        = 0x8
	wsOpPing         = 0x9
	wsOpPong         = 0xA

	wsMaxControlPayload = 125
	wsPingInterval      = 30 * time.Second
	wsPongTimeout       = 90 * time.Second
)

// WSProxyHandler returns a gin.HandlerFunc that upgrades the client connection
// to WebSocket, dials the target server, and relays frames bidirectionally.
func WSProxyHandler(targetURL string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic(fmt.Sprintf("wsproxy: invalid target URL %q: %v", targetURL, err))
	}

	return func(c *gin.Context) {
		handleWSProxy(c, target)
	}
}

func handleWSProxy(c *gin.Context, target *url.URL) {
	if !isWebSocketUpgrade(c.Request) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a websocket upgrade request"})
		return
	}

	clientConn, clientBuf, err := hijackClient(c.Writer)
	if err != nil {
		log.Printf("wsproxy: client hijack failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upgrade failed"})
		return
	}

	backendPath := strings.TrimPrefix(c.Request.URL.Path, "/vtuber-ws")
	if backendPath == "" {
		backendPath = "/"
	}
	if c.Request.URL.RawPath != "" {
		trimmed := strings.TrimPrefix(c.Request.URL.RawPath, "/vtuber-ws")
		if trimmed != "" {
			backendPath = trimmed
		}
	}
	if c.Request.URL.RawQuery != "" {
		backendPath += "?" + c.Request.URL.RawQuery
	}

	backendAddr := target.Host
	if !strings.Contains(backendAddr, ":") {
		backendAddr += ":80"
	}
	backendConn, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
	if err != nil {
		log.Printf("wsproxy: backend dial failed: %v", err)
		clientConn.Close()
		return
	}

	wsKey := c.Request.Header.Get("Sec-WebSocket-Key")
	if err := performBackendUpgrade(clientConn, clientBuf, backendConn, backendPath, target, wsKey, c.Request); err != nil {
		log.Printf("wsproxy: backend handshake failed: %v", err)
		clientConn.Close()
		backendConn.Close()
		return
	}

	wsActiveConns.Inc()
	defer wsActiveConns.Dec()

	closeOnce := sync.Once{}
	closeCh := make(chan struct{})
	closeConn := func() { closeOnce.Do(func() { close(closeCh) }) }

	go keepalive(clientConn, backendConn, closeConn)
	forwardWSConnection(clientConn, backendConn, closeConn)
	<-closeCh
	clientConn.Close()
	backendConn.Close()
}

func isWebSocketUpgrade(r *http.Request) bool {
	for _, v := range r.Header.Values("Connection") {
		if strings.EqualFold(strings.TrimSpace(v), "Upgrade") {
			return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
		}
	}
	return false
}

// sensitiveForwardHeaders 是不应透传给下游后端的敏感请求头(小写)。
// 这些头携带本服务与客户端之间的认证凭据,透传会给下游造成凭据泄露面。
var sensitiveForwardHeaders = map[string]bool{
	"authorization": true,
	"cookie":        true,
	"x-api-key":     true,
	"x-csrf-token":  true,
}

func isSensitiveHeader(lowerName string) bool {
	return sensitiveForwardHeaders[lowerName]
}

func hijackClient(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("ResponseWriter does not implement http.Hijacker")
	}
	conn, buf, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, err
	}
	return conn, buf, nil
}

func performBackendUpgrade(clientConn net.Conn, clientBuf *bufio.ReadWriter, backendConn net.Conn, path string, target *url.URL, wsKey string, clientReq *http.Request) error {
	var req strings.Builder
	fmt.Fprintf(&req, "GET %s HTTP/1.1\r\n", path)
	fmt.Fprintf(&req, "Host: %s\r\n", target.Host)
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	if wsKey != "" {
		fmt.Fprintf(&req, "Sec-WebSocket-Key: %s\r\n", wsKey)
	}
	req.WriteString("Sec-WebSocket-Version: 13\r\n")
	for k, vv := range clientReq.Header {
		lower := strings.ToLower(k)
		if lower == "host" || lower == "connection" || lower == "upgrade" ||
			lower == "sec-websocket-key" || lower == "sec-websocket-version" {
			continue
		}
		// 剔除敏感头:这些凭据属于本服务与客户端之间的认证,不应透传给下游 VTuber 后端,
		// 避免下游日志记录或二次转发导致凭据泄露。
		if isSensitiveHeader(lower) {
			continue
		}
		for _, v := range vv {
			fmt.Fprintf(&req, "%s: %s\r\n", k, v)
		}
	}
	req.WriteString("\r\n")

	if _, err := backendConn.Write([]byte(req.String())); err != nil {
		return fmt.Errorf("write upgrade request: %w", err)
	}

	backendReader := bufio.NewReaderSize(backendConn, 4096)
	respLine, err := readLine(backendReader)
	if err != nil {
		return fmt.Errorf("read status line: %w", err)
	}
	statusCode, err := parseHTTPStatusCode(respLine)
	if err != nil {
		return fmt.Errorf("parse status: %w (line: %q)", err, respLine)
	}
	if statusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("backend returned HTTP %d (expected 101)", statusCode)
	}

	hasUpgrade, hasConnectionUpgrade := false, false
	headers := make([]string, 0, 8)
	for {
		line, err := readLine(backendReader)
		if err != nil {
			return fmt.Errorf("read header: %w", err)
		}
		if line == "" {
			break
		}
		headers = append(headers, line)
		key, val := parseHeaderLine(line)
		lower := strings.ToLower(key)
		if lower == "upgrade" && strings.EqualFold(val, "websocket") {
			hasUpgrade = true
		}
		if lower == "connection" && strings.EqualFold(val, "Upgrade") {
			hasConnectionUpgrade = true
		}
	}
	if !hasUpgrade {
		return fmt.Errorf("backend response missing 'Upgrade: websocket' header")
	}
	if !hasConnectionUpgrade {
		return fmt.Errorf("backend response missing 'Connection: Upgrade' header")
	}

	if _, err := clientBuf.WriteString(respLine + "\r\n"); err != nil {
		return fmt.Errorf("write client status: %w", err)
	}
	for _, header := range headers {
		if _, err := clientBuf.WriteString(header + "\r\n"); err != nil {
			return fmt.Errorf("write client header: %w", err)
		}
	}
	if _, err := clientBuf.WriteString("\r\n"); err != nil {
		return fmt.Errorf("write client header terminator: %w", err)
	}
	if err := clientBuf.Flush(); err != nil {
		return fmt.Errorf("flush client handshake: %w", err)
	}

	if backendReader.Buffered() > 0 {
		if _, err := io.CopyN(clientConn, backendReader, int64(backendReader.Buffered())); err != nil {
			return fmt.Errorf("forward buffered backend data: %w", err)
		}
	}

	// Discard any buffered data from the hijacked client reader to avoid leaks.
	if clientBuf.Reader.Buffered() > 0 {
		clientBuf.Reader.Discard(clientBuf.Reader.Buffered())
	}
	return nil
}

// readLine reads a single line from the reader, stripping the trailing \r\n.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func parseHTTPStatusCode(line string) (int, error) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return 0, fmt.Errorf("malformed HTTP status line: %q", line)
	}
	var code int
	if _, err := fmt.Sscanf(parts[1], "%d", &code); err != nil {
		return 0, err
	}
	return code, nil
}

func parseHeaderLine(line string) (string, string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return line, ""
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:])
}

func forwardWSConnection(client, backend net.Conn, closeConn func()) {
	clientDone := make(chan struct{})
	backendDone := make(chan struct{})

	go func() {
		relay(client, backend, "client->backend")
		close(clientDone)
	}()
	go func() {
		relay(backend, client, "backend->client")
		close(backendDone)
	}()

	select {
	case <-clientDone:
	case <-backendDone:
	}
	closeConn()
}

// relay reads complete WebSocket frames from src and writes them to dst.
// Data frames (opcodes 0x0-0x2) are forwarded as opaque raw bytes.
// Control frames (close, ping, pong) are read fully, processed, and forwarded.
func relay(src, dst net.Conn, direction string) {
	for {
		hdr := make([]byte, 2)
		if _, err := io.ReadFull(src, hdr); err != nil {
			if err != io.EOF {
				log.Printf("wsproxy [%s]: read frame header: %v", direction, err)
			}
			return
		}

		opcode := hdr[0] & 0x0F
		isMasked := hdr[1]&0x80 != 0
		payloadLen := uint64(hdr[1] & 0x7F)

		switch payloadLen {
		case 126:
			ext := make([]byte, 2)
			if _, err := io.ReadFull(src, ext); err != nil {
				log.Printf("wsproxy [%s]: read 2-byte length: %v", direction, err)
				return
			}
			payloadLen = uint64(ext[0])<<8 | uint64(ext[1])
		case 127:
			ext := make([]byte, 8)
			if _, err := io.ReadFull(src, ext); err != nil {
				log.Printf("wsproxy [%s]: read 8-byte length: %v", direction, err)
				return
			}
			payloadLen = uint64(ext[0])<<56 | uint64(ext[1])<<48 | uint64(ext[2])<<40 | uint64(ext[3])<<32 |
				uint64(ext[4])<<24 | uint64(ext[5])<<16 | uint64(ext[6])<<8 | uint64(ext[7])
		}

		maskLen := 0
		if isMasked {
			maskLen = 4
		}

		if opcode == wsOpText || opcode == wsOpBinary || opcode == wsOpContinuation {
			if err := forwardOpaque(src, dst, hdr, payloadLen, maskLen); err != nil {
				log.Printf("wsproxy [%s]: forward data frame: %v", direction, err)
				return
			}
			continue
		}

		if payloadLen > wsMaxControlPayload {
			log.Printf("wsproxy [%s]: control frame payload too large (%d bytes), ignoring", direction, payloadLen)
			continue
		}

		var payload []byte
		if isMasked {
			maskKey, err := readMaskKey(src)
			if err != nil {
				log.Printf("wsproxy [%s]: read mask key: %v", direction, err)
				return
			}
			payload = make([]byte, payloadLen)
			if payloadLen > 0 {
				if _, err := io.ReadFull(src, payload); err != nil {
					log.Printf("wsproxy [%s]: read control payload: %v", direction, err)
					return
				}
				applyMask(payload, maskKey)
			}
		} else {
			payload = make([]byte, payloadLen)
			if payloadLen > 0 {
				if _, err := io.ReadFull(src, payload); err != nil {
					log.Printf("wsproxy [%s]: read control payload: %v", direction, err)
					return
				}
			}
		}

		switch opcode {
		case wsOpPing:
			pong := buildFrame(wsOpPong, false, payload)
			if _, err := src.Write(pong); err != nil {
				log.Printf("wsproxy [%s]: write pong: %v", direction, err)
				return
			}
			forwarded := buildFrame(wsOpPing, isMasked, payload)
			if _, err := dst.Write(forwarded); err != nil {
				log.Printf("wsproxy [%s]: forward ping: %v", direction, err)
				return
			}
		case wsOpPong:
			forwarded := buildFrame(wsOpPong, isMasked, payload)
			if _, err := dst.Write(forwarded); err != nil {
				log.Printf("wsproxy [%s]: forward pong: %v", direction, err)
				return
			}
		case wsOpClose:
			forwarded := buildFrame(wsOpClose, isMasked, payload)
			dst.Write(forwarded)
			return
		default:
			log.Printf("wsproxy [%s]: unknown control opcode 0x%X, ignoring", direction, opcode)
		}
	}
}

func forwardOpaque(src, dst net.Conn, hdr []byte, payloadLen uint64, maskLen int) error {
	total := int64(payloadLen) + int64(maskLen)
	extLen := 0
	switch {
	case payloadLen <= 125:
		extLen = 0
	case payloadLen <= 65535:
		extLen = 2
	default:
		extLen = 8
	}

	frame := make([]byte, 2+extLen+int(total))
	copy(frame, hdr)

	offset := 2
	switch extLen {
	case 2:
		frame[offset] = byte(payloadLen >> 8)
		frame[offset+1] = byte(payloadLen)
		offset += 2
	case 8:
		for i := 7; i >= 0; i-- {
			frame[offset+(7-i)] = byte(payloadLen >> (8 * uint(i)))
		}
		offset += 8
	}

	if _, err := io.ReadFull(src, frame[offset:]); err != nil {
		return err
	}
	_, err := dst.Write(frame)
	return err
}

func readMaskKey(r io.Reader) ([4]byte, error) {
	var key [4]byte
	_, err := io.ReadFull(r, key[:])
	return key, err
}

func applyMask(data []byte, key [4]byte) {
	for i := range data {
		data[i] ^= key[i%4]
	}
}

func buildFrame(opcode byte, masked bool, payload []byte) []byte {
	frame := []byte{0x80 | opcode, 0}

	if masked {
		frame[1] |= 0x80
	}

	length := len(payload)
	switch {
	case length <= 125:
		frame[1] |= byte(length)
	case length <= 65535:
		frame[1] |= 126
		frame = append(frame, byte(length>>8), byte(length))
	default:
		frame[1] |= 127
		var lenBytes [8]byte
		for i := 7; i >= 0; i-- {
			lenBytes[i] = byte(length & 0xFF)
			length >>= 8
		}
		frame = append(frame, lenBytes[:]...)
	}

	if masked {
		var key [4]byte
		maskedPayload := make([]byte, len(payload))
		copy(maskedPayload, payload)
		applyMask(maskedPayload, key)
		frame = append(frame, key[:]...)
		frame = append(frame, maskedPayload...)
	} else {
		frame = append(frame, payload...)
	}

	return frame
}

func keepalive(client, backend net.Conn, closeConn func()) {
	deadline := time.Now().Add(wsPongTimeout)
	client.SetDeadline(deadline)
	backend.SetDeadline(deadline)

	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	for range ticker.C {
		ping := buildFrame(wsOpPing, false, nil)
		if _, err := backend.Write(ping); err != nil {
			log.Printf("wsproxy: backend ping failed: %v", err)
			closeConn()
			return
		}

		maskedPing := buildFrame(wsOpPing, true, nil)
		if _, err := client.Write(maskedPing); err != nil {
			log.Printf("wsproxy: client ping failed: %v", err)
			closeConn()
			return
		}

		deadline = time.Now().Add(wsPongTimeout)
		client.SetDeadline(deadline)
		backend.SetDeadline(deadline)
	}
}
