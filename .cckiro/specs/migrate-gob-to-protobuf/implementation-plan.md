# Implementation Plan: Migrate from gob to Protocol Buffers

## 1. Overview

This document provides a step-by-step implementation plan for migrating the message encoding/decoding mechanism from Go's `encoding/gob` to Protocol Buffers (protobuf).

## 2. Prerequisites

### 2.1 Tools Required

All tools are already installed via devbox:
- ✅ `protoc` (protobuf@32.1)
- ✅ `protoc-gen-go` (protoc-gen-go@1.36.10)
- ✅ `google.golang.org/protobuf` (v1.36.10 in go.mod)

### 2.2 Current State Verification

Before starting, verify:
```bash
# Check current tests pass
go test ./pkg/protocol/...

# Check golangci-lint passes
golangci-lint run ./pkg/protocol/...

# Check build succeeds
go build ./...
```

## 3. Implementation Steps

### Step 1: Create Protocol Buffer Directory Structure

**Goal:** Create `pkg/protocol/pb/` subdirectory for protobuf files

**Actions:**
1. Create directory `pkg/protocol/pb/`

**Commands:**
```bash
mkdir -p pkg/protocol/pb
```

**Verification:**
```bash
ls -la pkg/protocol/pb/
```

**Commit:**
- Type: `build`
- Message: `build: create pb directory for protobuf files`
- Body:
  ```
  Create pkg/protocol/pb/ subdirectory to separate generated protobuf
  code from manual Go code. This prevents naming conflicts between
  protobuf-generated types and existing Go types.
  ```

**Files to stage:**
- (No files to commit, just directory creation - will be committed with .proto file in next step)

---

### Step 2: Create Protocol Buffer Schema

**Goal:** Define the Message structure in Protocol Buffers

**Actions:**
1. Create `pkg/protocol/pb/message.proto` with the schema

**File:** `pkg/protocol/pb/message.proto`
```protobuf
syntax = "proto3";

package protocol;

option go_package = "github.com/omochice/toy-socket-chat/pkg/protocol/pb";

// MessageType represents the type of message
enum MessageType {
  // Text message from a user
  MESSAGE_TYPE_TEXT = 0;
  // User joined notification
  MESSAGE_TYPE_JOIN = 1;
  // User left notification
  MESSAGE_TYPE_LEAVE = 2;
}

// Message represents a chat message
message Message {
  // Type of the message
  MessageType type = 1;
  // Username of the sender
  string sender = 2;
  // Content of the message (empty for JOIN/LEAVE)
  string content = 3;
}
```

**Verification:**
```bash
# Check file exists
cat pkg/protocol/pb/message.proto

# Verify syntax (will be done in next step with protoc)
```

**Commit:**
- Type: `build`
- Message: `build: add protocol buffer schema for Message`
- Body:
  ```
  Define Message and MessageType in protobuf schema to replace gob
  encoding. The schema maintains the same structure as the existing
  Go types (Type, Sender, Content) to ensure compatibility.

  Using proto3 syntax as it is simpler and is the current standard.
  Field numbers 1-3 are used for efficient encoding (1-byte tags).
  ```

**Files to stage:**
- `pkg/protocol/pb/message.proto`

---

### Step 3: Update devbox.json for Protobuf Generation

**Goal:** Update the protobuf generation script to use the new pb/ subdirectory

**Actions:**
1. Update `generate:proto` script in `devbox.json`

**Current:**
```json
"generate:proto": "protoc --go_out=. --go_opt=paths=source_relative pkg/protocol/message.proto"
```

**New:**
```json
"generate:proto": "protoc --go_out=. --go_opt=paths=source_relative --proto_path=pkg/protocol/pb pkg/protocol/pb/message.proto"
```

**Verification:**
```bash
# Check devbox.json is valid
cat devbox.json | jq .
```

**Commit:**
- Type: `build`
- Message: `build: update protobuf generation script for pb subdirectory`
- Body:
  ```
  Update the generate:proto script to point to the new
  pkg/protocol/pb/message.proto location. This ensures protoc generates
  code in the correct subdirectory, avoiding naming conflicts.
  ```

**Files to stage:**
- `devbox.json`

---

### Step 4: Generate Protobuf Code

**Goal:** Generate Go code from the protobuf schema

**Actions:**
1. Run protobuf code generation
2. Verify generated file

**Commands:**
```bash
devbox run generate:proto
```

**Expected output file:**
- `pkg/protocol/pb/message.pb.go`

