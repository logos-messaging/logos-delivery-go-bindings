package common

import (
	"encoding/json"
	"testing"
)

// storeQueryResponseCapture is a verbatim reply from a logos.dev store node,
// trimmed to one message. It pins the two shapes the decoder has to survive:
// Opt[T] fields rendered as {"oResultPrivate":...,"vResultPrivate":...}, and
// byte fields rendered as arrays of integers rather than base64.
const storeQueryResponseCapture = `{
  "requestId": "726177",
  "statusCode": 200,
  "statusDesc": "OK",
  "messages": [
    {
      "messageHash": "0x14865d35f6e381b1eb48960bf93aa638aedb9a0822ce44747148c78e759551f5",
      "message": {
        "oResultPrivate": true,
        "vResultPrivate": {
          "payload": [104, 101, 108, 108, 111],
          "contentTopic": "/atomic-swaps/1/offers/json",
          "meta": [],
          "version": 0,
          "timestamp": 1787589388212000000,
          "ephemeral": false,
          "proof": []
        }
      },
      "pubsubTopic": {"oResultPrivate": true, "vResultPrivate": "/waku/2/rs/3/7"}
    }
  ],
  "paginationCursor": {
    "oResultPrivate": true,
    "vResultPrivate": "0x14865d35f6e381b1eb48960bf93aa638aedb9a0822ce44747148c78e759551f5"
  }
}`

func TestStoreQueryResponseDecodesLibraryWireShape(t *testing.T) {
	var got StoreQueryResponse
	if err := json.Unmarshal([]byte(storeQueryResponseCapture), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.RequestId != "726177" {
		t.Errorf("RequestId = %q, want 726177", got.RequestId)
	}
	if got.StatusCode == nil || *got.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", got.StatusCode)
	}
	wantCursor := MessageHash("0x14865d35f6e381b1eb48960bf93aa638aedb9a0822ce44747148c78e759551f5")
	if got.PaginationCursor != wantCursor {
		t.Errorf("PaginationCursor = %q, want %q", got.PaginationCursor, wantCursor)
	}

	if got.Messages == nil || len(*got.Messages) != 1 {
		t.Fatalf("Messages = %v, want one message", got.Messages)
	}
	message := (*got.Messages)[0]

	if message.PubsubTopic != "/waku/2/rs/3/7" {
		t.Errorf("PubsubTopic = %q, want /waku/2/rs/3/7", message.PubsubTopic)
	}
	if string(message.Payload()) != "hello" {
		t.Errorf("Payload() = %q, want hello", message.Payload())
	}
	if message.ContentTopic() != "/atomic-swaps/1/offers/json" {
		t.Errorf("ContentTopic() = %q, want /atomic-swaps/1/offers/json", message.ContentTopic())
	}
}

// An empty Opt has to leave the field zeroed rather than fail the whole decode.
func TestStoreQueryResponseDecodesAbsentOptionals(t *testing.T) {
	const absent = `{
	  "requestId": "abc",
	  "messages": [{"messageHash": "0xdead", "message": {"oResultPrivate": false},
	                "pubsubTopic": {"oResultPrivate": false}}],
	  "paginationCursor": {"oResultPrivate": false}
	}`

	var got StoreQueryResponse
	if err := json.Unmarshal([]byte(absent), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.PaginationCursor != "" {
		t.Errorf("PaginationCursor = %q, want empty", got.PaginationCursor)
	}

	message := (*got.Messages)[0]
	if message.WakuMessage != nil {
		t.Errorf("WakuMessage = %v, want nil", message.WakuMessage)
	}
	if message.PubsubTopic != "" {
		t.Errorf("PubsubTopic = %q, want empty", message.PubsubTopic)
	}
	if message.Payload() != nil {
		t.Errorf("Payload() = %v, want nil", message.Payload())
	}
}

// The bare shape has to keep decoding, so normalising the library's encodings
// later is not a breaking change here.
func TestStoreQueryResponseDecodesBareShape(t *testing.T) {
	const bare = `{
	  "requestId": "abc",
	  "messages": [{"messageHash": "0xdead", "pubsubTopic": "/waku/2/rs/1/0",
	                "message": {"payload": "aGVsbG8=", "contentTopic": "/a/1/b/proto"}}],
	  "paginationCursor": "0xbeef"
	}`

	var got StoreQueryResponse
	if err := json.Unmarshal([]byte(bare), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.PaginationCursor != "0xbeef" {
		t.Errorf("PaginationCursor = %q, want 0xbeef", got.PaginationCursor)
	}

	message := (*got.Messages)[0]
	if message.PubsubTopic != "/waku/2/rs/1/0" {
		t.Errorf("PubsubTopic = %q, want /waku/2/rs/1/0", message.PubsubTopic)
	}
	if string(message.Payload()) != "hello" {
		t.Errorf("Payload() = %q, want hello", message.Payload())
	}
}
