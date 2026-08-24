package kernel

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
)

// TopicHealth reports how well a pubsub topic's relay mesh is populated.
type TopicHealth struct {
	PubsubTopic string `json:"pubsubTopic"`
	TopicHealth string `json:"topicHealth"`
}

// ConnectionChange reports a peer connecting to or disconnecting from the node.
type ConnectionChange struct {
	PeerID    peer.ID `json:"peerId"`
	PeerEvent string  `json:"peerEvent"`
}

// Messages is the stream of messages received on the pubsub topics the node
// relays. The channel is buffered and lossy: a message is dropped when a
// consumer falls behind rather than stalling the library's event thread.
func (n *Node) Messages() <-chan common.Envelope { return n.msgChan }

// TopicHealthChanges is the stream of relay mesh health changes. It has the
// same buffered, lossy delivery as Messages.
func (n *Node) TopicHealthChanges() <-chan TopicHealth { return n.topicHealthChan }

// ConnectionChanges is the stream of peer connect and disconnect events. It has
// the same buffered, lossy delivery as Messages.
func (n *Node) ConnectionChanges() <-chan ConnectionChange { return n.connectionChan }

// onEvent dispatches one raw kernel event onto its stream. It runs on the
// library's event thread.
func (n *Node) onEvent(eventJSON string) {
	var head struct {
		EventType string `json:"eventType"`
	}
	if err := json.Unmarshal([]byte(eventJSON), &head); err != nil {
		Error("could not unmarshal event for %s: %v", n.name, err)
		return
	}

	switch head.EventType {
	case "message":
		var envelope common.Envelope
		if err := json.Unmarshal([]byte(eventJSON), &envelope); err != nil {
			Error("could not parse message for %s: %v", n.name, err)
			return
		}
		n.deliverMessage(envelope)

	case "relay_topic_health_change":
		var health TopicHealth
		if err := json.Unmarshal([]byte(eventJSON), &health); err != nil {
			Error("could not parse topic health change for %s: %v", n.name, err)
			return
		}
		n.deliverTopicHealth(health)

	case "connection_change":
		var change ConnectionChange
		if err := json.Unmarshal([]byte(eventJSON), &change); err != nil {
			Error("could not parse connection change for %s: %v", n.name, err)
			return
		}
		n.deliverConnectionChange(change)
	}
}

// The delivery helpers below take the read lock so a Close racing with the
// event thread cannot send on a stream the close hooks are tearing down.

func (n *Node) deliverMessage(envelope common.Envelope) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.closed {
		return
	}
	select {
	case n.msgChan <- envelope:
	default:
		Warn("Can't deliver message for %s, Messages channel is full", n.name)
	}
}

func (n *Node) deliverTopicHealth(health TopicHealth) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.closed {
		return
	}
	select {
	case n.topicHealthChan <- health:
	default:
		Warn("Can't deliver topic health event for %s, channel is full", n.name)
	}
}

func (n *Node) deliverConnectionChange(change ConnectionChange) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.closed {
		return
	}
	select {
	case n.connectionChan <- change:
	default:
		Warn("Can't deliver connection change for %s, channel is full", n.name)
	}
}
