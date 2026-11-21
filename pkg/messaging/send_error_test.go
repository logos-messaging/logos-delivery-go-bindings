package messaging

import (
	"errors"
	"testing"
	"time"
)

func TestSendErrorKeepsTheKind(t *testing.T) {
	cause := errors.New("Send failed: node has no connected peer (NotConnected)")
	err := sendError(cause)
	var se *SendError
	if !errors.As(err, &se) {
		t.Fatalf("got %v, want a *SendError", err)
	}
	if se.Kind != SendNotConnected || se.Message != "node has no connected peer" {
		t.Errorf("got %+v", se)
	}
	if !errors.Is(err, cause) {
		t.Errorf("%v does not wrap %v", err, cause)
	}
}

func TestSendErrorReadsRetryAfter(t *testing.T) {
	err := sendError(errors.New("Send failed: failed to enqueue the message: the rate-limit budget of the epoch is spent, retry after: 42 s (RateLimited)"))
	var se *SendError
	if !errors.As(err, &se) {
		t.Fatalf("got %v, want a *SendError", err)
	}
	if se.Kind != SendRateLimited || se.RetryAfter != 42*time.Second {
		t.Errorf("got %+v", se)
	}
}

func TestSendErrorSpanningLines(t *testing.T) {
	err := sendError(errors.New("Send failed: failed to prepare the message: x\ny (NoShardMapsContentTopic)"))
	var se *SendError
	if !errors.As(err, &se) {
		t.Fatalf("got %v, want a *SendError", err)
	}
	if se.Kind != SendNoShardMapsContentTopic || se.Message != "failed to prepare the message: x\ny" {
		t.Errorf("got %+v", se)
	}
}

func TestSendErrorWithoutKind(t *testing.T) {
	err := sendError(errors.New("decode reply: unexpected EOF"))
	var se *SendError
	if errors.As(err, &se) {
		t.Fatalf("got a *SendError for %v", err)
	}
}
