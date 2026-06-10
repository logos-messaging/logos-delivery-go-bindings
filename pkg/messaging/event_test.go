package messaging

import (
	"bytes"
	"testing"
)

func TestDecodeEvent(t *testing.T) {
	t.Run("message_received with int-array payload", func(t *testing.T) {
		// liblogosdelivery serialises WakuMessage via std/json: payload/meta are
		// arrays of byte integers, not base64. "hi" = [104,105].
		const raw = `{"eventType":"message_received","messageHash":"0xabc",` +
			`"message":{"contentTopic":"/app/1/c/proto","payload":[104,105],` +
			`"meta":[1,2],"version":1,"timestamp":1717000000000000000,"ephemeral":true}}`
		ev, err := decodeEvent(raw)
		if err != nil {
			t.Fatalf("decodeEvent: %v", err)
		}
		got, ok := ev.(MessageReceivedEvent)
		if !ok {
			t.Fatalf("got %T, want MessageReceivedEvent", ev)
		}
		if got.MessageHash != "0xabc" {
			t.Errorf("messageHash = %q", got.MessageHash)
		}
		if !bytes.Equal(got.Message.Payload, []byte("hi")) {
			t.Errorf("payload = %v, want %v", got.Message.Payload, []byte("hi"))
		}
		if !bytes.Equal(got.Message.Meta, []byte{1, 2}) {
			t.Errorf("meta = %v", got.Message.Meta)
		}
		if got.Message.ContentTopic != "/app/1/c/proto" {
			t.Errorf("contentTopic = %q", got.Message.ContentTopic)
		}
		if got.Message.Version != 1 || got.Message.Timestamp != 1717000000000000000 || !got.Message.Ephemeral {
			t.Errorf("scalar fields wrong: %+v", got.Message)
		}
	})

	t.Run("message_received with base64 payload (robustness)", func(t *testing.T) {
		const raw = `{"eventType":"message_received","messageHash":"0x1",` +
			`"message":{"contentTopic":"/a/1/b/proto","payload":"aGk=","ephemeral":false}}`
		ev, err := decodeEvent(raw)
		if err != nil {
			t.Fatalf("decodeEvent: %v", err)
		}
		if got := ev.(MessageReceivedEvent); !bytes.Equal(got.Message.Payload, []byte("hi")) {
			t.Errorf("payload = %v", got.Message.Payload)
		}
	})

	t.Run("message_sent", func(t *testing.T) {
		ev, err := decodeEvent(`{"eventType":"message_sent","requestId":"req-1","messageHash":"0x9"}`)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := ev.(MessageSentEvent)
		if !ok || got.RequestID != "req-1" || got.MessageHash != "0x9" {
			t.Fatalf("got %#v", ev)
		}
	})

	t.Run("message_propagated", func(t *testing.T) {
		ev, err := decodeEvent(`{"eventType":"message_propagated","requestId":"req-2","messageHash":"0x8"}`)
		if err != nil {
			t.Fatal(err)
		}
		if got, ok := ev.(MessagePropagatedEvent); !ok || got.RequestID != "req-2" {
			t.Fatalf("got %#v", ev)
		}
	})

	t.Run("message_error", func(t *testing.T) {
		ev, err := decodeEvent(`{"eventType":"message_error","requestId":"req-3","messageHash":"0x7","error":"boom"}`)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := ev.(MessageErrorEvent)
		if !ok || got.RequestID != "req-3" || got.Err != "boom" {
			t.Fatalf("got %#v", ev)
		}
	})

	t.Run("connection_status_change", func(t *testing.T) {
		for in, want := range map[string]ConnectionStatus{
			"Connected":          Connected,
			"PartiallyConnected": PartiallyConnected,
			"Disconnected":       Disconnected,
		} {
			raw := `{"eventType":"connection_status_change","connectionStatus":"` + in + `"}`
			ev, err := decodeEvent(raw)
			if err != nil {
				t.Fatalf("%s: %v", in, err)
			}
			if got := ev.(ConnectionStatusEvent); got.Status != want {
				t.Errorf("%s -> %v, want %v", in, got.Status, want)
			}
		}
	})

	t.Run("unknown event type is ignored", func(t *testing.T) {
		ev, err := decodeEvent(`{"eventType":"something_new","foo":1}`)
		if err != nil || ev != nil {
			t.Fatalf("got ev=%v err=%v, want nil,nil", ev, err)
		}
	})
}
