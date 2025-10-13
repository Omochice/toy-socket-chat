package server

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

func TestUnifiedServer_Start(t *testing.T) {
	srv := NewUnifiedServer(":0", ":0")
	if srv == nil {
		t.Fatal("NewUnifiedServer returned nil")
	}

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Check if server is running
	tcpAddr := srv.TCPAddr()
	wsAddr := srv.WSAddr()
	if tcpAddr == "" {
		t.Fatal("TCP server address is empty")
	}
	if wsAddr == "" {
		t.Fatal("WebSocket server address is empty")
	}

	// Stop server
	srv.Stop()

	// Wait for server to stop
	select {
	case err := <-errChan:
		if err != nil && !strings.Contains(err.Error(), "Server stopped") {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Server did not stop in time")
	}
}

func TestUnifiedServer_TCPClient(t *testing.T) {
	srv := NewUnifiedServer(":0", ":0")
	go srv.Start()
	defer srv.Stop()

	time.Sleep(200 * time.Millisecond)

	// Connect TCP client
	tcpAddr := srv.TCPAddr()
	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		t.Fatalf("Failed to connect TCP client: %v", err)
	}
	defer conn.Close()

	// Wait for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Check client count
	if count := srv.ClientCount(); count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}
}

func TestUnifiedServer_WebSocketClient(t *testing.T) {
	srv := NewUnifiedServer(":0", ":0")
	go srv.Start()
	defer srv.Stop()

	time.Sleep(200 * time.Millisecond)

	// Connect WebSocket client
	wsAddr := srv.WSAddr()
	url := "ws://" + wsAddr + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket client: %v", err)
	}
	defer conn.Close()

	// Wait for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Check client count
	if count := srv.ClientCount(); count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}
}

func TestUnifiedServer_MixedClients(t *testing.T) {
	srv := NewUnifiedServer(":0", ":0")
	go srv.Start()
	defer srv.Stop()

	time.Sleep(200 * time.Millisecond)

	// Connect TCP client
	tcpAddr := srv.TCPAddr()
	tcpConn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		t.Fatalf("Failed to connect TCP client: %v", err)
	}
	defer tcpConn.Close()

	// Connect WebSocket client
	wsAddr := srv.WSAddr()
	url := "ws://" + wsAddr + "/ws"
	wsConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket client: %v", err)
	}
	defer wsConn.Close()

	// Wait for connections to be registered
	time.Sleep(100 * time.Millisecond)

	// Check client count (should have both TCP and WebSocket clients)
	if count := srv.ClientCount(); count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}
}

func TestUnifiedServer_CrossProtocolBroadcast(t *testing.T) {
	srv := NewUnifiedServer(":0", ":0")
	go srv.Start()
	defer srv.Stop()

	time.Sleep(200 * time.Millisecond)

	tcpAddr := srv.TCPAddr()
	wsAddr := srv.WSAddr()

	// Connect TCP client
	tcpConn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		t.Fatalf("Failed to connect TCP client: %v", err)
	}
	defer tcpConn.Close()

	// Send join message from TCP client
	joinMsg := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "tcp-alice",
	}
	joinData, _ := joinMsg.Encode()
	tcpConn.Write(joinData)

	time.Sleep(100 * time.Millisecond)

	// Connect WebSocket client
	wsURL := "ws://" + wsAddr + "/ws"
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket client: %v", err)
	}
	defer wsConn.Close()

	// Send join message from WebSocket client
	joinMsg2 := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "ws-bob",
	}
	joinData2, _ := joinMsg2.Encode()
	wsConn.WriteMessage(websocket.BinaryMessage, joinData2)

	// TCP client should receive WebSocket client's join message
	tcpBuf := make([]byte, 4096)
	tcpConn.SetReadDeadline(time.Now().Add(time.Second))
	n, err := tcpConn.Read(tcpBuf)
	if err != nil {
		t.Fatalf("Failed to receive message on TCP client: %v", err)
	}

	var receivedMsg protocol.Message
	if err := receivedMsg.Decode(tcpBuf[:n]); err != nil {
		t.Fatalf("Failed to decode message: %v", err)
	}

	if receivedMsg.Type != protocol.MessageTypeJoin {
		t.Errorf("Expected JOIN message, got %v", receivedMsg.Type)
	}
	if receivedMsg.Sender != "ws-bob" {
		t.Errorf("Expected sender 'ws-bob', got '%s'", receivedMsg.Sender)
	}

	// Send text message from TCP client
	textMsg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "tcp-alice",
		Content: "Hello from TCP!",
	}
	textData, _ := textMsg.Encode()
	tcpConn.Write(textData)

	// WebSocket client should receive TCP client's message
	wsConn.SetReadDeadline(time.Now().Add(time.Second))
	_, wsData, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to receive message on WebSocket client: %v", err)
	}

	var wsReceivedMsg protocol.Message
	if err := wsReceivedMsg.Decode(wsData); err != nil {
		t.Fatalf("Failed to decode WebSocket message: %v", err)
	}

	if wsReceivedMsg.Type != protocol.MessageTypeText {
		t.Errorf("Expected TEXT message, got %v", wsReceivedMsg.Type)
	}
	if wsReceivedMsg.Content != "Hello from TCP!" {
		t.Errorf("Expected content 'Hello from TCP!', got '%s'", wsReceivedMsg.Content)
	}
}

