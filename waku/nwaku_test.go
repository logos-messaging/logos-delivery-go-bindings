package waku

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/cenkalti/backoff/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/logos-messaging/logos-delivery-go-bindings/waku/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/waku/pb"
	"github.com/logos-messaging/logos-delivery-go-bindings/waku/store"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

// In order to run this test, you must run an nwaku node
//
// Using Docker:
//
//	IP_ADDRESS=$(hostname -I | awk '{print $1}');
// 	docker run \
// 	-p 61000:61000/tcp -p 8000:8000/udp -p 8646:8646/tcp harbor.status.im/wakuorg/nwaku:v0.33.0 \
// 	--discv5-discovery=true --cluster-id=16 --log-level=DEBUG --shard=64 --tcp-port=61000 \
// 	--nat=extip:${IP_ADDRESS} --discv5-udp-port=8000 --rest-address=0.0.0.0 --store --rest-port=8646

func TestBasicWaku(t *testing.T) {
	t.Skip("Skipping test as choosing this port will fail the CI")
	extNodeRestPort := 8646
	storeNodeInfo, err := GetNwakuInfo(nil, &extNodeRestPort)
	require.NoError(t, err)

	// ctx := context.Background()

	nwakuConfig := common.WakuConfig{
		Nodekey:         "11d0dcea28e86f81937a3bd1163473c7fbc0a0db54fd72914849bc47bdf78710",
		Relay:           true,
		LogLevel:        "DEBUG",
		DnsDiscoveryUrl: "enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.prod.status.nodes.status.im",
		DnsDiscovery:    true,
		Discv5Discovery: true,
		Staticnodes:     []string{storeNodeInfo.ListenAddresses[0]},
		ClusterID:       16,
		Shards:          []uint16{64},
	}

	storeNodeMa, err := ma.NewMultiaddr(storeNodeInfo.ListenAddresses[0])
	require.NoError(t, err)

	w, err := NewWakuNode(&nwakuConfig, "nwaku")
	require.NoError(t, err)
	require.NoError(t, w.Start())

	enr, err := w.ENR()
	require.NoError(t, err)
	require.NotNil(t, enr)

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	// Sanity check, not great, but it's probably helpful
	err = RetryWithBackOff(func() error {
		numConnected, err := w.GetNumConnectedPeers()
		if err != nil {
			return err
		}
		// Have to be connected to at least 3 nodes: the static node, the bootstrap node, and one discovered node
		if numConnected > 2 {
			return nil
		}
		return errors.New("no peers discovered")
	}, options)
	require.NoError(t, err)

	// Get local store node address
	storeNode, err := peer.AddrInfoFromString(storeNodeInfo.ListenAddresses[0])
	require.NoError(t, err)

	/*
		w.node.DialPeer(ctx, storeNode.Addrs[0], "")

		w.StorenodeCycle.SetStorenodeConfigProvider(newTestStorenodeConfigProvider(*storeNode))
	*/

	// Check that we are indeed connected to the store node
	connectedStoreNodes, err := w.GetPeerIDsByProtocol(store.StoreQueryID_v300)
	require.NoError(t, err)
	require.True(t, slices.Contains(connectedStoreNodes, storeNode.ID), "nwaku should be connected to the store node")

	// Disconnect from the store node
	err = w.DisconnectPeerByID(storeNode.ID)
	require.NoError(t, err)

	// Check that we are indeed disconnected
	connectedStoreNodes, err = w.GetPeerIDsByProtocol(store.StoreQueryID_v300)
	require.NoError(t, err)
	isDisconnected := !slices.Contains(connectedStoreNodes, storeNode.ID)
	require.True(t, isDisconnected, "nwaku should be disconnected from the store node")

	// Re-connect
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	err = w.Connect(ctx, storeNodeMa)
	require.NoError(t, err)

	// Check that we are connected again
	connectedStoreNodes, err = w.GetPeerIDsByProtocol(store.StoreQueryID_v300)
	require.NoError(t, err)
	require.True(t, slices.Contains(connectedStoreNodes, storeNode.ID), "nwaku should be connected to the store node")

	/* filter := &common.Filter{
		PubsubTopic:   w.cfg.DefaultShardPubsubTopic,
		Messages:      common.NewMemoryMessageStore(),
		ContentTopics: common.NewTopicSetFromBytes([][]byte{{1, 2, 3, 4}}),
	}

	_, err = w.Subscribe(filter)
	require.NoError(t, err)

	msgTimestamp := w.timestamp()
	contentTopic := maps.Keys(filter.ContentTopics)[0]

	time.Sleep(2 * time.Second)

	msgID, err := w.Send(w.cfg.DefaultShardPubsubTopic, &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5},
		ContentTopic: contentTopic.ContentTopic(),
		Version:      proto.Uint32(0),
		Timestamp:    &msgTimestamp,
	}, nil)

	require.NoError(t, err)
	require.NotEqual(t, msgID, "1")

	time.Sleep(1 * time.Second)

	messages := filter.Retrieve()
	require.Len(t, messages, 1)

	timestampInSeconds := msgTimestamp / int64(time.Second)
	marginInSeconds := 20

	options = func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 60 * time.Second
		b.InitialInterval = 500 * time.Millisecond
	}
	err = RetryWithBackOff(func() error {
		err := w.HistoryRetriever.Query(
			context.Background(),
			store.FilterCriteria{
				ContentFilter: protocol.NewContentFilter(w.cfg.DefaultShardPubsubTopic, contentTopic.ContentTopic()),
				TimeStart:     proto.Int64((timestampInSeconds - int64(marginInSeconds)) * int64(time.Second)),
				TimeEnd:       proto.Int64((timestampInSeconds + int64(marginInSeconds)) * int64(time.Second)),
			},
			*storeNode,
			10,
			nil, false,
		)

		return err

		// TODO-nwaku
		if err != nil || envelopeCount == 0 {
			// in case of failure extend timestamp margin up to 40secs
			if marginInSeconds < 40 {
				marginInSeconds += 5
			}
			return errors.New("no messages received from store node")
		}
		return nil

	}, options)
	require.NoError(t, err)

	time.Sleep(10 * time.Second)
	*/

	require.NoError(t, w.Stop())

}

