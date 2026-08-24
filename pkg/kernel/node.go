package kernel

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
)

// ErrClosed is returned by operations on a Node that has been closed.
var ErrClosed = errors.New("kernel: node is closed")

// ListenerID identifies one event listener registered on a Node.
type ListenerID uint64

// EventHandler receives the raw JSON of every event emitted under the name it
// was registered for. It runs on the library's event thread, so it must not
// block: hand work off to a buffered channel or a goroutine.
type EventHandler func(eventJSON string)

// Node is a logos-delivery node: the single owner of the library context that
// both API tiers share. The Kernel API is reached through the protocol facades
// (Relay, Store, Peers, Discovery); the Messaging API is reached through
// pkg/messaging, which builds a MessagingClient over a Node.
//
// The lifecycle is New -> Start -> ... -> Stop -> Close. Close is idempotent
// and releases the context, so it is safe to defer it right after New.
//
// A Node is safe for concurrent use.
type Node struct {
	h    ffi.Handle
	name string

	// config is the flat legacy configuration, when the node was built from
	// one. Nodes built from a Config leave it nil.
	config *common.WakuConfig

	msgChan         chan common.Envelope
	topicHealthChan chan TopicHealth
	connectionChan  chan ConnectionChange

	// mu guards the fields below, including against the event callbacks that
	// run on the library's event thread. It is only ever held briefly.
	mu         sync.RWMutex
	closed     bool
	started    bool
	listeners  []ListenerID
	closeHooks []func()
}

// Channel capacities for the kernel event streams. Events are dropped rather
// than blocked when a consumer falls behind, so the library's event thread is
// never stalled by a slow reader.
const (
	MsgChanBufferSize              = 1024
	TopicHealthChanBufferSize      = 1024
	ConnectionChangeChanBufferSize = 1024
)

// kernelEvents are the library's wire names for the events a Node consumes.
// The library registers one listener per name; the eventType inside each
// event's JSON is what the dispatcher switches on.
func kernelEvents() []string {
	return []string{
		"onReceivedMessage",
		"onTopicHealthChange",
		"onConnectionChange",
	}
}

// New builds a node from a layered configuration and returns it ready to
// Start. Release it with Close, started or not.
func New(cfg Config) (*Node, error) {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("kernel: marshal config: %w", err)
	}
	return newNode(string(cfgJSON), cfg.Name)
}

// NewFromWakuConfig builds a node from the legacy flat configuration blob. New
// is the preferred door: it takes the layered configuration the library
// expects, and a preset covers most of what this struct spells out by hand.
func NewFromWakuConfig(cfg *common.WakuConfig, name string) (*Node, error) {
	if cfg == nil {
		return nil, errors.New("kernel: config is nil")
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("kernel: marshal config: %w", err)
	}

	n, err := newNode(string(cfgJSON), name)
	if err != nil {
		return nil, err
	}
	n.config = cfg
	return n, nil
}

// newNode creates the library context and wires up the kernel event streams.
func newNode(configJSON, name string) (*Node, error) {
	Debug("Creating node %s", name)

	h, err := ffi.New(configJSON)
	if err != nil {
		Error("error creating node %s: %v", name, err)
		return nil, fmt.Errorf("kernel: create node: %w", err)
	}

	n := &Node{
		h:               h,
		name:            name,
		msgChan:         make(chan common.Envelope, MsgChanBufferSize),
		topicHealthChan: make(chan TopicHealth, TopicHealthChanBufferSize),
		connectionChan:  make(chan ConnectionChange, ConnectionChangeChanBufferSize),
	}

	// Register before Start so no event emitted during startup is missed.
	for _, name := range kernelEvents() {
		if _, err := n.AddEventListener(name, n.onEvent); err != nil {
			_ = n.Close()
			return nil, err
		}
	}

	Debug("Successfully created node %s", name)
	return n, nil
}

// Name is the label this node carries in log messages.
func (n *Node) Name() string { return n.name }

// Config returns the legacy flat configuration the node was built from, or nil
// when it was built from a Config.
func (n *Node) Config() *common.WakuConfig { return n.config }

