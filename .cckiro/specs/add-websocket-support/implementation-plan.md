# Implementation Plan: Add WebSocket Support

## Overview
This document outlines the step-by-step implementation plan for adding WebSocket support to the TCP socket chat application.

## Implementation Phases

### Phase 1: Dependencies and Interfaces

#### Task 1.1: Add nbio Dependency
**File**: `go.mod`

**Action**:
```bash
go get github.com/lesismal/nbio
```

**Verification**:
- Check that `go.mod` contains `github.com/lesismal/nbio` entry
- Run `go mod tidy` to ensure dependencies are clean
- Run `go build ./...` to verify no conflicts

**Dependencies**: None

---

#### Task 1.2: Create Server Connection Interface
**File**: `internal/server/connection.go` (new)

**Action**:
Create the `Connection` interface and basic types:

```go
package server

import (
    "net"
    "time"
)

// Connection represents a client connection (TCP or WebSocket)
type Connection interface {
    // RemoteAddr returns the remote address
    RemoteAddr() net.Addr

    // Write sends binary data to the client
    Write(data []byte) (int, error)

    // Read receives binary data from the client
    Read(buf []byte) (int, error)

    // Close closes the connection
    Close() error

    // SetReadDeadline sets the read deadline
    SetReadDeadline(t time.Time) error
}
```

**Verification**:
- File compiles without errors
- Interface is properly defined

**Dependencies**: Task 1.1

---

#### Task 1.3: Create Client Connection Interface
**File**: `internal/client/connection.go` (new)

**Action**:
Create the `ClientConnection` interface:

```go
package client

import "net"

// ClientConnection represents a connection to the server
type ClientConnection interface {
    // Write sends data to the server
    Write(data []byte) (int, error)

    // Read receives data from the server
    Read(buf []byte) (int, error)

    // Close closes the connection
    Close() error

    // RemoteAddr returns the server address
    RemoteAddr() net.Addr
}
```

**Verification**:
- File compiles without errors
- Interface is properly defined

**Dependencies**: None

---

### Phase 2: Server Implementation

#### Task 2.1: Implement Protocol Detection
**File**: `internal/server/protocol.go` (new)

**Action**:
Implement protocol detection logic:

```go
package server

import (
    "bufio"
    "bytes"
    "net"
)

type protocolType int

const (
    protocolTCP protocolType = iota
    protocolHTTP
)

// detectProtocol peeks at the first bytes to determine protocol type
func detectProtocol(conn net.Conn) (protocolType, *bufio.Reader, error) {
    reader := bufio.NewReader(conn)

    // Peek first 4 bytes
    peek, err := reader.Peek(4)
    if err != nil {
        return protocolTCP, reader, err
    }

    // Check for HTTP methods
    if bytes.HasPrefix(peek, []byte("GET ")) ||
       bytes.HasPrefix(peek, []byte("POST")) ||
       bytes.HasPrefix(peek, []byte("PUT ")) ||
       bytes.HasPrefix(peek, []byte("HEAD")) {
        return protocolHTTP, reader, nil
    }

    // Default to TCP
    return protocolTCP, reader, nil
}
```

**Verification**:
- Unit test with HTTP request data → returns `protocolHTTP`
- Unit test with binary protobuf data → returns `protocolTCP`
- Unit test with short/empty data → handles gracefully

**Dependencies**: None

---

#### Task 2.2: Implement TCPConnection
**File**: `internal/server/connection.go` (modify)

**Action**:
Add `TCPConnection` implementation:

```go
// TCPConnection wraps a net.Conn for TCP connections
type TCPConnection struct {
    conn net.Conn
}

// NewTCPConnection creates a new TCPConnection
func NewTCPConnection(conn net.Conn) *TCPConnection {
    return &TCPConnection{conn: conn}
}

func (tc *TCPConnection) RemoteAddr() net.Addr {
    return tc.conn.RemoteAddr()
}

func (tc *TCPConnection) Write(data []byte) (int, error) {
    return tc.conn.Write(data)
}

func (tc *TCPConnection) Read(buf []byte) (int, error) {
    return tc.conn.Read(buf)
}

func (tc *TCPConnection) Close() error {
    return tc.conn.Close()
}

func (tc *TCPConnection) SetReadDeadline(t time.Time) error {
    return tc.conn.SetReadDeadline(t)
}
```

