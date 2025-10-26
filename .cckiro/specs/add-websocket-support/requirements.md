# Requirements: Add WebSocket Support

## Overview
Add WebSocket support to the existing TCP socket chat application using the nbio library, while maintaining full backward compatibility with the current TCP socket implementation.

## Requirements

### R1: WebSocket Library Integration
**Description**: Integrate nbio library for WebSocket support

**Acceptance Criteria**:
- Add `github.com/lesismal/nbio` as a dependency to the project
- The library must be compatible with the existing Go version (1.25.1)
- No conflicts with existing dependencies (google.golang.org/protobuf)

### R2: Dual Protocol Server Support
**Description**: Server must accept both TCP socket and WebSocket connections

**Acceptance Criteria**:
- Server listens on a single port and handles both TCP and WebSocket protocols
- Server can distinguish between TCP and WebSocket connections automatically
- Both connection types can coexist simultaneously on the same server instance
- No degradation in performance for either protocol

### R3: Unified Entry Point
**Description**: TCP and WebSocket clients connect to the exact same endpoint

**Acceptance Criteria**:
- TCP socket clients connect to: `localhost:8080`
- WebSocket clients connect to: `ws://localhost:8080/`
- The server automatically detects the protocol type based on the connection handshake
- No separate ports or paths required for different protocols

### R4: Client Protocol Selection
**Description**: Client can choose between TCP and WebSocket via command-line argument

**Acceptance Criteria**:
- Add a new `-protocol` flag to the client CLI
- Supported values: `tcp` (default) or `ws`
- Example usage:
  - TCP: `./build/client -server localhost:8080 -username alice -protocol tcp`
  - WebSocket: `./build/client -server localhost:8080 -username alice -protocol ws`
- Invalid protocol values should result in a clear error message
- The `-protocol` flag is optional (defaults to `tcp` for backward compatibility)

### R5: Existing Functionality Preservation
**Description**: Maintain all current chat features for both protocols

**Acceptance Criteria**:
- Protobuf message format (pkg/protocol) is used by both TCP and WebSocket
- Message types (Join, Leave, Text) work identically for both protocols
- Message broadcasting works across both TCP and WebSocket clients
- A TCP client can chat with a WebSocket client and vice versa
- User join/leave notifications are sent to all clients regardless of protocol
- Graceful shutdown works for both connection types
- All existing tests continue to pass
- Existing client and server command-line flags remain unchanged (except for the new `-protocol` flag)

## Non-Functional Requirements

### NFR1: Backward Compatibility
- Existing TCP-only clients must continue to work without any changes
- Default behavior (no `-protocol` flag) uses TCP

### NFR2: Code Quality
- Follow existing code structure and patterns
- Add appropriate error handling for WebSocket-specific errors
- Include logging for protocol detection and connection type

### NFR3: Testing
- Existing integration tests must pass
- Add tests for WebSocket connections
- Add tests for mixed TCP/WebSocket scenarios

## Out of Scope
- WebSocket Secure (WSS) support with TLS/SSL
- WebSocket compression
- WebSocket subprotocols
- Binary WebSocket frames (only text frames will be supported initially)
- Authentication/authorization mechanisms
- Rate limiting or connection throttling
