package ws_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ws "github.com/omochice/toy-socket-chat/internal/client/ws"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
	"nhooyr.io/websocket"
)

func TestClient_ConnectAndDisconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("failed to accept: %v", err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// Wait for client to disconnect
		c.Read(context.Background())
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := ws.New(wsURL, "testuser")

	if client.IsConnected() {
		t.Error("expected IsConnected() to be false before Connect()")
	}

	err := client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if !client.IsConnected() {
		t.Error("expected IsConnected() to be true after Connect()")
	}

	client.Disconnect()

	if client.IsConnected() {
		t.Error("expected IsConnected() to be false after Disconnect()")
	}
}

func TestClient_SendMessage(t *testing.T) {
	received := make(chan []byte, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("failed to accept: %v", err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		_, data, err := c.Read(context.Background())
		if err != nil {
			return
		}
		received <- data
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := ws.New(wsURL, "testuser")

	err := client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Disconnect()

	err = client.SendMessage("hello")
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	select {
	case data := <-received:
		var msg protocol.Message
		if err := msg.Decode(data); err != nil {
			t.Fatalf("failed to decode message: %v", err)
		}
		if msg.Type != protocol.MessageTypeText {
			t.Errorf("expected message type %d, got %d", protocol.MessageTypeText, msg.Type)
		}
		if msg.Content != "hello" {
			t.Errorf("expected content %q, got %q", "hello", msg.Content)
		}
		if msg.Sender != "testuser" {
			t.Errorf("expected sender %q, got %q", "testuser", msg.Sender)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestClient_Join(t *testing.T) {
	received := make(chan []byte, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("failed to accept: %v", err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		_, data, err := c.Read(context.Background())
		if err != nil {
			return
		}
		received <- data
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := ws.New(wsURL, "testuser")

	err := client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Disconnect()

	err = client.Join()
	if err != nil {
		t.Fatalf("Join() error = %v", err)
	}

	select {
	case data := <-received:
		var msg protocol.Message
		if err := msg.Decode(data); err != nil {
			t.Fatalf("failed to decode message: %v", err)
		}
		if msg.Type != protocol.MessageTypeJoin {
			t.Errorf("expected message type %d, got %d", protocol.MessageTypeJoin, msg.Type)
		}
		if msg.Sender != "testuser" {
			t.Errorf("expected sender %q, got %q", "testuser", msg.Sender)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestClient_Leave(t *testing.T) {
	received := make(chan []byte, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("failed to accept: %v", err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		_, data, err := c.Read(context.Background())
		if err != nil {
			return
		}
		received <- data
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := ws.New(wsURL, "testuser")

	err := client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Disconnect()

	err = client.Leave()
	if err != nil {
		t.Fatalf("Leave() error = %v", err)
	}

	select {
	case data := <-received:
		var msg protocol.Message
		if err := msg.Decode(data); err != nil {
			t.Fatalf("failed to decode message: %v", err)
		}
		if msg.Type != protocol.MessageTypeLeave {
			t.Errorf("expected message type %d, got %d", protocol.MessageTypeLeave, msg.Type)
		}
		if msg.Sender != "testuser" {
			t.Errorf("expected sender %q, got %q", "testuser", msg.Sender)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestClient_ReceiveMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("failed to accept: %v", err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		msg := protocol.Message{
			Type:    protocol.MessageTypeText,
			Sender:  "server",
			Content: "welcome",
		}
		data, _ := msg.Encode()
		c.Write(context.Background(), websocket.MessageBinary, data)

		// Wait for client to disconnect
		c.Read(context.Background())
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := ws.New(wsURL, "testuser")

	err := client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer client.Disconnect()

	select {
	case msg := <-client.Messages():
		if msg.Type != protocol.MessageTypeText {
			t.Errorf("expected message type %d, got %d", protocol.MessageTypeText, msg.Type)
		}
		if msg.Content != "welcome" {
			t.Errorf("expected content %q, got %q", "welcome", msg.Content)
		}
		if msg.Sender != "server" {
			t.Errorf("expected sender %q, got %q", "server", msg.Sender)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestClient_SendMessage_NotConnected(t *testing.T) {
	client := ws.New("ws://localhost:9999", "testuser")

	err := client.SendMessage("hello")
	if err == nil {
		t.Error("expected error when sending without connection")
	}
}
