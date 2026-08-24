// Command messaging is a runnable demonstration of the Messaging API: it
// starts a node, subscribes to a content topic, sends a message on it and
// prints every event the node reports until interrupted.
//
// Build it against a local liblogosdelivery:
//
//	export LOGOS_DELIVERY_DIR=/path/to/logos-delivery
//	export CGO_CFLAGS="-I$LOGOS_DELIVERY_DIR/library/"
//	export CGO_LDFLAGS="-L$LOGOS_DELIVERY_DIR/build/ -Wl,-rpath,$LOGOS_DELIVERY_DIR/build/"
//	go run ./examples/messaging
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"
)

const contentTopic = "/logos-delivery-go-bindings/1/example/proto"

func main() {
	client, err := messaging.New(messaging.Config{
		Mode:   messaging.ModeCore,
		Preset: messaging.PresetLogosDev,
		MessagingOverrides: messaging.Overrides{
			"listen-address": "0.0.0.0",
			"tcp-port":       60000,
		},
	})
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("close: %v", err)
		}
	}()

	// Consume events for the client's whole lifetime. Events() is closed by
	// Close, which ends this goroutine.
	go printEvents(client.Events())

	if err := client.Start(); err != nil {
		log.Fatalf("start: %v", err)
	}
	log.Printf("node started")

	if err := client.Subscribe(contentTopic); err != nil {
		log.Fatalf("subscribe: %v", err)
	}
	log.Printf("subscribed to %s", contentTopic)

	// Give the node a moment to find peers before publishing.
	time.Sleep(5 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	requestID, err := client.Send(ctx, contentTopic, []byte("hello from logos-delivery-go-bindings"), false)
	if err != nil {
		log.Fatalf("send: %v", err)
	}
	log.Printf("sent, request id %s", requestID)

	// Run until interrupted so the delivery events have time to arrive.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Printf("shutting down")

	if err := client.Stop(); err != nil {
		log.Printf("stop: %v", err)
	}
}

// printEvents type-switches over the sealed Event interface. Keep the default
// branch: the event set grows over time.
func printEvents(events <-chan messaging.Event) {
	for ev := range events {
		switch e := ev.(type) {
		case messaging.MessageReceivedEvent:
			log.Printf("received %q on %s (hash %s)",
				e.Message.Payload, e.Message.ContentTopic, e.MessageHash)
		case messaging.MessageSentEvent:
			log.Printf("sent %s (hash %s)", e.RequestID, e.MessageHash)
		case messaging.MessagePropagatedEvent:
			log.Printf("propagated %s (hash %s)", e.RequestID, e.MessageHash)
		case messaging.MessageErrorEvent:
			log.Printf("error %s (hash %s): %s", e.RequestID, e.MessageHash, e.Err)
		case messaging.ConnectionStatusEvent:
			log.Printf("connection status: %s", e.Status)
		default:
			log.Printf("unhandled event %T", e)
		}
	}
}
