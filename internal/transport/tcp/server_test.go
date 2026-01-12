package tcp_test

import (
	"net"
	"testing"
	"time"

	"github.com/omochice/toy-socket-chat/internal/chat"
	"github.com/omochice/toy-socket-chat/internal/transport/tcp"
)

func TestServer_Start(t *testing.T) {
	hub := chat.NewHub()
	srv := tcp.New(":0", hub)

	go srv.Start()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn.Close()
}

func TestServer_Addr(t *testing.T) {
	hub := chat.NewHub()
	srv := tcp.New(":0", hub)

	go srv.Start()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	addr := srv.Addr()
	if addr == "" {
		t.Error("Addr() returned empty string")
	}
}

func TestServer_Stop(t *testing.T) {
	hub := chat.NewHub()
	srv := tcp.New(":0", hub)

	go srv.Start()

	time.Sleep(100 * time.Millisecond)

	srv.Stop()

	_, err := net.Dial("tcp", srv.Addr())
	if err == nil {
		t.Error("expected error after stop, got nil")
	}
}
