package client

import (
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// ClientConnection represents a connection to the server
type ClientConnection interface {
	// Write sends data to the server
	Write(data []byte) (int, error)

	// Read receives data from the server
	Read(buf []byte) (int, error)

	// Close closes the connection
	Close() error

	// RemoteAddr returns the server address
	RemoteAddr() net.Addr
}

// TCPClientConnection wraps net.Conn for TCP connections
type TCPClientConnection struct {
	conn net.Conn
}

// NewTCPClientConnection creates a new TCP connection wrapper
func NewTCPClientConnection(conn net.Conn) *TCPClientConnection {
	return &TCPClientConnection{conn: conn}
}

func (tc *TCPClientConnection) Write(data []byte) (int, error) {
	return tc.conn.Write(data)
}

func (tc *TCPClientConnection) Read(buf []byte) (int, error) {
	return tc.conn.Read(buf)
}

func (tc *TCPClientConnection) Close() error {
	return tc.conn.Close()
}

func (tc *TCPClientConnection) RemoteAddr() net.Addr {
	return tc.conn.RemoteAddr()
}

// WebSocketClientConnection wraps net.Conn for WebSocket connections using gobwas/ws
type WebSocketClientConnection struct {
	conn          net.Conn
	readBuffer    []byte
	readBufferPos int
	mu            sync.Mutex
}

// NewWebSocketClientConnection creates a new WebSocket connection wrapper
func NewWebSocketClientConnection(conn net.Conn) *WebSocketClientConnection {
	return &WebSocketClientConnection{conn: conn}
}

func (wc *WebSocketClientConnection) Write(data []byte) (int, error) {
	err := wsutil.WriteClientBinary(wc.conn, data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (wc *WebSocketClientConnection) Read(buf []byte) (int, error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if wc.readBufferPos < len(wc.readBuffer) {
		n := copy(buf, wc.readBuffer[wc.readBufferPos:])
		wc.readBufferPos += n
		if wc.readBufferPos >= len(wc.readBuffer) {
			wc.readBuffer = nil
			wc.readBufferPos = 0
		}
		return n, nil
	}

	data, err := wsutil.ReadServerBinary(wc.conn)
	if err != nil {
		return 0, err
	}

	n := copy(buf, data)
	if n < len(data) {
		wc.readBuffer = data[n:]
		wc.readBufferPos = 0
	}

	return n, nil
}

func (wc *WebSocketClientConnection) Close() error {
	_ = wsutil.WriteClientMessage(wc.conn, ws.OpClose, nil)
	return wc.conn.Close()
}

func (wc *WebSocketClientConnection) RemoteAddr() net.Addr {
	return wc.conn.RemoteAddr()
}
