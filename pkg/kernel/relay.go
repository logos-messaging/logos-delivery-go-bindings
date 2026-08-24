package kernel

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/pb"
)

// Relay is a Node's relay protocol surface: gossipsub subscriptions, publishing
// and the state of the relay mesh. Take one with Node.Relay.
type Relay struct{ n *Node }

// Subscribe joins the relay mesh for a pubsub topic. Messages received on it
// arrive on Node.Messages.
func (r Relay) Subscribe(pubsubTopic string) error {
	if err := r.n.check(); err != nil {
		return err
	}
	if pubsubTopic == "" {
		return errors.New("kernel: relay subscribe: pubsub topic is empty")
	}

	if err := ffi.RelaySubscribe(r.n.h, pubsubTopic); err != nil {
		Error("Failed to subscribe %s to %s: %v", r.n.name, pubsubTopic, err)
		return fmt.Errorf("kernel: relay subscribe %q: %w", pubsubTopic, err)
	}

	Debug("Successfully subscribed %s to %s", r.n.name, pubsubTopic)
	return nil
}

// Unsubscribe leaves the relay mesh for a pubsub topic.
func (r Relay) Unsubscribe(pubsubTopic string) error {
	if err := r.n.check(); err != nil {
		return err
	}
	if pubsubTopic == "" {
		return errors.New("kernel: relay unsubscribe: pubsub topic is empty")
	}

	if err := ffi.RelayUnsubscribe(r.n.h, pubsubTopic); err != nil {
		Error("Failed to unsubscribe %s from %s: %v", r.n.name, pubsubTopic, err)
		return fmt.Errorf("kernel: relay unsubscribe %q: %w", pubsubTopic, err)
	}

	Debug("Successfully unsubscribed %s from %s", r.n.name, pubsubTopic)
	return nil
}

// Publish publishes a message on a pubsub topic and returns its hash. A ctx
// without a deadline gets the package default of 30s.
func (r Relay) Publish(
	ctx context.Context, pubsubTopic string, message *pb.WakuMessage,
) (common.MessageHash, error) {
	if err := r.n.check(); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if message == nil {
		return "", errors.New("kernel: relay publish: message is nil")
	}

	// The library expects the WakuMessage wire format (camelCase keys); the
	// generated protobuf struct marshals content_topic, so marshal explicitly.
	jsonMsg, err := json.Marshal(struct {
		Payload      []byte  `json:"payload,omitempty"`
		ContentTopic string  `json:"contentTopic"`
		Version      *uint32 `json:"version,omitempty"`
		Timestamp    *int64  `json:"timestamp,omitempty"`
		Meta         []byte  `json:"meta,omitempty"`
		Ephemeral    *bool   `json:"ephemeral,omitempty"`
	}{
		Payload:      message.Payload,
		ContentTopic: message.ContentTopic,
		Version:      message.Version,
		Timestamp:    message.Timestamp,
		Meta:         message.Meta,
		Ephemeral:    message.Ephemeral,
	})
	if err != nil {
		return "", err
	}

	hash, err := ffi.RelayPublish(r.n.h, pubsubTopic, string(jsonMsg), timeoutMillis(ctx, requestTimeout))
	if err != nil {
		Error("Failed to publish from %s on %s: %v", r.n.name, pubsubTopic, err)
		return "", fmt.Errorf("kernel: relay publish: %w", err)
	}

	parsed, err := common.ToMessageHash(hash)
	if err != nil {
		return "", err
	}

	Debug("Successfully published from %s, messageHash: %s", r.n.name, parsed.String())
	return parsed, nil
}

// AddProtectedShard registers the public key allowed to sign messages on a
// protected shard.
func (r Relay) AddProtectedShard(clusterID, shardID uint16, pubkey *ecdsa.PublicKey) error {
	if err := r.n.check(); err != nil {
		return err
	}
	if pubkey == nil {
		return errors.New("kernel: add protected shard: pubkey is nil")
	}

	keyHex := hex.EncodeToString(crypto.FromECDSAPub(pubkey))
	if err := ffi.RelayAddProtectedShard(r.n.h, int(clusterID), int(shardID), keyHex); err != nil {
		return fmt.Errorf("kernel: add protected shard: %w", err)
	}
	return nil
}

// PeersInMesh returns the relay mesh peers for a pubsub topic.
func (r Relay) PeersInMesh(pubsubTopic string) (peer.IDSlice, error) {
	if err := r.n.check(); err != nil {
		return nil, err
	}

	list, err := ffi.GetPeersInMesh(r.n.h, pubsubTopic)
	if err != nil {
		return nil, fmt.Errorf("kernel: peers in mesh: %w", err)
	}
	return parsePeerIDs(list)
}

// NumPeersInMesh returns the relay mesh peer count for a pubsub topic.
func (r Relay) NumPeersInMesh(pubsubTopic string) (int, error) {
	if err := r.n.check(); err != nil {
		return 0, err
	}

	countStr, err := ffi.GetNumPeersInMesh(r.n.h, pubsubTopic)
	if err != nil {
		return 0, fmt.Errorf("kernel: num peers in mesh: %w", err)
	}
	return strconv.Atoi(countStr)
}

// ConnectedPeers returns the connected relay peers, optionally narrowed to one
// pubsub topic.
func (r Relay) ConnectedPeers(optPubsubTopic ...string) (peer.IDSlice, error) {
	if err := r.n.check(); err != nil {
		return nil, err
	}

	list, err := ffi.GetConnectedRelayPeers(r.n.h, optionalTopic(optPubsubTopic))
	if err != nil {
		return nil, fmt.Errorf("kernel: connected relay peers: %w", err)
	}
	return parsePeerIDs(list)
}

// NumConnectedPeers returns the connected relay peer count, optionally narrowed
// to one pubsub topic.
func (r Relay) NumConnectedPeers(optPubsubTopic ...string) (int, error) {
	if err := r.n.check(); err != nil {
		return 0, err
	}

	countStr, err := ffi.GetNumConnectedRelayPeers(r.n.h, optionalTopic(optPubsubTopic))
	if err != nil {
		return 0, fmt.Errorf("kernel: num connected relay peers: %w", err)
	}
	return strconv.Atoi(countStr)
}

func optionalTopic(topics []string) string {
	if len(topics) > 0 {
		return topics[0]
	}
	return ""
}
