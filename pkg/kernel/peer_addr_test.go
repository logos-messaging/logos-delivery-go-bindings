package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// The library parses its peer argument as a single multiaddress, so a peer
// advertising several addresses must still yield one dialable string.
func TestPeerAddrSendsOneMultiaddress(t *testing.T) {
	id, err := peer.Decode("16Uiu2HAmVGHwfEi4kiNvuK6xVwGB2WeHoZNU1FgTUgZ8QvxiMqQw")
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	peerInfo := peer.AddrInfo{ID: id, Addrs: []multiaddr.Multiaddr{
		multiaddr.StringCast("/ip4/127.0.0.1/tcp/60000"),
		multiaddr.StringCast("/ip4/10.0.0.1/tcp/60001"),
	}}

	got, err := peerAddr(peerInfo)
	if err != nil {
		t.Fatalf("peerAddr: %v", err)
	}

	want := "/ip4/127.0.0.1/tcp/60000/p2p/" + id.String()
	if got != want {
		t.Errorf("peerAddr() = %q, want %q", got, want)
	}
}

func TestPeerAddrRejectsAddresslessPeer(t *testing.T) {
	if _, err := peerAddr(peer.AddrInfo{}); err == nil {
		t.Error("peerAddr() on a peer with no addresses returned no error")
	}
}

// Zero milliseconds is not "no timeout": chronos expires immediately on it, so
// a context without a deadline has to fall back to the request timeout.
func TestTimeoutMillisFallsBackToRequestTimeout(t *testing.T) {
	if got, want := timeoutMillis(context.Background(), requestTimeout),
		int(requestTimeout.Milliseconds()); got != want {
		t.Errorf("timeoutMillis(Background) = %d, want %d", got, want)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if got := timeoutMillis(ctx, requestTimeout); got <= 0 || got > 60_000 {
		t.Errorf("timeoutMillis(1m) = %d, want (0, 60000]", got)
	}
}
