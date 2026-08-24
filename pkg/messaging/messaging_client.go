package messaging

import (
	"sync"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel"
)

// ErrClosed is returned by operations on a client whose node has been closed.
var ErrClosed = kernel.ErrClosed

// MessagingClient is a logos-delivery Messaging API client, mirroring the Nim
// MessagingClient: it drives a node and exposes the messaging surface over it —
// subscribe, unsubscribe, send, and a stream of delivery events.
//
// The node it drives is reached through Node, so the kernel protocols run
// against the same node the client sends on:
//
//	client.Send(ctx, topic, payload, false)
//	client.Node().Store().Query(ctx, request, peerInfo)
//
// The lifecycle is New -> Start -> ... -> Stop -> Close. Consume Events()
// concurrently for the whole lifetime; it is closed by Close.
//
// A MessagingClient is safe for concurrent use.
type MessagingClient struct {
	node   *kernel.Node
	events chan Event

	// mu guards closed and serialises it against the event callbacks, which
	// run on the library's event thread. It is only ever held briefly.
	mu     sync.RWMutex
	closed bool
}

// New creates a node from cfg and wires up its Messaging API event stream. The
// node is not started yet: call Start. Release it with Close, started or not.
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
		events: make(chan Event, kernel.EventChanBufferSize),
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

// onEvent runs on the library's event thread. It must not block, so a decoded
// event is dropped when the Events channel is full.
func (c *MessagingClient) onEvent(eventJSON string) {
	ev, err := decodeEvent(eventJSON)
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
