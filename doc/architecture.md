# Architecture

This document describes the technical architecture of the TCP Socket Chat system.

## Overview

The system consists of three main components:

1. **Message Protocol** (`pkg/protocol`) - Defines message format and encoding
2. **Server** (`internal/server`) - Manages connections and broadcasts messages
3. **Client** (`internal/client`) - Connects to server and handles user interaction

## System Architecture

```
┌─────────────┐         ┌─────────────┐
│   Client 1  │         │   Client 2  │
│  (Terminal) │         │  (Terminal) │
└──────┬──────┘         └──────┬──────┘
       │                       │
       │  TCP Connection       │
       │                       │
       └───────┬───────────────┘
               │
               │
       ┌───────▼────────┐
       │                │
       │   TCP Server   │
       │                │
       └────────────────┘
            │     │
         Broadcast
            │     │
       ┌────▼─────▼────┐
       │ Client Manager │
       │  (Goroutines)  │
       └────────────────┘
```

## Component Details

### Message Protocol (`pkg/protocol`)

The protocol package defines the message structure and encoding/decoding logic.

#### Message Structure

```go
type Message struct {
    Type    MessageType
    Sender  string
    Content string
}
```

#### Message Types

```go
const (
    MessageTypeText  MessageType = iota  // Regular chat message
    MessageTypeJoin                      // User joined notification
    MessageTypeLeave                     // User left notification
)
```

#### Encoding

Messages are encoded using Go's `encoding/gob` package, which provides:
- Efficient binary encoding
- Built-in support for Go types
- Automatic handling of complex structures

**Why `gob`?**
- Simple to use - no schema definition needed
- Type-safe encoding/decoding
- Good performance for Go-to-Go communication
- Built into the standard library

**Trade-offs:**
- Not interoperable with other languages
- Binary format (not human-readable)
- Requires both ends to be Go applications

For a production system, consider:
- Protocol Buffers for language interoperability
- JSON for human readability and debugging
- MessagePack for smaller message sizes

### Server (`internal/server`)

The server manages client connections and broadcasts messages.

#### Core Components

```go
type Server struct {
    address  string                  // Listen address (e.g., ":8080")
    listener net.Listener            // TCP listener
    clients  map[*Client]bool        // Connected clients
    mu       sync.RWMutex            // Protects clients map
    quit     chan struct{}           // Shutdown signal
    wg       sync.WaitGroup          // Goroutine coordination
}

type Client struct {
    conn     net.Conn                // TCP connection
    username string                  // User's name
    outgoing chan []byte             // Outgoing message queue
}
```

#### Connection Flow

1. **Accept Connection**
   ```
   listener.Accept() → New Client → Add to clients map → Spawn goroutines
   ```

2. **Client Handling** (2 goroutines per client)
   - **Reader goroutine**: Reads messages from TCP connection
   - **Writer goroutine**: Writes messages from outgoing channel to TCP connection

3. **Message Broadcasting**
   ```
   Client sends message → Server receives → Broadcast to all other clients
   ```

4. **Disconnect**
   ```
   Connection closed → Remove from clients map → Close channels → Cleanup
   ```

#### Concurrency Model

The server uses a concurrent model with the following characteristics:

**Per-Client Goroutines:**
- 1 goroutine for reading from client
- 1 goroutine for writing to client
- Allows independent read/write operations
- Prevents blocking on slow clients

**Synchronization:**
```go
mu sync.RWMutex  // Protects clients map
- Read lock: For broadcasting (reading the map)
- Write lock: For adding/removing clients (modifying the map)
```

**Message Queue:**
```go
outgoing chan []byte  // Buffered channel (size: 10)
- Decouples receiving from sending
- Prevents blocking on slow clients
- Drops messages if queue is full
```

**Graceful Shutdown:**
```go
quit chan struct{}   // Broadcast shutdown signal
wg sync.WaitGroup    // Wait for all goroutines
```

#### Thread Safety

All shared resources are protected:

1. **Clients Map**
   - Protected by `sync.RWMutex`
   - Multiple readers OR single writer
   - Prevents concurrent map access panics

2. **Client Channels**
   - Closed only after client is removed from map
   - Prevents sending to closed channels

3. **Connection State**
   - Each connection owned by one goroutine
   - No shared state between goroutines

### Client (`internal/client`)

The client connects to the server and manages message exchange.

#### Core Components

```go
type Client struct {
    address  string                  // Server address
    username string                  // User's name
    conn     net.Conn                // TCP connection
    messages chan protocol.Message   // Incoming messages
    mu       sync.RWMutex            // Protects conn
    done     chan struct{}           // Shutdown signal
    wg       sync.WaitGroup          // Goroutine coordination
}
```

#### Operation Flow

1. **Connect**
   ```
   Connect() → Dial TCP → Spawn receiver goroutine → Return
   ```