**Verification:**
```bash
# Check file exists
ls -la pkg/protocol/pb/message.pb.go

# Check file is valid Go code
go build ./pkg/protocol/pb/...
```

**Commit:**
- Type: `build`
- Message: `build: generate Go code from protobuf schema`
- Body:
  ```
  Generate message.pb.go using protoc. This file contains the
  protobuf-generated types and Marshal/Unmarshal methods that will be
  used internally for message serialization.

  The generated code is committed to ensure all contributors have
  identical generated code and to make builds reproducible.
  ```

**Files to stage:**
- `pkg/protocol/pb/message.pb.go`

---

### Step 5: Add go:generate Directive

**Goal:** Add automation for protobuf code regeneration

**Actions:**
1. Add `//go:generate` comment to `pkg/protocol/message.go`

**Add to top of `pkg/protocol/message.go` (after package declaration):**
```go
package protocol

//go:generate protoc --go_out=. --go_opt=paths=source_relative --proto_path=pb pb/message.proto

import (
	...
)
```

**Verification:**
```bash
# Test go:generate works
cd pkg/protocol
go generate
git diff  # Should show no changes if .pb.go is already up to date
```

**Commit:**
- Type: `build`
- Message: `build: add go:generate directive for protobuf code generation`
- Body:
  ```
  Add go:generate directive to automate protobuf code generation.
  Developers can run 'go generate ./...' to regenerate protobuf code
  when message.proto is modified.

  This makes the code generation process self-documenting and ensures
  the generation command is version-controlled alongside the code.
  ```

**Files to stage:**
- `pkg/protocol/message.go`

---

### Step 6: Add Tests for Conversion Functions (TDD Red Phase)

**Goal:** Write tests for conversion functions before implementing them

**Actions:**
1. Add tests for `toProto()` and `fromProto()` to `message_test.go`

**Add to `pkg/protocol/message_test.go`:**
```go
func TestMessage_toProto(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
		want *pb.Message
	}{
		{
			name: "convert text message to proto",
			msg: Message{
				Type:    MessageTypeText,
				Sender:  "user1",
				Content: "Hello",
			},
			want: &pb.Message{
				Type:    pb.MessageType_MESSAGE_TYPE_TEXT,
				Sender:  "user1",
				Content: "Hello",
			},
		},
		{
			name: "convert join message to proto",
			msg: Message{
				Type:    MessageTypeJoin,
				Sender:  "user2",
				Content: "",
			},
			want: &pb.Message{
				Type:    pb.MessageType_MESSAGE_TYPE_JOIN,
				Sender:  "user2",
				Content: "",
			},
		},
		{
			name: "convert leave message to proto",
			msg: Message{
				Type:    MessageTypeLeave,
				Sender:  "user3",
				Content: "",
			},
			want: &pb.Message{
				Type:    pb.MessageType_MESSAGE_TYPE_LEAVE,
				Sender:  "user3",
				Content: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.msg.toProto()
			if got.Type != tt.want.Type {
				t.Errorf("toProto() Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Sender != tt.want.Sender {
				t.Errorf("toProto() Sender = %v, want %v", got.Sender, tt.want.Sender)
			}
			if got.Content != tt.want.Content {
				t.Errorf("toProto() Content = %v, want %v", got.Content, tt.want.Content)
			}
		})
	}
}

func TestMessage_fromProto(t *testing.T) {
	tests := []struct {
		name  string
		proto *pb.Message
		want  Message
	}{
		{
			name: "convert proto text message to Message",
			proto: &pb.Message{
				Type:    pb.MessageType_MESSAGE_TYPE_TEXT,
				Sender:  "user1",
				Content: "Hello",
			},
			want: Message{
				Type:    MessageTypeText,
				Sender:  "user1",
				Content: "Hello",
			},
		},
		{
			name: "convert proto join message to Message",
			proto: &pb.Message{
				Type:    pb.MessageType_MESSAGE_TYPE_JOIN,
				Sender:  "user2",
				Content: "",
			},
			want: Message{
				Type:    MessageTypeJoin,
				Sender:  "user2",
				Content: "",
			},
		},
		{
			name: "convert proto leave message to Message",
			proto: &pb.Message{
				Type:    pb.MessageType_MESSAGE_TYPE_LEAVE,
				Sender:  "user3",
				Content: "",
			},
			want: Message{
				Type:    MessageTypeLeave,
				Sender:  "user3",
				Content: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Message
			got.fromProto(tt.proto)
			if got.Type != tt.want.Type {
				t.Errorf("fromProto() Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Sender != tt.want.Sender {
				t.Errorf("fromProto() Sender = %v, want %v", got.Sender, tt.want.Sender)
			}
			if got.Content != tt.want.Content {
				t.Errorf("fromProto() Content = %v, want %v", got.Content, tt.want.Content)
			}
		})
	}
}

func TestMessageTypeConversion(t *testing.T) {
	tests := []struct {
		name     string
		goType   MessageType
		protoType pb.MessageType
	}{
		{"text type", MessageTypeText, pb.MessageType_MESSAGE_TYPE_TEXT},
		{"join type", MessageTypeJoin, pb.MessageType_MESSAGE_TYPE_JOIN},
		{"leave type", MessageTypeLeave, pb.MessageType_MESSAGE_TYPE_LEAVE},
	}

	for _, tt := range tests {
		t.Run(tt.name+" to proto", func(t *testing.T) {
			got := messageTypeToProto(tt.goType)
			if got != tt.protoType {
				t.Errorf("messageTypeToProto(%v) = %v, want %v", tt.goType, got, tt.protoType)
			}
		})

		t.Run(tt.name+" from proto", func(t *testing.T) {
			got := messageTypeFromProto(tt.protoType)
			if got != tt.goType {
				t.Errorf("messageTypeFromProto(%v) = %v, want %v", tt.protoType, got, tt.goType)
			}
		})
	}
}
```

