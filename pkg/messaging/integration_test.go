//go:build integration

// Integration coverage for the Messaging API against a real network. It needs
// a built liblogosdelivery and outbound connectivity, so it is behind a build
// tag and is not part of the PR gate:
//
//	go test -tags integration -v ./pkg/messaging/...
package messaging

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/store"
)

// TestSendReceiveRoundTrip drives the full client surface against the Logos Dev
// network: create, start, subscribe, send, and observe the resulting events.
// A node relays its own published messages back to itself, so the message sent
// here is also the one received.
func TestSendReceiveRoundTrip(t *testing.T) {
	contentTopic := "/logos-delivery-go-bindings/1/it/proto"
	payload := []byte(fmt.Sprintf("round trip %d", time.Now().UnixNano()))

	client, err := New(Config{
		Mode:   ModeCore,
		Preset: PresetLogosDev,
		MessagingOverrides: Overrides{
			"listen-address": "0.0.0.0",
			"tcp-port":       60123,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := client.Subscribe(contentTopic); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Wait for the node to reach the network before publishing.
	waitFor(t, client, 60*time.Second, func(ev Event) bool {
		e, ok := ev.(ConnectionStatusEvent)
		return ok && e.Status == Connected
	}, "connected")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	requestID, err := client.Send(ctx, contentTopic, payload, false)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if requestID == "" {
		t.Fatal("Send returned an empty request id")
	}

	waitFor(t, client, 60*time.Second, func(ev Event) bool {
		e, ok := ev.(MessageReceivedEvent)
		return ok && e.Message.ContentTopic == contentTopic && bytes.Equal(e.Message.Payload, payload)
	}, "the published message back on Events()")

	waitFor(t, client, 60*time.Second, func(ev Event) bool {
		e, ok := ev.(MessagePropagatedEvent)
		return ok && e.RequestID == requestID
	}, "a propagation confirmation for the sent request id")
}

// waitFor drains the event stream until match accepts an event or time runs out.
func waitFor(t *testing.T, c *MessagingClient, timeout time.Duration, match func(Event) bool, what string) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-c.Events():
			if !ok {
				t.Fatalf("events channel closed while waiting for %s", what)
			}
			t.Logf("event: %#v", ev)
			if match(ev) {
				return
			}
		case <-deadline:
			t.Fatalf("timed out after %s waiting for %s", timeout, what)
		}
	}
}

// TestKernelAPIOnClientNode is what the Node accessor exists for: the Kernel
// API has to work against the very node the client is running, not a second
// one. It drives the identity, peer and store surfaces off client.Node().
func TestKernelAPIOnClientNode(t *testing.T) {
	client, err := New(Config{
		Mode:   ModeCore,
		Preset: PresetLogosDev,
		MessagingOverrides: Overrides{
			"listen-address": "0.0.0.0",
			"tcp-port":       60124,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitFor(t, client, 60*time.Second, func(ev Event) bool {
		e, ok := ev.(ConnectionStatusEvent)
		return ok && e.Status == Connected
	}, "connected")

	node := client.Node()

	peerID, err := node.Debug().PeerID()
	if err != nil {
		t.Fatalf("PeerID: %v", err)
	}
	if peerID == "" {
		t.Fatal("PeerID returned an empty id")
	}
	t.Logf("peer id: %s", peerID)

	version, err := node.Debug().Version()
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	t.Logf("version: %s", version)

	if _, err := node.Debug().ListenAddresses(); err != nil {
		t.Fatalf("ListenAddresses: %v", err)
	}
	if _, err := node.Debug().ENR(); err != nil {
		t.Fatalf("ENR: %v", err)
	}

	metrics, err := node.Debug().Metrics()
	if err != nil {
		t.Fatalf("Metrics: %v", err)
	}
	if len(metrics) == 0 {
		t.Fatal("Metrics returned nothing")
	}

	connected, err := node.Peers().NumConnected()
	if err != nil {
		t.Fatalf("NumConnected: %v", err)
	}
	if connected == 0 {
		t.Fatal("node reports Connected but has no peers")
	}
	t.Logf("connected peers: %d", connected)

	if _, err := node.Relay().NumConnectedPeers(); err != nil {
		t.Fatalf("Relay NumConnectedPeers: %v", err)
	}

	storePeer, ok := findStorePeer(t, node)
	if !ok {
		t.Skip("no connected store peer to query")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	limit := uint64(1)
	resp, err := node.Store().Query(ctx, &common.StoreQueryRequest{
		RequestId:       hex.EncodeToString([]byte("go-bindings-it")),
		IncludeData:     true,
		PaginationLimit: &limit,
	}, storePeer)
	if err != nil {
		t.Fatalf("Store query on the client's node: %v", err)
	}
	t.Logf("store query status: %v %s", resp.StatusCode, resp.StatusDesc)
}

// findStorePeer picks a connected peer that serves store queries, with the
// addresses needed to dial it.
func findStorePeer(t *testing.T, node *kernel.Node) (peer.AddrInfo, bool) {
	t.Helper()

	info, err := node.Peers().ConnectedInfo()
	if err != nil {
		t.Fatalf("ConnectedInfo: %v", err)
	}

	for id, data := range info {
		for _, protocol := range data.Protocols {
			if protocol == store.StoreQueryID_v300 && len(data.Addresses) > 0 {
				return peer.AddrInfo{ID: id, Addrs: data.Addresses}, true
			}
		}
	}
	return peer.AddrInfo{}, false
}
