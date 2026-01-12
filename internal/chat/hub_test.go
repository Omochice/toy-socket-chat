package chat_test

import (
	"testing"

	"github.com/omochice/toy-socket-chat/internal/chat"
)

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
