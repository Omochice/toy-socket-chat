package server_test

import (
	"net"
	"testing"
	"time"

	"github.com/omochice/toy-socket-chat/internal/server"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

func TestServer_Start(t *testing.T) {
	srv := server.New(":0") // Use port 0 to let OS assign a free port

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Try to get the actual address
	addr := srv.Addr()
	if addr == "" {
		t.Fatal("Server address is empty")
	}

	// Try to connect to the server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	_ = conn.Close()

	srv.Stop()

	// Wait for server to stop
	select {
	case err := <-errChan:
		if err != nil && err.Error() != "server stopped" {
			t.Errorf("Server.Start() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Server did not stop in time")
	}
}

func TestServer_ClientConnection(t *testing.T) {
	srv := server.New(":0")

	go func() {
		_ = srv.Start()
	}()
	defer srv.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	addr := srv.Addr()

	// Connect first client
	conn1, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer func() {
		_ = conn1.Close()
	}()

	// Send join message from client 1
	joinMsg := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "user1",
	}
	data, err := joinMsg.Encode()
	if err != nil {
		t.Fatalf("Failed to encode join message: %v", err)
	}

	// Write message length first (simple protocol: 4 bytes for length, then data)
	if _, err := conn1.Write(data); err != nil {
		t.Fatalf("Failed to send join message: %v", err)
	}

	// Wait a bit for message processing
	time.Sleep(100 * time.Millisecond)

	// Connect second client
	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer func() {
		_ = conn2.Close()
	}()

	// Send join message from client 2
	joinMsg2 := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "user2",
	}
	data2, err := joinMsg2.Encode()
	if err != nil {
		t.Fatalf("Failed to encode join message for client 2: %v", err)
	}
	if _, err := conn2.Write(data2); err != nil {
		t.Fatalf("Failed to send join message from client 2: %v", err)
	}

	// Wait for second client to be registered
	time.Sleep(100 * time.Millisecond)

	// Both connections should be maintained
	count := srv.ClientCount()
	if count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}
}

func TestServer_MessageBroadcast(t *testing.T) {
	srv := server.New(":0")

	go func() {
		_ = srv.Start()
	}()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	addr := srv.Addr()

	conn1, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer func() {
		_ = conn1.Close()
	}()

	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer func() {
		_ = conn2.Close()
	}()

	joinMsg1 := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "user1",
	}
	data1, err := joinMsg1.Encode()
	if err != nil {
		t.Fatalf("Failed to encode join message for client 1: %v", err)
	}
	if _, err := conn1.Write(data1); err != nil {
		t.Fatalf("Failed to send join message from client 1: %v", err)
	}

	joinMsg2 := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "user2",
	}
	data2, err := joinMsg2.Encode()
	if err != nil {
		t.Fatalf("Failed to encode join message for client 2: %v", err)
	}
	if _, err := conn2.Write(data2); err != nil {
		t.Fatalf("Failed to send join message from client 2: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	textMsg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "user1",
		Content: "Hello from user1",
	}
	textData, err := textMsg.Encode()
	if err != nil {
		t.Fatalf("Failed to encode text message: %v", err)
	}

	if _, err := conn1.Write(textData); err != nil {
		t.Fatalf("Failed to send text message: %v", err)
	}

	if err := conn2.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	buf := make([]byte, 1024)
	var receivedMsg protocol.Message

	for {
		n, err := conn2.Read(buf)
		if err != nil {
			t.Fatalf("Failed to read from conn2: %v", err)
		}

		if err := receivedMsg.Decode(buf[:n]); err != nil {
			t.Fatalf("Failed to decode received message: %v", err)
		}

		if receivedMsg.Type == protocol.MessageTypeText {
			break
		}
	}

	if receivedMsg.Sender != "user1" {
		t.Errorf("Expected sender 'user1', got %q", receivedMsg.Sender)
	}
	if receivedMsg.Content != "Hello from user1" {
		t.Errorf("Expected content 'Hello from user1', got %q", receivedMsg.Content)
	}
}

func TestServer_Stop(t *testing.T) {
	srv := server.New(":0")

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Stop the server
	srv.Stop()

	// Verify server stopped
	select {
	case err := <-errChan:
		if err != nil && err.Error() != "server stopped" {
			t.Errorf("Expected 'server stopped' error, got: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Server did not stop in time")
	}
}
