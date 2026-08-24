package kernel

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestVerifyNumConnectedRelayPeers(t *testing.T) {
	requiresNode(t)

	node1Cfg := DefaultWakuConfig
	node1Cfg.Relay = true
	node1, err := StartWakuNode(&node1Cfg)
	if err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}
	node2Cfg := DefaultWakuConfig
	node2Cfg.Relay = true
	node2, err := StartWakuNode(&node2Cfg)
	if err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	node3, err := StartWakuNode(nil)
	if err != nil {
		t.Fatalf("Failed to start node3: %v", err)
	}

	defer func() {
		_ = node1.Close()
		_ = node2.Close()
		_ = node3.Close()
	}()

	err = node2.Peers().ConnectTo(context.Background(), node1)
	if err != nil {
		t.Fatalf("Failed to connect node2 to node1: %v", err)
	}

	err = node3.Peers().ConnectTo(context.Background(), node1)
	if err != nil {
		t.Fatalf("Failed to connect node3 to node1: %v", err)
	}
	connectedPeersNode1, err := node1.Peers().Connected()
	if err != nil {
		t.Fatalf("Failed to get connected peers for node1: %v", err)
	}
	if len(connectedPeersNode1) != 2 {
		t.Fatalf("Expected 2 connected peers on node1, but got %d", len(connectedPeersNode1))
	}

	numRelayPeers, err := node1.Relay().NumConnectedPeers()
	if err != nil {
		t.Fatalf("Failed to get connected relay peers for node1: %v", err)
	}
	if numRelayPeers != 1 {
		t.Fatalf("Expected 1 relay peer on node1, but got %d", numRelayPeers)
	}

	t.Logf("Successfully connected node2 and node3 to node1. Relay Peers: %d, Total Peers: %d", numRelayPeers, len(connectedPeersNode1))
}

func TestVerifyConnectedRelayPeers(t *testing.T) {
	requiresNode(t)

	customShard := uint16(65)
	customPubsubTopic := FormatWakuRelayTopic(DEFAULT_CLUSTER_ID, customShard)
	node1Cfg := DefaultWakuConfig
	node1Cfg.Relay = true
	node1Cfg.Shards = []uint16{64, customShard}
	node1, err := StartWakuNode(&node1Cfg)
	if err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}
	node2Cfg := DefaultWakuConfig
	node2Cfg.Relay = true
	node2, err := StartWakuNode(&node2Cfg)
	if err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	node2PeerID, err := node2.Debug().PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 2")

	node3, err := StartWakuNode(nil)
	if err != nil {
		t.Fatalf("Failed to start node3: %v", err)
	}

	node4Cfg := DefaultWakuConfig
	node4Cfg.Relay = true
	node4Cfg.Shards = []uint16{customShard}
	node4, err := StartWakuNode(&node4Cfg)
	if err != nil {
		t.Fatalf("Failed to start node4: %v", err)
	}

	node4PeerID, err := node4.Debug().PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 4")

	defer func() {
		_ = node1.Close()
		_ = node2.Close()
		_ = node3.Close()
		_ = node4.Close()
	}()

	err = node2.Peers().ConnectTo(context.Background(), node1)
	if err != nil {
		t.Fatalf("Failed to connect node2 to node1: %v", err)
	}

	err = node3.Peers().ConnectTo(context.Background(), node1)
	if err != nil {
		t.Fatalf("Failed to connect node3 to node1: %v", err)
	}

	err = node4.Peers().ConnectTo(context.Background(), node1)
	if err != nil {
		t.Fatalf("Failed to connect node4 to node1: %v", err)
	}

	connectedPeersNode1, err := node1.Peers().Connected()
	if err != nil {
		t.Fatalf("Failed to get connected peers for node1: %v", err)
	}
	if len(connectedPeersNode1) != 3 {
		t.Fatalf("Expected 2 connected peers on node1, but got %d", len(connectedPeersNode1))
	}

	relayPeers, err := node1.Relay().ConnectedPeers()
	if err != nil {
		t.Fatalf("Failed to get connected relay peers for node1: %v", err)
	}
	if len(relayPeers) != 2 {
		t.Fatalf("Expected 2 relay peers on node1, but got %d", len(relayPeers))
	}
	require.True(t, slices.Contains(relayPeers, node2PeerID), "Node 2 should be included in node 1's connected relay peers")
	require.True(t, slices.Contains(relayPeers, node4PeerID), "Node 4 should be included in node 1's connected relay peers")

	relayPeersDefaultPubsub, err := node1.Relay().ConnectedPeers(DefaultPubsubTopic)
	if err != nil {
		t.Fatalf("Failed to get connected relay peers for node1 and default pubsub topic: %v", err)
	}
	if len(relayPeersDefaultPubsub) != 1 {
		t.Fatalf("Expected 1 relay peers on node1 for default pubsub topic, but got %d", len(relayPeersDefaultPubsub))
	}
	require.True(t, slices.Contains(relayPeersDefaultPubsub, node2PeerID), "Node 2 should be included in node 1's connected relay peers for the default pubsub topic")

	relayPeersCustomPubsub, err := node1.Relay().ConnectedPeers(customPubsubTopic)
	if err != nil {
		t.Fatalf("Failed to get connected relay peers for node1 and custom pubsub topic: %v", err)
	}
	if len(relayPeersCustomPubsub) != 1 {
		t.Fatalf("Expected 1 relay peers on node1 for custom pubsub topic, but got %d", len(relayPeersCustomPubsub))
	}
	require.True(t, slices.Contains(relayPeersCustomPubsub, node4PeerID), "Node 4 should be included in node 1's connected relay peers for the custom pubsub topic")

	t.Logf("Successfully connected node2, node3 and node4 to node1. Relay Peers: %d, Total Peers: %d", len(relayPeers), len(connectedPeersNode1))
}