// Start starts the node's protocols and services.
func (n *Node) Start() error {
	if err := n.check(); err != nil {
		return err
	}

	Debug("Starting %s", n.name)
	if err := ffi.Start(n.h); err != nil {
		Error("Failed to start %s: %v", n.name, err)
		return fmt.Errorf("kernel: start: %w", err)
	}

	n.mu.Lock()
	n.started = true
	n.mu.Unlock()

	Debug("Successfully started %s", n.name)
	return nil
}

// Stop stops the node. A stopped node can be started again.
func (n *Node) Stop() error {
	if err := n.check(); err != nil {
		return err
	}

	Debug("Stopping %s", n.name)
	if err := ffi.Stop(n.h); err != nil {
		Error("Failed to stop %s: %v", n.name, err)
		return fmt.Errorf("kernel: stop: %w", err)
	}

	n.mu.Lock()
	n.started = false
	n.mu.Unlock()

	Debug("Successfully stopped %s", n.name)
	return nil
}

// Close stops the node if it is running, releases the library context and runs
// the hooks registered with OnClose. It is idempotent. The node and every
// facade taken from it must not be used afterwards, and no other method may be
// in flight when it is called.
func (n *Node) Close() error {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return nil
	}
	n.closed = true
	started := n.started
	n.started = false
	listeners := n.listeners
	hooks := n.closeHooks
	n.listeners, n.closeHooks = nil, nil
	n.mu.Unlock()

	Debug("Closing %s", n.name)

	var errs []error
	if started {
		// Destroy regardless: a leaked context is worse than an unclean stop.
		if err := ffi.Stop(n.h); err != nil {
			errs = append(errs, fmt.Errorf("stop: %w", err))
		}
	}

	// Drop the listeners before the hooks tear down what they write to.
	for _, id := range listeners {
		if err := ffi.RemoveEventListener(n.h, ffi.ListenerID(id)); err != nil {
			Warn("failed to remove event listener for %v: %v", n.name, err)
		}
	}
	for _, hook := range hooks {
		hook()
	}

	if err := ffi.Destroy(n.h); err != nil {
		errs = append(errs, fmt.Errorf("destroy: %w", err))
	}

	if len(errs) > 0 {
		err := fmt.Errorf("kernel: close %s: %w", n.name, errors.Join(errs...))
		Error("%v", err)
		return err
	}

	Debug("Successfully closed %s", n.name)
	return nil
}

// Closed reports whether the node has been closed.
func (n *Node) Closed() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.closed
}

// OnClose registers fn to run while the node is closing, after its event
// listeners are removed and before the library context is released. Layers
// built on a Node use it to tear down their own state exactly once.
func (n *Node) OnClose(fn func()) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closeHooks = append(n.closeHooks, fn)
}

// AddEventListener registers fn to receive the named event, and returns the id
// that removes it again. Event names are the library's wire names, e.g.
// "onMessageReceived". Register before Start so no event is missed; a listener
// left registered at Close is removed with the node.
func (n *Node) AddEventListener(eventName string, fn EventHandler) (ListenerID, error) {
	if err := n.check(); err != nil {
		return 0, err
	}

	id, err := ffi.AddEventListener(n.h, eventName, func(ret int, msg string) {
		if ret != ffi.RetOK {
			Error("event listener %q on %s reported code %d: %v", eventName, n.name, ret, msg)
			return
		}
		fn(msg)
	})
	if err != nil {
		Error("error adding %s listener for %s: %v", eventName, n.name, err)
		return 0, fmt.Errorf("kernel: %w", err)
	}

	n.mu.Lock()
	n.listeners = append(n.listeners, ListenerID(id))
	n.mu.Unlock()
	return ListenerID(id), nil
}

// RemoveEventListener removes a listener previously added with
// AddEventListener.
func (n *Node) RemoveEventListener(id ListenerID) error {
	if err := n.check(); err != nil {
		return err
	}

	n.mu.Lock()
	for i, known := range n.listeners {
		if known == id {
			n.listeners = append(n.listeners[:i], n.listeners[i+1:]...)
			break
		}
	}
	n.mu.Unlock()

	if err := ffi.RemoveEventListener(n.h, ffi.ListenerID(id)); err != nil {
		return fmt.Errorf("kernel: %w", err)
	}
	return nil
}

