package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/omochice/toy-socket-chat/internal/chat"
	tcpclient "github.com/omochice/toy-socket-chat/internal/client/tcp"
	wsclient "github.com/omochice/toy-socket-chat/internal/client/ws"
	"github.com/omochice/toy-socket-chat/internal/transport/tcp"
	wstransport "github.com/omochice/toy-socket-chat/internal/transport/ws"
)

// TestIntegration_ServerClientCommunication tests end-to-end communication
func TestIntegration_ServerClientCommunication(t *testing.T) {
	hub := chat.NewHub()
	srv := tcp.New(":0", hub)
	go srv.Start()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	serverAddr := srv.Addr()
	if serverAddr == "" {
		t.Fatal("Server address is empty")
	}

	client1 := tcpclient.New(serverAddr, "user1")
	if err := client1.Connect(); err != nil {
		t.Fatalf("Client 1 failed to connect: %v", err)
	}
	defer client1.Disconnect()

	client2 := tcpclient.New(serverAddr, "user2")
	if err := client2.Connect(); err != nil {
		t.Fatalf("Client 2 failed to connect: %v", err)
	}
	defer client2.Disconnect()

	time.Sleep(200 * time.Millisecond)

	if count := hub.ClientCount(); count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}

	testMsg := "Hello from user1"
	if err := client1.SendMessage(testMsg); err != nil {
		t.Fatalf("Client 1 failed to send message: %v", err)
	}

	select {
	case msg := <-client2.Messages():
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
	hub := chat.NewHub()
	srv := tcp.New(":0", hub)
	go srv.Start()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	serverAddr := srv.Addr()

	clients := make([]*tcpclient.Client, 5)
	for i := 0; i < 5; i++ {
		c := tcpclient.New(serverAddr, fmt.Sprintf("user%d", i))
		if err := c.Connect(); err != nil {
			t.Fatalf("Client %d failed to connect: %v", i, err)
		}
		defer c.Disconnect()
		clients[i] = c
	}

	time.Sleep(300 * time.Millisecond)

	if count := hub.ClientCount(); count != 5 {
		t.Errorf("Expected 5 clients, got %d", count)
	}

	testMsg := "Broadcast message"
	if err := clients[0].SendMessage(testMsg); err != nil {
		t.Fatalf("Client 0 failed to send message: %v", err)
	}

	for i := 1; i < 5; i++ {
		select {
		case msg := <-clients[i].Messages():
			if msg.Content != testMsg {
				t.Errorf("Client %d: expected message %q, got %q", i, testMsg, msg.Content)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("Client %d did not receive broadcast message", i)
		}
	}
}

// TestIntegration_CrossTransport tests TCP and WebSocket clients communicating
func TestIntegration_CrossTransport(t *testing.T) {
	hub := chat.NewHub()

	tcpSrv := tcp.New(":0", hub)
	wsSrv := wstransport.New(":0", hub)

	go tcpSrv.Start()
	go wsSrv.Start()
	defer tcpSrv.Stop()
	defer wsSrv.Stop()

	time.Sleep(100 * time.Millisecond)

	tcpAddr := tcpSrv.Addr()
	wsAddr := "ws://" + wsSrv.Addr()

	// Connect TCP client
	tcpClient := tcpclient.New(tcpAddr, "tcp_user")
	if err := tcpClient.Connect(); err != nil {
		t.Fatalf("TCP client failed to connect: %v", err)
	}
	defer tcpClient.Disconnect()

	// Connect WebSocket client
	wsClient := wsclient.New(wsAddr, "ws_user")
	if err := wsClient.Connect(); err != nil {
		t.Fatalf("WebSocket client failed to connect: %v", err)
	}
	defer wsClient.Disconnect()

	time.Sleep(100 * time.Millisecond)

	// Verify both clients registered
	if count := hub.ClientCount(); count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}

	// TCP client sends message
	if err := tcpClient.SendMessage("hello from tcp"); err != nil {
		t.Fatalf("TCP client failed to send: %v", err)
	}

	// WebSocket client should receive
	select {
	case msg := <-wsClient.Messages():
		if msg.Content != "hello from tcp" {
			t.Errorf("Expected content %q, got %q", "hello from tcp", msg.Content)
		}
		if msg.Sender != "tcp_user" {
			t.Errorf("Expected sender %q, got %q", "tcp_user", msg.Sender)
		}
	case <-time.After(2 * time.Second):
		t.Error("WebSocket client did not receive message from TCP client")
	}

	// WebSocket client sends message
	if err := wsClient.SendMessage("hello from ws"); err != nil {
		t.Fatalf("WebSocket client failed to send: %v", err)
	}

	// TCP client should receive
	select {
	case msg := <-tcpClient.Messages():
		if msg.Content != "hello from ws" {
			t.Errorf("Expected content %q, got %q", "hello from ws", msg.Content)
		}
		if msg.Sender != "ws_user" {
			t.Errorf("Expected sender %q, got %q", "ws_user", msg.Sender)
		}
	case <-time.After(2 * time.Second):
		t.Error("TCP client did not receive message from WebSocket client")
	}
}
