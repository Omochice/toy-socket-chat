package client

import (
	"errors"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/quic-go/webtransport-go"
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
	// Write binary message using gobwas/ws (client side)
	err := wsutil.WriteClientBinary(wc.conn, data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (wc *WebSocketClientConnection) Read(buf []byte) (int, error) {
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

	// Read next WebSocket message using gobwas/ws (server messages)
	data, err := wsutil.ReadServerBinary(wc.conn)
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

func (wc *WebSocketClientConnection) Close() error {
	// Send close frame
	_ = wsutil.WriteClientMessage(wc.conn, ws.OpClose, nil)
	return wc.conn.Close()
}

func (wc *WebSocketClientConnection) RemoteAddr() net.Addr {
	return wc.conn.RemoteAddr()
}

// WebTransportClientConnection wraps a WebTransport session and its single
// bidirectional stream. The chat protocol is treated as an unframed byte stream
// over that stream, matching the framing assumption used for TCP.
type WebTransportClientConnection struct {
	session *webtransport.Session
	stream  *webtransport.Stream
}

// NewWebTransportClientConnection creates a new WebTransport connection wrapper
// over an established session and its bidirectional stream.
func NewWebTransportClientConnection(
	session *webtransport.Session,
	stream *webtransport.Stream,
) *WebTransportClientConnection {
	return &WebTransportClientConnection{session: session, stream: stream}
}

func (wtc *WebTransportClientConnection) Write(data []byte) (int, error) {
	return wtc.stream.Write(data)
}

func (wtc *WebTransportClientConnection) Read(buf []byte) (int, error) {
	return wtc.stream.Read(buf)
}

func (wtc *WebTransportClientConnection) Close() error {
	// Stream.Close only shuts the send direction, so the session must also be
	// torn down to release the underlying QUIC connection and receive side.
	streamErr := wtc.stream.Close()
	sessErr := wtc.session.CloseWithError(0, "")
	return errors.Join(streamErr, sessErr)
}

func (wtc *WebTransportClientConnection) RemoteAddr() net.Addr {
	return wtc.session.RemoteAddr()
}
