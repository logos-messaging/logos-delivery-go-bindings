package kernel

import (
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/require"
)

// waitForConnectionChange drains the node's connection-change events until the
// one carrying peerEvent arrives. The peer manager also emits
// EventMetadataUpdated when a peer connects, so the wanted event is not
// necessarily the first one on the channel.
func (n *WakuNode) waitForConnectionChange(t *testing.T, peerEvent string, timeout time.Duration) connectionChange {
	t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case change := <-n.ConnectionChangeChan:
			if change.PeerEvent == peerEvent {
				return change
			}
			Debug("Ignoring connection change %s while waiting for %s", change.PeerEvent, peerEvent)
		case <-deadline:
			t.Fatalf("timeout waiting for connection change %s on %s", peerEvent, n.nodeName)
			return connectionChange{}
		}
	}
}

// waitForRelayMesh blocks until every node holds at least minPeers gossipsub
// mesh peers on topic. The library only subscribes a node to its configured
// shards when the embedding app registers a relay handler, which the FFI layer
// never does, so callers must RelaySubscribe first or the mesh stays empty and
// publishing fails with NoPeersToPublish.
func waitForRelayMesh(t *testing.T, nodeList []*WakuNode, topic string, minPeers int) {
	t.Helper()

	Debug("Waiting for the relay mesh to form on topic %s", topic)

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	err := RetryWithBackOff(func() error {
		for _, node := range nodeList {
			numPeers, err := node.GetNumPeersInMesh(topic)
			if err != nil {
				return err
			}

			if numPeers < minPeers {
				return fmt.Errorf("node %s has %d mesh peers on %s, want %d",
					node.nodeName, numPeers, topic, minPeers)
			}
		}

		return nil
	}, options)
	require.NoError(t, err, "relay mesh did not form on %s", topic)

	Debug("Relay mesh formed on topic %s", topic)
}

// waitForMeshPeerCount blocks until the node's gossipsub mesh on topic holds
// exactly want peers, so a test can assert a count without guessing how long
// the mesh takes to settle.
func (n *WakuNode) waitForMeshPeerCount(t *testing.T, topic string, want int) {
	t.Helper()

	Debug("Waiting for %s to hold %d mesh peers on %s", n.nodeName, want, topic)

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	err := RetryWithBackOff(func() error {
		numPeers, err := n.GetNumPeersInMesh(topic)
		if err != nil {
			return err
		}

		if numPeers != want {
			return fmt.Errorf("node %s has %d mesh peers on %s, want %d",
				n.nodeName, numPeers, topic, want)
		}

		return nil
	}, options)
	require.NoError(t, err, "%s mesh did not settle at %d peers on %s", n.nodeName, want, topic)
}

// waitUntilOnline blocks until the node reports itself online. The health
// monitor derives that state on its own schedule, so it lags the connection
// that produced it.
func (n *WakuNode) waitUntilOnline(t *testing.T) {
	t.Helper()

	Debug("Waiting for %s to come online", n.nodeName)

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	err := RetryWithBackOff(func() error {
		online, err := n.IsOnline()
		if err != nil {
			return err
		}

		if !online {
			return fmt.Errorf("node %s is not online yet", n.nodeName)
		}

		return nil
	}, options)
	require.NoError(t, err, "%s did not come online", n.nodeName)
}

// subscribeAndWaitForMesh subscribes every node to topic and waits until each
// one has a mesh peer, the state a RelayPublish needs to reach anyone.
func subscribeAndWaitForMesh(t *testing.T, nodeList []*WakuNode, topic string) {
	t.Helper()

	require.NoError(t, SubscribeNodesToTopic(nodeList, topic), "failed to subscribe nodes to %s", topic)
	waitForRelayMesh(t, nodeList, topic, 1)
}
