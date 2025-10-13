package server

import (
	"bufio"
	"bytes"
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
	address     string
	tcpAddress  string
	wsAddress   string
	listener    net.Listener
	tcpListener net.Listener
	wsListener  net.Listener
	wsServer    *http.Server
	clients     map[*UnifiedClient]bool
	mu          sync.RWMutex
	quit        chan struct{}
	wg          sync.WaitGroup
	singlePort  bool
}

// NewUnifiedServer creates a new UnifiedServer instance
// If wsAddress is empty, both TCP and WebSocket will be handled on tcpAddress
func NewUnifiedServer(tcpAddress, wsAddress string) *UnifiedServer {
	singlePort := wsAddress == ""
	return &UnifiedServer{
		address:    tcpAddress,
		tcpAddress: tcpAddress,
		wsAddress:  wsAddress,
		clients:    make(map[*UnifiedClient]bool),
		quit:       make(chan struct{}),
		singlePort: singlePort,
	}
}

// Start starts both TCP and WebSocket servers
func (s *UnifiedServer) Start() error {
	if s.singlePort {
		// Single port mode: handle both protocols on one port
		listener, err := net.Listen("tcp", s.address)
		if err != nil {
			return fmt.Errorf("failed to start server: %w", err)
		}
		s.listener = listener
		log.Printf("Unified server started on %s (TCP and WebSocket)", listener.Addr().String())

		s.wg.Add(1)
		go s.acceptConnections()
	} else {
		// Dual port mode: separate ports for TCP and WebSocket
		tcpListener, err := net.Listen("tcp", s.tcpAddress)
		if err != nil {
			return fmt.Errorf("failed to start TCP server: %w", err)
		}
		s.tcpListener = tcpListener
		log.Printf("TCP server started on %s", tcpListener.Addr().String())

		wsListener, err := net.Listen("tcp", s.wsAddress)
		if err != nil {
			tcpListener.Close()
			return fmt.Errorf("failed to start WebSocket server: %w", err)
		}
		s.wsListener = wsListener
		log.Printf("WebSocket server started on %s", wsListener.Addr().String())

		s.wg.Add(1)
		go s.acceptTCPConnections()

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
	}

	// Wait for shutdown signal
	<-s.quit
	return fmt.Errorf("Server stopped")
}

// Stop stops the unified server
func (s *UnifiedServer) Stop() {
	close(s.quit)

	if s.listener != nil {
		s.listener.Close()
	}
	if s.tcpListener != nil {
		s.tcpListener.Close()
	}
	if s.wsServer != nil {
		s.wsServer.Close()
	}

	// Don't close client channels here - they will be closed by handleClient functions
	s.wg.Wait()
}

// Addr returns the server's listening address (for single port mode)
func (s *UnifiedServer) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
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

// acceptConnections accepts connections on single port and determines protocol
func (s *UnifiedServer) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.quit:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return
				default:
					log.Printf("Failed to accept connection: %v", err)
					continue
				}
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

// handleConnection determines whether the connection is HTTP (WebSocket) or TCP
func (s *UnifiedServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()

	// Peek at the first few bytes to determine protocol
	reader := bufio.NewReader(conn)
	prefix, err := reader.Peek(4)
	if err != nil {
		log.Printf("Failed to peek connection: %v", err)
		conn.Close()
		return
	}

	// Check if it's an HTTP request (WebSocket)
	// HTTP requests start with methods like "GET ", "POST", "PUT ", "HEAD", etc.
	isHTTP := bytes.HasPrefix(prefix, []byte("GET ")) ||
		bytes.HasPrefix(prefix, []byte("POST")) ||
		bytes.HasPrefix(prefix, []byte("PUT ")) ||
		bytes.HasPrefix(prefix, []byte("HEAD")) ||
		bytes.HasPrefix(prefix, []byte("OPTI")) || // OPTIONS
		bytes.HasPrefix(prefix, []byte("PATC")) || // PATCH
		bytes.HasPrefix(prefix, []byte("DELE")) || // DELETE
		bytes.HasPrefix(prefix, []byte("CONN"))    // CONNECT

	if isHTTP {
		// Handle as WebSocket (HTTP upgrade)
		s.handleHTTPConnection(conn, reader)
	} else {
		// Handle as raw TCP client
		s.handleRawTCPConnection(conn, reader)
	}
}

// handleHTTPConnection handles HTTP connections (WebSocket upgrades)
func (s *UnifiedServer) handleHTTPConnection(conn net.Conn, reader *bufio.Reader) {
	// Create an HTTP server to handle the WebSocket upgrade
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)

	// Wrap the connection with the buffered reader
	bufConn := &bufferedConn{
		Conn:   conn,
		reader: reader,
	}

	httpServer := &http.Server{Handler: mux}
	httpServer.Serve(&singleConnListener{conn: bufConn})
}

// handleRawTCPConnection handles raw TCP client connections
func (s *UnifiedServer) handleRawTCPConnection(conn net.Conn, reader *bufio.Reader) {
	client := &UnifiedClient{
		id:         fmt.Sprintf("tcp-%p", conn),
		outgoing:   make(chan []byte, 10),
		clientType: "tcp",
	}

	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.handleTCPClientWithReader(client, conn, reader)
}

// bufferedConn wraps a net.Conn with a bufio.Reader to preserve peeked data
type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (bc *bufferedConn) Read(p []byte) (int, error) {
	return bc.reader.Read(p)
}

// singleConnListener is a net.Listener that returns a single connection
type singleConnListener struct {
	conn net.Conn
	once sync.Once
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	var c net.Conn
	l.once.Do(func() {
		c = l.conn
	})
	if c != nil {
		return c, nil
	}
	return nil, io.EOF
}

func (l *singleConnListener) Close() error {
	return nil
}

func (l *singleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
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

// handleTCPClientWithReader handles a TCP client connection with a buffered reader
func (s *UnifiedServer) handleTCPClientWithReader(client *UnifiedClient, conn net.Conn, reader *bufio.Reader) {
	defer s.wg.Done()
	defer conn.Close()
	defer func() {
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
	}()

	// Start writer goroutine (write directly to the underlying connection)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for data := range client.outgoing {
			// Write directly to the underlying connection, not through the reader
			if rawConn, ok := conn.(interface{ Write([]byte) (int, error) }); ok {
				if _, err := rawConn.Write(data); err != nil {
					log.Printf("Failed to send message to TCP client: %v", err)
					return
				}
			}
		}
	}()

	defer func() {
		close(client.outgoing)
		<-writerDone
	}()

	// Read messages from client using buffered reader
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
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
