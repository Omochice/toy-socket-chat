package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

// UnifiedClient represents a generic client (TCP or WebSocket)
type UnifiedClient struct {
	id       string
	username string
	outgoing chan []byte
	clientType string // "tcp" or "websocket"
}

// UnifiedServer represents a server that handles both TCP and WebSocket connections
type UnifiedServer struct {
	tcpAddress string
	wsAddress  string
	tcpListener net.Listener
	wsListener  net.Listener
	wsServer    *http.Server
	clients     map[*UnifiedClient]bool
	mu          sync.RWMutex
	quit        chan struct{}
	wg          sync.WaitGroup
}

// NewUnifiedServer creates a new UnifiedServer instance
func NewUnifiedServer(tcpAddress, wsAddress string) *UnifiedServer {
	return &UnifiedServer{
		tcpAddress: tcpAddress,
		wsAddress:  wsAddress,
		clients:    make(map[*UnifiedClient]bool),
		quit:       make(chan struct{}),
	}
}

// Start starts both TCP and WebSocket servers
func (s *UnifiedServer) Start() error {
	// Start TCP server
	tcpListener, err := net.Listen("tcp", s.tcpAddress)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}
	s.tcpListener = tcpListener
	log.Printf("TCP server started on %s", tcpListener.Addr().String())

	// Start WebSocket server
	wsListener, err := net.Listen("tcp", s.wsAddress)
	if err != nil {
		tcpListener.Close()
		return fmt.Errorf("failed to start WebSocket server: %w", err)
	}
	s.wsListener = wsListener
	log.Printf("WebSocket server started on %s", wsListener.Addr().String())

	// Start TCP accept loop
	s.wg.Add(1)
	go s.acceptTCPConnections()

	// Start WebSocket server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	s.wsServer = &http.Server{
		Handler: mux,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.wsServer.Serve(wsListener); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-s.quit
	return fmt.Errorf("Server stopped")
}

// Stop stops the unified server
func (s *UnifiedServer) Stop() {
	close(s.quit)

	if s.tcpListener != nil {
		s.tcpListener.Close()
	}
	if s.wsServer != nil {
		s.wsServer.Close()
	}

	// Don't close client channels here - they will be closed by handleClient functions
	s.wg.Wait()
}

// TCPAddr returns the TCP server's listening address
func (s *UnifiedServer) TCPAddr() string {
	if s.tcpListener != nil {
		return s.tcpListener.Addr().String()
	}
	return ""
}

// WSAddr returns the WebSocket server's listening address
func (s *UnifiedServer) WSAddr() string {
	if s.wsListener != nil {
		return s.wsListener.Addr().String()
	}
	return ""
}

// ClientCount returns the number of connected clients
func (s *UnifiedServer) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// acceptTCPConnections accepts TCP connections
func (s *UnifiedServer) acceptTCPConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.quit:
			return
		default:
			conn, err := s.tcpListener.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return
				default:
					log.Printf("Failed to accept TCP connection: %v", err)
					continue
				}
			}

			client := &UnifiedClient{
				id:         fmt.Sprintf("tcp-%p", conn),
				outgoing:   make(chan []byte, 10),
				clientType: "tcp",
			}

			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()

			s.wg.Add(1)
			go s.handleTCPClient(client, conn)
		}
	}
}

// handleTCPClient handles a TCP client connection
func (s *UnifiedServer) handleTCPClient(client *UnifiedClient, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()
	defer func() {
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
	}()

	// Start writer goroutine
	writerDone := make(chan struct{})
	go func() {
		for data := range client.outgoing {
			if _, err := conn.Write(data); err != nil {
				log.Printf("Failed to send message to TCP client: %v", err)
				return
			}
		}
		close(writerDone)
	}()

	defer func() {
		close(client.outgoing)
		<-writerDone
	}()

	// Read messages from client
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from TCP client: %v", err)
			}
			return
		}

		if n > 0 {
			var msg protocol.Message
			if err := msg.Decode(buf[:n]); err != nil {
				log.Printf("Failed to decode message: %v", err)
				continue
			}

			switch msg.Type {
			case protocol.MessageTypeJoin:
				client.username = msg.Sender
				log.Printf("TCP user %s joined", msg.Sender)
				s.broadcast(buf[:n], client)
			case protocol.MessageTypeLeave:
				log.Printf("TCP user %s left", msg.Sender)
				s.broadcast(buf[:n], client)
				return
			case protocol.MessageTypeText:
				log.Printf("Message from TCP user %s: %s", msg.Sender, msg.Content)
				s.broadcast(buf[:n], client)
			}
		}
	}
}

// handleWebSocket handles WebSocket upgrade and client connections
func (s *UnifiedServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &UnifiedClient{
		id:         fmt.Sprintf("ws-%p", conn),
		outgoing:   make(chan []byte, 10),
		clientType: "websocket",
	}

	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.handleWebSocketClient(client, conn)
}

// handleWebSocketClient handles a WebSocket client connection
func (s *UnifiedServer) handleWebSocketClient(client *UnifiedClient, conn *websocket.Conn) {
	defer s.wg.Done()
	defer conn.Close()
	defer func() {
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
	}()

	// Start writer goroutine
	writerDone := make(chan struct{})
	go func() {
		for data := range client.outgoing {
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				log.Printf("Failed to send message to WebSocket client: %v", err)
				return
			}
		}
		close(writerDone)
	}()

	defer func() {
		close(client.outgoing)
		<-writerDone
	}()

	// Read messages from client
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		if messageType == websocket.BinaryMessage {
			var msg protocol.Message
			if err := msg.Decode(data); err != nil {
				log.Printf("Failed to decode message: %v", err)
				continue
			}

			switch msg.Type {
			case protocol.MessageTypeJoin:
				client.username = msg.Sender
				log.Printf("WebSocket user %s joined", msg.Sender)
				s.broadcast(data, client)
			case protocol.MessageTypeLeave:
				log.Printf("WebSocket user %s left", msg.Sender)
				s.broadcast(data, client)
				return
			case protocol.MessageTypeText:
				log.Printf("Message from WebSocket user %s: %s", msg.Sender, msg.Content)
				s.broadcast(data, client)
			}
		}
	}
}

// broadcast sends a message to all clients except the sender
func (s *UnifiedServer) broadcast(data []byte, sender *UnifiedClient) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if client != sender {
			select {
			case client.outgoing <- data:
			default:
				log.Printf("Client channel full, skipping")
			}
		}
	}
}