**Also add import:**
```go
import (
	"testing"

	"github.com/omochice/toy-socket-chat/pkg/protocol"
	pb "github.com/omochice/toy-socket-chat/pkg/protocol/pb"
)
```

**Verification:**
```bash
# Tests should FAIL (Red phase)
go test ./pkg/protocol/... -v
```

**Expected:** Tests fail because `toProto()`, `fromProto()`, `messageTypeToProto()`, and `messageTypeFromProto()` don't exist yet.

**Commit:**
- Type: `test`
- Message: `test: add tests for protobuf conversion functions`
- Body:
  ```
  Add tests for converting between Go types and protobuf types.
  These tests verify that:
  - Message.toProto() correctly converts Message to pb.Message
  - Message.fromProto() correctly populates Message from pb.Message
  - MessageType enum conversion works bidirectionally

  These tests are expected to fail (TDD Red phase) until the
  conversion functions are implemented.
  ```

**Files to stage:**
- `pkg/protocol/message_test.go`

---

### Step 7: Implement Conversion Functions (TDD Green Phase)

**Goal:** Implement conversion functions to make tests pass

**Actions:**
1. Add conversion functions to `pkg/protocol/message.go`

**Add to `pkg/protocol/message.go`:**
```go
import (
	"fmt"

	pb "github.com/omochice/toy-socket-chat/pkg/protocol/pb"
	"google.golang.org/protobuf/proto"
)

// toProto converts the Message to protobuf Message.
// This conversion isolates protobuf implementation details from the public API.
func (m *Message) toProto() *pb.Message {
	return &pb.Message{
		Type:    messageTypeToProto(m.Type),
		Sender:  m.Sender,
		Content: m.Content,
	}
}

// fromProto populates the Message from protobuf Message.
// This conversion isolates protobuf implementation details from the public API.
func (m *Message) fromProto(pbMsg *pb.Message) {
	m.Type = messageTypeFromProto(pbMsg.Type)
	m.Sender = pbMsg.Sender
	m.Content = pbMsg.Content
}

// messageTypeToProto converts MessageType to protobuf enum.
// Default case returns TEXT type rather than an error to ensure graceful
// degradation for unknown message types (safest option for chat system).
func messageTypeToProto(mt MessageType) pb.MessageType {
	switch mt {
	case MessageTypeText:
		return pb.MessageType_MESSAGE_TYPE_TEXT
	case MessageTypeJoin:
		return pb.MessageType_MESSAGE_TYPE_JOIN
	case MessageTypeLeave:
		return pb.MessageType_MESSAGE_TYPE_LEAVE
	default:
		return pb.MessageType_MESSAGE_TYPE_TEXT
	}
}

// messageTypeFromProto converts protobuf enum to MessageType.
// Default case returns MessageTypeText rather than an error to ensure graceful
// degradation for unknown enum values (safest option for chat system).
func messageTypeFromProto(pbType pb.MessageType) MessageType {
	switch pbType {
	case pb.MessageType_MESSAGE_TYPE_TEXT:
		return MessageTypeText
	case pb.MessageType_MESSAGE_TYPE_JOIN:
		return MessageTypeJoin
	case pb.MessageType_MESSAGE_TYPE_LEAVE:
		return MessageTypeLeave
	default:
		return MessageTypeText
	}
}
```

