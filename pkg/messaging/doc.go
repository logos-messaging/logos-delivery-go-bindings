// Package messaging is the high-level, idiomatic Go binding for the
// logos-delivery Messaging API: an opinionated layer over the kernel protocols
// that owns reliability, re-subscriptions, store-based catch-up and the
// messaging event surface.
//
// It mirrors the Nim MessagingClient. A MessagingClient owns a node, and the
// messaging operations are methods on it rather than free functions over a
// context handle:
//
//	client, err := messaging.New(messaging.Config{
//		Mode:   messaging.ModeCore,
//		Preset: messaging.PresetLogosDev,
//	})
//	if err != nil {
//		return err
//	}
//	defer client.Close()
//
//	go func() {
//		for ev := range client.Events() {
//			switch e := ev.(type) {
//			case messaging.MessageReceivedEvent:
//				log.Printf("received %q", e.Message.Payload)
//			case messaging.MessagePropagatedEvent:
//				log.Printf("%s propagated", e.RequestID)
//			}
//		}
//	}()
//
//	if err := client.Start(); err != nil {
//		return err
//	}
//	if err := client.Subscribe(topic); err != nil {
//		return err
//	}
//	requestID, err := client.Send(ctx, messaging.Envelope{
//		ContentTopic: topic,
//		Payload:      []byte("hello"),
//	})
//
// Send returns once the message is accepted, not once it is delivered.
// Delivery is reported asynchronously on Events(), correlated by RequestID:
// MessagePropagatedEvent when the message reaches neighbouring nodes,
// MessageSentEvent when store-based validation confirms it, and
// MessageErrorEvent when it fails.
//
// Events() never blocks the library: an event is dropped if the channel is
// full, so consume it from a dedicated goroutine for the client's lifetime.
//
// This package does not expose the underlying node handle or the kernel
// protocols; use pkg/kernel for those.
package messaging
