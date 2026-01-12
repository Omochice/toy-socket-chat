package ws_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/omochice/toy-socket-chat/internal/chat"
	"github.com/omochice/toy-socket-chat/internal/transport/ws"
	"nhooyr.io/websocket"
)

func TestServer_Start(t *testing.T) {
	hub := chat.NewHub()
	srv := ws.New(":0", hub)

	go srv.Start()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	wsURL := "ws://" + srv.Addr()
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn.Close(websocket.StatusNormalClosure, "")
}

func TestServer_Addr(t *testing.T) {
	hub := chat.NewHub()
	srv := ws.New(":0", hub)

	go srv.Start()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	addr := srv.Addr()
	if addr == "" {
		t.Error("Addr() returned empty string")
	}
	if !strings.Contains(addr, ":") {
		t.Errorf("Addr() = %q, expected host:port format", addr)
	}
}

func TestServer_Stop(t *testing.T) {
	hub := chat.NewHub()
	srv := ws.New(":0", hub)

	go srv.Start()

	time.Sleep(100 * time.Millisecond)

	srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := websocket.Dial(ctx, "ws://"+srv.Addr(), nil)
	if err == nil {
		t.Error("expected error after stop, got nil")
	}
}

func TestServer_ClientRegistration(t *testing.T) {
	hub := chat.NewHub()
	srv := ws.New(":0", hub)

	go srv.Start()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	wsURL := "ws://" + srv.Addr()
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("expected 1 client in hub, got %d", hub.ClientCount())
	}
}

func TestServer_MultipleClients(t *testing.T) {
	hub := chat.NewHub()
	srv := ws.New(":0", hub)

	go srv.Start()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	wsURL := "ws://" + srv.Addr()
	conns := make([]*websocket.Conn, 3)
	for i := range conns {
		conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		conns[i] = conn
	}
	defer func() {
		for _, conn := range conns {
			conn.Close(websocket.StatusNormalClosure, "")
		}
	}()

	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 3 {
		t.Errorf("expected 3 clients in hub, got %d", hub.ClientCount())
	}
}