func TestUnifiedServer_TCPToWebSocket(t *testing.T) {
	srv := NewUnifiedServer(":0", ":0")
	go srv.Start()
	defer srv.Stop()

	time.Sleep(200 * time.Millisecond)

	tcpAddr := srv.TCPAddr()
	wsAddr := srv.WSAddr()

	// Connect TCP client
	tcpConn, _ := net.Dial("tcp", tcpAddr)
	defer tcpConn.Close()

	// Send join
	joinMsg := protocol.Message{Type: protocol.MessageTypeJoin, Sender: "tcp-user"}
	joinData, _ := joinMsg.Encode()
	tcpConn.Write(joinData)

	time.Sleep(100 * time.Millisecond)

	// Connect WebSocket client
	wsConn, _, _ := websocket.DefaultDialer.Dial("ws://"+wsAddr+"/ws", nil)
	defer wsConn.Close()

	// Send join from WebSocket
	wsJoin := protocol.Message{Type: protocol.MessageTypeJoin, Sender: "ws-user"}
	wsJoinData, _ := wsJoin.Encode()
	wsConn.WriteMessage(websocket.BinaryMessage, wsJoinData)

	time.Sleep(100 * time.Millisecond)

	// Send message from TCP
	msg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "tcp-user",
		Content: "Message from TCP",
	}
	msgData, _ := msg.Encode()
	tcpConn.Write(msgData)

	// WebSocket should receive it
	wsConn.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("WebSocket failed to receive: %v", err)
	}

	var received protocol.Message
	received.Decode(data)
	if received.Content != "Message from TCP" {
		t.Errorf("Expected 'Message from TCP', got '%s'", received.Content)
	}
}

func TestUnifiedServer_WebSocketToTCP(t *testing.T) {
	srv := NewUnifiedServer(":0", ":0")
	go srv.Start()
	defer srv.Stop()

	time.Sleep(200 * time.Millisecond)

	tcpAddr := srv.TCPAddr()
	wsAddr := srv.WSAddr()

	// Connect WebSocket client
	wsConn, _, _ := websocket.DefaultDialer.Dial("ws://"+wsAddr+"/ws", nil)
	defer wsConn.Close()

	// Send join
	wsJoin := protocol.Message{Type: protocol.MessageTypeJoin, Sender: "ws-user"}
	wsJoinData, _ := wsJoin.Encode()
	wsConn.WriteMessage(websocket.BinaryMessage, wsJoinData)

	time.Sleep(100 * time.Millisecond)

	// Connect TCP client
	tcpConn, _ := net.Dial("tcp", tcpAddr)
	defer tcpConn.Close()

	// Send join from TCP
	tcpJoin := protocol.Message{Type: protocol.MessageTypeJoin, Sender: "tcp-user"}
	tcpJoinData, _ := tcpJoin.Encode()
	tcpConn.Write(tcpJoinData)

	time.Sleep(100 * time.Millisecond)

	// Send message from WebSocket
	msg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "ws-user",
		Content: "Message from WebSocket",
	}
	msgData, _ := msg.Encode()
	wsConn.WriteMessage(websocket.BinaryMessage, msgData)

	// TCP should receive it
	buf := make([]byte, 4096)
	tcpConn.SetReadDeadline(time.Now().Add(time.Second))
	n, err := tcpConn.Read(buf)
	if err != nil {
		t.Fatalf("TCP failed to receive: %v", err)
	}

	var received protocol.Message
	received.Decode(buf[:n])
	if received.Content != "Message from WebSocket" {
		t.Errorf("Expected 'Message from WebSocket', got '%s'", received.Content)
	}
}