**Verification**:
- Unit test: Create mock net.Conn, verify all methods delegate correctly
- Test Read/Write operations
- Test Close cleanup

**Dependencies**: Task 1.2

---

#### Task 2.3: Implement WebSocketConnection
**File**: `internal/server/connection.go` (modify)

**Action**:
Add `WebSocketConnection` implementation:

```go
import (
    "fmt"
    "github.com/lesismal/nbio/nbhttp/websocket"
)

// WebSocketConnection wraps a websocket.Conn for WebSocket connections
type WebSocketConnection struct {
    conn          *websocket.Conn
    readBuffer    []byte
    readBufferPos int
    mu            sync.Mutex
}

// NewWebSocketConnection creates a new WebSocketConnection
func NewWebSocketConnection(conn *websocket.Conn) *WebSocketConnection {
    return &WebSocketConnection{conn: conn}
}

func (wc *WebSocketConnection) RemoteAddr() net.Addr {
    return wc.conn.RemoteAddr()
}

func (wc *WebSocketConnection) Write(data []byte) (int, error) {
    err := wc.conn.WriteMessage(websocket.BinaryMessage, data)
    if err != nil {
        return 0, err
    }
    return len(data), nil
}

func (wc *WebSocketConnection) Read(buf []byte) (int, error) {
    wc.mu.Lock()
    defer wc.mu.Unlock()

    // Return buffered data if available
    if wc.readBufferPos < len(wc.readBuffer) {
        n := copy(buf, wc.readBuffer[wc.readBufferPos:])
        wc.readBufferPos += n
        if wc.readBufferPos >= len(wc.readBuffer) {
            wc.readBuffer = nil
            wc.readBufferPos = 0
        }
        return n, nil
    }

    // Read next WebSocket message
    messageType, data, err := wc.conn.ReadMessage()
    if err != nil {
        return 0, err
    }

    // Only accept binary messages
    if messageType != websocket.BinaryMessage {
        return 0, fmt.Errorf("expected binary message, got %v", messageType)
    }

    // Copy to output buffer
    n := copy(buf, data)
    if n < len(data) {
        // Buffer remaining data
        wc.readBuffer = data[n:]
        wc.readBufferPos = 0
    }

    return n, nil
}

func (wc *WebSocketConnection) Close() error {
    return wc.conn.Close()
}

func (wc *WebSocketConnection) SetReadDeadline(t time.Time) error {
    // WebSocket doesn't support read deadlines in the same way
    return nil
}
```

**Verification**:
- Unit test: Mock websocket.Conn
- Test binary message read/write
- Test message buffering for partial reads
- Test rejection of non-binary messages

**Dependencies**: Task 1.1, Task 1.2

---

#### Task 2.4: Implement WebSocket Upgrade Handler
**File**: `internal/server/websocket.go` (new)

**Action**:
Implement manual WebSocket upgrade:

