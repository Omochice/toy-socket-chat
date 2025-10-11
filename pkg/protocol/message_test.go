package protocol_test

import (
	"testing"

	"github.com/omochice/tcp-socket/pkg/protocol"
)

func TestMessage_Encode(t *testing.T) {
	tests := []struct {
		name     string
		msg      protocol.Message
		wantErr  bool
	}{
		{
			name: "encode text message successfully",
			msg: protocol.Message{
				Type:    protocol.MessageTypeText,
				Sender:  "user1",
				Content: "Hello, World!",
			},
			wantErr: false,
		},
		{
			name: "encode join message successfully",
			msg: protocol.Message{
				Type:    protocol.MessageTypeJoin,
				Sender:  "user2",
				Content: "",
			},
			wantErr: false,
		},
		{
			name: "encode leave message successfully",
			msg: protocol.Message{
				Type:    protocol.MessageTypeLeave,
				Sender:  "user3",
				Content: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.msg.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(data) == 0 {
				t.Error("Message.Encode() returned empty data")
			}
		})
	}
}

func TestMessage_Decode(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    protocol.Message
		wantErr bool
	}{
		{
			name: "decode text message successfully",
			data: func() []byte {
				msg := protocol.Message{
					Type:    protocol.MessageTypeText,
					Sender:  "user1",
					Content: "Hello, World!",
				}
				data, _ := msg.Encode()
				return data
			}(),
			want: protocol.Message{
				Type:    protocol.MessageTypeText,
				Sender:  "user1",
				Content: "Hello, World!",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got protocol.Message
			err := got.Decode(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Type != tt.want.Type {
					t.Errorf("Message.Decode() Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.Sender != tt.want.Sender {
					t.Errorf("Message.Decode() Sender = %v, want %v", got.Sender, tt.want.Sender)
				}
				if got.Content != tt.want.Content {
					t.Errorf("Message.Decode() Content = %v, want %v", got.Content, tt.want.Content)
				}
			}
		})
	}
}

func TestMessage_EncodeDecodeRoundTrip(t *testing.T) {
	original := protocol.Message{
		Type:    protocol.MessageTypeText,
		Sender:  "testuser",
		Content: "Test message content",
	}

	// Encode
	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Decode
	var decoded protocol.Message
	err = decoded.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Compare
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %v, want %v", decoded.Type, original.Type)
	}
	if decoded.Sender != original.Sender {
		t.Errorf("Sender mismatch: got %v, want %v", decoded.Sender, original.Sender)
	}
	if decoded.Content != original.Content {
		t.Errorf("Content mismatch: got %v, want %v", decoded.Content, original.Content)
	}
}

func TestMessageType_String(t *testing.T) {
	tests := []struct {
		name string
		mt   protocol.MessageType
		want string
	}{
		{"text type", protocol.MessageTypeText, "TEXT"},
		{"join type", protocol.MessageTypeJoin, "JOIN"},
		{"leave type", protocol.MessageTypeLeave, "LEAVE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mt.String(); got != tt.want {
				t.Errorf("MessageType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
