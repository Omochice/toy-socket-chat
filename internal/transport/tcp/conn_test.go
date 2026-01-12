package tcp_test

import (
	"context"
	"net"
	"testing"

	"github.com/omochice/toy-socket-chat/internal/chat"
	"github.com/omochice/toy-socket-chat/internal/transport/tcp"
)

func TestConn_ImplementsInterface(t *testing.T) {
	var _ chat.Conn = (*tcp.Conn)(nil)
}

func TestConn_Read(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := tcp.NewConn(client)

	go func() {
		server.Write([]byte("test message"))
		server.Close()
	}()

	data, err := conn.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "test message" {
		t.Errorf("Read() = %q, want %q", string(data), "test message")
	}
}

func TestConn_Write(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := tcp.NewConn(client)

	go func() {
		err := conn.Write(context.Background(), []byte("hello"))
		if err != nil {
			t.Errorf("Write() error = %v", err)
		}
	}()

	buf := make([]byte, 1024)
	n, err := server.Read(buf)
	if err != nil {
		t.Fatalf("server read error: %v", err)
	}
	if string(buf[:n]) != "hello" {
		t.Errorf("server received %q, want %q", string(buf[:n]), "hello")
	}
}

func TestConn_Close(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	conn := tcp.NewConn(client)

	err := conn.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	_, err = client.Read(make([]byte, 1))
	if err == nil {
		t.Error("expected error after close, got nil")
	}
}

func TestConn_RemoteAddr(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := tcp.NewConn(client)

	addr := conn.RemoteAddr()
	if addr == "" {
		t.Error("RemoteAddr() returned empty string")
	}
}
