package chat_test

import (
	"context"
	"io"
	"sync"

	"github.com/omochice/toy-socket-chat/internal/chat"
)

// mockConn is a mock implementation of chat.Conn for testing.
type mockConn struct {
	readCh     chan []byte
	readErr    error
	writtenMu  sync.Mutex
	written    [][]byte
	writeErr   error
	closed     bool
	remoteAddr string
}

func newMockConn(addr string) *mockConn {
	return &mockConn{
		readCh:     make(chan []byte, 10),
		remoteAddr: addr,
	}
}

func (m *mockConn) Read(ctx context.Context) ([]byte, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data, ok := <-m.readCh:
		if !ok {
			return nil, io.EOF
		}
		return data, nil
	}
}

func (m *mockConn) Write(ctx context.Context, data []byte) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writtenMu.Lock()
	defer m.writtenMu.Unlock()
	copied := make([]byte, len(data))
	copy(copied, data)
	m.written = append(m.written, copied)
	return nil
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) RemoteAddr() string {
	return m.remoteAddr
}

func (m *mockConn) GetWritten() [][]byte {
	m.writtenMu.Lock()
	defer m.writtenMu.Unlock()
	return m.written
}

// Compile-time check that mockConn implements chat.Conn
var _ chat.Conn = (*mockConn)(nil)
