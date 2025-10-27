package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

// Client represents a chat client
type Client struct {
	address  string
	username string
	protocol string
	conn     ClientConnection
	messages chan protocol.Message
	mu       sync.RWMutex
	done     chan struct{}
	wg       sync.WaitGroup
}

// New creates a new Client instance
func New(address, username, proto string) *Client {
	return &Client{
		address:  address,
		username: username,
		protocol: proto,
		messages: make(chan protocol.Message, 10),
		done:     make(chan struct{}),
	}
}

// Connect establishes a connection to the server
func (c *Client) Connect() error {
	var conn ClientConnection
	var err error

	switch c.protocol {
	case "ws":
		conn, err = c.connectWebSocket()
	case "tcp":
		fallthrough
	default:
		conn, err = c.connectTCP()
	}

	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Start receiving messages
	c.wg.Add(1)
	go c.receiveMessages()

	return nil
}

func (c *Client) connectTCP() (ClientConnection, error) {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect via TCP: %w", err)
	}
	return NewTCPClientConnection(conn), nil
}

func (c *Client) connectWebSocket() (ClientConnection, error) {
	// Use ws.Dial to establish WebSocket connection
	// ws.Dial returns (net.Conn, *bufio.Reader, Handshake, error)
	wsConn, _, _, err := ws.Dial(context.Background(), fmt.Sprintf("ws://%s/", c.address))
	if err != nil {
		return nil, fmt.Errorf("failed to connect via WebSocket: %w", err)
	}

	return NewWebSocketClientConnection(wsConn), nil
}

// Disconnect closes the connection to the server
func (c *Client) Disconnect() {
	c.mu.Lock()
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
		c.conn = nil
	}
	c.mu.Unlock()

	close(c.done)
	c.wg.Wait()
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil
}

// SendMessage sends a text message to the server
func (c *Client) SendMessage(content string) error {
	msg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  c.username,
		Content: content,
	}
	return c.send(msg)
}

// Join sends a join message to the server
func (c *Client) Join() error {
	msg := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: c.username,
	}
	return c.send(msg)
}

// Leave sends a leave message to the server
func (c *Client) Leave() error {
	msg := protocol.Message{
		Type:   protocol.MessageTypeLeave,
		Sender: c.username,
	}
	return c.send(msg)
}

// Messages returns the channel for receiving messages
func (c *Client) Messages() <-chan protocol.Message {
	return c.messages
}

// send sends a message to the server
func (c *Client) send(msg protocol.Message) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected to server")
	}

	data, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// receiveMessages continuously receives messages from the server
func (c *Client) receiveMessages() {
	defer c.wg.Done()

	buf := make([]byte, 4096)
	for {
		select {
		case <-c.done:
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			n, err := conn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from server: %v", err)
				}
				return
			}

			if n > 0 {
				var msg protocol.Message
				if err := msg.Decode(buf[:n]); err != nil {
					log.Printf("Failed to decode message: %v", err)
					continue
				}

				select {
				case c.messages <- msg:
				case <-c.done:
					return
				}
			}
		}
	}
}