**Note:** Keep existing gob-based `Encode()` and `Decode()` for now.

**Verification:**
```bash
# Tests should PASS (Green phase)
go test ./pkg/protocol/... -v

# Specifically verify conversion tests pass
go test ./pkg/protocol/... -v -run TestMessage_toProto
go test ./pkg/protocol/... -v -run TestMessage_fromProto
go test ./pkg/protocol/... -v -run TestMessageTypeConversion
```

**Commit:**
- Type: `refactor`
- Message: `refactor: add protobuf conversion functions`
- Body:
  ```
  Implement conversion layer between Go types and protobuf types:
  - toProto/fromProto for Message struct conversion
  - messageTypeToProto/messageTypeFromProto for enum conversion

  These functions isolate protobuf implementation details from the
  public API, maintaining the existing method signatures while enabling
  protobuf serialization internally.

  Unknown enum values default to TEXT type (graceful degradation)
  rather than returning errors, ensuring the chat system remains
  functional even when receiving unexpected message types.
  ```

**Files to stage:**
- `pkg/protocol/message.go`

---

### Step 8: Implement Protobuf-based Encode() (TDD Green Phase)

**Goal:** Replace gob encoding with protobuf encoding

**Actions:**
1. Update `Encode()` method in `pkg/protocol/message.go`

**Replace:**
```go
// Encode encodes the message into bytes using gob encoding
func (m *Message) Encode() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(m); err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}
	return buf.Bytes(), nil
}
```

**With:**
```go
// Encode encodes the message into bytes using protobuf
func (m *Message) Encode() ([]byte, error) {
	pbMsg := m.toProto()
	data, err := proto.Marshal(pbMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}
	return data, nil
}
```

**Note:** Don't remove gob imports yet, as `Decode()` still uses them.

**Verification:**
```bash
# Existing Encode tests should still pass
go test ./pkg/protocol/... -v -run TestMessage_Encode

# However, Decode and RoundTrip tests may fail now
# (expected until Decode is updated)
go test ./pkg/protocol/... -v
```

**Expected:** `TestMessage_Encode` passes, but `TestMessage_Decode` and `TestMessage_EncodeDecodeRoundTrip` may fail because encoding changed but decoding hasn't.

**Commit:**
- Type: `refactor`
- Message: `refactor: implement protobuf encoding in Message.Encode()`
- Body:
  ```
  Replace gob encoding with protobuf encoding in Message.Encode().
  The method signature remains unchanged to maintain API compatibility.

  Using protobuf instead of gob because:
  - Better cross-language compatibility
  - More efficient binary format
  - Explicit schema definition (message.proto)
  - Better tooling support

  Note: Decode() still uses gob, so round-trip tests will fail until
  Decode() is also updated to use protobuf.
  ```

**Files to stage:**
- `pkg/protocol/message.go`

---

### Step 9: Implement Protobuf-based Decode() (TDD Green Phase)

**Goal:** Replace gob decoding with protobuf decoding

**Actions:**
1. Update `Decode()` method in `pkg/protocol/message.go`

**Replace:**
```go
// Decode decodes bytes into a message using gob decoding
func (m *Message) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	if err := decoder.Decode(m); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}
	return nil
}
```

**With:**
```go
// Decode decodes bytes into a message using protobuf
func (m *Message) Decode(data []byte) error {
	pbMsg := &pb.Message{}
	if err := proto.Unmarshal(data, pbMsg); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}
	m.fromProto(pbMsg)
	return nil
}
```

**Verification:**
```bash
# ALL tests should now PASS
go test ./pkg/protocol/... -v

# Specifically verify round-trip test passes
go test ./pkg/protocol/... -v -run TestMessage_EncodeDecodeRoundTrip
```

**Expected:** All tests pass now that both encoding and decoding use protobuf.

**Commit:**
- Type: `refactor`
- Message: `refactor: implement protobuf decoding in Message.Decode()`
- Body:
  ```
  Replace gob decoding with protobuf decoding in Message.Decode().
  The method signature remains unchanged to maintain API compatibility.

  With both Encode() and Decode() now using protobuf, the migration
  from gob is functionally complete. All existing tests pass without
  modification to test logic.

  Using protobuf instead of gob for consistency with Encode() and to
  gain the benefits of protobuf's cross-language compatibility and
  explicit schema definition.
  ```

