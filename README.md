# Socket Chat

A Toy socket-based chat application built with Go, supporting both TCP and WebSocket connections.

## Overview

This project is a simple chat system that supports both TCP sockets and WebSocket connections.
It includes CLI tools (server and client) for both protocols, allowing multiple users to exchange messages in real-time.

## Features

- **Dual Protocol Support**: Both TCP sockets and WebSocket connections
- **Unified Server**: Single server handling both TCP and WebSocket clients in the same chat room
- **TCP Socket Communication**: Direct communication using raw TCP sockets
- **WebSocket Communication**: Browser-compatible WebSocket protocol (RFC 6455)
- **Cross-Protocol Messaging**: TCP clients can chat with WebSocket clients seamlessly
- **Multiple Client Support**: Multiple users can connect simultaneously
- **Message Broadcasting**: Messages from one user are delivered to all others
- **Join/Leave Notifications**: User join and leave events are notified to all participants
- **Concurrent Processing**: Efficient concurrent processing using Goroutines

## Requirements

- Go 1.25.2 or higher

## Installation

### Build

Build the server and client binaries:

```bash
# Build unified server (supports both TCP and WebSocket)
go build -o bin/server ./cmd/server

# Build clients
go build -o bin/client ./cmd/client
go build -o bin/websocket-client ./cmd/websocket-client
```

## Usage

### Starting the Server

The server runs in unified mode on a single port, handling both TCP and WebSocket clients in the same chat room.

```bash
./bin/server -port :8080
```

**Options:**
- `-port`: Port to listen on for both protocols (default: `:8080`)

The server automatically detects whether incoming connections are raw TCP or WebSocket (HTTP) by inspecting the first few bytes.

When the server starts, you'll see messages like this:
```
Starting unified server on :8080...
  Accepting both TCP socket and WebSocket connections
Unified server started on [::]:8080 (TCP and WebSocket)
```

### Connecting Clients

You can connect both TCP and WebSocket clients to the same port.

#### TCP Client

```bash
./bin/client -server localhost:8080 -username alice
```

**Options:**
- `-server`: Server address to connect to (default: `localhost:8080`)
- `-username`: Username to display in chat (required)

When the client connects, you'll see:
```
Connected to localhost:8080 as alice
Type your messages (or 'quit' to exit):
```

#### WebSocket Client

```bash
./bin/websocket-client -server ws://localhost:8080/ws -username bob
```

**Options:**
- `-server`: WebSocket server URL (default: `ws://localhost:8080/ws`)
- `-username`: Username to display in chat (required)

When the client connects, you'll see:
```
Connected to ws://localhost:8080/ws as bob
Type your messages (or 'quit' to exit):
```

**Note:** Both clients connect to the **same port** (8080). The server automatically detects the protocol by inspecting the connection headers.

### How to Chat

1. Type a message and press Enter to send it to all other connected users
2. Messages from other users are displayed in the format `[username]: message`
3. User join/leave events are notified in the format `*** username joined the chat ***`
4. To exit, type `quit` or `exit`

## Example Usage

### Cross-Protocol Chat Example

**Terminal 1: Starting the Server**

```bash
$ ./bin/server -port :8080
Starting unified server on :8080...
  Accepting both TCP socket and WebSocket connections
Unified server started on [::]:8080 (TCP and WebSocket)
TCP user alice joined
WebSocket user bob joined
Message from TCP user alice: Hello from TCP!
Message from WebSocket user bob: Hello from WebSocket!
```

**Terminal 2: TCP Client (alice)**

```bash
$ ./bin/client -server localhost:8080 -username alice
Connected to localhost:8080 as alice
Type your messages (or 'quit' to exit):
*** bob joined the chat ***
Hello from TCP!
[bob]: Hello from WebSocket!
```

**Terminal 3: WebSocket Client (bob)**

```bash
$ ./bin/websocket-client -server ws://localhost:8080/ws -username bob
Connected to ws://localhost:8080/ws as bob
Type your messages (or 'quit' to exit):
[alice]: Hello from TCP!
Hello from WebSocket!
```

## Troubleshooting

### Port Already in Use

```
Failed to start server: listen tcp :8080: bind: address already in use
```

Specify a different port:
```bash
./bin/server -port :9090
```

### Cannot Connect to Server

- Verify that the server is running
- Check if a firewall is blocking the port
- Verify that the server address and port number are correct

## Contributing

Interested in contributing? Check out [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines and best practices.

## Documentation

- [Architecture](doc/architecture.md) - Technical details about the system design
- [Testing Guide](doc/testing.md) - How to run and write tests

## License

[zlib](./LICENSE)
