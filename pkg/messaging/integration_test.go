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
	"fmt"
	"testing"
	"time"
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

	requestID, err := client.Send(ctx, Envelope{ContentTopic: contentTopic, Payload: payload})
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