func TestPeerExchange(t *testing.T) {

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)
	// start node that will be discovered by PeerExchange
	discV5NodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: true,
		ClusterID:       16,
		Shards:          []uint16{64},
		PeerExchange:    false,
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	discV5Node, err := NewWakuNode(&discV5NodeWakuConfig, "discV5Node")
	require.NoError(t, err)
	require.NoError(t, discV5Node.Start())

	discV5NodePeerId, err := discV5Node.PeerID()
	require.NoError(t, err)

	discv5NodeEnr, err := discV5Node.ENR()
	require.NoError(t, err)

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node which serves as PeerExchange server
	pxServerWakuConfig := common.WakuConfig{
		Relay:                true,
		LogLevel:             "DEBUG",
		Discv5Discovery:      true,
		ClusterID:            16,
		Shards:               []uint16{64},
		PeerExchange:         true,
		Discv5UdpPort:        udpPort,
		Discv5BootstrapNodes: []string{discv5NodeEnr.String()},
		TcpPort:              tcpPort,
	}

	pxServerNode, err := NewWakuNode(&pxServerWakuConfig, "pxServerNode")
	require.NoError(t, err)
	require.NoError(t, pxServerNode.Start())

	// Adding an extra second to make sure PX cache is not empty
	time.Sleep(2 * time.Second)

	serverNodeMa, err := pxServerNode.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, serverNodeMa)
	require.True(t, len(serverNodeMa) > 0)

	// Sanity check, not great, but it's probably helpful
	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	// Check that pxServerNode has discV5Node in its Peer Store
	err = RetryWithBackOff(func() error {
		peers, err := pxServerNode.GetPeerIDsFromPeerStore()

		if err != nil {
			return err
		}

		if slices.Contains(peers, discV5NodePeerId) {
			return nil
		}

		return errors.New("pxServer is missing the discv5 node in its peer store")
	}, options)
	require.NoError(t, err)

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start light node which uses PeerExchange to discover peers
	pxClientWakuConfig := common.WakuConfig{
		Relay:            false,
		LogLevel:         "DEBUG",
		Discv5Discovery:  false,
		ClusterID:        16,
		Shards:           []uint16{64},
		PeerExchange:     true,
		Discv5UdpPort:    udpPort,
		TcpPort:          tcpPort,
		PeerExchangeNode: serverNodeMa[0].String(),
	}

	lightNode, err := NewWakuNode(&pxClientWakuConfig, "lightNode")
	require.NoError(t, err)
	require.NoError(t, lightNode.Start())

	pxServerPeerId, err := pxServerNode.PeerID()
	require.NoError(t, err)

	// Check that the light node discovered the discV5Node and has both nodes in its peer store
	err = RetryWithBackOff(func() error {
		peers, err := lightNode.GetPeerIDsFromPeerStore()
		if err != nil {
			return err
		}

		if slices.Contains(peers, discV5NodePeerId) && slices.Contains(peers, pxServerPeerId) {
			return nil
		}
		return errors.New("lightnode is missing peers")
	}, options)
	require.NoError(t, err)

	// Now perform the PX request manually to see if it also works
	err = RetryWithBackOff(func() error {
		numPeersReceived, err := lightNode.PeerExchangeRequest(1)
		if err != nil {
			return err
		}

		if numPeersReceived == 1 {
			return nil
		}
		return errors.New("Peer Exchange is not returning peers")
	}, options)
	require.NoError(t, err)

	// Stop nodes
	require.NoError(t, lightNode.Stop())
	require.NoError(t, pxServerNode.Stop())
	require.NoError(t, discV5Node.Stop())

}

