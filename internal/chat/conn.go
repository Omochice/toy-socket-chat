// Package chat provides the core chat domain logic shared by all transports.
package chat

import "context"

// Conn abstracts a bidirectional connection for both TCP and WebSocket.
// This interface isolates transport details from chat logic.
type Conn interface {
	// Read reads a single message frame (protobuf bytes).
	// Returns io.EOF when connection is closed.
	Read(ctx context.Context) ([]byte, error)

	// Write sends a single message frame (protobuf bytes).
	Write(ctx context.Context, data []byte) error

	// Close closes the connection.
	Close() error

	// RemoteAddr returns the remote address for logging.
	RemoteAddr() string
}
