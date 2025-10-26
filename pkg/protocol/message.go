package protocol

//go:generate protoc --go_out=pb --go_opt=paths=source_relative --proto_path=../../proto ../../proto/message.proto

import (
	"fmt"

	"github.com/omochice/toy-socket-chat/pkg/protocol/pb"
	"google.golang.org/protobuf/proto"
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

// Encode encodes the message into bytes using protobuf
func (m *Message) Encode() ([]byte, error) {
	pbMsg := m.toProto()
	data, err := proto.Marshal(pbMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}
	return data, nil
}

// Decode decodes bytes into a message using protobuf
func (m *Message) Decode(data []byte) error {
	pbMsg := &pb.Message{}
	if err := proto.Unmarshal(data, pbMsg); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}
	m.fromProto(pbMsg)
	return nil
}

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
