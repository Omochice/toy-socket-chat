package chat_test

import (
	"testing"
	"time"

	"github.com/omochice/toy-socket-chat/internal/chat"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

func createJoinMessage(t *testing.T, username string) []byte {
	t.Helper()
	msg := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: username,
	}
	data, err := msg.Encode()
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}
	return data
}

func createTextMessage(t *testing.T, sender, content string) []byte {
	t.Helper()
	msg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  sender,
		Content: content,
	}
	data, err := msg.Encode()
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}
	return data
}

func TestHub_Register(t *testing.T) {
	hub := chat.NewHub()
	client := &chat.Client{
		Conn:     &mockConn{remoteAddr: "127.0.0.1:1234"},
		Username: "testuser",
		Outgoing: make(chan []byte, 10),
	}

	hub.Register(client)

	if got := hub.ClientCount(); got != 1 {
		t.Errorf("ClientCount() = %d, want 1", got)
	}
}

func TestHub_Register_MultipleClients(t *testing.T) {
	hub := chat.NewHub()

	for i := 0; i < 3; i++ {
		client := &chat.Client{
			Conn:     &mockConn{remoteAddr: "127.0.0.1:1234"},
			Username: "user",
			Outgoing: make(chan []byte, 10),
		}
		hub.Register(client)
	}

	if got := hub.ClientCount(); got != 3 {
		t.Errorf("ClientCount() = %d, want 3", got)
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := chat.NewHub()
	client := &chat.Client{
		Conn:     &mockConn{remoteAddr: "127.0.0.1:1234"},
		Username: "testuser",
		Outgoing: make(chan []byte, 10),
	}

	hub.Register(client)
	hub.Unregister(client)

	if got := hub.ClientCount(); got != 0 {
		t.Errorf("ClientCount() = %d, want 0", got)
	}
}

func TestHub_Unregister_NonExistentClient(t *testing.T) {
	hub := chat.NewHub()
	client := &chat.Client{
		Conn:     &mockConn{remoteAddr: "127.0.0.1:1234"},
		Username: "testuser",
		Outgoing: make(chan []byte, 10),
	}

	hub.Unregister(client)

	if got := hub.ClientCount(); got != 0 {
		t.Errorf("ClientCount() = %d, want 0", got)
	}
}

func TestHub_Broadcast(t *testing.T) {
	hub := chat.NewHub()
	sender := &chat.Client{
		Conn:     &mockConn{remoteAddr: "127.0.0.1:1234"},
		Username: "sender",
		Outgoing: make(chan []byte, 10),
	}
	receiver := &chat.Client{
		Conn:     &mockConn{remoteAddr: "127.0.0.1:5678"},
		Username: "receiver",
		Outgoing: make(chan []byte, 10),
	}

	hub.Register(sender)
	hub.Register(receiver)

	hub.Broadcast([]byte("hello"), sender)

	select {
	case msg := <-receiver.Outgoing:
		if string(msg) != "hello" {
			t.Errorf("Broadcast() got %q, want %q", string(msg), "hello")
		}
	default:
		t.Error("Broadcast() receiver did not get message")
	}

	select {
	case <-sender.Outgoing:
		t.Error("Broadcast() sender should not receive own message")
	default:
	}
}

func TestHub_Broadcast_MultipleReceivers(t *testing.T) {
	hub := chat.NewHub()
	sender := &chat.Client{
		Conn:     &mockConn{remoteAddr: "127.0.0.1:1234"},
		Username: "sender",
		Outgoing: make(chan []byte, 10),
	}

	receivers := make([]*chat.Client, 3)
	for i := range receivers {
		receivers[i] = &chat.Client{
			Conn:     &mockConn{remoteAddr: "127.0.0.1:5678"},
			Username: "receiver",
			Outgoing: make(chan []byte, 10),
		}
		hub.Register(receivers[i])
	}
	hub.Register(sender)

	hub.Broadcast([]byte("hello all"), sender)

	for i, receiver := range receivers {
		select {
		case msg := <-receiver.Outgoing:
			if string(msg) != "hello all" {
				t.Errorf("receiver[%d] got %q, want %q", i, string(msg), "hello all")
			}
		default:
			t.Errorf("receiver[%d] did not get message", i)
		}
	}
}

func TestHub_HandleClient_BroadcastsJoinMessage(t *testing.T) {
	hub := chat.NewHub()

	senderConn := newMockConn("127.0.0.1:1234")
	sender := &chat.Client{
		Conn:     senderConn,
		Username: "",
		Outgoing: make(chan []byte, 10),
	}

	receiverConn := newMockConn("127.0.0.1:5678")
	receiver := &chat.Client{
		Conn:     receiverConn,
		Username: "receiver",
		Outgoing: make(chan []byte, 10),
	}

	hub.Register(sender)
	hub.Register(receiver)

	done := make(chan struct{})
	go func() {
		hub.HandleClient(sender)
		close(done)
	}()

	joinMsg := createJoinMessage(t, "sender")
	senderConn.readCh <- joinMsg
	close(senderConn.readCh)

	select {
	case msg := <-receiver.Outgoing:
		var decoded protocol.Message
		if err := decoded.Decode(msg); err != nil {
			t.Fatalf("failed to decode broadcast message: %v", err)
		}
		if decoded.Type != protocol.MessageTypeJoin {
			t.Errorf("expected JOIN message, got %v", decoded.Type)
		}
		if decoded.Sender != "sender" {
			t.Errorf("expected sender 'sender', got %q", decoded.Sender)
		}
	case <-done:
		t.Error("HandleClient finished before broadcasting")
	}

	<-done

	if sender.Username != "sender" {
		t.Errorf("expected username 'sender', got %q", sender.Username)
	}
}

func TestHub_HandleClient_BroadcastsTextMessage(t *testing.T) {
	hub := chat.NewHub()

	senderConn := newMockConn("127.0.0.1:1234")
	sender := &chat.Client{
		Conn:     senderConn,
		Username: "sender",
		Outgoing: make(chan []byte, 10),
	}

	receiverConn := newMockConn("127.0.0.1:5678")
	receiver := &chat.Client{
		Conn:     receiverConn,
		Username: "receiver",
		Outgoing: make(chan []byte, 10),
	}

	hub.Register(sender)
	hub.Register(receiver)

	done := make(chan struct{})
	go func() {
		hub.HandleClient(sender)
		close(done)
	}()

	textMsg := createTextMessage(t, "sender", "Hello, World!")
	senderConn.readCh <- textMsg
	close(senderConn.readCh)

	select {
	case msg := <-receiver.Outgoing:
		var decoded protocol.Message
		if err := decoded.Decode(msg); err != nil {
			t.Fatalf("failed to decode broadcast message: %v", err)
		}
		if decoded.Type != protocol.MessageTypeText {
			t.Errorf("expected TEXT message, got %v", decoded.Type)
		}
		if decoded.Content != "Hello, World!" {
			t.Errorf("expected content 'Hello, World!', got %q", decoded.Content)
		}
	case <-done:
		t.Error("HandleClient finished before broadcasting")
	}

	<-done
}

func TestHub_HandleClient_UnregistersOnDisconnect(t *testing.T) {
	hub := chat.NewHub()

	conn := newMockConn("127.0.0.1:1234")
	client := &chat.Client{
		Conn:     conn,
		Username: "testuser",
		Outgoing: make(chan []byte, 10),
	}

	hub.Register(client)

	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount())
	}

	done := make(chan struct{})
	go func() {
		hub.HandleClient(client)
		close(done)
	}()

	close(conn.readCh)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("HandleClient did not finish")
	}

	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", hub.ClientCount())
	}
}

func TestHub_Stop(t *testing.T) {
	hub := chat.NewHub()

	conn1 := newMockConn("127.0.0.1:1234")
	client1 := &chat.Client{
		Conn:     conn1,
		Username: "user1",
		Outgoing: make(chan []byte, 10),
	}
	conn2 := newMockConn("127.0.0.1:5678")
	client2 := &chat.Client{
		Conn:     conn2,
		Username: "user2",
		Outgoing: make(chan []byte, 10),
	}

	hub.Register(client1)
	hub.Register(client2)

	if hub.ClientCount() != 2 {
		t.Fatalf("expected 2 clients, got %d", hub.ClientCount())
	}

	hub.Stop()

	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients after stop, got %d", hub.ClientCount())
	}

	if !conn1.closed {
		t.Error("expected conn1 to be closed")
	}
	if !conn2.closed {
		t.Error("expected conn2 to be closed")
	}
}
