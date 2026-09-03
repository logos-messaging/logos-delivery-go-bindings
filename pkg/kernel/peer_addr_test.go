package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
)

const testPeerID = "16Uiu2HAmVGHwfEi4kiNvuK6xVwGB2WeHoZNU1FgTUgZ8QvxiMqQw"

// The library reads the peer id out of the multiaddress it is given, so an
// address without a /p2p component cannot identify a peer.
func TestPeerAddrStringRequiresPeerID(t *testing.T) {
	qualified := "/ip4/127.0.0.1/tcp/60000/p2p/" + testPeerID

	got, err := peerAddrString(multiaddr.StringCast(qualified))
	if err != nil {
		t.Fatalf("peerAddrString(%s): %v", qualified, err)
	}
	if got != qualified {
		t.Errorf("peerAddrString() = %q, want %q", got, qualified)
	}

	if _, err := peerAddrString(multiaddr.StringCast("/ip4/127.0.0.1/tcp/60000")); err == nil {
		t.Error("peerAddrString() on an address with no /p2p component returned no error")
	}
	if _, err := peerAddrString(nil); err == nil {
		t.Error("peerAddrString(nil) returned no error")
	}
}

// Zero milliseconds is not "no timeout": chronos expires on it immediately, so
// a context without a deadline has to fall back to the request timeout.
func TestContextTimeoutFallsBackToRequestTimeout(t *testing.T) {
	if got, want := getContextTimeoutMilliseconds(context.Background()),
		int(requestTimeout.Milliseconds()); got != want {
		t.Errorf("getContextTimeoutMilliseconds(Background) = %d, want %d", got, want)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if got := getContextTimeoutMilliseconds(ctx); got <= 0 || got > 60_000 {
		t.Errorf("getContextTimeoutMilliseconds(1m) = %d, want (0, 60000]", got)
	}

	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpired()
	if got := getContextTimeoutMilliseconds(expired); got != 0 {
		t.Errorf("getContextTimeoutMilliseconds(expired) = %d, want 0", got)
	}
}
