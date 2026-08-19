package messaging

// ContentTopic is an application-level message category, e.g.
// "/my-app/1/chat/proto". It mirrors the Nim ContentTopic, which is a string.
type ContentTopic = string

// RequestID correlates a Send call with the MessageSentEvent /
// MessagePropagatedEvent / MessageErrorEvent it later produces. It mirrors the
// Nim RequestId.
type RequestID string

func (id RequestID) String() string { return string(id) }

// Envelope is an outgoing message, mirroring the Nim MessageEnvelope.
//
// The Nim envelope also carries an opaque `meta` marker, but the C surface's
// send call only reads contentTopic, payload and ephemeral, so there is no way
// to set it from here yet.
type Envelope struct {
	// ContentTopic is the topic the message is published on. Required.
	ContentTopic ContentTopic
	// Payload is the message body. It crosses the C boundary base64-encoded.
	Payload []byte
	// Ephemeral marks the message as transient, so stores do not retain it.
	Ephemeral bool
}
