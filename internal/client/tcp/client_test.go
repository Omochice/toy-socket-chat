package tcp_test

import (
	"net"
	"testing"
	"time"

	"github.com/omochice/toy-socket-chat/internal/client/tcp"
)

func startMockServer(t *testing.T) (string, func()) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				go func(c net.Conn) {
					defer c.Close()
					buf := make([]byte, 4096)
					for {
						n, err := c.Read(buf)
						if err != nil {
							return
						}
						if n > 0 {
							c.Write(buf[:n])
						}
					}
				}(conn)
			}
		}
	}()

	cleanup := func() {
		close(done)
		listener.Close()
	}

	return listener.Addr().String(), cleanup
}

func TestClient_Connect(t *testing.T) {
	addr, cleanup := startMockServer(t)
	defer cleanup()

	c := tcp.New(addr, "testuser")
	err := c.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !c.IsConnected() {
		t.Error("Client should be connected")
	}

	c.Disconnect()

	if c.IsConnected() {
		t.Error("Client should be disconnected")
	}
}

func TestClient_SendMessage(t *testing.T) {
	addr, cleanup := startMockServer(t)
	defer cleanup()

	c := tcp.New(addr, "testuser")
	err := c.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect()

	err = c.SendMessage("Hello, World!")
	if err != nil {
		t.Errorf("Failed to send message: %v", err)
	}
}

func TestClient_ReceiveMessage(t *testing.T) {
	addr, cleanup := startMockServer(t)
	defer cleanup()

	c := tcp.New(addr, "testuser")
	err := c.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect()

	msgChan := c.Messages()
	testMsg := "Test message"
	err = c.SendMessage(testMsg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for echo
	select {
	case msg := <-msgChan:
		if msg.Content != testMsg {
			t.Errorf("Expected message %q, got %q", testMsg, msg.Content)
		}
		if msg.Sender != "testuser" {
			t.Errorf("Expected sender %q, got %q", "testuser", msg.Sender)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestClient_SendWithoutConnection(t *testing.T) {
	c := tcp.New("localhost:9999", "testuser")

	// Try to send without connecting
	err := c.SendMessage("This should fail")
	if err == nil {
		t.Error("Expected error when sending without connection")
	}
}

func TestClient_Join(t *testing.T) {
	addr, cleanup := startMockServer(t)
	defer cleanup()

	c := tcp.New(addr, "testuser")
	err := c.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect()

	err = c.Join()
	if err != nil {
		t.Errorf("Failed to send join message: %v", err)
	}
}

func TestClient_Leave(t *testing.T) {
	addr, cleanup := startMockServer(t)
	defer cleanup()

	c := tcp.New(addr, "testuser")
	err := c.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	err = c.Leave()
	if err != nil {
		t.Errorf("Failed to send leave message: %v", err)
	}

	c.Disconnect()
}
