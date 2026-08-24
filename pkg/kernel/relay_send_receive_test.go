package kernel

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestRelaySendReceive is an end-to-end check against the single liblogosdelivery
// library: two relay nodes are connected, one publishes a random payload and the
// other must receive it. It exercises the unified node lifecycle and the Kernel
// relay ops (waku_relay_subscribe/publish) over the one library.
func TestRelaySendReceive(t *testing.T) {
	const clusterID, shardID = 16, 64

	newNode := func(name string) *Node {
		node, err := StartWakuNode(name, &common.WakuConfig{
			Relay:           true,
			LogLevel:        "ERROR",
			Discv5Discovery: false,
			ClusterID:       clusterID,
			Shards:          []uint16{shardID},
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = node.Close() })
		return node
	}

	sender := newNode("sender")
	receiver := newNode("receiver")

	topic := FormatWakuRelayTopic(clusterID, shardID)
	require.NoError(t, sender.Relay().Subscribe(topic))
	require.NoError(t, receiver.Relay().Subscribe(topic))

	// Dial the receiver from the sender using the receiver's listen multiaddr
	// (it already embeds the peer id).
	addrs, err := receiver.ListenAddresses()
	require.NoError(t, err)
	require.NotEmpty(t, addrs)

	connCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, sender.Peers().Connect(connCtx, addrs[0]))
	require.Eventually(t, func() bool {
		n, _ := sender.Peers().NumConnected()
		return n >= 1
	}, 15*time.Second, time.Second, "sender never connected to the receiver")

	payload := make([]byte, 16)
	_, err = rand.Read(payload)
	require.NoError(t, err)

	publish := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// May fail with NoPeersToPublish until gossipsub grafts the mesh; retried.
		_, _ = sender.Relay().Publish(ctx, topic, &pb.WakuMessage{
			Payload:      payload,
			ContentTopic: "/kernel-test/1/relay/proto",
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().UnixNano()),
		})
	}

	// Publish immediately, then retry each second while waiting for delivery —
	// the relay mesh takes a moment to form after the connection is established.
	publish()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case env := <-receiver.Messages():
			if string(env.Message().GetPayload()) == string(payload) {
				return // received our exact message — success
			}
		case <-ticker.C:
			publish()
		case <-deadline:
			t.Fatal("timed out waiting for the receiver to get the message")
		}
	}
}
