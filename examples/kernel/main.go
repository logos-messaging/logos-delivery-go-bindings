// Command kernel is a small runnable check that the bindings work against the
// single liblogosdelivery library: it drives the unified node lifecycle
// (logosdelivery_create_node/start/stop/destroy) and a few low-level Kernel
// (waku_*) operations through pkg/kernel.
//
// Build/run requires liblogosdelivery at link time, e.g.:
//
//	export LOGOS_DELIVERY_DIR=/abs/path/to/logos-delivery   # built with `make liblogosdelivery`
//	export CGO_CFLAGS="-I$LOGOS_DELIVERY_DIR/library"
//	export CGO_LDFLAGS="-L$LOGOS_DELIVERY_DIR/build -Wl,-rpath,$LOGOS_DELIVERY_DIR/build"
//	go run ./examples/kernel
package main

import (
	"log"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
)

func main() {
	const clusterID = 16

	// A single, isolated relay node — no bootstrap/discovery needed.
	cfg := &common.WakuConfig{
		Relay:              true,
		ClusterID:          clusterID,
		Shards:             []uint16{0},
		NumShardsInNetwork: 8,
		LogLevel:           "ERROR",
	}

	// StartWakuNode creates (logosdelivery_create_node) + starts
	// (logosdelivery_start_node) the node, allocating a free TCP port.
	node, err := kernel.StartWakuNode("kernel-example", cfg)
	if err != nil {
		log.Fatalf("start node: %v", err)
	}
	defer func() {
		// logosdelivery_stop_node + logosdelivery_destroy
		if err := node.StopAndDestroy(); err != nil {
			log.Printf("stop/destroy: %v", err)
		}
	}()

	// A few low-level kernel calls to prove the kernel surface links and runs
	// over the single library.
	version, err := node.Version() // waku_version
	if err != nil {
		log.Fatalf("version: %v", err)
	}
	log.Printf("node version: %s", version)

	addrs, err := node.ListenAddresses() // waku_listen_addresses
	if err != nil {
		log.Fatalf("listen addresses: %v", err)
	}
	log.Printf("listening on: %v", addrs)

	online, err := node.IsOnline() // waku_is_online
	if err != nil {
		log.Fatalf("is online: %v", err)
	}
	log.Printf("online: %v", online)

	// Subscribe to / unsubscribe from a relay (pubsub) topic
	// (waku_relay_subscribe / waku_relay_unsubscribe).
	topic := kernel.FormatWakuRelayTopic(clusterID, 0)
	if err := node.RelaySubscribe(topic); err != nil {
		log.Fatalf("relay subscribe: %v", err)
	}
	log.Printf("subscribed to %s", topic)

	if err := node.RelayUnsubscribe(topic); err != nil {
		log.Fatalf("relay unsubscribe: %v", err)
	}
	log.Printf("unsubscribed from %s", topic)

	log.Print("OK — single liblogosdelivery library: node lifecycle + kernel ops work")
}