func TestRelayMessageTransmission(t *testing.T) {
	requiresNode(t)
	logDebug("Starting TestRelayMessageTransmission")

	logDebug("Creating Sender Node with Relay enabled")
	senderConfig := DefaultWakuConfig
	senderConfig.Relay = true

	senderNode, err := StartWakuNode(&senderConfig)
	require.NoError(t, err, "Failed to start SenderNode")
	defer func() { _ = senderNode.Close() }()

	logDebug("Creating Receiver Node with Relay enabled")
	receiverConfig := DefaultWakuConfig
	receiverConfig.Relay = true

	// Set the Receiver Node's discovery bootstrap node as SenderNode
	enrSender, err := senderNode.Debug().ENR()
	require.NoError(t, err, "Failed to get ENR for SenderNode")
	receiverConfig.Discv5BootstrapNodes = []string{enrSender.String()}

	receiverNode, err := StartWakuNode(&receiverConfig)
	require.NoError(t, err, "Failed to start ReceiverNode")
	defer func() { _ = receiverNode.Close() }()

	logDebug("Waiting for nodes to auto-connect via Discv5")
	err = WaitForAutoConnection([]*Node{senderNode, receiverNode})
	require.NoError(t, err, "Nodes did not auto-connect within timeout")

	logDebug("Creating and publishing message")
	message := senderNode.CreateMessage()
	var msgHash string

	err = RetryWithBackOff(func() error {
		var err error
		msgHashObj, err := senderNode.Relay().Publish(context.Background(), DefaultPubsubTopic, message)
		if err == nil {
			msgHash = msgHashObj.String()
		}
		return err
	})
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	logDebug("Verifying message reception")
	err = RetryWithBackOff(func() error {
		msgHashObj, _ := common.ToMessageHash(msgHash)
		return receiverNode.VerifyMessageReceived(message, msgHashObj)
	})
	require.NoError(t, err, "Message verification failed")

	logDebug("TestRelayMessageTransmission completed successfully")
}

func TestRelayMessageBroadcast(t *testing.T) {
	requiresNode(t)
	logDebug("Starting TestRelayMessageBroadcast")

	numPeers := 5
	nodes := make([]*Node, numPeers)
	nodeNames := []string{"SenderNode", "PeerNode1", "PeerNode2", "PeerNode3", "PeerNode4"}
	defaultPubsubTopic := DefaultPubsubTopic

	for i := 0; i < numPeers; i++ {
		logDebug("Creating node %s", nodeNames[i])
		nodeConfig := DefaultWakuConfig
		nodeConfig.Relay = true
		if i > 0 {
			enrPrevNode, err := nodes[i-1].Debug().ENR()
			require.NoError(t, err, "Failed to get ENR for node %s", nodeNames[i-1])
			nodeConfig.Discv5BootstrapNodes = []string{enrPrevNode.String()}
		}
		node, err := StartWakuNode(&nodeConfig)
		require.NoError(t, err)
		defer func() { _ = node.Close() }()
		nodes[i] = node
	}

	WaitForAutoConnection(nodes)

	senderNode := nodes[0]
	logDebug("SenderNode is publishing a message")
	message := senderNode.CreateMessage()
	msgHash, err := senderNode.Relay().Publish(context.Background(), defaultPubsubTopic, message)
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	logDebug("Waiting to ensure message delivery")
	time.Sleep(3 * time.Second)

	logDebug("Verifying message reception for each node")
	for i, node := range nodes {
		logDebug("Verifying message for node %s", nodeNames[i])
		err := node.VerifyMessageReceived(message, msgHash)
		require.NoError(t, err, "message verification failed for node: %s", nodeNames[i])
	}

	logDebug("TestRelayMessageBroadcast completed successfully")
}

