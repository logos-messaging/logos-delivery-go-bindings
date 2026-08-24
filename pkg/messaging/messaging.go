package messaging

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel"
)

// ContentTopic names the application-level channel a message belongs to, by
// convention "/<app>/<version>/<name>/<encoding>".
type ContentTopic = string

// RequestID correlates a send with the delivery events it later produces.
type RequestID string

func (id RequestID) String() string { return string(id) }

// Subscribe starts receiving messages published on a content topic. They
// arrive as MessageReceivedEvent on Events().
func (c *MessagingClient) Subscribe(contentTopic ContentTopic) error {
	if err := ffi.Subscribe(kernel.Handle(c.node), contentTopic); err != nil {
		return fmt.Errorf("messaging: subscribe %q: %w", contentTopic, err)
	}
	return nil
}

// Unsubscribe stops receiving messages published on a content topic.
func (c *MessagingClient) Unsubscribe(contentTopic ContentTopic) error {
	if err := ffi.Unsubscribe(kernel.Handle(c.node), contentTopic); err != nil {
		return fmt.Errorf("messaging: unsubscribe %q: %w", contentTopic, err)
	}
	return nil
}

// wireEnvelope is the send call's JSON shape: the C surface reads exactly these
// three fields, with the payload base64-encoded.
type wireEnvelope struct {
	ContentTopic string `json:"contentTopic"`
	Payload      string `json:"payload"`
	Ephemeral    bool   `json:"ephemeral"`
}

// Send publishes payload on contentTopic and returns the RequestID that
// correlates it with the MessageSentEvent, MessagePropagatedEvent or
// MessageErrorEvent it produces. An ephemeral message is transient, so stores
// do not retain it.
//
// Returning marks the message accepted by the send service, not delivered:
// delivery is reported on Events(). If ctx is cancelled while the library is
// still working, Send returns ctx.Err() and the message may still go out.
func (c *MessagingClient) Send(
	ctx context.Context, contentTopic ContentTopic, payload []byte, ephemeral bool,
) (RequestID, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	msg, err := json.Marshal(wireEnvelope{
		ContentTopic: contentTopic,
		Payload:      base64.StdEncoding.EncodeToString(payload),
		Ephemeral:    ephemeral,
	})
	if err != nil {
		return "", fmt.Errorf("messaging: marshal message: %w", err)
	}

	type result struct {
		id  string
		err error
	}
	// Buffered: the call outlives a cancelled ctx, and must not block on exit.
	done := make(chan result, 1)
	go func() {
		id, err := ffi.Send(kernel.Handle(c.node), string(msg))
		done <- result{id, err}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-done:
		if r.err != nil {
			return "", fmt.Errorf("messaging: send: %w", r.err)
		}
		return RequestID(r.id), nil
	}
}
