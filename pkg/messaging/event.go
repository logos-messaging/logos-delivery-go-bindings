package messaging

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// ConnectionStatus reports the node's overall connectivity.
type ConnectionStatus int

const (
	Disconnected ConnectionStatus = iota
	PartiallyConnected
	Connected
)

func (s ConnectionStatus) String() string {
	switch s {
	case Disconnected:
		return "Disconnected"
	case PartiallyConnected:
		return "PartiallyConnected"
	case Connected:
		return "Connected"
	default:
		return fmt.Sprintf("ConnectionStatus(%d)", int(s))
	}
}

// Message is a received message (the underlying WakuMessage).
type Message struct {
	ContentTopic ContentTopic
	Payload      []byte
	Meta         []byte
	Version      uint32
	// Timestamp is sender-generated, in nanoseconds.
	Timestamp int64
	Ephemeral bool
}

// Event is the sealed interface implemented by every Messaging API event
// delivered on MessagingClient.Events(). Consumers type-switch over the concrete types.
type Event interface {
	isMessagingEvent()
}

// MessageReceivedEvent is emitted when a message is received from the network.
type MessageReceivedEvent struct {
	MessageHash string
	Message     Message
}

// MessageSentEvent is emitted when a message has been accepted and queued for
// delivery by the send service.
type MessageSentEvent struct {
	RequestID   RequestID
	MessageHash string
}

// MessagePropagatedEvent is emitted when a message has been propagated to
// neighbouring nodes.
type MessagePropagatedEvent struct {
	RequestID   RequestID
	MessageHash string
}

// MessageErrorEvent is emitted when sending or propagating a message fails.
type MessageErrorEvent struct {
	RequestID   RequestID
	MessageHash string
	Err         string
}

// ConnectionStatusEvent is emitted when the node's connectivity changes.
type ConnectionStatusEvent struct {
	Status ConnectionStatus
}

func (MessageReceivedEvent) isMessagingEvent()   {}
func (MessageSentEvent) isMessagingEvent()       {}
func (MessagePropagatedEvent) isMessagingEvent() {}
func (MessageErrorEvent) isMessagingEvent()      {}
func (ConnectionStatusEvent) isMessagingEvent()  {}

// wireBytes decodes a byte field that liblogosdelivery serialises with
// std/json defaults. On receive (WakuMessage) that is a JSON array of byte
// integers; we also accept a base64 string and null for robustness.
type wireBytes []byte

func (b *wireBytes) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*b = nil
		return nil
	}
	if data[0] == '[' {
		var nums []int
		if err := json.Unmarshal(data, &nums); err != nil {
			return err
		}
		out := make([]byte, len(nums))
		for i, n := range nums {
			out[i] = byte(n)
		}
		*b = out
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	*b = dec
	return nil
}

// decodeEvent parses a flat event JSON string from liblogosdelivery into a
// typed Event. Unknown event types return a nil Event and no error so callers
// can ignore them.
func decodeEvent(eventJSON string) (Event, error) {
	var head struct {
		EventType string `json:"eventType"`
	}
	if err := json.Unmarshal([]byte(eventJSON), &head); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}

	switch head.EventType {
	case "message_received":
		var e struct {
			MessageHash string `json:"messageHash"`
			Message     struct {
				ContentTopic string    `json:"contentTopic"`
				Payload      wireBytes `json:"payload"`
				Meta         wireBytes `json:"meta"`
				Version      uint32    `json:"version"`
				Timestamp    int64     `json:"timestamp"`
				Ephemeral    bool      `json:"ephemeral"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(eventJSON), &e); err != nil {
			return nil, fmt.Errorf("decode message_received: %w", err)
		}
		return MessageReceivedEvent{
			MessageHash: e.MessageHash,
			Message: Message{
				ContentTopic: e.Message.ContentTopic,
				Payload:      e.Message.Payload,
				Meta:         e.Message.Meta,
				Version:      e.Message.Version,
				Timestamp:    e.Message.Timestamp,
				Ephemeral:    e.Message.Ephemeral,
			},
		}, nil

	case "message_sent":
		var e struct {
			RequestID   string `json:"requestId"`
			MessageHash string `json:"messageHash"`
		}
		if err := json.Unmarshal([]byte(eventJSON), &e); err != nil {
			return nil, fmt.Errorf("decode message_sent: %w", err)
		}
		return MessageSentEvent{RequestID: RequestID(e.RequestID), MessageHash: e.MessageHash}, nil

	case "message_propagated":
		var e struct {
			RequestID   string `json:"requestId"`
			MessageHash string `json:"messageHash"`
		}
		if err := json.Unmarshal([]byte(eventJSON), &e); err != nil {
			return nil, fmt.Errorf("decode message_propagated: %w", err)
		}
		return MessagePropagatedEvent{RequestID: RequestID(e.RequestID), MessageHash: e.MessageHash}, nil

	case "message_error":
		var e struct {
			RequestID   string `json:"requestId"`
			MessageHash string `json:"messageHash"`
			Error       string `json:"error"`
		}
		if err := json.Unmarshal([]byte(eventJSON), &e); err != nil {
			return nil, fmt.Errorf("decode message_error: %w", err)
		}
		return MessageErrorEvent{RequestID: RequestID(e.RequestID), MessageHash: e.MessageHash, Err: e.Error}, nil

	case "connection_status_change":
		var e struct {
			ConnectionStatus string `json:"connectionStatus"`
		}
		if err := json.Unmarshal([]byte(eventJSON), &e); err != nil {
			return nil, fmt.Errorf("decode connection_status_change: %w", err)
		}
		return ConnectionStatusEvent{Status: parseConnectionStatus(e.ConnectionStatus)}, nil

	default:
		return nil, nil
	}
}

func parseConnectionStatus(s string) ConnectionStatus {
	switch s {
	case "Connected":
		return Connected
	case "PartiallyConnected":
		return PartiallyConnected
	default:
		return Disconnected
	}
}