```go
package server

import (
    "bufio"
    "crypto/sha1"
    "encoding/base64"
    "fmt"
    "net"
    "net/http"
    "strings"

    "github.com/lesismal/nbio/nbhttp/websocket"
)

// upgradeWebSocket performs WebSocket handshake and returns WebSocket connection
func (s *Server) upgradeWebSocket(rawConn net.Conn, reader *bufio.Reader) (Connection, error) {
    // Parse HTTP request
    req, err := http.ReadRequest(reader)
    if err != nil {
        return nil, fmt.Errorf("failed to read HTTP request: %w", err)
    }

    // Validate WebSocket upgrade request
    if !isWebSocketUpgrade(req) {
        return nil, fmt.Errorf("not a WebSocket upgrade request")
    }

    // Compute accept key
    key := req.Header.Get("Sec-WebSocket-Key")
    acceptKey := computeAcceptKey(key)

    // Send upgrade response
    response := fmt.Sprintf(
        "HTTP/1.1 101 Switching Protocols\r\n"+
        "Upgrade: websocket\r\n"+
        "Connection: Upgrade\r\n"+
        "Sec-WebSocket-Accept: %s\r\n"+
        "\r\n",
        acceptKey,
    )

    if _, err := rawConn.Write([]byte(response)); err != nil {
        return nil, fmt.Errorf("failed to write upgrade response: %w", err)
    }

    // Create WebSocket connection using nbio
    wsConn := websocket.NewConn(rawConn, true, 1024, 1024)

    return NewWebSocketConnection(wsConn), nil
}

func isWebSocketUpgrade(req *http.Request) bool {
    return req.Method == "GET" &&
        strings.ToLower(req.Header.Get("Upgrade")) == "websocket" &&
        strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

func computeAcceptKey(key string) string {
    const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
    h := sha1.New()
    h.Write([]byte(key))
    h.Write([]byte(websocketGUID))
    return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
```

**Verification**:
- Unit test: Valid WebSocket upgrade request → success
- Unit test: Invalid request → error
- Unit test: Missing headers → error
- Test accept key computation against known values

**Dependencies**: Task 1.1, Task 2.3

---

#### Task 2.5: Modify Server Structure
**File**: `internal/server/server.go` (modify)

**Action**:
1. Update `Client` struct to use `Connection` interface:
```go
type Client struct {
    conn     Connection  // Changed from net.Conn
    username string
    outgoing chan []byte
}
```

2. Modify `Start()` method to delegate to `handleConnection`:
```go
func (s *Server) Start() error {
    listener, err := net.Listen("tcp", s.address)
    if err != nil {
        return fmt.Errorf("failed to start server: %w", err)
    }
    s.listener = listener

    log.Printf("Server started on %s", listener.Addr().String())

    for {
        select {
        case <-s.quit:
            return fmt.Errorf("server stopped")
        default:
            conn, err := listener.Accept()
            if err != nil {
                select {
                case <-s.quit:
                    return fmt.Errorf("server stopped")
                default:
                    log.Printf("Failed to accept connection: %v", err)
                    continue
                }
            }

            // Handle connection in goroutine
            go s.handleConnection(conn)
        }
    }
}
```

3. Add new `handleConnection` method:
```go
func (s *Server) handleConnection(rawConn net.Conn) {
    // Detect protocol
    protocol, reader, err := detectProtocol(rawConn)
    if err != nil {
        log.Printf("Protocol detection failed: %v", err)
        rawConn.Close()
        return
    }

    var conn Connection

    switch protocol {
    case protocolHTTP:
        // WebSocket upgrade
        conn, err = s.upgradeWebSocket(rawConn, reader)
        if err != nil {
            log.Printf("WebSocket upgrade failed: %v", err)
            rawConn.Close()
            return
        }
        log.Printf("WebSocket connection from %s", conn.RemoteAddr())

    case protocolTCP:
        // Wrap as TCP connection
        conn = NewTCPConnection(rawConn)
        log.Printf("TCP connection from %s", conn.RemoteAddr())
    }

    // Create client
    client := &Client{
        conn:     conn,
        outgoing: make(chan []byte, 10),
    }

    s.mu.Lock()
    s.clients[client] = true
    s.mu.Unlock()

    s.wg.Add(1)
    go s.handleClient(client)
}
```

**Verification**:
- Existing tests still pass (TCP connections)
- Manual test: TCP client can connect
- Manual test: WebSocket client can connect
- Both connection types appear in client list

**Dependencies**: Task 2.1, Task 2.2, Task 2.3, Task 2.4

