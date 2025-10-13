package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	conn     *websocket.Conn
	username string
	outgoing chan []byte
}

// WebSocketServer represents a WebSocket chat server
type WebSocketServer struct {
	address  string
	listener net.Listener
	server   *http.Server
	clients  map[*WebSocketClient]bool
	mu       sync.RWMutex
	quit     chan struct{}
	wg       sync.WaitGroup
}

// NewWebSocketServer creates a new WebSocketServer instance
func NewWebSocketServer(address string) *WebSocketServer {
	return &WebSocketServer{
		address: address,
		clients: make(map[*WebSocketClient]bool),
		quit:    make(chan struct{}),
	}
}

// Start starts the WebSocket server
func (s *WebSocketServer) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)

	s.server = &http.Server{
		Handler: mux,
	}

	log.Printf("WebSocket server started on %s", listener.Addr().String())

	errChan := make(chan error, 1)
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for either error or quit signal
	select {
	case err := <-errChan:
		return fmt.Errorf("failed to start server: %w", err)
	case <-s.quit:
		return fmt.Errorf("Server stopped")
	}
}

// Stop stops the WebSocket server
func (s *WebSocketServer) Stop() {
	close(s.quit)

	if s.server != nil {
		s.server.Close()
	}

	s.mu.Lock()
	for client := range s.clients {
		client.conn.Close()
	}
	s.mu.Unlock()

	s.wg.Wait()
}

// Addr returns the server's listening address
func (s *WebSocketServer) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// ClientCount returns the number of connected clients
func (s *WebSocketServer) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// handleWebSocket handles WebSocket upgrade and client connections
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &WebSocketClient{
		conn:     conn,
		outgoing: make(chan []byte, 10),
	}

	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.handleClient(client)
}

// handleClient handles a single WebSocket client connection
func (s *WebSocketServer) handleClient(client *WebSocketClient) {
	defer s.wg.Done()
	defer func() {
		close(client.outgoing)
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
		client.conn.Close()
	}()

	// Start goroutine to send messages to client
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for data := range client.outgoing {
			if err := client.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				log.Printf("Failed to send message to client: %v", err)
				return
			}
		}
	}()

	// Read messages from client
	for {
		messageType, data, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		if messageType == websocket.BinaryMessage {
			// Decode message
			var msg protocol.Message
			if err := msg.Decode(data); err != nil {
				log.Printf("Failed to decode message: %v", err)
				continue
			}

			// Handle different message types
			switch msg.Type {
			case protocol.MessageTypeJoin:
				client.username = msg.Sender
				log.Printf("User %s joined", msg.Sender)
				s.broadcast(data, client)
			case protocol.MessageTypeLeave:
				log.Printf("User %s left", msg.Sender)
				s.broadcast(data, client)
				return
			case protocol.MessageTypeText:
				log.Printf("Message from %s: %s", msg.Sender, msg.Content)
				s.broadcast(data, client)
			}
		}
	}
}

// broadcast sends a message to all clients except the sender
func (s *WebSocketServer) broadcast(data []byte, sender *WebSocketClient) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if client != sender {
			select {
			case client.outgoing <- data:
			default:
				// Channel is full, skip this client
				log.Printf("Client channel full, skipping")
			}
		}
	}
}
