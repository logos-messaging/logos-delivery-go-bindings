package messaging

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
)

// eventBufferSize bounds the buffered Events channel. Events are dropped (never
// blocked) if a consumer falls behind, so the FFI worker thread is never stalled.
const eventBufferSize = 1024

// MessagingClient is a logos-delivery Messaging API client (the Nim
// MessagingClient). Create it with New, then Start it; consume incoming events
// from Events(); release it with Close.
type MessagingClient struct {
	h         ffi.Handle
	events    chan Event
	closeOnce sync.Once
}

// New creates (but does not start) a MessagingClient from cfg.
func New(cfg Config) (*MessagingClient, error) {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	h, err := ffi.New(string(cfgJSON))
	if err != nil {
		return nil, err
	}
	c := &MessagingClient{h: h, events: make(chan Event, eventBufferSize)}
	ffi.SetEventHandler(h, c.onEvent)
	return c, nil
}

// onEvent runs on the FFI worker thread: decode and hand off without blocking.
func (c *MessagingClient) onEvent(ret int, msg string) {
	if ret != ffi.RetOK {
		return
	}
	ev, err := decodeEvent(msg)
	if err != nil || ev == nil {
		return
	}
	select {
	case c.events <- ev:
	default:
		// Consumer not keeping up; drop rather than block the worker thread.
	}
}

// Start starts the client's protocols and Messaging API services.
func (c *MessagingClient) Start() error { return ffi.Start(c.h) }

// Stop stops the client. It can be started again.
func (c *MessagingClient) Stop() error { return ffi.Stop(c.h) }

// Close stops tracking events and releases the underlying node context. The
// Events channel is closed. The MessagingClient must not be used afterwards.
func (c *MessagingClient) Close() error {
	err := ffi.Destroy(c.h)
	c.closeOnce.Do(func() { close(c.events) })
	return err
}

// Subscribe subscribes to a content topic so messages on it are received.
func (c *MessagingClient) Subscribe(topic ContentTopic) error {
	return ffi.Subscribe(c.h, topic)
}

// Unsubscribe stops receiving messages on a content topic.
func (c *MessagingClient) Unsubscribe(topic ContentTopic) error {
	return ffi.Unsubscribe(c.h, topic)
}

// Send publishes env and returns the RequestID to correlate with the later
// MessageSentEvent / MessagePropagatedEvent / MessageErrorEvent. The send is
// fire-and-queue: ctx cancellation is honoured before dispatch only.
func (c *MessagingClient) Send(ctx context.Context, env Envelope) (RequestID, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	msg, err := json.Marshal(struct {
		ContentTopic string `json:"contentTopic"`
		Payload      string `json:"payload"`
		Ephemeral    bool   `json:"ephemeral"`
	}{
		ContentTopic: string(env.ContentTopic),
		Payload:      base64.StdEncoding.EncodeToString(env.Payload),
		Ephemeral:    env.Ephemeral,
	})
	if err != nil {
		return "", fmt.Errorf("marshal envelope: %w", err)
	}
	id, err := ffi.Send(c.h, string(msg))
	if err != nil {
		return "", err
	}
	return RequestID(id), nil
}

// Events returns the channel of incoming Messaging API events. Type-switch over
// the concrete Event types. The channel is closed by Close.
func (c *MessagingClient) Events() <-chan Event { return c.events }