func TestDnsDiscover(t *testing.T) {

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	nameserver := "8.8.8.8"
	nodeWakuConfig := common.WakuConfig{
		Relay:         true,
		LogLevel:      "DEBUG",
		ClusterID:     16,
		Shards:        []uint16{64},
		Discv5UdpPort: udpPort,
		TcpPort:       tcpPort,
	}

	node, err := NewWakuNode(&nodeWakuConfig, "node")
	require.NoError(t, err)
	require.NoError(t, node.Start())
	sampleEnrTree := "enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.prod.status.nodes.status.im"

	ctx, cancel := context.WithTimeout(context.TODO(), requestTimeout)
	defer cancel()
	res, err := node.DnsDiscovery(ctx, sampleEnrTree, nameserver)
	require.NoError(t, err)
	require.True(t, len(res) > 1, "multiple nodes should be returned from the DNS Discovery query")
	// Stop nodes
	require.NoError(t, node.Stop())
}

func TestDial(t *testing.T) {

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node that will initiate the dial
	dialerNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	dialerNode, err := NewWakuNode(&dialerNodeWakuConfig, "dialerNode")
	require.NoError(t, err)
	require.NoError(t, dialerNode.Start())

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node that will receive the dial
	receiverNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	receiverNode, err := NewWakuNode(&receiverNodeWakuConfig, "receiverNode")
	require.NoError(t, err)
	require.NoError(t, receiverNode.Start())
	receiverMultiaddr, err := receiverNode.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, receiverMultiaddr)
	require.True(t, len(receiverMultiaddr) > 0)
	// Check that both nodes start with no connected peers
	dialerPeerCount, err := dialerNode.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, dialerPeerCount == 0, "Dialer node should have no connected peers")
	receiverPeerCount, err := receiverNode.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, receiverPeerCount == 0, "Receiver node should have no connected peers")
	// Dial
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	err = dialerNode.Connect(ctx, receiverMultiaddr[0])
	require.NoError(t, err)
	time.Sleep(1 * time.Second)
	// Check that both nodes now have one connected peer
	dialerPeerCount, err = dialerNode.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, dialerPeerCount == 1, "Dialer node should have 1 peer")
	receiverPeerCount, err = receiverNode.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, receiverPeerCount == 1, "Receiver node should have 1 peer")
	// Stop nodes
	require.NoError(t, dialerNode.Stop())
	require.NoError(t, receiverNode.Stop())
}

