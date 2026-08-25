package messaging

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel"
)

// eventBufferSize bounds the buffered Events channel. Events are dropped rather
// than blocked when a consumer falls behind, so the library's event thread is
// never stalled by a slow reader.
const eventBufferSize = 1024

// ErrClosed is returned by operations on a MessagingClient whose node has been
// closed.
var ErrClosed = kernel.ErrClosed

// MessagingClient is a logos-delivery Messaging API client, mirroring the Nim
// MessagingClient: it drives a node and exposes the messaging surface over it —
// subscribe, unsubscribe, send, and a stream of delivery events.
//
// The node is reached through Node, so the kernel protocols run against the
// same node the client sends on.
//
// The lifecycle is New -> Start -> ... -> Stop -> Close. Consume Events()
// concurrently for the whole lifetime; it is closed by Close.
//
// A MessagingClient is safe for concurrent use.
type MessagingClient struct {
	node *kernel.Node
	h    ffi.Handle

	events chan Event

	// mu guards closed and serialises it against the event callbacks, which
	// run on the library's event thread. It is only ever held briefly.
	mu     sync.RWMutex
	closed bool
}

// New creates a node from cfg and wires up its event stream. The node is not
// started yet: call Start. Release it with Close, started or not.
func New(cfg Config) (*MessagingClient, error) {
	node, err := kernel.New(cfg)
	if err != nil {
		return nil, err
	}
	return Attach(node)
}

// Attach adds the Messaging API event stream to an existing node, for a caller
// that built the node itself. The client takes over the node's lifetime:
// closing either one closes both.
func Attach(node *kernel.Node) (*MessagingClient, error) {
	c := &MessagingClient{
		node:   node,
		h:      kernel.Handle(node),
		events: make(chan Event, eventBufferSize),
	}

	// Runs after the node drops its listeners, so nothing sends afterwards.
	node.OnClose(func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.closed = true
		close(c.events)
	})

	// Register before Start so no event emitted during startup is missed.
	for _, name := range messagingEvents() {
		if _, err := node.AddEventListener(name, c.onEvent); err != nil {
			_ = node.Close()
			return nil, err
		}
	}
	return c, nil
}

// Node returns the node this client drives, so the kernel protocols can be
// used against it. Its lifetime is the client's: Close either one and both are
// done.
func (c *MessagingClient) Node() *kernel.Node { return c.node }

// onEvent runs on the library's event thread. It must not block, so a decoded
// event is dropped when the Events channel is full.
func (c *MessagingClient) onEvent(msg string) {
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
func (c *MessagingClient) Start() error { return c.node.Start() }

// Stop stops the node. A stopped client can be started again.
func (c *MessagingClient) Stop() error { return c.node.Stop() }

// Close releases the node and closes the Events channel. It is idempotent, and
// tears a running node down, so Stop beforehand is optional. The client must
// not be used afterwards, and no other method may be in flight when it is
// called.
func (c *MessagingClient) Close() error { return c.node.Close() }

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
	if err := c.check(); err != nil {
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
		return "", fmt.Errorf("messaging: marshal message: %w", err)
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
