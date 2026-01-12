package server

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// upgradeWebSocket performs WebSocket handshake and returns WebSocket connection
func (s *Server) upgradeWebSocket(rawConn net.Conn, reader *bufio.Reader) (Connection, error) {
	req, err := http.ReadRequest(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP request: %w", err)
	}

	if !isWebSocketUpgrade(req) {
		return nil, fmt.Errorf("not a WebSocket upgrade request")
	}

	key := req.Header.Get("Sec-WebSocket-Key")
	acceptKey := computeAcceptKey(key)

	response := fmt.Sprintf(
		"HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n"+
			"\r\n",
		acceptKey,
	)

	if _, err := rawConn.Write([]byte(response)); err != nil {
		return nil, fmt.Errorf("failed to write upgrade response: %w", err)
	}

	// After handshake completes, the connection is ready for WebSocket framing
	// gobwas/ws will handle the framing in WebSocketConnection
	return NewWebSocketConnection(rawConn), nil
}

func isWebSocketUpgrade(req *http.Request) bool {
	return req.Method == "GET" &&
		strings.ToLower(req.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

func computeAcceptKey(key string) string {
	const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte(websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
