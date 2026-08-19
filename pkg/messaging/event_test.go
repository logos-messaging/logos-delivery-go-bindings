package messaging

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"
)

// The JSON documents below are verbatim captures from a running
// liblogosdelivery node, so these tests pin the decoder to the real wire
// format rather than to an assumption about it.
const (
	connectionStatusJSON  = `{"eventType":"connection_status_change","connectionStatus":"Connected"}`
	messageReceivedJSON   = `{"eventType":"message_received","messageHash":"0x270090e9d88219b9e8f8a51820664ff2a972e9e101cdd87b584d547d40582118","message":{"payload":[104,101,108,108,111],"contentTopic":"/logos-delivery-go-bindings/1/raw/proto","meta":[],"version":0,"timestamp":1787098057353072384,"ephemeral":false,"proof":[]}}`
	messagePropagatedJSON = `{"eventType":"message_propagated","requestId":"f9620781ac7c85234b41","messageHash":"0x270090e9d88219b9e8f8a51820664ff2a972e9e101cdd87b584d547d40582118"}`
	messageSentJSON       = `{"eventType":"message_sent","requestId":"f9620781ac7c85234b41","messageHash":"0x270090e9d88219b9e8f8a51820664ff2a972e9e101cdd87b584d547d40582118"}`
	messageErrorJSON      = `{"eventType":"message_error","requestId":"f9620781ac7c85234b41","messageHash":"0x2700","error":"Unable to send within retry time window"}`
)

func TestDecodeMessageReceived(t *testing.T) {
	ev, err := decodeEvent(messageReceivedJSON)
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	e, ok := ev.(MessageReceivedEvent)
	if !ok {
		t.Fatalf("got %T, want MessageReceivedEvent", ev)
	}
	if want := "0x270090e9d88219b9e8f8a51820664ff2a972e9e101cdd87b584d547d40582118"; e.MessageHash != want {
		t.Errorf("MessageHash = %q, want %q", e.MessageHash, want)
	}
	// The payload crosses as a JSON array of byte integers, not base64.
	if !bytes.Equal(e.Message.Payload, []byte("hello")) {
		t.Errorf("Payload = %q, want %q", e.Message.Payload, "hello")
	}
	if want := "/logos-delivery-go-bindings/1/raw/proto"; e.Message.ContentTopic != want {
		t.Errorf("ContentTopic = %q, want %q", e.Message.ContentTopic, want)
	}
	if len(e.Message.Meta) != 0 {
		t.Errorf("Meta = %v, want empty", e.Message.Meta)
	}
	if e.Message.Timestamp != 1787098057353072384 {
		t.Errorf("Timestamp = %d, want 1787098057353072384", e.Message.Timestamp)
	}
	if e.Message.Version != 0 || e.Message.Ephemeral {
		t.Errorf("Version/Ephemeral = %d/%v, want 0/false", e.Message.Version, e.Message.Ephemeral)
	}
}

func TestDecodeSendLifecycleEvents(t *testing.T) {
	const requestID = RequestID("f9620781ac7c85234b41")

	sent, err := decodeEvent(messageSentJSON)
	if err != nil {
		t.Fatalf("decodeEvent(sent): %v", err)
	}
	if e, ok := sent.(MessageSentEvent); !ok || e.RequestID != requestID {
		t.Errorf("got %#v, want MessageSentEvent with request id %s", sent, requestID)
	}

	propagated, err := decodeEvent(messagePropagatedJSON)
	if err != nil {
		t.Fatalf("decodeEvent(propagated): %v", err)
	}
	if e, ok := propagated.(MessagePropagatedEvent); !ok || e.RequestID != requestID {
		t.Errorf("got %#v, want MessagePropagatedEvent with request id %s", propagated, requestID)
	}

	failed, err := decodeEvent(messageErrorJSON)
	if err != nil {
		t.Fatalf("decodeEvent(error): %v", err)
	}
	e, ok := failed.(MessageErrorEvent)
	if !ok {
		t.Fatalf("got %T, want MessageErrorEvent", failed)
	}
	if e.RequestID != requestID {
		t.Errorf("RequestID = %s, want %s", e.RequestID, requestID)
	}
	if want := "Unable to send within retry time window"; e.Err != want {
		t.Errorf("Err = %q, want %q", e.Err, want)
	}
}

func TestDecodeConnectionStatus(t *testing.T) {
	for _, tc := range []struct {
		name string
		want ConnectionStatus
	}{
		{"Connected", Connected},
		{"PartiallyConnected", PartiallyConnected},
		{"Disconnected", Disconnected},
	} {
		raw := `{"eventType":"connection_status_change","connectionStatus":"` + tc.name + `"}`
		ev, err := decodeEvent(raw)
		if err != nil {
			t.Fatalf("decodeEvent(%s): %v", tc.name, err)
		}
		e, ok := ev.(ConnectionStatusEvent)
		if !ok {
			t.Fatalf("got %T, want ConnectionStatusEvent", ev)
		}
		if e.Status != tc.want {
			t.Errorf("Status = %v, want %v", e.Status, tc.want)
		}
		if e.Status.String() != tc.name {
			t.Errorf("Status.String() = %q, want %q", e.Status.String(), tc.name)
		}
	}
}

// An unknown eventType is not an error: a listener registered for a wider set
// of events must be able to ignore what it does not model.
func TestDecodeUnknownEventIsIgnored(t *testing.T) {
	ev, err := decodeEvent(`{"eventType":"relay_topic_health_change","pubsubTopic":"/waku/2/rs/3/0","topicHealth":"SufficientlyHealthy"}`)
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if ev != nil {
		t.Errorf("got %#v, want nil", ev)
	}
}

func TestDecodeMalformedEventIsAnError(t *testing.T) {
	if _, err := decodeEvent(`not json`); err == nil {
		t.Error("decodeEvent(garbage) succeeded, want an error")
	}
	if _, err := decodeEvent(`{"eventType":"message_received","message":{"payload":"!!!not base64!!!"}}`); err == nil {
		t.Error("decodeEvent(bad payload) succeeded, want an error")
	}
}

// wireBytes also accepts the base64 and null encodings, so a field that moves
// to the encoding the send path uses keeps decoding.
func TestWireBytesAlternativeEncodings(t *testing.T) {
	var b wireBytes
	if err := json.Unmarshal([]byte(`"`+base64.StdEncoding.EncodeToString([]byte("hello"))+`"`), &b); err != nil {
		t.Fatalf("base64: %v", err)
	}
	if !bytes.Equal(b, []byte("hello")) {
		t.Errorf("base64 decoded to %q, want %q", b, "hello")
	}

	b = wireBytes("stale")
	if err := json.Unmarshal([]byte(`null`), &b); err != nil {
		t.Fatalf("null: %v", err)
	}
	if b != nil {
		t.Errorf("null decoded to %v, want nil", b)
	}
}
