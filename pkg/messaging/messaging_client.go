package messaging

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
)

// eventBufferSize bounds the buffered Events channel. Events are dropped rather
// than blocked when a consumer falls behind, so the library's event thread is
// never stalled by a slow reader.
const eventBufferSize = 1024

// ErrClosed is returned by operations on a MessagingClient that has been
// closed.
var ErrClosed = errors.New("messaging: client is closed")

// MessagingClient is a logos-delivery Messaging API client, mirroring the Nim
// MessagingClient: it owns a node and exposes the messaging surface over it —
// subscribe, unsubscribe, send, and a stream of delivery events.
//
// The lifecycle is New -> Start -> ... -> Stop -> Close. Consume Events()
// concurrently for the whole lifetime; it is closed by Close.
//
// A MessagingClient is safe for concurrent use.
type MessagingClient struct {
	h ffi.Handle

	events chan Event

	// mu guards closed and serialises it against the event callbacks, which
	// run on the library's event thread. It is only ever held briefly.
	mu        sync.RWMutex
	closed    bool
	listeners []ffi.ListenerID
}

// New creates a node from cfg and wires up its event stream. The node is not
// started yet: call Start. Release it with Close, started or not.
func New(cfg Config) (*MessagingClient, error) {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("messaging: marshal config: %w", err)
	}

	h, err := ffi.New(string(cfgJSON))
	if err != nil {
		return nil, fmt.Errorf("messaging: create node: %w", err)
	}

	c := &MessagingClient{h: h, events: make(chan Event, eventBufferSize)}

	// Register before Start so no event emitted during startup is missed.
	for _, name := range messagingEvents {
		id, err := ffi.AddEventListener(h, name, c.onEvent)
		if err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("messaging: %w", err)
		}
		c.listeners = append(c.listeners, id)
	}
	return c, nil
}

// onEvent runs on the library's event thread. It must not block, so a decoded
// event is dropped when the Events channel is full.
func (c *MessagingClient) onEvent(ret int, msg string) {
	if ret != ffi.RetOK {
		return
	}
	ev, err := decodeEvent(msg)
	if err != nil || ev == nil {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return
	}
	select {
	case c.events <- ev:
	default:
		// Consumer is not keeping up. Dropping is the contract: blocking here
		// would stall the library's event thread.
	}
}

// Events returns the stream of Messaging API events. Type-switch over the
// concrete Event types. The channel is closed by Close.
func (c *MessagingClient) Events() <-chan Event { return c.events }

// Start starts the node's protocols and Messaging API services.
func (c *MessagingClient) Start() error {
	if err := c.check(); err != nil {
		return err
	}
	if err := ffi.Start(c.h); err != nil {
		return fmt.Errorf("messaging: start: %w", err)
	}
	return nil
}

// Stop stops the node. A stopped client can be started again.
func (c *MessagingClient) Stop() error {
	if err := c.check(); err != nil {
		return err
	}
	if err := ffi.Stop(c.h); err != nil {
		return fmt.Errorf("messaging: stop: %w", err)
	}
	return nil
}

// Close releases the node and closes the Events channel. It is idempotent, and
// tears a running node down, so Stop beforehand is optional. The client must
// not be used afterwards, and no other method may be in flight when it is
// called.
func (c *MessagingClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	listeners := c.listeners
	c.listeners = nil
	c.mu.Unlock()

	// Drop the listeners before the context goes away, so no callback can
	// arrive after the channel is closed.
	var errs []error
	for _, id := range listeners {
		if err := ffi.RemoveEventListener(c.h, id); err != nil {
			errs = append(errs, err)
		}
	}
	if err := ffi.Destroy(c.h); err != nil {
		errs = append(errs, err)
	}
	close(c.events)

	if len(errs) > 0 {
		return fmt.Errorf("messaging: close: %w", errors.Join(errs...))
	}
	return nil
}

// Subscribe starts receiving messages published on a content topic. They arrive
// as MessageReceivedEvent on Events().
func (c *MessagingClient) Subscribe(topic ContentTopic) error {
	if err := c.check(); err != nil {
		return err
	}
	if err := ffi.Subscribe(c.h, topic); err != nil {
		return fmt.Errorf("messaging: subscribe %q: %w", topic, err)
	}
	return nil
}

// Unsubscribe stops receiving messages published on a content topic.
func (c *MessagingClient) Unsubscribe(topic ContentTopic) error {
	if err := c.check(); err != nil {
		return err
	}
	if err := ffi.Unsubscribe(c.h, topic); err != nil {
		return fmt.Errorf("messaging: unsubscribe %q: %w", topic, err)
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

// Send publishes env and returns the RequestID that correlates it with the
// MessageSentEvent, MessagePropagatedEvent or MessageErrorEvent it produces.
//
// Returning marks the message accepted by the send service, not delivered:
// delivery is reported on Events(). If ctx is cancelled while the library is
// still working, Send returns ctx.Err() and the message may still go out.
func (c *MessagingClient) Send(ctx context.Context, env Envelope) (RequestID, error) {
	if err := c.check(); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	msg, err := json.Marshal(wireEnvelope{
		ContentTopic: env.ContentTopic,
		Payload:      base64.StdEncoding.EncodeToString(env.Payload),
		Ephemeral:    env.Ephemeral,
	})
	if err != nil {
		return "", fmt.Errorf("messaging: marshal envelope: %w", err)
	}

	type result struct {
		id  string
		err error
	}
	// Buffered: the call outlives a cancelled ctx, and must not block on exit.
	done := make(chan result, 1)
	go func() {
		id, err := ffi.Send(c.h, string(msg))
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

// check reports whether the client is still usable.
func (c *MessagingClient) check() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return ErrClosed
	}
	return nil
}