func TestSendmsgInvalidPayload(t *testing.T) {
	requiresNode(t)
	logDebug("Starting TestInvalidMessageFormat")

	defaultPubsubTopic := DefaultPubsubTopic

	logDebug("Creating nodes")
	senderNodeConfig := DefaultWakuConfig
	senderNodeConfig.Relay = true
	senderNode, err := StartWakuNode(&senderNodeConfig)
	require.NoError(t, err)
	defer func() { _ = senderNode.Close() }()

	receiverNodeConfig := DefaultWakuConfig
	receiverNodeConfig.Relay = true
	enrNode2, err := senderNode.Debug().ENR()
	if err != nil {
		require.Error(t, err, "Can't find node ENR")
	}
	receiverNodeConfig.Discv5BootstrapNodes = []string{enrNode2.String()}
	receiverNode, err := StartWakuNode(&receiverNodeConfig)
	require.NoError(t, err)
	defer func() { _ = receiverNode.Close() }()

	err = WaitForAutoConnection([]*Node{senderNode, receiverNode})
	require.NoError(t, err, "Nodes did not auto-connect within timeout")

	logDebug("SenderNode is publishing an invalid message")
	invalidMessage := &pb.WakuMessage{
		Payload:      []byte{},
		ContentTopic: "test-content-topic",
		Version:      proto.Uint32(0),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	}

	message := senderNode.CreateMessage(invalidMessage)
	var msgHash common.MessageHash
	msgHash, err = senderNode.Relay().Publish(context.Background(), defaultPubsubTopic, message)

	logDebug("Verifying if message was sent or failed")
	if err != nil {
		logDebug("Message was not sent due to invalid format: %v", err)

	} else {
		logDebug("Message was unexpectedly sent: %s", msgHash.String())
		require.Fail(t, "message with invalid format should not be sent")
	}

	logDebug("TestInvalidMessageFormat completed")
}

func TestRelayNodesNotConnectedDirectly(t *testing.T) {
	requiresNode(t)
	logDebug("Starting TestRelayNodesNotConnectedDirectly")

	logDebug("Creating Sender Node with Relay enabled")
	senderConfig := DefaultWakuConfig
	senderConfig.Relay = true
	senderNode, err := StartWakuNode(&senderConfig)
	require.NoError(t, err)
	defer func() { _ = senderNode.Close() }()

	logDebug("Creating Relay-Enabled Receiver Node (Node2)")
	node2Config := DefaultWakuConfig
	node2Config.Relay = true

	// Use static nodes instead of ENR
	node1Address, err := senderNode.Debug().ListenAddresses()
	require.NoError(t, err, "Failed to get sender node address")
	node2Config.Staticnodes = []string{node1Address[0].String()}

	node2, err := StartWakuNode(&node2Config)
	require.NoError(t, err)
	defer func() { _ = node2.Close() }()

	logDebug("Creating Relay-Enabled Receiver Node (Node3)")
	node3Config := DefaultWakuConfig
	node3Config.Relay = true

	// Use static nodes instead of Discv5
	node2Address, err := node2.Debug().ListenAddresses()
	require.NoError(t, err, "Failed to get node2 address")
	node3Config.Staticnodes = []string{node2Address[0].String()}

	node3, err := StartWakuNode(&node3Config)
	require.NoError(t, err)
	defer func() { _ = node3.Close() }()

	logDebug("Waiting for nodes to connect before proceeding")
	err = WaitForAutoConnection([]*Node{senderNode, node2, node3})
	require.NoError(t, err, "Nodes did not connect within timeout")

	logDebug("SenderNode is publishing a message")
	message := senderNode.CreateMessage()
	msgHash, err := senderNode.Relay().Publish(context.Background(), DefaultPubsubTopic, message)
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	logDebug("Verifying that Node2 received the message")
	err = node2.VerifyMessageReceived(message, msgHash)
	require.NoError(t, err, "Node2 should have received the message")

	logDebug("Verifying that Node3 received the message")
	err = node3.VerifyMessageReceived(message, msgHash)
	require.NoError(t, err, "Node3 should have received the message")

	logDebug("TestRelayNodesNotConnectedDirectly completed successfully")
}

