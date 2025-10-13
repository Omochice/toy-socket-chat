package client

import (
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

// WebSocketClient represents a WebSocket chat client
type WebSocketClient struct {
	address    string
	username   string
	conn       *websocket.Conn
	messages   chan protocol.Message
	mu         sync.RWMutex
	done       chan struct{}
	doneOnce   sync.Once
	wg         sync.WaitGroup
	isShutdown bool
}

// NewWebSocketClient creates a new WebSocketClient instance
func NewWebSocketClient(address, username string) *WebSocketClient {
	return &WebSocketClient{
		address:  address,
		username: username,
		messages: make(chan protocol.Message, 10),
		done:     make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to the server
func (c *WebSocketClient) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.address, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Start receiving messages
	c.wg.Add(1)
	go c.receiveMessages()

	return nil
}

// Disconnect closes the WebSocket connection to the server
func (c *WebSocketClient) Disconnect() {
	c.mu.Lock()
	if c.isShutdown {
		c.mu.Unlock()
		return
	}
	c.isShutdown = true
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	c.doneOnce.Do(func() {
		close(c.done)
	})
	c.wg.Wait()
}

// IsConnected returns whether the client is connected
func (c *WebSocketClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil
}

// SendMessage sends a text message to the server
func (c *WebSocketClient) SendMessage(content string) error {
	msg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  c.username,
		Content: content,
	}
	return c.send(msg)
}

// Join sends a join message to the server
func (c *WebSocketClient) Join() error {
	msg := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: c.username,
	}
	return c.send(msg)
}

// Leave sends a leave message to the server
func (c *WebSocketClient) Leave() error {
	msg := protocol.Message{
		Type:   protocol.MessageTypeLeave,
		Sender: c.username,
	}
	return c.send(msg)
}

// Messages returns the channel for receiving messages
func (c *WebSocketClient) Messages() <-chan protocol.Message {
	return c.messages
}

// send sends a message to the server
func (c *WebSocketClient) send(msg protocol.Message) error {
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

	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// receiveMessages continuously receives messages from the server
func (c *WebSocketClient) receiveMessages() {
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

				select {
				case c.messages <- msg:
				case <-c.done:
					return
				}
			}
		}
	}
}
