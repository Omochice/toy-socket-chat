# Testing Guide

This document describes the testing strategy and provides guidance on running and writing tests.

## Testing Philosophy

This project follows **Test-Driven Development (TDD)**. Tests are written before implementation, ensuring:
- Code is testable by design
- Clear specifications through tests
- High test coverage
- Confidence in refactoring

## Test Hierarchy

```
┌─────────────────────────────────────┐
│      Integration Tests              │  End-to-end scenarios
│      (test/)                         │
├─────────────────────────────────────┤
│      Unit Tests                      │  Individual components
│      (*_test.go in each package)    │
└─────────────────────────────────────┘
```

## Test Structure

### Unit Tests

Unit tests are located alongside the code they test:

```
internal/server/
├── server.go          # Implementation
└── server_test.go     # Tests

pkg/protocol/
├── message.go         # Implementation
└── message_test.go    # Tests
```

### Integration Tests

Integration tests are in the `test/` directory:

```
test/
└── integration_test.go    # Full system tests
```

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run with Verbose Output

```bash
go test ./... -v
```

### Run Specific Package

```bash
# Protocol tests
go test ./pkg/protocol -v

# Server tests
go test ./internal/server -v

# Client tests
go test ./internal/client -v

# Integration tests
go test ./test -v
```

### Run with Coverage

```bash
# Coverage report
go test -cover ./...

# Detailed coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run with Race Detector

```bash
go test -race ./...
```

This detects data races in concurrent code. Always run this before committing changes.

### Run Specific Test

```bash
# Run single test function
go test ./pkg/protocol -run TestMessage_Encode

# Run tests matching pattern
go test ./internal/server -run TestServer_
```

### Set Timeout

```bash
# Default timeout is 10 minutes
go test ./... -timeout 30s
```

## Test Organization

### Package Structure

Tests are organized by package:

- **`pkg/protocol`**: Message encoding/decoding tests
- **`internal/server`**: Server functionality tests
- **`internal/client`**: Client functionality tests
- **`test/`**: Integration tests

### Test File Naming

- Test files: `*_test.go`
- Same package: `package foo` → `package foo_test` or `package foo`
  - `package foo`: White-box testing (access private members)
  - `package foo_test`: Black-box testing (only public API)

## Writing Tests

### Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
func TestMessage_Encode(t *testing.T) {
    tests := []struct {
        name    string
        msg     protocol.Message
        wantErr bool
    }{
        {
            name: "encode text message",
            msg: protocol.Message{
                Type:    protocol.MessageTypeText,
                Sender:  "user1",
                Content: "Hello",
            },
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            data, err := tt.msg.Encode()
            if (err != nil) != tt.wantErr {
                t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
            }
            // More assertions...
        })
    }
}
```

### Test Naming Convention

```go
// Function_Scenario
func TestServer_Start(t *testing.T)
func TestClient_Connect(t *testing.T)
func TestMessage_Encode(t *testing.T)

// Function_Scenario_ExpectedResult
func TestServer_Stop_ClosesAllConnections(t *testing.T)
func TestClient_SendMessage_WithoutConnection_ReturnsError(t *testing.T)
```

### Assertions

Use clear, descriptive assertions:

```go
// Good
if got != want {
    t.Errorf("ClientCount() = %d, want %d", got, want)
}

// Better - shows context
if count := srv.ClientCount(); count != 2 {
    t.Errorf("Expected 2 clients after connections, got %d", count)
}
```

### Setup and Cleanup

Use `defer` for cleanup:

```go
func TestServer_ClientConnection(t *testing.T) {
    srv := server.New(":0")
    go srv.Start()
    defer srv.Stop()  // Always cleanup

    // Test code...
}
```

### Testing Concurrent Code

Use channels and timeouts:

```go
func TestClient_ReceiveMessage(t *testing.T) {
    // Setup...
    msgChan := client.Messages()

    // Send test message...

    // Wait with timeout
    select {
    case msg := <-msgChan:
        if msg.Content != expected {
            t.Errorf("Expected %q, got %q", expected, msg.Content)
        }
    case <-time.After(2 * time.Second):
        t.Fatal("Timeout waiting for message")
    }
}
```

## Test Coverage by Package

### Protocol Package (`pkg/protocol`)

Tests cover:
- ✅ Message encoding
- ✅ Message decoding
- ✅ Round-trip encoding/decoding
- ✅ MessageType string representation
- ✅ Error cases

Test coverage is approximately 100%.

### Server Package (`internal/server`)

Tests cover:
- ✅ Server start/stop
- ✅ Client connection handling
- ✅ Message broadcasting
- ✅ Multiple client connections
- ✅ Client disconnection
- ✅ Graceful shutdown

Test coverage is approximately 90%.

### Client Package (`internal/client`)

Tests cover:
- ✅ Connection establishment
- ✅ Message sending
- ✅ Message receiving
- ✅ Join/leave messages
- ✅ Error handling (no connection)
- ✅ Disconnection

