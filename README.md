# TCP Socket Chat

A Toy TCP socket-based chat application built with Go.

## Overview

This project is a simple chat system using TCP sockets.
It includes two CLI tools (server and client) that allow multiple users to exchange messages in real-time.

## Features

- Multiple Transports: Raw TCP sockets, WebSocket, and WebTransport (HTTP/3 over QUIC) are all supported, and clients on any of them share the same chat
- Multiple Client Support: Multiple users can connect simultaneously
- Message Broadcasting: Messages from one user are delivered to all others
- Join/Leave Notifications: User join and leave events are notified to all participants
- Concurrent Processing: Efficient concurrent processing using Goroutines

## Build

```bash
devbox run build
```

The server and client binaries will be placed in the `build/` directory.

See [CONTRIBUTING.md](./CONTRIBUTING.md) for more information.

## Usage

### Starting the Server

First, start the server:

```bash
./build/server -port :8080
```

Options:
- `-port`: Port for the server to listen on (default: `:8080`)
- `-cert`: Path to a TLS certificate PEM file (enables WebTransport; requires `-key`)
- `-key`: Path to a TLS private key PEM file (enables WebTransport; requires `-cert`)

Without `-cert`/`-key`, the server accepts TCP and WebSocket connections only. See [WebTransport (HTTP/3 over QUIC)](#webtransport-http3-over-quic) below for how to enable the WebTransport endpoint.

When the server starts, you'll see a message like this:
```
Starting server on :8080...
Server started on [::]:8080
```

### Connecting a Client

In a separate terminal, start the client:

```bash
./build/client -server localhost:8080 -username alice
```

Options:
- `-server`: Server address to connect to (default: `localhost:8080`)
- `-username`: Username to display in chat (required)
- `-protocol`: Transport to use: `tcp`, `ws`, or `wt` (default: `tcp`)
- `-ca`: Path to a PEM CA certificate to trust when verifying the server (only used with `-protocol wt`; without it, the system trust store is used)

When the client connects, you'll see a message like this:
```
Connected to localhost:8080 as alice
Type your messages (or 'quit' to exit):
```

### How to Chat

1. Type a message and press Enter to send it to all other connected users
2. Messages from other users are displayed in the format `[username]: message`
3. User join/leave events are notified in the format `*** username joined the chat ***`
4. To exit, type `quit` or `exit`

### WebTransport (HTTP/3 over QUIC)

WebTransport requires TLS, so a certificate and key are needed. [mkcert](https://github.com/FiloSottile/mkcert) generates a certificate trusted by a local development CA, which the client can then be pointed at explicitly.

`mkcert` is included in the devbox environment.

1. Generate a certificate and key for `localhost`:

   ```bash
   mkcert -cert-file server.pem -key-file server-key.pem localhost 127.0.0.1 ::1
   ```

   The first run creates a local CA under mkcert's CAROOT and uses it to sign the certificate. `mkcert -install` is not required, because the client below is pointed at the CA directly with `-ca` instead of relying on the system trust store.

2. Start the server with the generated certificate and key:

   ```bash
   ./build/server -cert server.pem -key server-key.pem
   ```

   WebTransport is served on the same port number as the TCP listener, over UDP.

3. Connect a client over WebTransport, trusting mkcert's local CA:

   ```bash
   ./build/client -protocol wt -ca "$(mkcert -CAROOT)/rootCA.pem" -username alice
   ```

WebTransport clients join the same chat as TCP and WebSocket clients; messages are broadcast across all three transports.

## Example Usage

### Three-User Chat Example

**Terminal 1: Starting the Server**

```bash
$ ./build/server -port :8080
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
$ ./build/client -server localhost:8080 -username alice
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
$ ./build/client -server localhost:8080 -username bob
Connected to localhost:8080 as bob
Type your messages (or 'quit' to exit):
[alice]: Hello everyone!
Hi alice!
*** carol joined the chat ***
[carol]: Hey guys!
```

**Terminal 4: Connecting as carol**

```bash
$ ./build/client -server localhost:8080 -username carol
Connected to localhost:8080 as carol
Type your messages (or 'quit' to exit):
[alice]: Hello everyone!
[bob]: Hi alice!
Hey guys!
```

## Troubleshooting

### Port Already in Use

```
Failed to start server: listen tcp :8080: bind: address already in use
```

Specify a different port number:
```bash
./build/server -port :9090
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
