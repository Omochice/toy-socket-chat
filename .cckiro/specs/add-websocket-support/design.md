# Design: Add WebSocket Support

## Overview
This document describes the design for adding WebSocket support to the TCP socket chat application while maintaining the exact same entry point for both protocols.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Server                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │            net.Listener (:8080)                       │  │
│  └─────────────┬─────────────────────────────────────────┘  │
│                │                                              │
│                ▼                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │      Protocol Detection (peek first bytes)           │    │
│  └──────────┬──────────────────────────┬─────────────────┘  │
│             │                           │                     │
│    HTTP Request?                  Binary data?                │
│             │                           │                     │
│             ▼                           ▼                     │
│  ┌──────────────────┐      ┌──────────────────────┐         │
│  │  WebSocket Path  │      │   TCP Socket Path    │         │
│  │  (nbio upgrader) │      │  (existing handler)  │         │
│  └──────────────────┘      └──────────────────────┘         │
│             │                           │                     │
│             └───────────┬───────────────┘                     │
│                         ▼                                     │
│            ┌─────────────────────┐                           │
│            │  Unified Client      │                           │
│            │  Management          │                           │
│            │  (broadcast, etc.)   │                           │
│            └─────────────────────┘                           │
└─────────────────────────────────────────────────────────────┘

┌──────────────┐                              ┌──────────────┐
│ TCP Client   │                              │  WS Client   │
│ (net.Conn)   │                              │ (websocket)  │
└──────────────┘                              └──────────────┘
```

## Component Design

### 1. Server Components

#### 1.1 Protocol Detection Layer

**Location**: `internal/server/server.go` (new code)

**Responsibility**: Detect protocol type by peeking at the first bytes of the connection

**Implementation**:
```go
type protocolType int

const (
    protocolTCP protocolType = iota
    protocolHTTP
)