---

### Phase 3: Client Implementation

#### Task 3.1: Implement TCPClientConnection
**File**: `internal/client/connection.go` (modify)

**Action**:
```go
package client

import "net"

// TCPClientConnection wraps net.Conn for TCP connections
type TCPClientConnection struct {
    conn net.Conn
}

// NewTCPClientConnection creates a new TCP connection wrapper
func NewTCPClientConnection(conn net.Conn) *TCPClientConnection {
    return &TCPClientConnection{conn: conn}
}

func (tc *TCPClientConnection) Write(data []byte) (int, error) {
    return tc.conn.Write(data)
}

func (tc *TCPClientConnection) Read(buf []byte) (int, error) {
    return tc.conn.Read(buf)
}

func (tc *TCPClientConnection) Close() error {
    return tc.conn.Close()
}

func (tc *TCPClientConnection) RemoteAddr() net.Addr {
    return tc.conn.RemoteAddr()
}
```

**Verification**:
- Unit test with mock net.Conn
- Verify all methods delegate correctly

**Dependencies**: Task 1.3

---

#### Task 3.2: Implement WebSocketClientConnection
**File**: `internal/client/connection.go` (modify)

**Action**:
```go
import (
    "fmt"
    "sync"
    "github.com/lesismal/nbio/nbhttp/websocket"
)

// WebSocketClientConnection wraps websocket.Conn for WebSocket connections
type WebSocketClientConnection struct {
    conn          *websocket.Conn
    readBuffer    []byte
    readBufferPos int
    mu            sync.Mutex
}

// NewWebSocketClientConnection creates a new WebSocket connection wrapper
func NewWebSocketClientConnection(conn *websocket.Conn) *WebSocketClientConnection {
    return &WebSocketClientConnection{conn: conn}
}

func (wc *WebSocketClientConnection) Write(data []byte) (int, error) {
    err := wc.conn.WriteMessage(websocket.BinaryMessage, data)
    if err != nil {
        return 0, err
    }
    return len(data), nil
}

func (wc *WebSocketClientConnection) Read(buf []byte) (int, error) {
    wc.mu.Lock()
    defer wc.mu.Unlock()

    // Return buffered data if available
    if wc.readBufferPos < len(wc.readBuffer) {
        n := copy(buf, wc.readBuffer[wc.readBufferPos:])
        wc.readBufferPos += n
        if wc.readBufferPos >= len(wc.readBuffer) {
            wc.readBuffer = nil
            wc.readBufferPos = 0
        }
        return n, nil
    }

    // Read next message
    messageType, data, err := wc.conn.ReadMessage()
    if err != nil {
        return 0, err
    }

    if messageType != websocket.BinaryMessage {
        return 0, fmt.Errorf("expected binary message, got %v", messageType)
    }

    n := copy(buf, data)
    if n < len(data) {
        wc.readBuffer = data[n:]
        wc.readBufferPos = 0
    }

    return n, nil
}

func (wc *WebSocketClientConnection) Close() error {
    // Send close frame
    wc.conn.WriteMessage(websocket.CloseMessage,
        websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
    return wc.conn.Close()
}

func (wc *WebSocketClientConnection) RemoteAddr() net.Addr {
    return wc.conn.RemoteAddr()
}
```

**Verification**:
- Unit test with mock websocket.Conn
- Test message buffering
- Test close handshake

**Dependencies**: Task 1.1, Task 1.3

---

#### Task 3.3: Modify Client Structure
**File**: `internal/client/client.go` (modify)

**Action**:
1. Update `Client` struct:
```go
type Client struct {
    address  string
    username string
    protocol string            // NEW: "tcp" or "ws"
    conn     ClientConnection  // Changed from net.Conn
    messages chan protocol.Message
    mu       sync.RWMutex
    done     chan struct{}
    wg       sync.WaitGroup
}
```

