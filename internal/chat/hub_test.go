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