// Relay is the relay protocol surface.
func (n *Node) Relay() Relay { return Relay{n} }

// Store is the store protocol surface.
func (n *Node) Store() Store { return Store{n} }

// Peers is the peer management surface.
func (n *Node) Peers() Peers { return Peers{n} }

// Discovery is the peer discovery surface: DiscV5, DNS discovery and peer
// exchange.
func (n *Node) Discovery() Discovery { return Discovery{n} }

// PeerID returns the node's own peer id.
func (n *Node) PeerID() (peer.ID, error) {
	if err := n.check(); err != nil {
		return "", err
	}

	idStr, err := ffi.GetMyPeerID(n.h)
	if err != nil {
		return "", fmt.Errorf("kernel: peer id: %w", err)
	}

	id, err := peer.Decode(idStr)
	if err != nil {
		return "", fmt.Errorf("kernel: decode peer id: %w", err)
	}
	return id, nil
}

// ListenAddresses returns the multiaddresses the node listens on.
func (n *Node) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	if err := n.check(); err != nil {
		return nil, err
	}

	addrs, err := ffi.ListenAddresses(n.h)
	if err != nil {
		return nil, fmt.Errorf("kernel: listen addresses: %w", err)
	}
	return parseMultiaddrs(addrs)
}

// ENR returns the node's own ENR record.
func (n *Node) ENR() (*enode.Node, error) {
	if err := n.check(); err != nil {
		return nil, err
	}

	enrStr, err := ffi.GetMyENR(n.h)
	if err != nil {
		return nil, fmt.Errorf("kernel: enr: %w", err)
	}

	record, err := enode.Parse(enode.ValidSchemes, enrStr)
	if err != nil {
		return nil, fmt.Errorf("kernel: parse enr: %w", err)
	}
	return record, nil
}

// Version returns the library version the node runs.
func (n *Node) Version() (string, error) {
	if err := n.check(); err != nil {
		return "", err
	}

	version, err := ffi.Version(n.h)
	if err != nil {
		return "", fmt.Errorf("kernel: version: %w", err)
	}
	return version, nil
}

// IsOnline reports whether the node considers itself connected to the network.
func (n *Node) IsOnline() (bool, error) {
	if err := n.check(); err != nil {
		return false, err
	}

	online, err := ffi.IsOnline(n.h)
	if err != nil {
		return false, fmt.Errorf("kernel: is online: %w", err)
	}
	return online == "true", nil
}

// Metrics returns the node's metrics in Prometheus text format.
func (n *Node) Metrics() (string, error) {
	if err := n.check(); err != nil {
		return "", err
	}

	metrics, err := ffi.GetMetrics(n.h)
	if err != nil {
		return "", fmt.Errorf("kernel: metrics: %w", err)
	}
	if metrics == "" {
		return "", errors.New("kernel: metrics: empty response")
	}
	return metrics, nil
}

// check reports whether the node is still usable.
func (n *Node) check() error {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.closed {
		return ErrClosed
	}
	return nil
}

// parseMultiaddrs splits and parses the comma-separated multiaddress lists the
// library returns. An empty list yields no addresses rather than an error.
func parseMultiaddrs(list string) ([]multiaddr.Multiaddr, error) {
	if list == "" {
		return nil, nil
	}

	parts := strings.Split(list, ",")
	addrs := make([]multiaddr.Multiaddr, 0, len(parts))
	for _, part := range parts {
		addr, err := multiaddr.NewMultiaddr(part)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}

// parsePeerIDs splits and parses the comma-separated peer id lists the library
// returns. An empty list yields no peers rather than an error.
func parsePeerIDs(list string) (peer.IDSlice, error) {
	if list == "" {
		return nil, nil
	}

	parts := strings.Split(list, ",")
	peers := make(peer.IDSlice, 0, len(parts))
	for _, part := range parts {
		id, err := peer.Decode(part)
		if err != nil {
			return nil, err
		}
		peers = append(peers, id)
	}
	return peers, nil
}
