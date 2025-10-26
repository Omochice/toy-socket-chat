package protocol

//go:generate protoc --go_out=. --go_opt=paths=source_relative --proto_path=pb pb/message.proto

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// MessageType represents the type of message
type MessageType int

const (
	MessageTypeText MessageType = iota
	MessageTypeJoin
	MessageTypeLeave
)

// String returns the string representation of MessageType
func (mt MessageType) String() string {
	switch mt {
	case MessageTypeText:
		return "TEXT"
	case MessageTypeJoin:
		return "JOIN"
	case MessageTypeLeave:
		return "LEAVE"
	default:
		return "UNKNOWN"
	}
}

// Message represents a chat message
type Message struct {
	Type    MessageType
	Sender  string
	Content string
}

// Encode encodes the message into bytes using gob encoding
func (m *Message) Encode() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(m); err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}
	return buf.Bytes(), nil
}

// Decode decodes bytes into a message using gob decoding
func (m *Message) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	if err := decoder.Decode(m); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}
	return nil
}