func TestUnifiedServer_SinglePort(t *testing.T) {
	srv := NewUnifiedServer(":0", "")
	go srv.Start()
	defer srv.Stop()

	time.Sleep(200 * time.Millisecond)

	addr := srv.Addr()
	if addr == "" {
		t.Fatal("Server address is empty")
	}

	// Connect TCP client
	tcpConn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect TCP client: %v", err)
	}
	defer tcpConn.Close()

	// Send join message from TCP client
	joinMsg := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "tcp-user",
	}
	joinData, _ := joinMsg.Encode()
	tcpConn.Write(joinData)

	time.Sleep(100 * time.Millisecond)

	// Connect WebSocket client to the same port (no /ws path)
	wsURL := "ws://" + addr
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket client: %v", err)
	}
	defer wsConn.Close()

	// Send join message from WebSocket client
	wsJoin := protocol.Message{
		Type:   protocol.MessageTypeJoin,
		Sender: "ws-user",
	}
	wsJoinData, _ := wsJoin.Encode()
	wsConn.WriteMessage(websocket.BinaryMessage, wsJoinData)

	// TCP should receive the join notification
	buf := make([]byte, 4096)
	tcpConn.SetReadDeadline(time.Now().Add(time.Second))
	n, err := tcpConn.Read(buf)
	if err != nil {
		t.Fatalf("TCP failed to receive join: %v", err)
	}
	var joinReceived protocol.Message
	joinReceived.Decode(buf[:n])
	if joinReceived.Type != protocol.MessageTypeJoin || joinReceived.Sender != "ws-user" {
		t.Logf("Received join: type=%v, sender=%s", joinReceived.Type, joinReceived.Sender)
	}

	time.Sleep(100 * time.Millisecond)

	// Check client count (should have both clients)
	if count := srv.ClientCount(); count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}

	// Send message from TCP to WebSocket
	textMsg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "tcp-user",
		Content: "Hello from TCP!",
	}
	textData, _ := textMsg.Encode()
	tcpConn.Write(textData)

	// WebSocket should receive it
	wsConn.SetReadDeadline(time.Now().Add(time.Second))
	_, wsData, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("WebSocket failed to receive: %v", err)
	}

	var wsReceived protocol.Message
	wsReceived.Decode(wsData)
	if wsReceived.Content != "Hello from TCP!" {
		t.Errorf("Expected 'Hello from TCP!', got '%s'", wsReceived.Content)
	}

	// Send message from WebSocket to TCP
	wsMsg := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "ws-user",
		Content: "Hello from WebSocket!",
	}
	wsMsgData, _ := wsMsg.Encode()
	wsConn.WriteMessage(websocket.BinaryMessage, wsMsgData)

	// TCP should receive it
	tcpConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = tcpConn.Read(buf)
	if err != nil {
		t.Fatalf("TCP failed to receive: %v", err)
	}

	var tcpReceived protocol.Message
	if err := tcpReceived.Decode(buf[:n]); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}
	if tcpReceived.Content != "Hello from WebSocket!" {
		t.Errorf("Expected 'Hello from WebSocket!', got '%s'", tcpReceived.Content)
	}
}