2. Update `New()` function:
```go
func New(address, username, protocol string) *Client {
    return &Client{
        address:  address,
        username: username,
        protocol: protocol,  // NEW
        messages: make(chan protocol.Message, 10),
        done:     make(chan struct{}),
    }
}
```

3. Modify `Connect()` method:
```go
func (c *Client) Connect() error {
    var conn ClientConnection
    var err error

    switch c.protocol {
    case "ws":
        conn, err = c.connectWebSocket()
    case "tcp":
        fallthrough
    default:
        conn, err = c.connectTCP()
    }

    if err != nil {
        return err
    }

    c.mu.Lock()
    c.conn = conn
    c.mu.Unlock()

    // Start receiving messages
    c.wg.Add(1)
    go c.receiveMessages()

    return nil
}
```

4. Add helper methods:
```go
func (c *Client) connectTCP() (ClientConnection, error) {
    conn, err := net.Dial("tcp", c.address)
    if err != nil {
        return nil, fmt.Errorf("failed to connect via TCP: %w", err)
    }
    return NewTCPClientConnection(conn), nil
}

func (c *Client) connectWebSocket() (ClientConnection, error) {
    url := fmt.Sprintf("ws://%s/", c.address)

    dialer := &websocket.Dialer{
        Timeout: 10 * time.Second,
    }

    conn, _, err := dialer.Dial(url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to connect via WebSocket: %w", err)
    }

    return NewWebSocketClientConnection(conn), nil
}
```

**Verification**:
- Existing unit tests pass (with protocol="tcp")
- New test: Create client with protocol="ws"
- Test both connection methods work

**Dependencies**: Task 3.1, Task 3.2

---

#### Task 3.4: Update Client CLI
**File**: `cmd/client/main.go` (modify)

**Action**:
1. Add protocol flag:
```go
func main() {
    serverAddr := flag.String("server", "localhost:8080", "Server address (e.g., localhost:8080)")
    username := flag.String("username", "", "Username for chat")
    protocol := flag.String("protocol", "tcp", "Protocol to use (tcp or ws)")
    flag.Parse()

    if *username == "" {
        log.Fatal("Username is required. Use -username flag")
    }

    // Validate protocol
    if *protocol != "tcp" && *protocol != "ws" {
        log.Fatalf("Invalid protocol: %s. Use 'tcp' or 'ws'", *protocol)
    }

    // Create client with protocol
    c := client.New(*serverAddr, *username, *protocol)

    // ... rest remains the same
}
```

**Verification**:
- Test with `-protocol tcp` → connects via TCP
- Test with `-protocol ws` → connects via WebSocket
- Test with `-protocol invalid` → error message
- Test with no `-protocol` flag → defaults to TCP

**Dependencies**: Task 3.3

---

### Phase 4: Testing and Integration

#### Task 4.1: Update Existing Unit Tests
**Files**:
- `internal/server/server_test.go`
- `internal/client/client_test.go`

**Action**:
- Review all existing tests
- Update client creation to pass "tcp" protocol
- Ensure all tests still pass
- Add protocol detection tests

**Verification**:
```bash
go test ./internal/server/...
go test ./internal/client/...
go test ./pkg/protocol/...
```

**Dependencies**: All previous tasks

---

#### Task 4.2: Add WebSocket Unit Tests
**Files**:
- `internal/server/connection_test.go` (new)
- `internal/client/connection_test.go` (new)
- `internal/server/protocol_test.go` (new)

**Action**:
Create tests for:
1. Protocol detection
2. WebSocketConnection read/write
3. WebSocketClientConnection read/write
4. WebSocket upgrade process

**Verification**:
All new tests pass

**Dependencies**: Task 4.1

---

#### Task 4.3: Add Integration Tests
**File**: `test/websocket_integration_test.go` (new)

**Action**:
Create integration tests:
1. WebSocket client → server → receive message
2. TCP client + WebSocket client → both receive messages
3. Multiple WebSocket clients
4. Join/leave notifications across protocols

