// Command kernel is a small runnable check that the bindings work against the
// single liblogosdelivery library. It brings up two relay nodes, connects them,
// publishes a message from one, and confirms the other receives it — end-to-end
// send/receive over the unified lifecycle (logosdelivery_create_node/start/
// stop/destroy) and the Kernel relay ops (waku_relay_publish, plus the mesh the
// node forms for its configured shard).
//
// Build/run requires liblogosdelivery at link time, e.g.:
//
//	export LOGOS_DELIVERY_DIR=/abs/path/to/logos-delivery   # built with `make liblogosdelivery`
//	export CGO_CFLAGS="-I$LOGOS_DELIVERY_DIR/library"
//	export CGO_LDFLAGS="-L$LOGOS_DELIVERY_DIR/build -Wl,-rpath,$LOGOS_DELIVERY_DIR/build"
//	go run ./examples/kernel
package main

import (
	"context"
	"log"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/pb"
	"google.golang.org/protobuf/proto"
)

const (
	clusterID = 16
	shardID   = 64
)

func newNode(name string) *kernel.WakuNode {
	// A relay node on cluster 16, static shard 0 — the node auto-subscribes to
	// its configured shard, so no explicit RelaySubscribe or discovery is needed.
	node, err := kernel.StartWakuNode(name, &common.WakuConfig{
		Relay:           true,
		LogLevel:        "ERROR",
		Discv5Discovery: false,
		ClusterID:       clusterID,
		Shards:          []uint16{shardID},
	})
	if err != nil {
		log.Fatalf("start %s: %v", name, err)
	}
	return node
}

func stopAndDestroy(node *kernel.WakuNode) {
	if err := node.StopAndDestroy(); err != nil {
		log.Printf("stop/destroy: %v", err)
	}
}

func main() {
	sender := newNode("sender")
	defer stopAndDestroy(sender)
	receiver := newNode("receiver")
	defer stopAndDestroy(receiver)

	topic := kernel.FormatWakuRelayTopic(clusterID, shardID)
	if err := sender.RelaySubscribe(topic); err != nil {
		log.Fatalf("sender subscribe: %v", err)
	}
	if err := receiver.RelaySubscribe(topic); err != nil {
		log.Fatalf("receiver subscribe: %v", err)
	}

	// Dial the receiver from the sender using the receiver's listen multiaddr
	// (it already embeds the peer id).
	addrs, err := receiver.ListenAddresses()
	if err != nil || len(addrs) == 0 {
		log.Fatalf("receiver listen addresses: %v", err)
	}
	connCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := sender.Connect(connCtx, addrs[0]); err != nil {
		log.Fatalf("connect sender -> receiver: %v", err)
	}
	// The dial is asynchronous; wait for the peer to actually connect.
	for i := 0; ; i++ {
		if n, _ := sender.GetNumConnectedPeers(); n >= 1 {
			log.Printf("nodes connected (%d peer)", n)
			break
		}
		if i >= 15 {
			log.Fatal("no peer connected after 15s")
		}
		time.Sleep(1 * time.Second)
	}

	payload := []byte("hello over relay")
	publish := func() {
		msg := &pb.WakuMessage{
			Payload:      payload,
			ContentTopic: "/kernel-example/1/relay/proto",
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().UnixNano()),
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if hash, err := sender.RelayPublish(ctx, msg, topic); err != nil {
			log.Printf("publish (retrying): %v", err)
		} else {
			log.Printf("sender published, hash: %s", hash.String())
		}
	}

	// Publish immediately, then retry each second — gossipsub needs a moment to
	// graft the mesh after the connection is established.
	publish()
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case env := <-receiver.MsgChan:
			if got := env.Message().GetPayload(); string(got) == string(payload) {
				log.Printf("receiver got %q on %s — send/receive OK over the single library", got, env.PubsubTopic())
				return
			}
		case <-ticker.C:
			publish()
		case <-deadline:
			log.Fatal("timed out waiting for the receiver to get the message")
		}
	}
}
