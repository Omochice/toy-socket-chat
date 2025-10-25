# Requirements: Migrate from gob to Protocol Buffers

## 1. Overview

This document defines the requirements for migrating the message encoding/decoding mechanism from Go's `encoding/gob` to Protocol Buffers (protobuf).

## 2. Goals

### 2.1 Primary Goals

- **G1**: Replace gob encoding with Protocol Buffers for message serialization
- **G2**: Maintain existing functionality and API surface
- **G3**: Ensure all existing tests continue to pass
- **G4**: Follow TDD (Test-Driven Development) principles throughout the migration
- **G5**: Commit in meaningful, atomic units following conventional commit format

### 2.2 Non-Goals

- Changing the message structure or adding new fields
- Modifying the TCP communication protocol beyond encoding format
- Supporting backward compatibility with gob-encoded messages (clean break)

## 3. Current State

### 3.1 Implementation

- Location: `pkg/protocol/message.go`
- Encoding method: `encoding/gob`
- Message structure:
    - `Type`: MessageType (int enum: TEXT=0, JOIN=1, LEAVE=2)
    - `Sender`: string
    - `Content`: string

### 3.2 Existing Tests

1. `pkg/protocol/message_test.go`:
    - `TestMessage_Encode`: Tests encoding various message types
    - `TestMessage_Decode`: Tests decoding messages
    - `TestMessage_EncodeDecodeRoundTrip`: Tests encode→decode cycle
    - `TestMessageType_String`: Tests MessageType string representation

2. `internal/server/server_test.go`: Server integration tests
3. `internal/client/client_test.go`: Client integration tests
4. `test/integration_test.go`: End-to-end integration tests

## 4. Requirements

### 4.1 Functional Requirements

#### FR1: Protocol Buffer Schema Definition

- **FR1.1**: Create `.proto` file defining the Message structure
- **FR1.2**: Define MessageType enum in proto (TEXT=0, JOIN=1, LEAVE=2)
- **FR1.3**: Define Message with fields: type, sender, content
- **FR1.4**: Use proto3 syntax
- **FR1.5**: Place proto file in appropriate location (e.g., `pkg/protocol/message.proto`)

#### FR2: Code Generation

- **FR2.1**: Generate Go code from proto file using `protoc`
- **FR2.2**: Generated code must be committed to the repository
- **FR2.3**: Provide mechanism to regenerate code (e.g., Make target, go:generate comment)

#### FR3: Encoding/Decoding Implementation

- **FR3.1**: Replace `Message.Encode()` implementation to use protobuf marshaling
- **FR3.2**: Replace `Message.Decode()` implementation to use protobuf unmarshaling
- **FR3.3**: Maintain the same method signatures:
    - `func (m *Message) Encode() ([]byte, error)`
    - `func (m *Message) Decode(data []byte) error`
- **FR3.4**: Return appropriate errors with context

#### FR4: MessageType String Representation

- **FR4.1**: Maintain `MessageType.String()` method
- **FR4.2**: Return same string values: "TEXT", "JOIN", "LEAVE", "UNKNOWN"

#### FR5: Backward Compatibility

- **FR5.1**: No backward compatibility with gob required (clean migration)
- **FR5.2**: All components (client, server, tests) will migrate together

### 4.2 Testing Requirements

#### TR1: Test-Driven Development Approach

- **TR1.1**: Follow Red-Green-Refactor cycle for each change
- **TR1.2**: Write failing tests first (Red)
- **TR1.3**: Implement minimum code to pass tests (Green)
- **TR1.4**: Refactor while keeping tests green (Refactor)

#### TR2: Test Coverage

- **TR2.1**: All existing tests must pass after migration
- **TR2.2**: No reduction in test coverage
- **TR2.3**: Add tests for protobuf-specific edge cases if needed

#### TR3: Test Strategy

- **TR3.1**: Start with unit tests (`pkg/protocol/message_test.go`)
- **TR3.2**: Progress to integration tests
- **TR3.3**: Verify end-to-end tests last

### 4.3 Commit Strategy Requirements

#### CR1: Conventional Commit Format

All commits must follow the conventional commit format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

#### CR2: Commit Type Classification

Commit types must be chosen based on **user impact**, not internal technical changes:

- **feat**: Changes that add new functionality visible to users
    - Example: Adding new message types, new API methods

- **refactor**: Internal changes that don't affect functionality or user experience
    - Example: Replacing gob with protobuf (same functionality, different implementation)

- **test**: Adding or modifying tests
    - Example: Adding new test cases

- **build**: Changes to build system, dependencies, or code generation
    - Example: Adding protoc generation step

- **docs**: Documentation changes
    - Example: Updating architecture.md to reflect protobuf usage

- **fix**: Bug fixes that affect user experience
    - **NOT** for fixing internal implementation issues that users don't see
    - Example: Fixing a message encoding bug that causes message loss

- **chore(deps)**: Adding dependencies
    - Example: updating go.mod

#### CR3: Atomic Commits