Test coverage is approximately 85%.

### Integration Tests (`test/`)

Tests cover:
- ✅ End-to-end communication
- ✅ Multiple client scenarios
- ✅ Message broadcasting between clients
- ✅ Join/leave notifications

## Mock Objects

### Mock Server

For client tests, a mock TCP server is used:

```go
func startMockServer(t *testing.T) (string, func()) {
    listener, err := net.Listen("tcp", ":0")
    if err != nil {
        t.Fatalf("Failed to start mock server: %v", err)
    }

    // Server logic...

    cleanup := func() {
        listener.Close()
    }

    return listener.Addr().String(), cleanup
}
```

### Why Mock?

- **Isolation**: Test client without depending on real server
- **Control**: Simulate specific server behaviors
- **Speed**: Faster than spinning up full server
- **Reliability**: No network dependencies

## Test Data

### Test Messages

Use descriptive test data:

```go
testMsg := protocol.Message{
    Type:    protocol.MessageTypeText,
    Sender:  "testuser",
    Content: "Test message for scenario X",
}
```

### Test Ports

Always use port `:0` to let OS assign free ports:

```go
srv := server.New(":0")  // OS assigns free port
addr := srv.Addr()       // Get actual address
```

This prevents port conflicts in tests.

## Common Patterns

### Testing Goroutines

```go
func TestBackgroundWorker(t *testing.T) {
    done := make(chan struct{})

    go func() {
        defer close(done)
        // Work...
    }()

    select {
    case <-done:
        // Success
    case <-time.After(time.Second):
        t.Fatal("Goroutine didn't complete")
    }
}
```

### Testing Error Cases

```go
func TestFunction_ErrorCase(t *testing.T) {
    err := someFunction()
    if err == nil {
        t.Fatal("Expected error, got nil")
    }

    // Check error message
    expectedMsg := "expected error message"
    if err.Error() != expectedMsg {
        t.Errorf("Error message = %q, want %q", err.Error(), expectedMsg)
    }
}
```

### Subtests

Group related tests:

```go
func TestMessage(t *testing.T) {
    t.Run("Encoding", func(t *testing.T) {
        // Encoding tests...
    })

    t.Run("Decoding", func(t *testing.T) {
        // Decoding tests...
    })
}
```

## Debugging Tests

### Run Single Test with Verbose Output

```bash
go test ./pkg/protocol -v -run TestMessage_Encode
```

### Print Debug Information

```go
func TestDebug(t *testing.T) {
    result := someFunction()
    t.Logf("Debug: result = %+v", result)  // Only shows if test fails

    if result != expected {
        t.Errorf("Failed: got %v, want %v", result, expected)
    }
}
```

### Use `-race` Flag

Always check for race conditions:

```bash
go test -race ./...
```

## Continuous Integration

### Pre-Commit Checks

Before committing, always run:

```bash
# Format code
gofmt -w .

# Run all tests
go test ./...

# Check for races
go test -race ./...

# Check coverage
go test -cover ./...
```

### CI Pipeline

Recommended CI checks:
1. Format check (`gofmt`)
2. Lint check (`golangci-lint`)
3. Unit tests
4. Integration tests
5. Race detection
6. Coverage report

## Best Practices

### What to Do

- Write tests before implementation following TDD
- Use table-driven tests for multiple scenarios
- Test edge cases and error conditions
- Use meaningful test names
- Keep tests independent
- Use `t.Helper()` for test helpers
- Clean up resources with `defer`

### What to Avoid

- Skipping error checks in tests
- Using fixed ports (use `:0` instead)
- Sharing state between tests
- Using `time.Sleep` for synchronization (use channels instead)
- Testing implementation details (test behavior)
- Ignoring flaky tests (fix them)

## Test Metrics

### Current Status

```
Package                          Coverage
----------------------------------------
pkg/protocol                     100%
internal/server                  90%
internal/client                  85%
test/ (integration)              N/A
----------------------------------------
Overall                          ~90%
```

### Goals

- Maintain >80% coverage for all packages
- All public APIs must have tests
- All bug fixes must include regression tests
- Integration tests for all user-facing features

## Troubleshooting

### Test Timeout

If tests timeout:
1. Check for goroutine leaks
2. Ensure channels are closed
3. Add explicit timeouts to test operations
4. Use `-timeout` flag: `go test -timeout 30s ./...`

### Flaky Tests

If tests fail intermittently:
1. Check for race conditions (`go test -race`)
2. Look for timing dependencies
3. Ensure proper synchronization
4. Use channels instead of `time.Sleep`

### Port Conflicts

If tests fail with "address already in use":
1. Always use `:0` for test servers
2. Ensure cleanup in `defer`
3. Check for leaked server instances

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Go Testing Best Practices](https://go.dev/doc/code)
- [Advanced Testing with Go](https://www.youtube.com/watch?v=8hQG7QlcLBk)
