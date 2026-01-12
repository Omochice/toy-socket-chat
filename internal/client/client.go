// Package client defines the common interface for chat clients.
package client

import "github.com/omochice/toy-socket-chat/pkg/protocol"

// Client defines the interface for chat clients.
// Both TCP and WebSocket implementations satisfy this interface.
type Client interface {
	Connect() error
	Disconnect()
	IsConnected() bool
	SendMessage(content string) error
	Join() error
	Leave() error
	Messages() <-chan protocol.Message
}
