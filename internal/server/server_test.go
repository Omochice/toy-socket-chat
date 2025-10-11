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
	conn.Close()

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

	go srv.Start()
	defer srv.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	addr := srv.Addr()

	// Connect first client
	conn1, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer conn1.Close()

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
	defer conn2.Close()

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

	go srv.Start()
	defer srv.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	addr := srv.Addr()

	// Connect two clients
	conn1, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer conn2.Close()

	// Wait for connections to be established
	time.Sleep(100 * time.Millisecond)

	// Send a text message from client 1
	textMsg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "user1",
		Content: "Hello from user1",
	}
	data, err := textMsg.Encode()
	if err != nil {
		t.Fatalf("Failed to encode text message: %v", err)
	}

	if _, err := conn1.Write(data); err != nil {
		t.Fatalf("Failed to send text message: %v", err)
	}

	// Client 2 should receive the message
	// This is a basic test - in real implementation, we'd need proper message framing
	time.Sleep(200 * time.Millisecond)

	// Test passes if no errors occurred during broadcast
	// More sophisticated testing would involve reading from conn2
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
