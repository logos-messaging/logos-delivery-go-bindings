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
func messagingEvents() []string {
	return []string{
		wireMessageReceived,
		wireMessageSent,
		wireMessagePropagated,
		wireMessageError,
		wireConnectionStatusChange,
	}
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

// Event is the interface every event delivered on MessagingClient.Events()
// implements. Consumers type-switch over the concrete types; the set only
// grows, so keep a default branch.
//
// It is sealed: isMessagingEvent is unexported, so only types declared in this
// package can satisfy Event. That is what makes the type switch trustworthy —
// no other package can introduce an Event, and every value on the channel is
// one of the types below. Adding an event type is then a backwards-compatible
// change, because callers cannot have exhaustively matched on a closed set
// they do not control.
type Event interface {
	isMessagingEvent()
}

// MessageSource tells a message delivered as it was published from one
// recovered from a store peer.
type MessageSource string

const (
	// SourceLive is a message delivered as it was published, over relay or
	// filter.
	SourceLive MessageSource = "live"
	// SourceHistory is a message recovered from a store peer, at start or after
	// a connectivity gap.
	SourceHistory MessageSource = "history"
)

// MessageReceivedEvent is emitted when a message arrives on a subscribed
// content topic, live or recovered from store.
type MessageReceivedEvent struct {
	MessageHash string
	Message     Message
	Source      MessageSource
}

// MessageSentEvent is emitted when the node confirms the delivery of a message:
// a store node confirms it, or, for a send over mix, the mix exit's reply. It
// follows the MessagePropagatedEvent, and comes only with the reliability
// option on and never for an ephemeral message.
type MessageSentEvent struct {
	MessageHash string
}

// MessagePropagatedEvent is emitted when a message has reached neighbouring
// nodes on the network.
type MessagePropagatedEvent struct {
	MessageHash string
}

// MessageErrorKind tells why a message that Send accepted failed.
type MessageErrorKind string

const (
	// NotPropagated: no peer took the message within the delivery window.
	NotPropagated MessageErrorKind = "NotPropagated"
	// NotStored: the message propagated, but no store node reported it.
	NotStored MessageErrorKind = "NotStored"
)

// MessageErrorEvent is emitted when a message that Send accepted fails.
type MessageErrorEvent struct {
	MessageHash string
	Kind        MessageErrorKind
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

// wireBytes decodes a byte field as liblogosdelivery serialises it, which is
// not base64. Only the send path is base64: logosdelivery_send decodes the
// payload it is given, and the channel events base64-encode explicitly. The
// messaging events do neither — a received WakuMessage is rendered by Nim's
// std/json, whose default for seq[byte] is an array of integers, so `hello`
// arrives as [104,101,108,108,111] (see the captures in event_test.go).
//
// Base64 strings and null decode too, so the day the library normalises its
// encodings this keeps working instead of silently dropping every message.
type wireBytes []byte

func (b *wireBytes) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*b = nil
		return nil
	}
	if data[0] == '[' {
		// Not []byte: encoding/json reads that from a base64 string only.
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
			Source string `json:"source"`
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
			Source: MessageSource(e.Source),
		}, nil

	case "message_sent":
		var e struct {
			MessageHash string `json:"messageHash"`
		}
		if err := decode(&e); err != nil {
			return nil, err
		}
		return MessageSentEvent{MessageHash: e.MessageHash}, nil

	case "message_propagated":
		var e struct {
			MessageHash string `json:"messageHash"`
		}
		if err := decode(&e); err != nil {
			return nil, err
		}
		return MessagePropagatedEvent{MessageHash: e.MessageHash}, nil

	case "message_error":
		var e struct {
			MessageHash string `json:"messageHash"`
			Kind        string `json:"kind"`
			Error       string `json:"error"`
		}
		if err := decode(&e); err != nil {
			return nil, err
		}
		return MessageErrorEvent{
			MessageHash: e.MessageHash,
			Kind:        MessageErrorKind(e.Kind),
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
