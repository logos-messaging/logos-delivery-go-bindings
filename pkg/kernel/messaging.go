package kernel

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
)

// ContentTopic names the application-level channel a message belongs to, by
// convention "/<app>/<version>/<name>/<encoding>".
type ContentTopic = string

// RequestID correlates a send with the delivery events it later produces.
type RequestID string

func (id RequestID) String() string { return string(id) }

// Messaging is a Node's Messaging API surface: the stable send and subscribe
// tier that sits above relay. Take one with Node.Messaging.
//
// It carries the operations only. For the delivery events they produce, use
// pkg/messaging, whose MessagingClient pairs this surface with a typed event
// stream.
type Messaging struct{ n *Node }

// Messaging is the Messaging API surface.
func (n *Node) Messaging() Messaging { return Messaging{n} }

// Subscribe starts receiving messages published on a content topic.
func (m Messaging) Subscribe(contentTopic ContentTopic) error {
	if err := m.n.check(); err != nil {
		return err
	}

	if err := ffi.Subscribe(m.n.h, contentTopic); err != nil {
		return fmt.Errorf("kernel: subscribe %q: %w", contentTopic, err)
	}
	return nil
}

// Unsubscribe stops receiving messages published on a content topic.
func (m Messaging) Unsubscribe(contentTopic ContentTopic) error {
	if err := m.n.check(); err != nil {
		return err
	}

	if err := ffi.Unsubscribe(m.n.h, contentTopic); err != nil {
		return fmt.Errorf("kernel: unsubscribe %q: %w", contentTopic, err)
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
// correlates it with the delivery events it produces. An ephemeral message is
// transient, so stores do not retain it.
//
// Returning marks the message accepted by the send service, not delivered.
// If ctx is cancelled while the library is still working, Send returns
// ctx.Err() and the message may still go out.
func (m Messaging) Send(
	ctx context.Context, contentTopic ContentTopic, payload []byte, ephemeral bool,
) (RequestID, error) {
	if err := m.n.check(); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	msg, err := json.Marshal(wireEnvelope{
		ContentTopic: contentTopic,
		Payload:      base64.StdEncoding.EncodeToString(payload),
		Ephemeral:    ephemeral,
	})
	if err != nil {
		return "", fmt.Errorf("kernel: marshal message: %w", err)
	}

	type result struct {
		id  string
		err error
	}
	// Buffered: the call outlives a cancelled ctx, and must not block on exit.
	done := make(chan result, 1)
	go func() {
		id, err := ffi.Send(m.n.h, string(msg))
		done <- result{id, err}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-done:
		if r.err != nil {
			return "", fmt.Errorf("kernel: send: %w", r.err)
		}
		return RequestID(r.id), nil
	}
}
