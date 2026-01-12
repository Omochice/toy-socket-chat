package ws_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/omochice/toy-socket-chat/internal/transport/ws"
	"nhooyr.io/websocket"
)

func TestConn_Read(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("failed to accept websocket: %v", err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		err = c.Write(context.Background(), websocket.MessageBinary, []byte("test message"))
		if err != nil {
			t.Fatalf("failed to write: %v", err)
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsConn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	conn := ws.NewConn(wsConn)

	data, err := conn.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "test message" {
		t.Errorf("Read() = %q, want %q", string(data), "test message")
	}
}

func TestConn_Write(t *testing.T) {
	received := make(chan []byte, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("failed to accept websocket: %v", err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		_, data, err := c.Read(context.Background())
		if err != nil {
			t.Fatalf("failed to read: %v", err)
		}
		received <- data
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsConn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	conn := ws.NewConn(wsConn)

	err = conn.Write(context.Background(), []byte("hello"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	data := <-received
	if string(data) != "hello" {
		t.Errorf("server received %q, want %q", string(data), "hello")
	}
}

func TestConn_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		c.Read(context.Background())
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsConn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	conn := ws.NewConn(wsConn)

	err = conn.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestConn_RemoteAddr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		c.Read(context.Background())
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsConn, resp, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	conn := ws.NewConnWithAddr(wsConn, resp.Request.URL.Host)

	addr := conn.RemoteAddr()
	if addr == "" {
		t.Error("RemoteAddr() returned empty string")
	}
}
