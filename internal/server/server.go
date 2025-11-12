package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

// Client represents a connected client
type Client struct {
	conn     Connection
	username string
	outgoing chan []byte
}

// Server represents a TCP chat server
type Server struct {
	address  string
	listener net.Listener
	clients  map[*Client]bool
	mu       sync.RWMutex
	quit     chan struct{}
	wg       sync.WaitGroup
}

// New creates a new Server instance
func New(address string) *Server {
	return &Server{
		address: address,
		clients: make(map[*Client]bool),
		quit:    make(chan struct{}),
	}
}

// Start starts the TCP server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.listener = listener

	log.Printf("Server started on %s", listener.Addr().String())

	for {
		select {
		case <-s.quit:
			return fmt.Errorf("server stopped")
		default:
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return fmt.Errorf("server stopped")
				default:
					log.Printf("Failed to accept connection: %v", err)
					continue
				}
			}

			// Handle connection in goroutine with protocol detection
			go s.handleConnection(conn)
		}
	}
}

// Stop stops the server
func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Printf("Error closing listener: %v", err)
		}
	}

	s.mu.Lock()
	for client := range s.clients {
		if err := client.conn.Close(); err != nil {
			log.Printf("Error closing client connection: %v", err)
		}
	}
	s.mu.Unlock()

	s.wg.Wait()
}

// Addr returns the server's listening address
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// ClientCount returns the number of connected clients
func (s *Server) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// handleConnection detects protocol and creates appropriate Connection
func (s *Server) handleConnection(rawConn net.Conn) {
	// Detect protocol
	protocol, reader, err := detectProtocol(rawConn)
	if err != nil {
		log.Printf("Protocol detection failed: %v", err)
		if closeErr := rawConn.Close(); closeErr != nil {
			log.Printf("Error closing connection: %v", closeErr)
		}
		return
	}

	var conn Connection

	switch protocol {
	case protocolHTTP:
		// WebSocket upgrade
		conn, err = s.upgradeWebSocket(rawConn, reader)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			if closeErr := rawConn.Close(); closeErr != nil {
				log.Printf("Error closing connection: %v", closeErr)
			}
			return
		}
		log.Printf("WebSocket connection from %s", conn.RemoteAddr())

	case protocolTCP:
		// Wrap as TCP connection with buffered reader
		// Since we peeked at the data, we need to use the buffered reader
		conn = NewTCPConnectionWithReader(rawConn, reader)
		log.Printf("TCP connection from %s", conn.RemoteAddr())
	}

	// Create client
	client := &Client{
		conn:     conn,
		outgoing: make(chan []byte, 10),
	}

	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.handleClient(client)
}

// handleClient handles a single client connection
func (s *Server) handleClient(client *Client) {
	defer s.wg.Done()
	defer func() {
		close(client.outgoing) // Close channel before removing client
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
		if err := client.conn.Close(); err != nil {
			log.Printf("Error closing client connection: %v", err)
		}
	}()

	// Start goroutine to send messages to client
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for data := range client.outgoing {
			if _, err := client.conn.Write(data); err != nil {
				log.Printf("Failed to send message to client: %v", err)
				return
			}
		}
	}()

	// Read messages from client
	buf := make([]byte, 4096)
	for {
		n, err := client.conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from client: %v", err)
			}
			return
		}

		if n > 0 {
			// Decode message
			var msg protocol.Message
			if err := msg.Decode(buf[:n]); err != nil {
				log.Printf("Failed to decode message: %v", err)
				continue
			}

			// Handle different message types
			switch msg.Type {
			case protocol.MessageTypeJoin:
				client.username = msg.Sender
				log.Printf("User %s joined", msg.Sender)
				s.broadcast(buf[:n], client)
			case protocol.MessageTypeLeave:
				log.Printf("User %s left", msg.Sender)
				s.broadcast(buf[:n], client)
				return
			case protocol.MessageTypeText:
				log.Printf("Message from %s: %s", msg.Sender, msg.Content)
				s.broadcast(buf[:n], client)
			}
		}
	}
}

// broadcast sends a message to all clients except the sender
func (s *Server) broadcast(data []byte, sender *Client) {
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
