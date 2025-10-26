# Contributing to TCP Socket Chat

Thank you for your interest in contributing to this project! This document provides guidelines and best practices for development.

## Prerequisites

This repository uses [jetify-com/devbox](https://github.com/jetify-com/devbox).

The following documents assume that you are attaching to the shell using `devbox shell`.

## Build

```bash
devbox run build
```

## Development Philosophy

This project follows **Test-Driven Development (TDD)** principles. All code must be developed following the Red-Green-Refactor cycle.

### TDD Workflow

1. **Red Phase**: Write a failing test first
   - Define what you want to implement through tests
   - Run tests and verify they fail
   - Commit with prefix `test:`

2. **Green Phase**: Write minimal code to pass the test
   - Implement just enough to make the test pass
   - Run tests and verify they pass
   - Commit with prefix `feat:` or `fix:`

3. **Refactor Phase**: Improve the code
   - Clean up code while keeping tests green
   - Optimize, extract functions, improve readability
   - Commit with prefix `refactor:`

### Example TDD Cycle

```bash
# 1. Write test (Red)
# Edit: pkg/protocol/message_test.go
go test ./pkg/protocol
# Test fails ✗
git add pkg/protocol/message_test.go
git commit -m "test: add message encoding tests"

# 2. Implement (Green)
# Edit: pkg/protocol/message.go
go test ./pkg/protocol
# Test passes ✓
git add pkg/protocol/message.go
git commit -m "feat: implement message encoding"

# 3. Refactor (optional)
# Improve code quality
git add pkg/protocol/message.go
git commit -m "refactor: extract encoding logic"
```

## Commit Message Convention

This project uses [Conventional Commits](https://www.conventionalcommits.org/).

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: A new feature
- `fix`: A bug fix
- `test`: Adding or updating tests
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code (formatting, etc)
- `perf`: A code change that improves performance
- `build`: Changes that affect the build system or external dependencies
- `ci`: Changes to CI configuration files and scripts

### Examples

```
feat: add message broadcasting to server

test: add integration tests for multi-client communication

fix: resolve race condition in client disconnect

docs: update API documentation for protocol package
```

## Code Style

### Go Guidelines

- Follow standard Go conventions and idioms
- Use `gofmt` to format code
- Keep functions small and focused
- Use meaningful variable and function names
- Add comments for public APIs

### Concurrency

- Always protect shared state with mutexes
- Use channels for communication between goroutines
- Avoid goroutine leaks - ensure all goroutines terminate properly
- Use `sync.WaitGroup` for coordinating goroutine completion

### Error Handling

- Always check and handle errors
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Return errors rather than panicking
- Log errors at appropriate levels

## Testing Guidelines

### Test Organization

- Keep tests in `*_test.go` files
- Use table-driven tests for multiple test cases
- Name test functions descriptively: `TestFunctionName_Scenario`

### Test Coverage

- Write unit tests for all packages
- Write integration tests for cross-component functionality
- Aim for high test coverage, but focus on meaningful tests

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./pkg/protocol -v

# Run with race detector
go test -race ./...
```

## Pull Request Process

1. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Develop using TDD**:
   - Write tests first
   - Implement functionality
   - Ensure all tests pass

3. **Verify code quality**:
   ```bash
   # Run all tests
   go test ./...

   # Check formatting
   gofmt -d .

   # Run linters (if available)
   golangci-lint run
   ```

4. **Commit with conventional commits**:
   ```bash
   git add <files>
   git commit -m "feat: add your feature"
   ```

5. **Push and create PR**:
   ```bash
   git push origin feature/your-feature-name
   ```

6. **PR Requirements**:
   - All tests must pass
   - Code must follow TDD approach (visible in commit history)
   - Commits must follow conventional commit format
   - Code must be properly formatted
   - Documentation must be updated if needed

## Project Structure

Understanding the project structure helps with contributions:

```
toy-socket-chat/
├── cmd/                    # Command-line applications
│   ├── server/            # Server executable
│   └── client/            # Client executable
├── internal/              # Private application code
│   ├── server/           # Server implementation
│   └── client/           # Client implementation
├── pkg/                   # Public library code
│   └── protocol/         # Message protocol (reusable)
├── test/                  # Integration tests
├── doc/                   # Documentation
└── CONTRIBUTING.md        # This file
```

### Package Guidelines

- `cmd/`: Entry points, CLI parsing, minimal logic
- `internal/`: Core business logic, not importable by external projects
- `pkg/`: Reusable packages that could be imported by other projects
- `test/`: End-to-end and integration tests

## Getting Help

- Check existing documentation in `doc/`
- Review commit history to understand TDD patterns
- Look at existing tests for examples
- Open an issue for questions or discussions

## Code Review Guidelines

### For Contributors

- Keep PRs focused and small
- Provide clear descriptions
- Respond to feedback promptly
- Update your PR based on review comments

### For Reviewers

- Be constructive and respectful
- Check for TDD compliance (test commits before implementation commits)
- Verify test coverage
- Ensure code follows project conventions
- Test the changes locally if needed

## License

By contributing, you agree that your contributions will be licensed under the [zlib](./LICENSE) License.