func TestRelay(t *testing.T) {
	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node that will send the message
	senderNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	senderNode, err := NewWakuNode(&senderNodeWakuConfig, "senderNode")
	require.NoError(t, err)
	require.NoError(t, senderNode.Start())

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node that will receive the message
	receiverNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}
	receiverNode, err := NewWakuNode(&receiverNodeWakuConfig, "receiverNode")
	require.NoError(t, err)
	require.NoError(t, receiverNode.Start())
	receiverMultiaddr, err := receiverNode.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, receiverMultiaddr)
	require.True(t, len(receiverMultiaddr) > 0)

	// Dial so they become peers
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	err = senderNode.Connect(ctx, receiverMultiaddr[0])
	require.NoError(t, err)
	time.Sleep(1 * time.Second)
	// Check that both nodes now have one connected peer
	senderPeerCount, err := senderNode.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, senderPeerCount == 1, "Dialer node should have 1 peer")
	receiverPeerCount, err := receiverNode.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, receiverPeerCount == 1, "Receiver node should have 1 peer")

	message := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3, 4, 5, 6},
		ContentTopic: "test-content-topic",
		Version:      proto.Uint32(0),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	}
	// send message
	pubsubTopic := FormatWakuRelayTopic(senderNodeWakuConfig.ClusterID, senderNodeWakuConfig.Shards[0])
	ctx2, cancel2 := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel2()
	senderNode.RelayPublish(ctx2, message, pubsubTopic)

	// Wait to receive message
	select {
	case envelope := <-receiverNode.MsgChan:
		require.NotNil(t, envelope, "Envelope should be received")
		require.Equal(t, message.Payload, envelope.Message().Payload, "Received payload should match")
		require.Equal(t, message.ContentTopic, envelope.Message().ContentTopic, "Content topic should match")
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout: No message received within 10 seconds")
	}

	// Stop nodes
	require.NoError(t, senderNode.Stop())
	require.NoError(t, receiverNode.Stop())
}

func TestTopicHealth(t *testing.T) {
	clusterId := uint16(16)
	shardId := uint16(64)

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node1
	wakuConfig1 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	node1, err := NewWakuNode(&wakuConfig1, "node1")
	require.NoError(t, err)
	require.NoError(t, node1.Start())

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node2
	wakuConfig2 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}
	node2, err := NewWakuNode(&wakuConfig2, "node2")
	require.NoError(t, err)
	require.NoError(t, node2.Start())
	multiaddr2, err := node2.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, multiaddr2)
	require.True(t, len(multiaddr2) > 0)

	// node1 dials node2 so they become peers
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	err = node1.Connect(ctx, multiaddr2[0])
	require.NoError(t, err)
	time.Sleep(1 * time.Second)
	// Check that both nodes now have one connected peer
	peerCount1, err := node1.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, peerCount1 == 1, "node1 should have 1 peer")
	peerCount2, err := node2.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, peerCount2 == 1, "node2 should have 1 peer")

	// Wait to receive topic health update
	select {
	case topicHealth := <-node2.TopicHealthChan:
		require.NotNil(t, topicHealth, "topicHealth should be updated")
		require.Equal(t, topicHealth.TopicHealth, "MinimallyHealthy", "Topic health should be MinimallyHealthy")
		require.Equal(t, topicHealth.PubsubTopic, FormatWakuRelayTopic(clusterId, shardId), "PubsubTopic should match configured cluster and shard")
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout: No topic health event received within 10 seconds")
	}

	// Stop nodes
	require.NoError(t, node1.Stop())
	require.NoError(t, node2.Stop())

}

