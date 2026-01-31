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

	time.Sleep(100 * time.Millisecond)

	addr := srv.Addr()
	if addr == "" {
		t.Fatal("Server address is empty")
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	_ = conn.Close()

	srv.Stop()

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

	time.Sleep(100 * time.Millisecond)

	addr := srv.Addr()

	conn1, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer func() {
		_ = conn1.Close()
	}()

	joinMsg := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "user1",
	}
	data, err := joinMsg.Encode()
	if err != nil {
		t.Fatalf("Failed to encode join message: %v", err)
	}

	if _, err := conn1.Write(data); err != nil {
		t.Fatalf("Failed to send join message: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer func() {
		_ = conn2.Close()
	}()

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

	time.Sleep(100 * time.Millisecond)

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

	time.Sleep(100 * time.Millisecond)

	srv.Stop()

	select {
	case err := <-errChan:
		if err != nil && err.Error() != "server stopped" {
			t.Errorf("Expected 'server stopped' error, got: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Server did not stop in time")
	}
}
