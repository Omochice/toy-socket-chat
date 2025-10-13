# Socket Chat

A Toy socket-based chat application built with Go, supporting both TCP and WebSocket connections.

## Overview

This project is a simple chat system that supports both TCP sockets and WebSocket connections.
It includes CLI tools (server and client) for both protocols, allowing multiple users to exchange messages in real-time.

## Features

- **Dual Protocol Support**: Both TCP sockets and WebSocket connections
- **TCP Socket Communication**: Direct communication using raw TCP sockets
- **WebSocket Communication**: Browser-compatible WebSocket protocol (RFC 6455)
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
# Build TCP server and client
go build -o bin/server ./cmd/server
go build -o bin/client ./cmd/client

# Build WebSocket server and client
go build -o bin/websocket-server ./cmd/websocket-server
go build -o bin/websocket-client ./cmd/websocket-client
```

## Usage

### TCP Socket Version

#### Starting the TCP Server

First, start the TCP server:

```bash
./bin/server -port :8080
```

Options:
- `-port`: Port for the server to listen on (default: `:8080`)

When the server starts, you'll see a message like this:
```
Starting server on :8080...
Server started on [::]:8080
```

#### Connecting a TCP Client

In a separate terminal, start the client:

```bash
./bin/client -server localhost:8080 -username alice
```

Options:
- `-server`: Server address to connect to (default: `localhost:8080`)
- `-username`: Username to display in chat (required)

When the client connects, you'll see a message like this:
```
Connected to localhost:8080 as alice
Type your messages (or 'quit' to exit):
```

### WebSocket Version

#### Starting the WebSocket Server

Start the WebSocket server:

```bash
./bin/websocket-server -port :8080
```

Options:
- `-port`: Port for the server to listen on (default: `:8080`)

When the server starts, you'll see a message like this:
```
Starting WebSocket server on :8080...
WebSocket server started on [::]:8080
```

#### Connecting a WebSocket Client

In a separate terminal, start the WebSocket client:

```bash
./bin/websocket-client -server ws://localhost:8080/ws -username alice
```

Options:
- `-server`: WebSocket server URL (default: `ws://localhost:8080/ws`)
- `-username`: Username to display in chat (required)

When the client connects, you'll see a message like this:
```
Connected to ws://localhost:8080/ws as alice
Type your messages (or 'quit' to exit):
```

### How to Chat

1. Type a message and press Enter to send it to all other connected users
2. Messages from other users are displayed in the format `[username]: message`
3. User join/leave events are notified in the format `*** username joined the chat ***`
4. To exit, type `quit` or `exit`

## Example Usage

### Three-User Chat Example (TCP)

**Terminal 1: Starting the TCP Server**

```bash
$ ./bin/server -port :8080
Starting server on :8080...
Server started on [::]:8080
User alice joined
User bob joined
Message from alice: Hello everyone!
User carol joined
Message from bob: Hi alice!
Message from carol: Hey guys!
```

**Terminal 2: Connecting as alice**

```bash
$ ./bin/client -server localhost:8080 -username alice
Connected to localhost:8080 as alice
Type your messages (or 'quit' to exit):
*** bob joined the chat ***
Hello everyone!
[bob]: Hi alice!
*** carol joined the chat ***
[carol]: Hey guys!
```

**Terminal 3: Connecting as bob**

```bash
$ ./bin/client -server localhost:8080 -username bob
Connected to localhost:8080 as bob
Type your messages (or 'quit' to exit):
[alice]: Hello everyone!
Hi alice!
*** carol joined the chat ***
[carol]: Hey guys!
```

**Terminal 4: Connecting as carol**

```bash
$ ./bin/client -server localhost:8080 -username carol
Connected to localhost:8080 as carol
Type your messages (or 'quit' to exit):
[alice]: Hello everyone!
[bob]: Hi alice!
Hey guys!
```

### Three-User Chat Example (WebSocket)

**Terminal 1: Starting the WebSocket Server**

```bash
$ ./bin/websocket-server -port :8080
Starting WebSocket server on :8080...
WebSocket server started on [::]:8080
User alice joined
User bob joined
Message from alice: Hello everyone!
```

**Terminal 2: Connecting as alice**

```bash
$ ./bin/websocket-client -server ws://localhost:8080/ws -username alice
Connected to ws://localhost:8080/ws as alice
Type your messages (or 'quit' to exit):
*** bob joined the chat ***
Hello everyone!
[bob]: Hi alice!
```

**Terminal 3: Connecting as bob**

```bash
$ ./bin/websocket-client -server ws://localhost:8080/ws -username bob
Connected to ws://localhost:8080/ws as bob
Type your messages (or 'quit' to exit):
[alice]: Hello everyone!
Hi alice!
```

## Troubleshooting

### Port Already in Use

```
Failed to start server: listen tcp :8080: bind: address already in use
```

Specify a different port number:
```bash
# For TCP server
./bin/server -port :9090

# For WebSocket server
./bin/websocket-server -port :9090
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