func TestConnectionChange(t *testing.T) {
	clusterId := uint16(16)
	shardId := uint16(64)

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node1
	wakuConfig1 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	node1, err := NewWakuNode(&wakuConfig1, "node1")
	require.NoError(t, err)
	require.NoError(t, node1.Start())

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node2
	wakuConfig2 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}
	node2, err := NewWakuNode(&wakuConfig2, "node2")
	require.NoError(t, err)
	require.NoError(t, node2.Start())
	multiaddr2, err := node2.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, multiaddr2)
	require.True(t, len(multiaddr2) > 0)

	// node1 dials node2 so they become peers
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	err = node1.Connect(ctx, multiaddr2[0])
	require.NoError(t, err)
	time.Sleep(1 * time.Second)
	// Check that both nodes now have one connected peer
	peerCount1, err := node1.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, peerCount1 == 1, "node1 should have 1 peer")
	peerCount2, err := node2.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, peerCount2 == 1, "node2 should have 1 peer")

	peerId1, err := node1.PeerID()
	require.NoError(t, err)

	// Wait to receive connectionChange event
	select {
	case connectionChange := <-node2.ConnectionChangeChan:
		require.NotNil(t, connectionChange, "connectionChange should be updated")
		require.Equal(t, connectionChange.PeerEvent, "Joined", "connectionChange Joined event should be emitted")
		require.Equal(t, connectionChange.PeerId, peerId1, "connectionChange event should contain node 1's peerId")
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout: No connectionChange event received within 10 seconds")
	}

	// Disconnect from node1
	err = node2.DisconnectPeerByID(peerId1)
	require.NoError(t, err)

	// Wait to receive connectionChange event
	select {
	case connectionChange := <-node2.ConnectionChangeChan:
		require.NotNil(t, connectionChange, "connectionChange should be updated")
		require.Equal(t, connectionChange.PeerEvent, "Left", "connectionChange Left event should be emitted")
		require.Equal(t, connectionChange.PeerId, peerId1, "connectionChange event should contain node 1's peerId")
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout: No connectionChange event received within 10 seconds")
	}

	// Stop nodes
	require.NoError(t, node1.Stop())
	require.NoError(t, node2.Stop())
}

func TestStore(t *testing.T) {

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node that will send the message
	senderNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		Store:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
		LegacyStore:     false,
	}

	senderNode, err := NewWakuNode(&senderNodeWakuConfig, "senderNode")
	require.NoError(t, err)
	require.NoError(t, senderNode.Start())

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node that will receive the message
	receiverNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		Store:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
		LegacyStore:     false,
	}
	receiverNode, err := NewWakuNode(&receiverNodeWakuConfig, "receiverNode")
	require.NoError(t, err)
	require.NoError(t, receiverNode.Start())
	receiverMultiaddr, err := receiverNode.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, receiverMultiaddr)
	require.True(t, len(receiverMultiaddr) > 0)

	// Dial so they become peers
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	err = senderNode.Connect(ctx, receiverMultiaddr[0])
	require.NoError(t, err)
	time.Sleep(1 * time.Second)
	// Check that both nodes now have one connected peer
	senderPeerCount, err := senderNode.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, senderPeerCount == 1, "Dialer node should have 1 peer")
	receiverPeerCount, err := receiverNode.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, receiverPeerCount == 1, "Receiver node should have 1 peer")

	// Send 8 messages
	numMessages := 8
	paginationLimit := 5
	timeStart := proto.Int64(time.Now().UnixNano())
	hashes := []common.MessageHash{}
	pubsubTopic := FormatWakuRelayTopic(senderNodeWakuConfig.ClusterID, senderNodeWakuConfig.Shards[0])

	for i := 0; i < numMessages; i++ {
		message := &pb.WakuMessage{
			Payload:      []byte{byte(i)}, // Include message number in payload
			ContentTopic: "test-content-topic",
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().UnixNano()),
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel2()

		hash, err := senderNode.RelayPublish(ctx2, message, pubsubTopic)
		require.NoError(t, err)
		hashes = append(hashes, hash)
	}

	// Wait to receive all 8 messages
	receivedCount := 0
	receivedMessages := make(map[byte]bool)

	// Use a timeout for the entire receive operation
	timeoutChan := time.After(10 * time.Second)

	for receivedCount < numMessages {
		select {
		case envelope := <-receiverNode.MsgChan:
			require.NotNil(t, envelope, "Envelope should be received")

			payload := envelope.Message().Payload
			msgNum := payload[0]

			// Check if we've already received this message number
			if !receivedMessages[msgNum] {
				receivedMessages[msgNum] = true
				receivedCount++
			}

			require.Equal(t, "test-content-topic", envelope.Message().ContentTopic, "Content topic should match")

		case <-timeoutChan:
			t.Fatalf("Timeout: Only received %d messages out of 8 within 10 seconds", receivedCount)
		}
	}

	// Verify we received all messages
	for i := 0; i < numMessages; i++ {
		require.True(t, receivedMessages[byte(i)], fmt.Sprintf("Message %d was not received", i))
	}

	// Now send store query
	storeReq1 := common.StoreQueryRequest{
		IncludeData:       true,
		ContentTopics:     &[]string{"test-content-topic"},
		PaginationLimit:   proto.Uint64(uint64(paginationLimit)),
		PaginationForward: true,
		TimeStart:         timeStart,
	}

	storeNodeAddrInfo, err := peer.AddrInfoFromString(receiverMultiaddr[0].String())
	require.NoError(t, err)

	ctx3, cancel3 := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel3()

	res1, err := senderNode.StoreQuery(ctx3, &storeReq1, *storeNodeAddrInfo)
	require.NoError(t, err)

	storedMessages1 := *res1.Messages
	for i := 0; i < paginationLimit; i++ {
		require.True(t, storedMessages1[i].MessageHash == hashes[i], fmt.Sprintf("Stored message does not match received message for index %d", i))
	}

	// Now let's query the second page
	storeReq2 := common.StoreQueryRequest{
		IncludeData:       true,
		ContentTopics:     &[]string{"test-content-topic"},
		PaginationLimit:   proto.Uint64(uint64(paginationLimit)),
		PaginationForward: true,
		TimeStart:         timeStart,
		PaginationCursor:  &res1.PaginationCursor,
	}

	ctx4, cancel4 := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel4()

	res2, err := senderNode.StoreQuery(ctx4, &storeReq2, *storeNodeAddrInfo)
	require.NoError(t, err)

	storedMessages2 := *res2.Messages
	for i := 0; i < len(storedMessages2); i++ {
		require.True(t, storedMessages2[i].MessageHash == hashes[i+paginationLimit], fmt.Sprintf("Stored message does not match received message for index %d", i))
	}

	// Now let's query for two specific message hashes
	storeReq3 := common.StoreQueryRequest{
		IncludeData:   true,
		ContentTopics: &[]string{"test-content-topic"},
		MessageHashes: &[]common.MessageHash{hashes[0], hashes[2]},
	}

	ctx5, cancel5 := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel5()

	res3, err := senderNode.StoreQuery(ctx5, &storeReq3, *storeNodeAddrInfo)
	require.NoError(t, err)

	storedMessages3 := *res3.Messages
	require.True(t, storedMessages3[0].MessageHash == hashes[0], "Stored message does not match queried message")
	require.True(t, storedMessages3[1].MessageHash == hashes[2], "Stored message does not match queried message")

	// Stop nodes
	require.NoError(t, senderNode.Stop())
	require.NoError(t, receiverNode.Stop())
}

