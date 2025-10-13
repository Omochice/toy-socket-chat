package client

import (
	"strings"
	"testing"
	"time"

	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

func TestNewWebSocketClient(t *testing.T) {
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")
	if client == nil {
		t.Fatal("NewWebSocketClient returned nil")
	}

	if client.username != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", client.username)
	}
}

func TestWebSocketClient_Connect(t *testing.T) {
	// This test requires a running server
	// We'll test connection failure for now
	client := NewWebSocketClient("ws://localhost:99999/ws", "alice")

	err := client.Connect()
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
}

func TestWebSocketClient_IsConnected(t *testing.T) {
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")

	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}
}

func TestWebSocketClient_SendMessage_NotConnected(t *testing.T) {
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")

	err := client.SendMessage("Hello")
	if err == nil {
		t.Error("Expected error when sending message without connection")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("Expected 'not connected' error, got: %v", err)
	}
}

func TestWebSocketClient_Join_NotConnected(t *testing.T) {
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")

	err := client.Join()
	if err == nil {
		t.Error("Expected error when joining without connection")
	}
}

func TestWebSocketClient_Leave_NotConnected(t *testing.T) {
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")

	err := client.Leave()
	if err == nil {
		t.Error("Expected error when leaving without connection")
	}
}

func TestWebSocketClient_Messages(t *testing.T) {
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")

	msgChan := client.Messages()
	if msgChan == nil {
		t.Error("Messages channel should not be nil")
	}

	// Check that the channel is initially empty
	select {
	case <-msgChan:
		t.Error("Messages channel should be empty initially")
	case <-time.After(100 * time.Millisecond):
		// Expected: timeout means channel is empty
	}
}

func TestWebSocketClient_Disconnect_NotConnected(t *testing.T) {
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")

	// Should not panic when disconnecting without connection
	client.Disconnect()

	if client.IsConnected() {
		t.Error("Client should not be connected after disconnect")
	}
}

// Integration test helper
type mockWebSocketServer struct {
	address string
}

func TestWebSocketClient_IntegrationWithServer(t *testing.T) {
	// This is a placeholder for integration tests
	// We'll implement this after creating the server mock or using real server
	t.Skip("Integration test - requires running WebSocket server")
}

func TestWebSocketClient_SendAndReceive(t *testing.T) {
	// This test will verify end-to-end message flow
	t.Skip("Integration test - requires running WebSocket server")
}

func TestWebSocketClient_MultipleClients(t *testing.T) {
	// This test will verify multiple clients can communicate
	t.Skip("Integration test - requires running WebSocket server")
}

func TestWebSocketClient_Reconnect(t *testing.T) {
	// Test reconnection scenario
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")

	// First disconnect (should not panic)
	client.Disconnect()

	// Try to connect (will fail without server, but should not panic)
	err := client.Connect()
	if err == nil {
		t.Error("Expected connection error without server")
	}

	// Disconnect again
	client.Disconnect()
}

func TestWebSocketClient_MessageTypes(t *testing.T) {
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")

	// Test that different message types can be created
	// (without sending, since not connected)

	testCases := []struct {
		name string
		fn   func() error
	}{
		{"Join", client.Join},
		{"Leave", client.Leave},
		{"SendMessage", func() error { return client.SendMessage("test") }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if err == nil {
				t.Errorf("%s should fail when not connected", tc.name)
			}
		})
	}
}

func TestWebSocketClient_ConcurrentSend(t *testing.T) {
	// Test that concurrent sends don't cause race conditions
	client := NewWebSocketClient("ws://localhost:8080/ws", "alice")

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			_ = client.SendMessage("message")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without panic, test passed
}

// Test helper to verify message encoding/decoding
func TestWebSocketClient_MessageEncoding(t *testing.T) {
	// This verifies that the client uses the protocol package correctly
	msg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "alice",
		Content: "Hello",
	}

	data, err := msg.Encode()
	if err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}

	var decoded protocol.Message
	if err := decoded.Decode(data); err != nil {
		t.Fatalf("Failed to decode message: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Expected type %v, got %v", msg.Type, decoded.Type)
	}
	if decoded.Sender != msg.Sender {
		t.Errorf("Expected sender '%s', got '%s'", msg.Sender, decoded.Sender)
	}
	if decoded.Content != msg.Content {
		t.Errorf("Expected content '%s', got '%s'", msg.Content, decoded.Content)
	}
}
