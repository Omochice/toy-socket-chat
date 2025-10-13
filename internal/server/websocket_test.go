package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

func TestWebSocketServer_Start(t *testing.T) {
	srv := NewWebSocketServer(":0")
	if srv == nil {
		t.Fatal("NewWebSocketServer returned nil")
	}

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Check if server is running
	addr := srv.Addr()
	if addr == "" {
		t.Fatal("Server address is empty")
	}

	// Stop server
	srv.Stop()

	// Wait for server to stop
	select {
	case err := <-errChan:
		if err != nil && !strings.Contains(err.Error(), "Server stopped") {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Server did not stop in time")
	}
}

func TestWebSocketServer_ClientConnection(t *testing.T) {
	srv := NewWebSocketServer(":0")

	// Start server in background
	go srv.Start()
	defer srv.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Connect client
	addr := srv.Addr()
	url := "ws://" + addr + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Check client count
	time.Sleep(100 * time.Millisecond)
	if count := srv.ClientCount(); count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}
}

func TestWebSocketServer_MessageBroadcast(t *testing.T) {
	srv := NewWebSocketServer(":0")

	// Start server in background
	go srv.Start()
	defer srv.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	addr := srv.Addr()
	url := "ws://" + addr + "/ws"

	// Connect first client
	conn1, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer conn1.Close()

	// Send join message from client 1
	joinMsg := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "alice",
	}
	joinData, _ := joinMsg.Encode()
	if err := conn1.WriteMessage(websocket.BinaryMessage, joinData); err != nil {
		t.Fatalf("Failed to send join message: %v", err)
	}

	// Wait for join to be processed
	time.Sleep(100 * time.Millisecond)

	// Connect second client
	conn2, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer conn2.Close()

	// Send join message from client 2
	joinMsg2 := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "bob",
	}
	joinData2, _ := joinMsg2.Encode()
	if err := conn2.WriteMessage(websocket.BinaryMessage, joinData2); err != nil {
		t.Fatalf("Failed to send join message from client 2: %v", err)
	}

	// Client 1 should receive bob's join message
	conn1.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to receive message on client 1: %v", err)
	}

	var receivedMsg protocol.Message
	if err := receivedMsg.Decode(data); err != nil {
		t.Fatalf("Failed to decode message: %v", err)
	}

	if receivedMsg.Type != protocol.MessageTypeJoin {
		t.Errorf("Expected JOIN message, got %v", receivedMsg.Type)
	}
	if receivedMsg.Sender != "bob" {
		t.Errorf("Expected sender 'bob', got '%s'", receivedMsg.Sender)
	}

	// Send text message from client 2
	textMsg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "bob",
		Content: "Hello, alice!",
	}
	textData, _ := textMsg.Encode()
	if err := conn2.WriteMessage(websocket.BinaryMessage, textData); err != nil {
		t.Fatalf("Failed to send text message: %v", err)
	}

	// Client 1 should receive the text message
	conn1.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to receive text message on client 1: %v", err)
	}

	var textReceivedMsg protocol.Message
	if err := textReceivedMsg.Decode(data); err != nil {
		t.Fatalf("Failed to decode text message: %v", err)
	}

	if textReceivedMsg.Type != protocol.MessageTypeText {
		t.Errorf("Expected TEXT message, got %v", textReceivedMsg.Type)
	}
	if textReceivedMsg.Content != "Hello, alice!" {
		t.Errorf("Expected content 'Hello, alice!', got '%s'", textReceivedMsg.Content)
	}
}

func TestWebSocketServer_HandleUpgrade(t *testing.T) {
	srv := NewWebSocketServer(":0")

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleWebSocket(w, r)
	}))
	defer testServer.Close()

	// Try to connect via WebSocket
	url := "ws" + strings.TrimPrefix(testServer.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to upgrade to WebSocket: %v", err)
	}
	defer conn.Close()

	// Connection successful
	if conn == nil {
		t.Error("WebSocket connection is nil")
	}
}
