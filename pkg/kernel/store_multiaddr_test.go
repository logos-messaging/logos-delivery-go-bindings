package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestStoreQueryWithSeveralPeerAddresses(t *testing.T) {
	requiresNode(t)

	storeConfig := DefaultWakuConfig
	storeConfig.Relay = true
	storeConfig.Store = true

	storeNode, err := StartWakuNode(&storeConfig)
	require.NoError(t, err, "Failed to start StoreNode")
	defer func() { _ = storeNode.Close() }()

	clientConfig := DefaultWakuConfig
	clientConfig.Relay = true

	client, err := StartWakuNode(&clientConfig)
	require.NoError(t, err, "Failed to start Client")
	defer func() { _ = client.Close() }()

	require.NoError(t, client.Peers().ConnectTo(context.Background(), storeNode))

	storeAddrs, err := storeNode.Debug().ListenAddresses()
	require.NoError(t, err)
	require.NotEmpty(t, storeAddrs)

	storeInfo, err := peer.AddrInfoFromString(storeAddrs[0].String())
	require.NoError(t, err)

	// Unreachable, and first, so one address alone would not reach the peer.
	storeInfo.Addrs = append([]multiaddr.Multiaddr{
		multiaddr.StringCast("/ip4/127.0.0.1/tcp/1"),
	}, storeInfo.Addrs...)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, err = client.Store().Query(ctx, &DefaultStoreQueryRequest, *storeInfo)
	require.NoError(t, err, "store query with several addresses for one peer")
}
