// Command messaging is a minimal demonstration of the Messaging API binding:
// create a client, start it, subscribe to a content topic, send a message, and
// print the events that arrive.
//
// Build/run requires liblogosdelivery at link time, e.g.:
//
//	export LOGOS_DELIVERY_DIR=/abs/path/to/logos-delivery
//	export CGO_CFLAGS="-I$LOGOS_DELIVERY_DIR/liblogosdelivery"
//	export CGO_LDFLAGS="-L$LOGOS_DELIVERY_DIR/build -Wl,-rpath,$LOGOS_DELIVERY_DIR/build"
//	go run ./examples/messaging
package main

import (
	"context"
	"log"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"
)

func main() {
	const contentTopic = "/my-app/1/chat/proto"

	cfg := messaging.Config{
		Relay:              true,
		ClusterID:          16,
		Shards:             []uint16{0},
		NumShardsInNetwork: 8,
		LogLevel:           "INFO",
	}

	client, err := messaging.New(cfg)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("close: %v", err)
		}
	}()

	if err := client.Start(); err != nil {
		log.Fatalf("start: %v", err)
	}
	defer func() {
		if err := client.Stop(); err != nil {
			log.Printf("stop: %v", err)
		}
	}()

	if err := client.Subscribe(contentTopic); err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	// Print incoming events until the program exits.
	go func() {
		for ev := range client.Events() {
			switch e := ev.(type) {
			case messaging.MessageReceivedEvent:
				log.Printf("received on %s: %q", e.Message.ContentTopic, e.Message.Payload)
			case messaging.MessageSentEvent:
				log.Printf("sent: req=%s hash=%s", e.RequestID, e.MessageHash)
			case messaging.MessagePropagatedEvent:
				log.Printf("propagated: req=%s", e.RequestID)
			case messaging.MessageErrorEvent:
				log.Printf("error: req=%s %s", e.RequestID, e.Err)
			case messaging.ConnectionStatusEvent:
				log.Printf("connection status: %s", e.Status)
			}
		}
	}()

	reqID, err := client.Send(context.Background(), messaging.Envelope{
		ContentTopic: contentTopic,
		Payload:      []byte("hello logos"),
	})
	if err != nil {
		log.Fatalf("send: %v", err)
	}
	log.Printf("queued message, request id: %s", reqID)

	time.Sleep(5 * time.Second)
}
