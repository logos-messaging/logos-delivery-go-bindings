package kernel

import (
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/require"
)

// waitForAutoConnection blocks until every node has at least one connected
// peer.
func waitForAutoConnection(t *testing.T, nodeList []*Node) {
	t.Helper()

	require.NoError(t, WaitForAutoConnection(nodeList), "nodes did not connect")
}

// waitForConnectionChange drains the node's connection-change events until the
// one carrying peerEvent arrives. The peer manager also emits
// EventMetadataUpdated when a peer connects, so the wanted event is not
// necessarily the first one on the channel.
func (n *Node) waitForConnectionChange(t *testing.T, peerEvent string, timeout time.Duration) ConnectionChange {
	t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case change := <-n.ConnectionChanges():
			if change.PeerEvent == peerEvent {
				return change
			}
			logDebug("Ignoring connection change %s while waiting for %s", change.PeerEvent, peerEvent)
		case <-deadline:
			t.Fatalf("timeout waiting for connection change %s", peerEvent)
			return ConnectionChange{}
		}
	}
}

// waitForRelayMesh blocks until every node holds at least minPeers gossipsub
// mesh peers on topic. The library only subscribes a node to its configured
// shards when the embedding app registers a relay handler, which the FFI layer
// never does, so callers must subscribe first or the mesh stays empty and
// publishing fails with NoPeersToPublish.
func waitForRelayMesh(t *testing.T, nodeList []*Node, topic string, minPeers int) {
	t.Helper()

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	err := RetryWithBackOff(func() error {
		for i, node := range nodeList {
			numPeers, err := node.Relay().NumPeersInMesh(topic)
			if err != nil {
				return err
			}

			if numPeers < minPeers {
				return fmt.Errorf("node %d has %d mesh peers on %s, want %d", i, numPeers, topic, minPeers)
			}
		}

		return nil
	}, options)
	require.NoError(t, err, "relay mesh did not form on %s", topic)
}

// waitForMeshPeerCount blocks until the node's gossipsub mesh on topic holds
// exactly want peers, so a test can assert a count without guessing how long
// the mesh takes to settle.
func (n *Node) waitForMeshPeerCount(t *testing.T, topic string, want int) {
	t.Helper()

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	err := RetryWithBackOff(func() error {
		numPeers, err := n.Relay().NumPeersInMesh(topic)
		if err != nil {
			return err
		}

		if numPeers != want {
			return fmt.Errorf("node has %d mesh peers on %s, want %d", numPeers, topic, want)
		}

		return nil
	}, options)
	require.NoError(t, err, "mesh did not settle at %d peers on %s", want, topic)
}

// waitUntilOnline blocks until the node reports itself online. The health
// monitor derives that state on its own schedule, so it lags the connection
// that produced it.
func (n *Node) waitUntilOnline(t *testing.T) {
	t.Helper()

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	err := RetryWithBackOff(func() error {
		online, err := n.Debug().IsOnline()
		if err != nil {
			return err
		}

		if !online {
			return fmt.Errorf("node is not online yet")
		}

		return nil
	}, options)
	require.NoError(t, err, "node did not come online")
}

// subscribeAndWaitForMesh subscribes every node to topic and waits until each
// one has a mesh peer, the state a relay publish needs to reach anyone.
func subscribeAndWaitForMesh(t *testing.T, nodeList []*Node, topic string) {
	t.Helper()

	require.NoError(t, SubscribeNodesToTopic(nodeList, topic), "failed to subscribe nodes to %s", topic)
	waitForRelayMesh(t, nodeList, topic, 1)
}
