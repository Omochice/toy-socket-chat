package protocol

import (
	"testing"

	pb "github.com/omochice/toy-socket-chat/pkg/protocol/pb"
)

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
		name      string
		goType    MessageType
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