func TestRelaySubscribeAndPeerCountChange(t *testing.T) {
	requiresNode(t)
	logDebug("Starting test to verify relay subscription and peer count change after stopping a node")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	logDebug("Creating Node1 with Relay enabled")
	node1, err := StartWakuNode(&node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node1Address, err := node1.Debug().ListenAddresses()
	require.NoError(t, err, "Failed to get listening address for Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Staticnodes = []string{node1Address[0].String()}

	logDebug("Creating Node2 with Node1 as a static node")
	node2, err := StartWakuNode(&node2Config)
	require.NoError(t, err, "Failed to start Node2")

	// Commented till we configure external IPs
	//node2Address, err := node2.Debug().ListenAddresses()
	//require.NoError(t, err, "Failed to get listening address for Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = true
	node3Config.Staticnodes = []string{node1Address[0].String()}

	logDebug("Creating Node3 with Node1 as a static node")
	node3, err := StartWakuNode(&node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		logDebug("Stopping and destroying all Waku nodes")
		_ = node1.Close()
		_ = node2.Close()
	}()

	defaultPubsubTopic := DefaultPubsubTopic
	logDebug("Default pubsub topic retrieved: %s", defaultPubsubTopic)

	logDebug("Waiting for nodes to connect via static node configuration")
	err = WaitForAutoConnection([]*Node{node1, node2, node3})
	require.NoError(t, err, "Nodes did not connect within timeout")

	logDebug("Waiting for peer connections to stabilize")
	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 10 * time.Second
	}
	require.NoError(t, RetryWithBackOff(func() error {
		numPeers, err := node1.Relay().NumConnectedPeers(defaultPubsubTopic)
		if err != nil {
			return err
		}
		if numPeers != 2 {
			return fmt.Errorf("expected 2 relay peers, got %d", numPeers)
		}
		return nil
	}, options), "Peers did not stabilize in time")

	logDebug("Stopping Node3")
	_ = node3.Close()

	logDebug("Waiting for network to update after Node3 stops")
	require.NoError(t, RetryWithBackOff(func() error {
		numPeers, err := node1.Relay().NumConnectedPeers(defaultPubsubTopic)
		if err != nil {
			return err
		}
		if numPeers != 1 {
			return fmt.Errorf("expected 1 relay peer after stopping Node3, got %d", numPeers)
		}
		return nil
	}, options), "Peer count did not update after stopping Node3")

	logDebug("Test successfully verified peer count changes as expected after stopping Node3")
}

func TestRelaySubscribeFailsWhenRelayDisabled(t *testing.T) {
	requiresNode(t)
	logDebug("Starting test to verify that subscribing to a topic fails when Relay is disabled")

	nodeConfig := DefaultWakuConfig
	nodeConfig.Relay = false

	logDebug("Creating Node with Relay disabled")
	node, err := StartWakuNode(&nodeConfig)
	require.NoError(t, err, "Failed to start Node")

	defer func() {
		logDebug("Stopping and destroying the Waku node")
		_ = node.Close()
	}()

	defaultPubsubTopic := DefaultPubsubTopic
	logDebug("Attempting to subscribe to the default pubsub topic: %s", defaultPubsubTopic)

	err = node.Relay().Subscribe(defaultPubsubTopic)

	logDebug("Verifying that subscription failed")
	require.Error(t, err, "Expected RelaySubscribe to return an error when Relay is disabled")

	logDebug("Test successfully verified that RelaySubscribe fails when Relay is disabled")
}

func TestRelayDisabledNodeDoesNotReceiveMessages(t *testing.T) {
	requiresNode(t)
	logDebug("Starting test to verify that a node with Relay disabled does not receive messages")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	logDebug("Creating Node1 with Relay enabled")
	node1, err := StartWakuNode(&node1Config)
	require.NoError(t, err, "Failed to start Node1")

	enrNode1, err := node1.Debug().ENR()
	require.NoError(t, err, "Failed to get ENR for Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Discv5BootstrapNodes = []string{enrNode1.String()}

	logDebug("Creating Node2 with Node1 as Discv5 bootstrap")
	node2, err := StartWakuNode(&node2Config)
	require.NoError(t, err, "Failed to start Node2")

	enrNode2, err := node2.Debug().ENR()
	require.NoError(t, err, "Failed to get ENR for Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = false
	node3Config.Discv5BootstrapNodes = []string{enrNode2.String()}

	logDebug("Creating Node3 with Node2 as Discv5 bootstrap")
	node3, err := StartWakuNode(&node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		logDebug("Stopping and destroying all Waku nodes")
		_ = node1.Close()
		_ = node2.Close()
		_ = node3.Close()
	}()

	defaultPubsubTopic := DefaultPubsubTopic
	logDebug("Default pubsub topic retrieved: %s", defaultPubsubTopic)

	err = SubscribeNodesToTopic([]*Node{node1, node2}, defaultPubsubTopic)
	require.NoError(t, err, "Failed to subscribe nodes to the topic")

	logDebug("Waiting for nodes to auto-connect via Discv5")
	err = WaitForAutoConnection([]*Node{node1, node2})
	require.NoError(t, err, "Nodes did not auto-connect within timeout")

	logDebug("Creating and publishing message from Node1")
	message := node1.CreateMessage()
	msgHash, err := node1.Relay().Publish(context.Background(), defaultPubsubTopic, message)
	require.NoError(t, err, "Failed to publish message from Node1")

	logDebug("Waiting to ensure message delivery")
	time.Sleep(3 * time.Second)

	logDebug("Verifying that Node2 received the message")
	err = node2.VerifyMessageReceived(message, msgHash)
	require.NoError(t, err, "Node2 should have received the message")

	logDebug("Verifying that Node3 did NOT receive the message")
	err = node3.VerifyMessageReceived(message, msgHash)
	require.Error(t, err, "Node3 should NOT have received the message")

	logDebug("Test successfully verified that Node3 did not receive the message")
}

func TestPublishWithLargePayload(t *testing.T) {
	requiresNode(t)
	logDebug("Starting test to verify message publishing with a payload close to 150KB")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	logDebug("Creating Node1 with Relay enabled")
	node1, err := StartWakuNode(&node1Config)
	require.NoError(t, err, "Failed to start Node1")

	enrNode1, err := node1.Debug().ENR()
	require.NoError(t, err, "Failed to get ENR for Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Discv5BootstrapNodes = []string{enrNode1.String()}

	logDebug("Creating Node2 with Node1 as Discv5 bootstrap")
	node2, err := StartWakuNode(&node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		logDebug("Stopping and destroying all Waku nodes")
		_ = node1.Close()
		_ = node2.Close()
	}()

	defaultPubsubTopic := DefaultPubsubTopic
	logDebug("Default pubsub topic retrieved: %s", defaultPubsubTopic)

	err = SubscribeNodesToTopic([]*Node{node1, node2}, defaultPubsubTopic)
	require.NoError(t, err, "Failed to subscribe nodes to the topic")

	logDebug("Waiting for nodes to auto-connect via Discv5")
	err = WaitForAutoConnection([]*Node{node1, node2})
	require.NoError(t, err, "Nodes did not auto-connect within timeout")

	payloadLength := 1024 * 100 // 100KB raw, approximately 150KB when base64 encoded
	logDebug("Generating a large payload of %d bytes", payloadLength)

	largePayload := make([]byte, payloadLength)
	for i := range largePayload {
		largePayload[i] = 'a'
	}

	message := node1.CreateMessage(&pb.WakuMessage{
		Payload:      largePayload,
		ContentTopic: "test-content-topic",
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	})

	logDebug("Publishing message from Node1 with large payload")
	msgHash, err := node1.Relay().Publish(context.Background(), defaultPubsubTopic, message)
	require.NoError(t, err, "Failed to publish message from Node1")

	logDebug("Waiting to ensure message propagation")
	time.Sleep(2 * time.Second)

	logDebug("Verifying that Node2 received the message")
	err = node2.VerifyMessageReceived(message, msgHash)
	require.NoError(t, err, "Node2 should have received the message")

	logDebug("Test successfully verified message publishing with a large payload")
}