func TestParallelPings(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node that will initiate the dial
	dialerNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	dialerNode, err := NewWakuNode(&dialerNodeWakuConfig, "dialerNode")
	require.NoError(t, err)
	require.NoError(t, dialerNode.Start())

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	receiverNodeWakuConfig1 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	receiverNode1, err := NewWakuNode(&receiverNodeWakuConfig1, "receiverNode1")
	require.NoError(t, err)
	require.NoError(t, receiverNode1.Start())
	receiverMultiaddr1, err := receiverNode1.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, receiverMultiaddr1)
	require.True(t, len(receiverMultiaddr1) > 0)

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	receiverNodeWakuConfig2 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	receiverNode2, err := NewWakuNode(&receiverNodeWakuConfig2, "receiverNode2")
	require.NoError(t, err)
	require.NoError(t, receiverNode2.Start())
	receiverMultiaddr2, err := receiverNode2.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, receiverMultiaddr2)
	require.True(t, len(receiverMultiaddr2) > 0)

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	receiverNodeWakuConfig3 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	receiverNode3, err := NewWakuNode(&receiverNodeWakuConfig3, "receiverNode3")
	require.NoError(t, err)
	require.NoError(t, receiverNode3.Start())
	receiverMultiaddr3, err := receiverNode3.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, receiverMultiaddr3)
	require.True(t, len(receiverMultiaddr3) > 0)

	receiverNodes := []string{receiverMultiaddr1[0].String(), receiverMultiaddr2[0].String(), receiverMultiaddr3[0].String()}

	// node.PingPeer(ctx, peerInfo)
	for _, receiverNode := range receiverNodes {

		addrInfo, err := peer.AddrInfoFromString(receiverNode)
		require.NoError(t, err)

		go func(peerInfo peer.AddrInfo) {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()

			_, err := dialerNode.PingPeer(ctx, peerInfo)
			if err != nil { // pinging storenodes might fail, but we don't care
				logger.Warn("failed pinging node", zap.Stringer("peerId", addrInfo.ID), zap.Error(err))
			}
		}(*addrInfo)

	}

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}
	err = RetryWithBackOff(func() error {
		dialerPeerCount, err := dialerNode.GetNumConnectedPeers()

		if err != nil {
			return err
		}

		if dialerPeerCount == 3 {
			return nil
		}

		return fmt.Errorf("dialerNode should have 3 peers but it has %d", dialerPeerCount)
	}, options)
	require.NoError(t, err)

	// Stop nodes
	require.NoError(t, dialerNode.Stop())
}

