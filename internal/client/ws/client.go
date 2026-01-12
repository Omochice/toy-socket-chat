// Package ws provides a WebSocket client for the chat server.
package ws

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/omochice/toy-socket-chat/pkg/protocol"
	"nhooyr.io/websocket"
)

// Client represents a WebSocket chat client.
type Client struct {
	address  string
	username string
	conn     *websocket.Conn
	messages chan protocol.Message
	mu       sync.RWMutex
	done     chan struct{}
	wg       sync.WaitGroup
}

// New creates a new WebSocket Client instance.
func New(address, username string) *Client {
	return &Client{
		address:  address,
		username: username,
		messages: make(chan protocol.Message, 10),
		done:     make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to the server.
func (c *Client) Connect() error {
	conn, _, err := websocket.Dial(context.Background(), c.address, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	c.wg.Add(1)
	go c.receiveMessages()

	return nil
}

// Disconnect closes the WebSocket connection.
func (c *Client) Disconnect() {
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "")
		c.conn = nil
	}
	c.mu.Unlock()

	close(c.done)
	c.wg.Wait()
}

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil
}

// SendMessage sends a text message to the server.
func (c *Client) SendMessage(content string) error {
	msg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  c.username,
		Content: content,
	}
	return c.send(msg)
}

// Join sends a join message to the server.
func (c *Client) Join() error {
	msg := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: c.username,
	}
	return c.send(msg)
}

// Leave sends a leave message to the server.
func (c *Client) Leave() error {
	msg := protocol.Message{
		Type:   protocol.MessageTypeLeave,
		Sender: c.username,
	}
	return c.send(msg)
}

// Messages returns the channel for receiving messages.
func (c *Client) Messages() <-chan protocol.Message {
	return c.messages
}

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

	if err := conn.Write(context.Background(), websocket.MessageBinary, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (c *Client) receiveMessages() {
	defer c.wg.Done()

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

			_, data, err := conn.Read(context.Background())
			if err != nil {
				select {
				case <-c.done:
					return
				default:
					log.Printf("Error reading from server: %v", err)
					return
				}
			}

			var msg protocol.Message
			if err := msg.Decode(data); err != nil {
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