**Files to stage:**
- `pkg/protocol/message.go`

---

### Step 10: Remove gob Imports and Clean Up (TDD Refactor Phase)

**Goal:** Remove unused gob-related imports

**Actions:**
1. Remove `bytes` and `encoding/gob` imports from `message.go`

**Current imports:**
```go
import (
	"bytes"
	"encoding/gob"
	"fmt"

	pb "github.com/omochice/toy-socket-chat/pkg/protocol/pb"
	"google.golang.org/protobuf/proto"
)
```

**New imports:**
```go
import (
	"fmt"

	pb "github.com/omochice/toy-socket-chat/pkg/protocol/pb"
	"google.golang.org/protobuf/proto"
)
```

**Verification:**
```bash
# Verify no gob imports remain
grep -r "encoding/gob" pkg/protocol/message.go
# Should return nothing

# Verify tests still pass
go test ./pkg/protocol/... -v

# Verify build succeeds
go build ./pkg/protocol/...
```

**Commit:**
- Type: `refactor`
- Message: `refactor: remove gob dependencies from message.go`
- Body:
  ```
  Remove unused gob and bytes imports after migrating to protobuf.
  All encoding/decoding now uses protobuf, making these imports
  unnecessary.

  Verify no gob-related code remains in the protocol package.
  ```

**Files to stage:**
- `pkg/protocol/message.go`

---

### Step 11: Run Integration Tests

**Goal:** Verify that server, client, and end-to-end tests still pass

**Actions:**
1. Run all integration tests

**Commands:**
```bash
# Run server tests
go test ./internal/server/... -v

# Run client tests
go test ./internal/client/... -v

# Run end-to-end tests
go test ./test/... -v

# Run all tests
go test ./... -v
```

**Verification:**
- All tests should pass
- If any tests fail, investigate and fix

**Expected:** All tests pass because we maintained the existing API surface (`Encode()` and `Decode()` signatures unchanged).

**Commit:** No commit needed if all tests pass. If fixes are required, commit with appropriate type.

---

### Step 12: Run golangci-lint

**Goal:** Verify code quality and linting rules

**Actions:**
1. Run golangci-lint

**Commands:**
```bash
# Run golangci-lint (using devbox script)
devbox run check:golangci-lint

# Or directly
golangci-lint run ./...
```

**Verification:**
- No linting errors should be reported
- If errors are found, fix them

**Commit:** If fixes are required:
- Type: `style` (for formatting) or `refactor` (for code changes)
- Message: Describe what was fixed
- Body: Explain why the change was needed

---

### Step 13: Update Documentation

**Goal:** Document the protobuf usage in project documentation

**Actions:**
1. Update `architecture.md` (if it exists) or create developer documentation

**File:** Update relevant documentation (e.g., `docs/architecture.md`, `CONTRIBUTING.md`, or `README.md`)

**Add/Update sections:**
```markdown
## Message Serialization

Messages are serialized using Protocol Buffers (protobuf) for transmission over TCP connections.

### Schema Definition

The message schema is defined in `pkg/protocol/pb/message.proto`:
- `MessageType`: Enum for message types (TEXT, JOIN, LEAVE)
- `Message`: Message structure with type, sender, and content fields

### Code Generation

Protobuf code is auto-generated from the schema. To regenerate:

```bash
# Using devbox
devbox run generate:proto

# Or using go generate
go generate ./pkg/protocol/...
```

Generated code is committed to the repository to ensure reproducible builds.

### API Usage

The public API (`Message.Encode()` and `Message.Decode()`) remains unchanged. Protobuf is used internally for serialization.
```

**Verification:**
```bash
# Verify markdown is well-formed
cat docs/architecture.md  # or relevant doc file
```

**Commit:**
- Type: `docs`
- Message: `docs: update documentation for protobuf usage`
- Body:
  ```
  Document the migration from gob to protobuf in architecture docs.
  Explain the protobuf schema location, code generation process,
  and how to regenerate protobuf code when the schema changes.

  This helps future developers understand the serialization mechanism
  and know how to modify the message schema if needed.
  ```

**Files to stage:**
- Documentation files (e.g., `docs/architecture.md`, `CONTRIBUTING.md`, `README.md`)

---

## 4. Post-Implementation Verification

After completing all steps, perform final verification:

