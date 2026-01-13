package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/omochice/toy-socket-chat/internal/client"
	"github.com/omochice/toy-socket-chat/internal/server"
)

// TestIntegration_ServerClientCommunication tests end-to-end communication
func TestIntegration_ServerClientCommunication(t *testing.T) {
	srv := server.New(":0")
	go func() {
		_ = srv.Start()
	}()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	serverAddr := srv.Addr()
	if serverAddr == "" {
		t.Fatal("Server address is empty")
	}

	client1 := client.New(serverAddr, "user1", "tcp")
	if err := client1.Connect(); err != nil {
		t.Fatalf("Client 1 failed to connect: %v", err)
	}
	defer client1.Disconnect()

	if err := client1.Join(); err != nil {
		t.Fatalf("Client 1 failed to join: %v", err)
	}

	client2 := client.New(serverAddr, "user2", "tcp")
	if err := client2.Connect(); err != nil {
		t.Fatalf("Client 2 failed to connect: %v", err)
	}
	defer client2.Disconnect()

	if err := client2.Join(); err != nil {
		t.Fatalf("Client 2 failed to join: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if count := srv.ClientCount(); count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}

	select {
	case msg := <-client1.Messages():
		if msg.Type != 1 { // MessageTypeJoin
			t.Logf("Received message type %d (expected join)", msg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Log("Client 1 did not receive join message from user2")
	}

	testMsg := "Hello from user1"
	if err := client1.SendMessage(testMsg); err != nil {
		t.Fatalf("Client 1 failed to send message: %v", err)
	}

	select {
	case msg := <-client2.Messages():
		// Skip join messages if received
		for msg.Type != 0 { // MessageTypeText
			select {
			case msg = <-client2.Messages():
			case <-time.After(1 * time.Second):
				t.Fatal("Timeout waiting for text message")
			}
		}
		if msg.Content != testMsg {
			t.Errorf("Expected message %q, got %q", testMsg, msg.Content)
		}
		if msg.Sender != "user1" {
			t.Errorf("Expected sender %q, got %q", "user1", msg.Sender)
		}
	case <-time.After(2 * time.Second):
		t.Error("Client 2 did not receive message from Client 1")
	}

	testMsg2 := "Hello from user2"
	if err := client2.SendMessage(testMsg2); err != nil {
		t.Fatalf("Client 2 failed to send message: %v", err)
	}

	select {
	case msg := <-client1.Messages():
		// Skip non-text messages if received
		for msg.Type != 0 { // MessageTypeText
			select {
			case msg = <-client1.Messages():
			case <-time.After(1 * time.Second):
				t.Fatal("Timeout waiting for text message from user2")
			}
		}
		if msg.Content != testMsg2 {
			t.Errorf("Expected message %q, got %q", testMsg2, msg.Content)
		}
		if msg.Sender != "user2" {
			t.Errorf("Expected sender %q, got %q", "user2", msg.Sender)
		}
	case <-time.After(2 * time.Second):
		t.Error("Client 1 did not receive message from Client 2")
	}
}

// TestIntegration_MultipleClients tests with multiple clients
func TestIntegration_MultipleClients(t *testing.T) {
	srv := server.New(":0")
	go func() {
		_ = srv.Start()
	}()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	serverAddr := srv.Addr()

	clients := make([]*client.Client, 5)
	for i := 0; i < 5; i++ {
		c := client.New(serverAddr, fmt.Sprintf("user%d", i), "tcp")
		if err := c.Connect(); err != nil {
			t.Fatalf("Client %d failed to connect: %v", i, err)
		}
		defer c.Disconnect()

		if err := c.Join(); err != nil {
			t.Fatalf("Client %d failed to join: %v", i, err)
		}

		clients[i] = c
	}

	time.Sleep(300 * time.Millisecond)

	if count := srv.ClientCount(); count != 5 {
		t.Errorf("Expected 5 clients, got %d", count)
	}

	testMsg := "Broadcast message"
	if err := clients[0].SendMessage(testMsg); err != nil {
		t.Fatalf("Client 0 failed to send message: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	for i := 0; i < 5; i++ {
		for {
			select {
			case <-clients[i].Messages():
			default:
				goto drained
			}
		}
	drained:
	}

	if err := clients[0].SendMessage(testMsg); err != nil {
		t.Fatalf("Client 0 failed to resend message: %v", err)
	}

	for i := 1; i < 5; i++ {
		select {
		case msg := <-clients[i].Messages():
			for msg.Type != 0 {
				select {
				case msg = <-clients[i].Messages():
				case <-time.After(1 * time.Second):
					t.Fatalf("Client %d timeout waiting for text message", i)
				}
			}
			if msg.Content != testMsg {
				t.Errorf("Client %d: expected message %q, got %q", i, testMsg, msg.Content)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("Client %d did not receive broadcast message", i)
		}
	}
}
