package chat

import (
	"context"
	"io"
	"log"
	"sync"

	"github.com/omochice/toy-socket-chat/pkg/protocol"
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

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
}

// Broadcast sends data to all clients except the sender.
func (h *Hub) Broadcast(data []byte, sender *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client != sender {
			select {
			case client.Outgoing <- data:
			default:
			}
		}
	}
}

// ClientCount returns number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// HandleClient manages a single client's message loop.
// It reads messages from the client's connection, processes them,
// and broadcasts to other clients. Returns when connection closes.
func (h *Hub) HandleClient(client *Client) {
	defer func() {
		h.Unregister(client)
		client.Conn.Close()
	}()

	ctx := context.Background()
	for {
		data, err := client.Conn.Read(ctx)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from client %s: %v", client.Conn.RemoteAddr(), err)
			}
			return
		}

		var msg protocol.Message
		if err := msg.Decode(data); err != nil {
			log.Printf("Failed to decode message from %s: %v", client.Conn.RemoteAddr(), err)
			continue
		}

		switch msg.Type {
		case protocol.MessageTypeJoin:
			client.Username = msg.Sender
			log.Printf("User %s joined from %s", msg.Sender, client.Conn.RemoteAddr())
			h.Broadcast(data, client)
		case protocol.MessageTypeLeave:
			log.Printf("User %s left", msg.Sender)
			h.Broadcast(data, client)
			return
		case protocol.MessageTypeText:
			log.Printf("Message from %s: %s", msg.Sender, msg.Content)
			h.Broadcast(data, client)
		}
	}
}
