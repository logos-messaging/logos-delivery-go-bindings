package messaging

// ContentTopic is an application-level message category, e.g.
// "/my-app/1/chat/proto". It mirrors the Nim ContentTopic, which is a string.
type ContentTopic = string

// RequestID correlates a Send call with the MessageSentEvent /
// MessagePropagatedEvent / MessageErrorEvent it later produces. It mirrors the
// Nim RequestId.
type RequestID string

func (id RequestID) String() string { return string(id) }
