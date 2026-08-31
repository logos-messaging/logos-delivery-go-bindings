package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestPeerAddrSendsOneMultiaddress(t *testing.T) {
	id, err := peer.Decode("16Uiu2HAmVGHwfEi4kiNvuK6xVwGB2WeHoZNU1FgTUgZ8QvxiMqQw")
	require.NoError(t, err)

	peerInfo := peer.AddrInfo{ID: id, Addrs: []multiaddr.Multiaddr{
		multiaddr.StringCast("/ip4/127.0.0.1/tcp/60000"),
		multiaddr.StringCast("/ip4/10.0.0.1/tcp/60001"),
	}}

	addr, err := peerAddr(peerInfo)
	require.NoError(t, err)
	require.Equal(t, "/ip4/127.0.0.1/tcp/60000/p2p/"+id.String(), addr)
}

func TestPeerAddrRejectsAddresslessPeer(t *testing.T) {
	_, err := peerAddr(peer.AddrInfo{})
	require.Error(t, err)
}

func TestGetContextTimeoutMillisecondsWithoutDeadline(t *testing.T) {
	require.Equal(t,
		int(requestTimeout.Milliseconds()),
		getContextTimeoutMilliseconds(context.Background()))
}

func TestGetContextTimeoutMillisecondsWithDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	timeoutMs := getContextTimeoutMilliseconds(ctx)
	require.Positive(t, timeoutMs)
	require.LessOrEqual(t, timeoutMs, 60_000)
}

func TestGetContextTimeoutMillisecondsExpired(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	require.Zero(t, getContextTimeoutMilliseconds(ctx))
}
