package messaging

// ContentTopic is an application-level message category, e.g.
// "/my-app/1/chat/proto".
type ContentTopic = string

// RequestID correlates a Send call with its later MessageSentEvent /
// MessagePropagatedEvent / MessageErrorEvent.
type RequestID string

// Envelope is an outgoing Messaging API message.
type Envelope struct {
	ContentTopic ContentTopic
	Payload      []byte
	// Ephemeral marks the message as transient (not stored).
	Ephemeral bool
}
