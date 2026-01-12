// Package ws provides WebSocket transport implementation for the chat server.
package ws

import (
	"context"

	"nhooyr.io/websocket"
)

// Conn adapts nhooyr.io/websocket to chat.Conn interface.
type Conn struct {
	conn       *websocket.Conn
	remoteAddr string
}

// NewConn wraps a websocket.Conn with empty remote address.
func NewConn(conn *websocket.Conn) *Conn {
	return &Conn{conn: conn}
}

// NewConnWithAddr wraps a websocket.Conn with the specified remote address.
func NewConnWithAddr(conn *websocket.Conn, addr string) *Conn {
	return &Conn{conn: conn, remoteAddr: addr}
}

// Read implements chat.Conn.
// Reads a binary message from the WebSocket connection.
func (c *Conn) Read(ctx context.Context) ([]byte, error) {
	_, data, err := c.conn.Read(ctx)
	return data, err
}

// Write implements chat.Conn.
// Writes a binary message to the WebSocket connection.
func (c *Conn) Write(ctx context.Context, data []byte) error {
	return c.conn.Write(ctx, websocket.MessageBinary, data)
}

// Close implements chat.Conn.
func (c *Conn) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

// RemoteAddr implements chat.Conn.
func (c *Conn) RemoteAddr() string {
	return c.remoteAddr
}