### 4.1 Test Suite
```bash
# Run all tests
go test ./... -v

# Check test coverage (optional)
go test ./pkg/protocol/... -cover
```

### 4.2 Build Verification
```bash
# Build all packages
go build ./...

# Run application (if applicable)
# Manual testing: start server, connect client, send messages
```

### 4.3 Linting
```bash
# Run golangci-lint
devbox run check:golangci-lint
```

### 4.4 Code Review Checklist
- [ ] All unit tests pass (`go test ./pkg/protocol/...`)
- [ ] All integration tests pass (`go test ./...`)
- [ ] golangci-lint passes (`devbox run check:golangci-lint`)
- [ ] Build succeeds (`go build ./...`)
- [ ] No gob imports remain in `pkg/protocol/message.go`
- [ ] Generated protobuf code is committed (`pkg/protocol/pb/message.pb.go`)
- [ ] Documentation is updated
- [ ] All commits follow conventional commit format
- [ ] All commit messages include WHY in the body
- [ ] go:generate directive is present in `message.go`
- [ ] devbox.json `generate:proto` script is updated

### 4.5 Success Criteria Verification

From requirements.md:

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

---

## 5. Rollback Plan

If issues are discovered during implementation:

### 5.1 Rollback Individual Steps

Each step is independently committable and revertable:

```bash
# Revert the last commit
git revert HEAD

# Revert a specific commit
git revert <commit-hash>

# Revert multiple commits (interactive)
git rebase -i HEAD~<number-of-commits>
```

### 5.2 Complete Rollback

To completely roll back the migration:

```bash
# Find the commit before the migration started
git log --oneline

# Reset to that commit (DANGEROUS - loses all changes)
git reset --hard <commit-before-migration>

# Or create a revert commit for each migration commit (SAFER)
git revert <commit-range>
```

### 5.3 Critical Issues

If critical issues are found after merging:

1. Create a hotfix branch
2. Revert the merge commit
3. Fix issues in a new feature branch
4. Re-merge after verification

---

## 6. Timeline and Effort Estimate

| Step | Description | Estimated Time |
|------|-------------|----------------|
| 1-5 | Setup and code generation | 30 minutes |
| 6-7 | Conversion functions (TDD) | 30 minutes |
| 8-10 | Encode/Decode migration | 30 minutes |
| 11-12 | Testing and linting | 15 minutes |
| 13 | Documentation | 15 minutes |
| **Total** | **Complete migration** | **~2 hours** |

**Note:** This assumes no unexpected issues. Add buffer time for debugging if needed.

---

## 7. Dependencies and Blockers

### 7.1 Dependencies
- ✅ protoc installed (available in devbox)
- ✅ protoc-gen-go installed (available in devbox)
- ✅ google.golang.org/protobuf in go.mod

### 7.2 Potential Blockers
- **None identified** - all prerequisites are met

---

## 8. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Tests fail after migration | Low | High | Follow TDD strictly, verify after each step |
| Integration issues | Low | High | Run integration tests after encode/decode changes |
| Performance regression | Low | Low | Not a primary concern (can benchmark later if needed) |
| Naming conflicts | Very Low | Medium | Already mitigated by using pb/ subdirectory |
| Generated code conflicts | Very Low | Low | Never manually edit .pb.go files |

---

## 9. Open Questions

### 9.1 Should we add godoc examples?

**Status:** Optional, not required for this migration

**Rationale:** Existing tests serve as examples. Can be added later if needed.

### 9.2 Should we benchmark performance?

**Status:** Out of scope for this migration (per requirements)

**Rationale:** Can be added later if performance becomes a concern.

---

## 10. Next Steps After Completion

After successful migration:

1. Monitor for any issues in production (if applicable)
2. Consider adding protobuf validation (optional enhancement)
3. Consider adding version field to protobuf schema (optional enhancement)
4. Document lessons learned
5. **Execute cleanup phase:** Delete `.cckiro/specs/migrate-gob-to-protobuf` and commit with `chore(spec):` prefix

---

## 11. Summary

This implementation plan provides a step-by-step guide to migrate from gob to protobuf encoding. The migration:

- Maintains existing API surface (no breaking changes)
- Follows TDD principles (Red-Green-Refactor)
- Uses atomic commits with conventional commit messages
- Includes comprehensive testing at each step
- Documents all changes and decisions

**Total commits:** ~13 commits (may vary based on issues encountered)

**Total time:** ~2 hours (estimated)

**Risk level:** Low (TDD approach, small incremental changes, full test coverage)
