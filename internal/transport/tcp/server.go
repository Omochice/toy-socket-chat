package tcp

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/omochice/toy-socket-chat/internal/chat"
)

// Server handles TCP connections and delegates to Hub.
type Server struct {
	address  string
	listener net.Listener
	hub      *chat.Hub
	quit     chan struct{}
	wg       sync.WaitGroup
}

// New creates a TCP server that uses the provided Hub.
func New(address string, hub *chat.Hub) *Server {
	return &Server{
		address: address,
		hub:     hub,
		quit:    make(chan struct{}),
	}
}

// Start starts accepting TCP connections.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}
	s.listener = listener

	log.Printf("TCP server started on %s", listener.Addr().String())

	for {
		select {
		case <-s.quit:
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return nil
				default:
					log.Printf("Failed to accept TCP connection: %v", err)
					continue
				}
			}

			client := &chat.Client{
				Conn:     NewConn(conn),
				Outgoing: make(chan []byte, 10),
			}

			s.hub.Register(client)

			s.wg.Add(2)
			go s.handleClient(client)
			go s.writeLoop(client)
		}
	}
}

// Stop stops the TCP server.
func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
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

func (s *Server) handleClient(client *chat.Client) {
	defer s.wg.Done()
	defer close(client.Outgoing)
	s.hub.HandleClient(client)
}

func (s *Server) writeLoop(client *chat.Client) {
	defer s.wg.Done()
	for data := range client.Outgoing {
		if err := client.Conn.Write(nil, data); err != nil {
			log.Printf("Failed to write to client: %v", err)
			return
		}
	}
}
