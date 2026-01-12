package chat_test

import (
	"context"

	"github.com/omochice/toy-socket-chat/internal/chat"
)

// mockConn is a mock implementation of chat.Conn for testing.
type mockConn struct {
	readData   []byte
	readErr    error
	writeData  []byte
	writeErr   error
	closed     bool
	remoteAddr string
}

func (m *mockConn) Read(ctx context.Context) ([]byte, error) {
	return m.readData, m.readErr
}

func (m *mockConn) Write(ctx context.Context, data []byte) error {
	m.writeData = data
	return m.writeErr
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) RemoteAddr() string {
	return m.remoteAddr
}

// Compile-time check that mockConn implements chat.Conn
var _ chat.Conn = (*mockConn)(nil)