2. **Send Message**
   ```
   SendMessage() → Encode → Write to connection
   ```

3. **Receive Messages**
   ```
   Background goroutine: Read from connection → Decode → Send to messages channel
   ```

4. **Disconnect**
   ```
   Disconnect() → Close connection → Signal done → Wait for goroutines
   ```

#### Concurrency Model

**Background Receiver:**
- Single goroutine continuously reads from TCP connection
- Decodes messages and sends to `messages` channel
- Application reads from channel when ready

**Thread Safety:**
```go
mu sync.RWMutex  // Protects connection access
- Used when checking connection state
- Prevents concurrent access to connection
```

## Data Flow

### Sending a Message

```
User Input
    ↓
Client.SendMessage()
    ↓
protocol.Message.Encode()
    ↓
TCP Write (client → server)
    ↓
Server receives (reader goroutine)
    ↓
Server.broadcast()
    ↓
For each connected client:
    ↓
Write to client.outgoing channel
    ↓
Writer goroutine reads channel
    ↓
TCP Write (server → client)
    ↓
Client receiver goroutine reads
    ↓
protocol.Message.Decode()
    ↓
Send to messages channel
    ↓
Application reads message
    ↓
Display to user
```

### Connection Lifecycle

```
Client Side:                    Server Side:
    ↓                               ↓
Connect()                       listener.Accept()
    ↓                               ↓
net.Dial()  ─────────────────→  New connection
    ↓                               ↓
Start receiver goroutine        Create Client object
    ↓                               ↓
Send JOIN message  ───────────→  Add to clients map
    ↓                               ↓
                                Start reader goroutine
                                Start writer goroutine
    ↓                               ↓
Chat messages   ←────────────→  Broadcast messages
    ↓                               ↓
Send LEAVE message ───────────→ Remove from clients
    ↓                               ↓
Disconnect()                    Close connection
    ↓                               ↓
conn.Close()    ←───────────────  Cleanup goroutines
```

## Error Handling

### Server Errors

1. **Accept Errors**
   - Log error and continue accepting
   - Check for shutdown signal

2. **Read Errors**
   - Treat as client disconnect
   - Clean up client resources
   - Don't crash server

3. **Broadcast Errors**
   - Skip failed client
   - Log error
   - Continue broadcasting to others

### Client Errors

1. **Connection Errors**
   - Return error to caller
   - Don't start receiver goroutine

2. **Read Errors**
   - Log error
   - Exit receiver goroutine
   - Application should detect disconnect

3. **Send Errors**
   - Return error to caller
   - Let application handle

## Performance Considerations

### Buffering

- **Client outgoing channel**: Buffered (10 messages)
  - Prevents blocking on slow clients
  - Trade-off: May drop messages if client is very slow

- **TCP connection**: OS-level buffering
  - No explicit buffering in application

### Scalability

Current limitations:
- Single-threaded broadcast (sequentially sends to each client)
- All clients in memory
- No message persistence

For production scale:
- Use pub/sub system (Redis, NATS)
- Horizontal scaling with load balancer
- Separate connection handling from message routing

### Memory Usage

Per client:
- Connection object: ~1KB
- Goroutines: ~4KB each (2 per client)
- Message buffer: ~1KB (channel buffer)

Estimated: ~10KB per connected client

## Design Decisions

### Why Two Goroutines Per Client?

**Alternative 1**: Single goroutine for both read and write
- Problem: Blocking on write prevents reading
- Problem: Blocking on read prevents writing

**Alternative 2**: Non-blocking I/O with select
- More complex code
- Less idiomatic Go
- Harder to maintain

**Chosen**: Separate goroutines
- Simple, idiomatic Go
- Each goroutine has single responsibility
- Easy to reason about

### Why Mutex Instead of Channels for Clients Map?

**Alternative**: Use channels to serialize access
```go
type clientOp struct {
    op      string
    client  *Client
    result  chan bool
}
```

**Trade-offs:**
- Channels: More idiomatic, prevents forgetting locks
- Mutex: Simpler, less overhead, more straightforward

**Chosen**: Mutex
- Clearer intent (protecting data structure)
- Less boilerplate code
- Better performance for read-heavy workload

### Why `encoding/gob`?

See "Encoding" section above for detailed rationale.

## Future Improvements

### Short Term
1. Add connection timeout handling
2. Implement heartbeat/ping messages
3. Add message size limits
4. Better error recovery

### Long Term
1. Support for private messages
2. Chat rooms/channels
3. Message history persistence
4. Authentication and authorization
5. TLS encryption
6. Rate limiting
7. Message acknowledgments

## References

- Go Concurrency Patterns: https://go.dev/blog/pipelines
- Effective Go: https://go.dev/doc/effective_go
- TCP Socket Programming: https://pkg.go.dev/net
