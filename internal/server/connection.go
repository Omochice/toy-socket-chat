package server

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// Connection represents a client connection (TCP or WebSocket)
type Connection interface {
	// RemoteAddr returns the remote address
	RemoteAddr() net.Addr

	// Write sends binary data to the client
	Write(data []byte) (int, error)

	// Read receives binary data from the client
	Read(buf []byte) (int, error)

	// Close closes the connection
	Close() error

	// SetReadDeadline sets the read deadline
	SetReadDeadline(t time.Time) error
}

// TCPConnection wraps a net.Conn for TCP connections
type TCPConnection struct {
	conn   net.Conn
	reader io.Reader
}

// NewTCPConnection creates a new TCPConnection
func NewTCPConnection(conn net.Conn) *TCPConnection {
	return &TCPConnection{
		conn:   conn,
		reader: conn,
	}
}

// NewTCPConnectionWithReader creates a new TCPConnection with a buffered reader
// This is used when we've already peeked at the connection for protocol detection
func NewTCPConnectionWithReader(conn net.Conn, reader *bufio.Reader) *TCPConnection {
	return &TCPConnection{
		conn:   conn,
		reader: reader,
	}
}

func (tc *TCPConnection) RemoteAddr() net.Addr {
	return tc.conn.RemoteAddr()
}

func (tc *TCPConnection) Write(data []byte) (int, error) {
	return tc.conn.Write(data)
}

func (tc *TCPConnection) Read(buf []byte) (int, error) {
	return tc.reader.Read(buf)
}

func (tc *TCPConnection) Close() error {
	return tc.conn.Close()
}

func (tc *TCPConnection) SetReadDeadline(t time.Time) error {
	return tc.conn.SetReadDeadline(t)
}

// WebSocketConnection wraps a net.Conn for WebSocket connections using gobwas/ws
type WebSocketConnection struct {
	conn          net.Conn
	readBuffer    []byte
	readBufferPos int
	mu            sync.Mutex
}

// NewWebSocketConnection creates a new WebSocketConnection
func NewWebSocketConnection(conn net.Conn) *WebSocketConnection {
	return &WebSocketConnection{conn: conn}
}

func (wc *WebSocketConnection) RemoteAddr() net.Addr {
	return wc.conn.RemoteAddr()
}

func (wc *WebSocketConnection) Write(data []byte) (int, error) {
	// Write binary message using gobwas/ws
	err := wsutil.WriteServerBinary(wc.conn, data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (wc *WebSocketConnection) Read(buf []byte) (int, error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	// Return buffered data if available
	if wc.readBufferPos < len(wc.readBuffer) {
		n := copy(buf, wc.readBuffer[wc.readBufferPos:])
		wc.readBufferPos += n
		if wc.readBufferPos >= len(wc.readBuffer) {
			wc.readBuffer = nil
			wc.readBufferPos = 0
		}
		return n, nil
	}

	// Read next WebSocket message using gobwas/ws
	data, err := wsutil.ReadClientBinary(wc.conn)
	if err != nil {
		return 0, err
	}

	// Copy to output buffer
	n := copy(buf, data)
	if n < len(data) {
		// Buffer remaining data
		wc.readBuffer = data[n:]
		wc.readBufferPos = 0
	}

	return n, nil
}

func (wc *WebSocketConnection) Close() error {
	// Send close frame
	_ = wsutil.WriteServerMessage(wc.conn, ws.OpClose, nil)
	return wc.conn.Close()
}

func (wc *WebSocketConnection) SetReadDeadline(t time.Time) error {
	return wc.conn.SetReadDeadline(t)
}