- **CR3.1**: Each commit must represent a single, meaningful change
- **CR3.2**: Commits should be as small as possible while remaining meaningful
- **CR3.3**: Each commit should leave the codebase in a working state (compilable)
- **CR3.4**: Tests may fail between commits (TDD Red phase), but this should be documented

#### CR4: Commit Sequence Example

Example commit sequence for this migration:

1. `build: add protocol buffer schema for Message`
    - Add message.proto file

2. `build: generate Go code from protobuf schema`
    - Add generated .pb.go file
    - Update go.mod if needed

3. `test: add failing test for protobuf encoding`
    - Modify existing test to expect protobuf encoding (Red phase)

4. `refactor: implement protobuf encoding in Message.Encode()`
    - Implement protobuf-based encoding (Green phase)

5. `test: add failing test for protobuf decoding`
    - Modify existing test to expect protobuf decoding (Red phase)

6. `refactor: implement protobuf decoding in Message.Decode()`
    - Implement protobuf-based decoding (Green phase)

7. `refactor: remove gob import from message.go`
    - Clean up unused imports

8. `docs: update architecture.md to document protobuf usage`
    - Update documentation

### 4.4 Implementation Requirements

#### IR1: Code Quality

- **IR1.1**: Follow existing code style and conventions
- **IR1.2**: Pass all linter checks (golangci-lint)
- **IR1.3**: Maintain or improve code readability
- **IR1.4**: Code comments MUST explain WHY NOT an alternative approach was chosen
  - Do NOT write WHAT or HOW the code does (should be obvious from code)
  - Do NOT write obvious implementation details
  - DO write reasons for choosing this approach over alternatives
- **IR1.5**: Preserve godoc comments for exported types, functions, and methods
  - Public API documentation must remain complete
  - Follow Go documentation conventions
- **IR1.6**: Test code comments SHOULD clearly describe WHAT is being tested
  - Test comments are an exception to the "why not" rule
  - Test names and comments should make test purpose clear
- **IR1.7**: Write WHY the change was made in commit message body
  - Code comments explain "why not alternative X"
  - Commit messages explain "why this change is needed"

#### IR2: Dependencies

- **IR2.1**: Use `google.golang.org/protobuf` (already in go.mod v1.36.10)
- **IR2.2**: No additional dependencies required
- **IR2.3**: protoc and protoc-gen-go are pre-installed

#### IR3: Build Process

- **IR3.1**: Document protobuf code generation process
- **IR3.2**: Provide automation for code generation (devbox, or go:generate)
- **IR3.3**: Ensure `devbox run build` continues to work

## 5. Success Criteria

The migration is considered successful when:

1. ✅ All existing tests pass
2. ✅ Message encoding uses protobuf instead of gob
3. ✅ Message decoding uses protobuf instead of gob
4. ✅ No gob-related imports remain in message.go
5. ✅ All commits follow conventional commit format
6. ✅ All commits are atomic and meaningful
7. ✅ golangci-lint passes
8. ✅ Documentation is updated
9. ✅ Code generation process is documented
10. ✅ Integration tests pass (server, client, end-to-end)

## 6. Out of Scope

The following are explicitly out of scope for this migration:

1. Adding new message types or fields
2. Changing message structure
3. Modifying TCP framing or protocol
4. Performance optimization
5. Supporting both gob and protobuf simultaneously
6. Migration path for existing gob-encoded data
7. Backward compatibility with gob format

## 7. Constraints

- **C1**: Must use TDD approach (test-first development)
- **C2**: Must follow conventional commit format with user-impact-based types
- **C3**: Must maintain existing API surface (method signatures)
- **C4**: Must keep all existing tests passing
- **C5**: Must not use `git add .` (stage specific files only)
- **C6**: Must write commit messages in English
- **C7**: All commits must include WHY the change was made in the commit body
- **C8**: Code comments must explain WHY NOT alternatives were chosen, not WHAT/HOW
- **C9**: Must preserve godoc comments for public API

## 8. Assumptions

- **A1**: protoc and protoc-gen-go are correctly installed
- **A2**: No external systems depend on gob format
- **A3**: Clean migration (no gradual rollout needed)
- **A4**: All components will be updated simultaneously

## 9. Dependencies

- `google.golang.org/protobuf` v1.36.10 (already installed)
- `protoc` compiler (installed)
- `protoc-gen-go` plugin (installed)

## 10. Risks and Mitigations

| Risk                                                          | Impact | Mitigation                                      |
| ------------------------------------------------------------- | ------ | ----------------------------------------------- |
| Protobuf encoding differs from gob, breaking existing clients | High   | Clean migration, update all components together |
| Generated code conflicts with manual code                     | Medium | Use separate files, clear naming conventions    |
| Test failures during migration                                | Medium | Follow TDD, commit frequently                   |
| Performance regression                                        | Low    | Not a primary concern for this migration        |

## 11. Next Steps

After this requirements document is approved:

1. Create design document (design phase)
2. Create implementation plan (implementation planning phase)
3. Begin implementation (implementation phase) following TDD
