package ws

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/omochice/toy-socket-chat/internal/chat"
	"nhooyr.io/websocket"
)

// Server handles WebSocket connections and delegates to Hub.
type Server struct {
	address  string
	listener net.Listener
	hub      *chat.Hub
	server   *http.Server
	quit     chan struct{}
	wg       sync.WaitGroup
}

// New creates a WebSocket server that uses the provided Hub.
func New(address string, hub *chat.Hub) *Server {
	return &Server{
		address: address,
		hub:     hub,
		quit:    make(chan struct{}),
	}
}

// Start starts accepting WebSocket connections.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWebSocket)

	s.server = &http.Server{Handler: mux}

	log.Printf("WebSocket server started on %s", listener.Addr().String())

	return s.server.Serve(listener)
}

// Stop stops the WebSocket server.
func (s *Server) Stop() {
	close(s.quit)
	if s.server != nil {
		s.server.Shutdown(context.Background())
	}
	s.wg.Wait()
}

// Addr returns the listening address.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	wsConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("Failed to accept WebSocket connection: %v", err)
		return
	}

	client := &chat.Client{
		Conn:     NewConnWithAddr(wsConn, r.RemoteAddr),
		Outgoing: make(chan []byte, 10),
	}

	s.hub.Register(client)

	s.wg.Add(2)
	go s.handleClient(client)
	go s.writeLoop(client)
}

func (s *Server) handleClient(client *chat.Client) {
	defer s.wg.Done()
	defer close(client.Outgoing)
	s.hub.HandleClient(client)
}

func (s *Server) writeLoop(client *chat.Client) {
	defer s.wg.Done()
	for data := range client.Outgoing {
		if err := client.Conn.Write(context.Background(), data); err != nil {
			log.Printf("Failed to write to WebSocket client: %v", err)
			return
		}
	}
}
