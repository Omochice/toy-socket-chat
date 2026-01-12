package chat

import (
	"sync"
)

// Client represents a connected client with transport-agnostic connection.
type Client struct {
	Conn     Conn
	Username string
	Outgoing chan []byte
}

// Hub manages all connected clients and handles broadcast.
// Both TCP and WebSocket servers share a single Hub instance.
type Hub struct {
	clients map[*Client]bool
	mu      sync.RWMutex
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
}

// ClientCount returns number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