func TestOnline(t *testing.T) {

	clusterId := uint16(16)
	shardId := uint16(64)

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node1
	wakuConfig1 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	node1, err := NewWakuNode(&wakuConfig1, "node1")
	require.NoError(t, err)
	require.NoError(t, node1.Start())

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node2
	wakuConfig2 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}
	node2, err := NewWakuNode(&wakuConfig2, "node2")
	require.NoError(t, err)
	require.NoError(t, node2.Start())
	multiaddr2, err := node2.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, multiaddr2)
	require.True(t, len(multiaddr2) > 0)

	// node1 dials node2 so they become peers
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	err = node1.Connect(ctx, multiaddr2[0])
	require.NoError(t, err)
	time.Sleep(1 * time.Second)
	// Check that both nodes now have one connected peer
	peerCount1, err := node1.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, peerCount1 == 1, "node1 should have 1 peer")
	peerCount2, err := node2.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, peerCount2 == 1, "node2 should have 1 peer")

	isOnline, err := node1.IsOnline()
	require.NoError(t, err)
	require.True(t, isOnline, "node1 should be online")

	// Stop nodes
	require.NoError(t, node1.Stop())
	require.NoError(t, node2.Stop())
}

func TestDisconnectAllPeers(t *testing.T) {

	clusterId := uint16(16)
	shardId := uint16(64)

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node1
	wakuConfig1 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	node1, err := NewWakuNode(&wakuConfig1, "node1")
	require.NoError(t, err)
	require.NoError(t, node1.Start())
	defer node1.Stop()

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node2
	wakuConfig2 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}
	node2, err := NewWakuNode(&wakuConfig2, "node2")
	require.NoError(t, err)
	require.NoError(t, node2.Start())
	defer node2.Stop()
	multiaddr2, err := node2.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, multiaddr2)
	require.True(t, len(multiaddr2) > 0)

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node3
	wakuConfig3 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}
	node3, err := NewWakuNode(&wakuConfig3, "node3")
	require.NoError(t, err)
	require.NoError(t, node3.Start())
	defer node3.Stop()
	multiaddr3, err := node3.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, multiaddr3)
	require.True(t, len(multiaddr3) > 0)

	tcpPort, udpPort, err = GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	// start node4
	wakuConfig4 := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       clusterId,
		Shards:          []uint16{shardId},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}
	node4, err := NewWakuNode(&wakuConfig4, "node4")
	require.NoError(t, err)
	require.NoError(t, node4.Start())
	defer node4.Stop()
	multiaddr4, err := node4.ListenAddresses()
	require.NoError(t, err)
	require.NotNil(t, multiaddr4)
	require.True(t, len(multiaddr4) > 0)

	to_dial := []ma.Multiaddr{multiaddr2[0], multiaddr3[0], multiaddr4[0]}

	for _, addr := range to_dial {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		err = node1.Connect(ctx, addr)
		require.NoError(t, err)
	}

	time.Sleep(1 * time.Second)

	peerCount1, err := node1.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, peerCount1 == 3, "node1 should have 3 peers")

	err = node1.DisconnectAllPeers()
	require.NoError(t, err)
	peerCount1, err = node1.GetNumConnectedPeers()
	require.NoError(t, err)
	require.True(t, peerCount1 == 0, "node1 should have 0 peers")

}