**Example**:
```go
func TestWebSocketIntegration(t *testing.T) {
    // Start server
    srv := server.New(":0")
    go srv.Start()
    defer srv.Stop()

    addr := srv.Addr()

    // Connect WebSocket client
    wsClient := client.New(addr, "wsuser", "ws")
    err := wsClient.Connect()
    require.NoError(t, err)
    defer wsClient.Disconnect()

    // Connect TCP client
    tcpClient := client.New(addr, "tcpuser", "tcp")
    err = tcpClient.Connect()
    require.NoError(t, err)
    defer tcpClient.Disconnect()

    // Test cross-protocol messaging
    // ...
}
```

**Verification**:
```bash
go test ./test/...
```

**Dependencies**: Task 4.2

---

#### Task 4.4: Build and Manual Testing
**Action**:
1. Build both binaries:
```bash
devbox run build
```

2. Manual test scenarios:

**Scenario 1: TCP client only**
```bash
# Terminal 1
./build/server -port :8080

# Terminal 2
./build/client -server localhost:8080 -username alice -protocol tcp
```

**Scenario 2: WebSocket client only**
```bash
# Terminal 1
./build/server -port :8080

# Terminal 2
./build/client -server localhost:8080 -username bob -protocol ws
```

**Scenario 3: Mixed TCP + WebSocket**
```bash
# Terminal 1
./build/server -port :8080

# Terminal 2
./build/client -server localhost:8080 -username alice -protocol tcp

# Terminal 3
./build/client -server localhost:8080 -username bob -protocol ws

# Verify both can send and receive messages from each other
```

**Scenario 4: Default behavior (backward compatibility)**
```bash
# Terminal 1
./build/server -port :8080

# Terminal 2 (no -protocol flag)
./build/client -server localhost:8080 -username charlie
```

**Verification Checklist**:
- [ ] TCP client connects successfully
- [ ] WebSocket client connects successfully
- [ ] Messages from TCP client reach WebSocket client
- [ ] Messages from WebSocket client reach TCP client
- [ ] Join notifications work cross-protocol
- [ ] Leave notifications work cross-protocol
- [ ] Multiple clients of each type work
- [ ] Default protocol is TCP (backward compatible)
- [ ] Server logs show correct protocol detection
- [ ] Graceful shutdown works for both protocols

**Dependencies**: Task 4.3

---

## Testing Strategy Summary

### Unit Tests
- Protocol detection logic
- Connection interface implementations
- WebSocket upgrade process
- Message encoding/decoding (existing)

### Integration Tests
- End-to-end WebSocket connection
- Cross-protocol message delivery
- Multiple client scenarios
- Join/leave notifications

### Manual Tests
- Real client/server interaction
- User experience verification
- Performance observation
- Error handling verification

## Rollback Plan

If issues are encountered during implementation:

1. **After Phase 1**: No rollback needed (only added dependencies and interfaces)
2. **After Phase 2**: Revert server changes, keep TCP-only functionality
3. **After Phase 3**: Revert client changes, keep TCP-only functionality
4. **After Phase 4**: Review test failures, fix incrementally

## Success Criteria

Implementation is complete when:

1. ✅ All existing tests pass
2. ✅ New WebSocket tests pass
3. ✅ Integration tests pass
4. ✅ Manual test scenarios all succeed
5. ✅ Build completes without errors or warnings
6. ✅ Backward compatibility verified (old TCP clients work)
7. ✅ Cross-protocol messaging works
8. ✅ Code follows existing patterns and style
9. ✅ Error handling is comprehensive
10. ✅ Logging is informative

## Estimated Implementation Time

- Phase 1: 30 minutes
- Phase 2: 2-3 hours
- Phase 3: 1-2 hours
- Phase 4: 1-2 hours
- **Total**: 5-8 hours

## Notes

- Implement one task at a time
- Run tests after each task
- Commit after each phase
- Document any deviations from the plan
- Update this plan if requirements change