func detectProtocol(conn net.Conn) (protocolType, *bufio.Reader, error) {
    // Create buffered reader to peek without consuming data
    reader := bufio.NewReader(conn)

    // Peek first 4 bytes to detect HTTP
    // HTTP requests start with: "GET ", "POST", "PUT ", "HEAD", etc.
    // TCP protobuf messages start with binary data
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

    // Default to TCP for binary data
    return protocolTCP, reader, nil
}
```

**Why this approach**:
- Non-destructive: `Peek()` doesn't consume data from the stream
- Simple: HTTP requests always start with ASCII method names
- Backward compatible: Binary protobuf data will never match HTTP patterns

#### 1.2 Connection Abstraction Layer

**Location**: `internal/server/connection.go` (new file)

**Responsibility**: Provide a unified interface for both TCP and WebSocket connections

**Interface**:
```go
// Connection represents a client connection (TCP or WebSocket)
type Connection interface {
    // RemoteAddr returns the remote address
    RemoteAddr() net.Addr

    // Write sends binary data to the client
    Write(data []byte) (int, error)

    // Read receives binary data from the client
    // For WebSocket, this blocks until a message is received
    Read(buf []byte) (int, error)

    // Close closes the connection
    Close() error

    // SetReadDeadline sets the read deadline
    SetReadDeadline(t time.Time) error
}
```

**Implementations**:

1. `TCPConnection` (wraps `net.Conn`):
```go
type TCPConnection struct {
    conn net.Conn
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

2. `WebSocketConnection` (wraps `*websocket.Conn`):
```go
type WebSocketConnection struct {
    conn          *websocket.Conn
    readBuffer    []byte
    readBufferPos int
}

func (wc *WebSocketConnection) RemoteAddr() net.Addr {
    return wc.conn.RemoteAddr()
}

func (wc *WebSocketConnection) Write(data []byte) (int, error) {
    // Send as binary WebSocket message
    err := wc.conn.WriteMessage(websocket.BinaryMessage, data)
    if err != nil {
        return 0, err
    }
    return len(data), nil
}

func (wc *WebSocketConnection) Read(buf []byte) (int, error) {
    // If we have buffered data, return it first
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

    // Only accept binary messages (protobuf)
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
    // This is acceptable for our use case
    return nil
}
```

#### 1.3 Modified Server Structure

**Location**: `internal/server/server.go` (modified)

**Changes**:
1. Replace `net.Conn` with `Connection` interface in `Client` struct
2. Modify `Start()` to perform protocol detection
3. Add WebSocket upgrade logic

**Key modifications**:
```go
type Client struct {
    conn     Connection  // Changed from net.Conn
    username string
    outgoing chan []byte
}

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
                // ... error handling
            }

            // Detect protocol
            go s.handleConnection(conn)
        }
    }
}

func (s *Server) handleConnection(rawConn net.Conn) {
    protocol, reader, err := detectProtocol(rawConn)
    if err != nil {
        log.Printf("Protocol detection failed: %v", err)
        rawConn.Close()
        return
    }

    var conn Connection

    switch protocol {
    case protocolHTTP:
        // Handle WebSocket upgrade
        conn, err = s.upgradeWebSocket(rawConn, reader)
        if err != nil {
            log.Printf("WebSocket upgrade failed: %v", err)
            rawConn.Close()
            return
        }
    case protocolTCP:
        // Wrap as TCP connection
        conn = &TCPConnection{conn: rawConn}
    }

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

#### 1.4 WebSocket Upgrade Handler

**Location**: `internal/server/websocket.go` (new file)

**Responsibility**: Handle HTTP request parsing and WebSocket upgrade

**Implementation**:
```go
func (s *Server) upgradeWebSocket(rawConn net.Conn, reader *bufio.Reader) (Connection, error) {
    // Parse HTTP request manually
    req, err := http.ReadRequest(reader)
    if err != nil {
        return nil, fmt.Errorf("failed to read HTTP request: %w", err)
    }

    // Check if it's a WebSocket upgrade request
    if !isWebSocketUpgrade(req) {
        return nil, fmt.Errorf("not a WebSocket upgrade request")
    }

    // Perform WebSocket handshake manually
    // Write upgrade response
    key := req.Header.Get("Sec-WebSocket-Key")
    acceptKey := computeAcceptKey(key)

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

    // Wrap in nbio WebSocket connection
    wsConn := websocket.NewConn(rawConn, true, 1024, 1024)

    return &WebSocketConnection{conn: wsConn}, nil
}

func isWebSocketUpgrade(req *http.Request) bool {
    return req.Method == "GET" &&
        strings.ToLower(req.Header.Get("Upgrade")) == "websocket" &&
        strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

func computeAcceptKey(key string) string {
    h := sha1.New()
    h.Write([]byte(key))
    h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
    return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
```

### 2. Client Components

#### 2.1 Connection Interface for Client

**Location**: `internal/client/connection.go` (new file)

**Responsibility**: Abstract TCP and WebSocket connections on client side

**Interface**:
```go
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

#### 2.2 Modified Client Structure

**Location**: `internal/client/client.go` (modified)

**Changes**:
1. Add `protocol` field to specify connection type
2. Replace `net.Conn` with `ClientConnection` interface
3. Modify `Connect()` to support both protocols

**Key modifications**:
```go
type Client struct {
    address  string
    username string
    protocol string  // "tcp" or "ws"
    conn     ClientConnection  // Changed from net.Conn
    messages chan protocol.Message
    mu       sync.RWMutex
    done     chan struct{}
    wg       sync.WaitGroup
}

func New(address, username, protocol string) *Client {
    return &Client{
        address:  address,
        username: username,
        protocol: protocol,  // New field
        messages: make(chan protocol.Message, 10),
        done:     make(chan struct{}),
    }
}

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

func (c *Client) connectTCP() (ClientConnection, error) {
    conn, err := net.Dial("tcp", c.address)
    if err != nil {
        return nil, fmt.Errorf("failed to connect via TCP: %w", err)
    }
    return &TCPClientConnection{conn: conn}, nil
}

func (c *Client) connectWebSocket() (ClientConnection, error) {
    // Parse address to create WebSocket URL
    url := fmt.Sprintf("ws://%s/", c.address)

    dialer := &websocket.Dialer{
        Timeout: 10 * time.Second,
    }

    conn, _, err := dialer.Dial(url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to connect via WebSocket: %w", err)
    }

    return &WebSocketClientConnection{conn: conn}, nil
}
```

#### 2.3 Client Connection Implementations

**Location**: `internal/client/connection.go` (new file)

**TCP Implementation**:
```go
type TCPClientConnection struct {
    conn net.Conn
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

**WebSocket Implementation**:
```go
type WebSocketClientConnection struct {
    conn          *websocket.Conn
    readBuffer    []byte
    readBufferPos int
}

func (wc *WebSocketClientConnection) Write(data []byte) (int, error) {
    err := wc.conn.WriteMessage(websocket.BinaryMessage, data)
    if err != nil {
        return 0, err
    }
    return len(data), nil
}

func (wc *WebSocketClientConnection) Read(buf []byte) (int, error) {
    // Similar to server-side WebSocketConnection.Read()
    // Handle buffering for partial reads
    if wc.readBufferPos < len(wc.readBuffer) {
        n := copy(buf, wc.readBuffer[wc.readBufferPos:])
        wc.readBufferPos += n
        if wc.readBufferPos >= len(wc.readBuffer) {
            wc.readBuffer = nil
            wc.readBufferPos = 0
        }
        return n, nil
    }

    messageType, data, err := wc.conn.ReadMessage()
    if err != nil {
        return 0, err
    }

    if messageType != websocket.BinaryMessage {
        return 0, fmt.Errorf("expected binary message")
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

#### 2.4 CLI Flag Addition

**Location**: `cmd/client/main.go` (modified)

**Changes**:
Add `-protocol` flag with validation

```go
func main() {
    serverAddr := flag.String("server", "localhost:8080", "Server address")
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

    // ... rest of main()
}
```

## Data Flow

### Message Flow (Both Protocols)

```
Client                      Server
  │                           │
  │   1. Connect              │
  │─────────────────────────>│
  │                           │ 2. Protocol Detection
  │                           │    (peek first bytes)
  │                           │
  │   3. Protocol-specific    │
  │      handshake            │
  │<─────────────────────────>│
  │                           │
  │   4. Protobuf Message     │
  │      (Join)               │
  │─────────────────────────>│
  │                           │ 5. Broadcast to all
  │   6. Receive broadcasted  │    clients (both TCP
  │      messages             │    and WebSocket)
  │<─────────────────────────│
  │                           │
```

### Protocol-Specific Details

**TCP Flow**:
1. Client: `net.Dial("tcp", "localhost:8080")`
2. Server: Accepts connection, peeks first bytes (binary protobuf data)
3. Server: Wraps connection in `TCPConnection`
4. Both: Exchange protobuf binary messages directly

**WebSocket Flow**:
1. Client: WebSocket dial to `ws://localhost:8080/`
2. Client: Sends HTTP GET request with Upgrade headers
3. Server: Peeks first bytes, detects "GET " prefix
4. Server: Parses HTTP request, performs WebSocket handshake
5. Server: Wraps connection in `WebSocketConnection`
6. Both: Exchange protobuf messages in WebSocket binary frames

## Protocol Message Format

Both protocols use the same protobuf message format:

```protobuf
message Message {
  MessageType type = 1;
  string sender = 2;
  string content = 3;
}

enum MessageType {
  MESSAGE_TYPE_TEXT = 0;
  MESSAGE_TYPE_JOIN = 1;
  MESSAGE_TYPE_LEAVE = 2;
}
```

**TCP Transport**: Raw protobuf binary data
**WebSocket Transport**: Protobuf binary data wrapped in WebSocket binary frames

## Error Handling

### Protocol Detection Errors
- If peek fails: Log error, close connection
- If neither HTTP nor valid protobuf: Log error, close connection
- Default to TCP for ambiguous cases

### WebSocket-Specific Errors
- Invalid HTTP request: Return 400 Bad Request
- Missing Upgrade headers: Return 400 Bad Request
- Handshake failure: Log error, close connection
- Non-binary message received: Log warning, ignore message

### Client Connection Errors
- Invalid protocol flag: Exit with error message
- WebSocket dial failure: Return clear error message
- Connection closed: Clean up resources, notify user

## Backward Compatibility

1. **Existing TCP Clients**: Continue to work without any changes
   - No new flags required
   - Protocol defaults to "tcp"
   - Server automatically detects TCP protocol

2. **Existing Server Code**: Core logic remains unchanged
   - Same protobuf message handling
   - Same broadcast mechanism
   - Same client management

3. **Protocol Layer**: New `Connection` interface is transparent
   - Existing message encoding/decoding unchanged
   - Same `protocol.Message` struct

## Testing Strategy

### Unit Tests
1. Protocol detection logic
   - HTTP request detection
   - Binary data detection
   - Edge cases (short reads, malformed data)

2. Connection interface implementations
   - TCPConnection read/write
   - WebSocketConnection read/write with framing
   - Buffer management in WebSocketConnection

### Integration Tests
1. TCP client → TCP server (existing test, should still pass)
2. WebSocket client → server
3. Mixed: TCP client + WebSocket client → server
4. Cross-protocol message delivery
5. Concurrent connections (multiple of each type)

### Manual Testing
1. Start server
2. Connect with TCP client, send messages
3. Connect with WebSocket client, send messages
4. Verify both clients receive messages from each other
5. Test join/leave notifications across protocols

## Dependencies

### New Dependencies
- `github.com/lesismal/nbio`: For WebSocket support
  - Only use `nbio/nbhttp/websocket` package
  - Client: Use `websocket.Dialer`
  - Server: Use `websocket.Conn` directly after manual upgrade

### Existing Dependencies (unchanged)
- `google.golang.org/protobuf`: For message serialization

## Implementation Notes

### Why Manual HTTP Parsing?
- We need protocol detection before any HTTP framework takes over
- Manual parsing gives us full control over the upgrade process
- Allows us to use buffered reader for peeking without data loss

### Why Not Use nbio HTTP Engine?
- nbio's HTTP engine doesn't support raw TCP on the same port
- We need lower-level control for protocol detection
- Manual approach is simpler for this specific use case

### WebSocket Path
- Client connects to `ws://localhost:8080/`
- Server accepts any path (we only check for Upgrade header)
- This ensures "same entry point" requirement is met

## Security Considerations

### Protocol Detection
- Peek operation has timeout to prevent DoS
- Invalid data results in immediate connection close
- No sensitive information in protocol detection logic

### WebSocket
- Accept all origins in development (should be restricted in production)
- Validate message types (only accept binary)
- Implement proper close handshake

### General
- No authentication in this phase (out of scope)
- Message size limits apply to both protocols
- Connection limits apply equally to both protocols
