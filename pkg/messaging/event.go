package messaging

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Wire names of the events a MessagingClient listens for. They are the names
// liblogosdelivery registers listeners under, and are unrelated to the
// eventType each event's JSON carries.
const (
	wireMessageReceived        = "onMessageReceived"
	wireMessageSent            = "onMessageSent"
	wireMessagePropagated      = "onMessagePropagated"
	wireMessageError           = "onMessageError"
	wireConnectionStatusChange = "onConnectionStatusChange"
)

// messagingEvents is the set of events a MessagingClient subscribes to.
var messagingEvents = []string{
	wireMessageReceived,
	wireMessageSent,
	wireMessagePropagated,
	wireMessageError,
	wireConnectionStatusChange,
}

// ConnectionStatus reports the node's overall connectivity. It mirrors the Nim
// ConnectionStatus enum.
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

// Message is a message received from the network: the underlying WakuMessage.
type Message struct {
	ContentTopic ContentTopic
	Payload      []byte
	// Meta is an opaque wire-format marker stamped by higher layers.
	Meta []byte
	// Version discriminates payload encryption schemes.
	Version uint32
	// Timestamp is sender-generated, in nanoseconds.
	Timestamp int64
	Ephemeral bool
}

// Event is the sealed interface every event delivered on
// MessagingClient.Events() implements. Consumers type-switch over the concrete
// types; the set only grows, so keep a default branch.
type Event interface {
	isMessagingEvent()
}

// MessageReceivedEvent is emitted when a message arrives from the network on a
// subscribed content topic.
type MessageReceivedEvent struct {
	MessageHash string
	Message     Message
}

// MessageSentEvent is emitted when a message has been accepted by the send
// service and queued for delivery.
type MessageSentEvent struct {
	RequestID   RequestID
	MessageHash string
}

// MessagePropagatedEvent is emitted when a message has reached neighbouring
// nodes on the network.
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

// ConnectionStatusEvent is emitted when the node's overall connectivity
// changes.
type ConnectionStatusEvent struct {
	Status ConnectionStatus
}

func (MessageReceivedEvent) isMessagingEvent()   {}
func (MessageSentEvent) isMessagingEvent()       {}
func (MessagePropagatedEvent) isMessagingEvent() {}
func (MessageErrorEvent) isMessagingEvent()      {}
func (ConnectionStatusEvent) isMessagingEvent()  {}

// wireBytes decodes a byte field as liblogosdelivery serialises it. A received
// WakuMessage crosses the boundary through Nim's std/json, which renders
// seq[byte] as an array of integers; base64 strings and null are accepted too,
// so a field that switches to the encoding used on the send path still decodes.
type wireBytes []byte

func (b *wireBytes) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*b = nil
		return nil
	}
	if data[0] == '[' {
		// Not []byte: encoding/json reads that from a base64 string, never
		// from an array.
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

// decodeEvent parses one flat event JSON document into a typed Event. An event
// whose eventType is not part of the Messaging surface decodes to a nil Event
// and no error, so a listener registered for a wider set can ignore it.
func decodeEvent(eventJSON string) (Event, error) {
	var head struct {
		EventType string `json:"eventType"`
	}
	if err := json.Unmarshal([]byte(eventJSON), &head); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}

	decode := func(v any) error {
		if err := json.Unmarshal([]byte(eventJSON), v); err != nil {
			return fmt.Errorf("decode %s: %w", head.EventType, err)
		}
		return nil
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
		if err := decode(&e); err != nil {
			return nil, err
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
		if err := decode(&e); err != nil {
			return nil, err
		}
		return MessageSentEvent{RequestID: RequestID(e.RequestID), MessageHash: e.MessageHash}, nil

	case "message_propagated":
		var e struct {
			RequestID   string `json:"requestId"`
			MessageHash string `json:"messageHash"`
		}
		if err := decode(&e); err != nil {
			return nil, err
		}
		return MessagePropagatedEvent{RequestID: RequestID(e.RequestID), MessageHash: e.MessageHash}, nil

	case "message_error":
		var e struct {
			RequestID   string `json:"requestId"`
			MessageHash string `json:"messageHash"`
			Error       string `json:"error"`
		}
		if err := decode(&e); err != nil {
			return nil, err
		}
		return MessageErrorEvent{
			RequestID:   RequestID(e.RequestID),
			MessageHash: e.MessageHash,
			Err:         e.Error,
		}, nil

	case "connection_status_change":
		var e struct {
			ConnectionStatus string `json:"connectionStatus"`
		}
		if err := decode(&e); err != nil {
			return nil, err
		}
		return ConnectionStatusEvent{Status: parseConnectionStatus(e.ConnectionStatus)}, nil

	default:
		return nil, nil
	}
}
