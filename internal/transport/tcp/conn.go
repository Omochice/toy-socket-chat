// Package tcp provides TCP transport implementation for the chat server.
package tcp

import (
	"context"
	"net"
)

// Conn adapts net.Conn to chat.Conn interface.
type Conn struct {
	conn net.Conn
}

// NewConn wraps a net.Conn.
func NewConn(conn net.Conn) *Conn {
	return &Conn{conn: conn}
}

// Read implements chat.Conn.
// Reads available bytes from the TCP connection.
func (c *Conn) Read(ctx context.Context) ([]byte, error) {
	buf := make([]byte, 4096)
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// Write implements chat.Conn.
func (c *Conn) Write(ctx context.Context, data []byte) error {
	_, err := c.conn.Write(data)
	return err
}

// Close implements chat.Conn.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// RemoteAddr implements chat.Conn.
func (c *Conn) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}
