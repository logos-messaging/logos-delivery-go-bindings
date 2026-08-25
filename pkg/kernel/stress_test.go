//go:build stress
// +build stress

package kernel

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/stretchr/testify/require"

	//	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
)

func TestStressMemoryUsageForThreeNodes(t *testing.T) {
	requiresNode(t)
	testName := t.Name()
	var err error
	captureMemory(t.Name(), "start")
	node1Cfg := DefaultWakuConfig
	node2Cfg := DefaultWakuConfig
	node3Cfg := DefaultWakuConfig

	node1, err := NewFromWakuConfig(&node1Cfg)
	require.NoError(t, err)
	node2, err := NewFromWakuConfig(&node2Cfg)
	require.NoError(t, err)
	node3, err := NewFromWakuConfig(&node3Cfg)
	require.NoError(t, err)

	captureMemory(t.Name(), "before nodes start")

	err = node1.Start()
	require.NoError(t, err)
	err = node2.Start()
	require.NoError(t, err)
	err = node3.Start()
	require.NoError(t, err)

	captureMemory(t.Name(), "after nodes run")

	time.Sleep(2 * time.Second)

	node1.Close()
	node2.Close()
	node3.Close()

	runtime.GC()
	time.Sleep(1 * time.Second)
	runtime.GC()

	captureMemory(t.Name(), "at end")

	logDebug("[%s] Test completed successfully", testName)
}

func TestStressStoreQuery5kMessagesWithPagination(t *testing.T) {
	requiresNode(t)
	logDebug("Starting test")
	runtime.GC()
	nodeConfig := DefaultWakuConfig
	nodeConfig.Relay = true
	nodeConfig.Store = true

	logDebug("Creating 2 nodes")
	wakuNode, err := StartWakuNode(&nodeConfig)
	require.NoError(t, err, "Failed to start Waku node")

	node2, err := StartWakuNode(&nodeConfig)
	require.NoError(t, err, "Failed to start Waku node")
	node2.Peers().ConnectTo(context.Background(), wakuNode)

	time.Sleep(200 * time.Millisecond)

	defer func() {
		logDebug("Stopping and destroying Waku node")
		wakuNode.Close()
		node2.Close()
	}()

	iterations := 2500

	captureMemory(t.Name(), "at start")

	queryTimestamp := proto.Int64(time.Now().UnixNano())

	for i := 0; i < iterations; i++ {
		message := wakuNode.CreateMessage()
		message.Payload = []byte(fmt.Sprintf("Test endurance message payload %d", i))
		hash, err := wakuNode.Relay().Publish(context.Background(), DefaultPubsubTopic, message)
		require.NoError(t, err, "Failed to publish message")

		err = node2.VerifyMessageReceived(message, hash)
		require.NoError(t, err, "node2 failed to receive message %d", i)
		err = wakuNode.VerifyMessageReceived(message, hash)
		require.NoError(t, err, "wakuNode failed to receive message %d", i)

		if i%10 == 0 {

			storeQueryRequest := &common.StoreQueryRequest{
				TimeStart:         queryTimestamp,
				IncludeData:       true,
				PaginationLimit:   proto.Uint64(50),
				PaginationForward: false,
			}

			storedmsgs, err := wakuNode.GetStoredMessages(node2, storeQueryRequest)
			require.NoError(t, err, "Failed to query store messages")
			require.Greater(t, len(*storedmsgs.Messages), 0, "Expected at least one stored message")
		}
		logDebug("##Iteration #%d", i)
	}

	captureMemory(t.Name(), "at end")

	logDebug("[%s] Test completed successfully", t.Name())
}

func TestStressHighThroughput10kPublish(t *testing.T) {
	requiresNode(t)
	node1Cfg := DefaultWakuConfig
	node1Cfg.Relay = true

	node1, err := StartWakuNode(&node1Cfg)
	require.NoError(t, err, "failed to start node1")
	defer node1.Close()

	node2Cfg := DefaultWakuConfig
	node2Cfg.Relay = true

	node2, err := StartWakuNode(&node2Cfg)
	require.NoError(t, err, "failed to start node2")
	defer node2.Close()

	require.NoError(t, node1.Peers().ConnectTo(context.Background(), node2), "failed to connect peers")

	captureMemory(t.Name(), "at start")

	const totalMessages = 1000
	var pubsubTopic = DefaultPubsubTopic

	for i := 0; i < totalMessages; i++ {
		msg := node1.CreateMessage()
		msg.Payload = []byte(fmt.Sprintf("high-throughput message #%d", i))

		hash, err := node1.Relay().Publish(context.Background(), pubsubTopic, msg)
		require.NoError(t, err, "publish failed @%d", i)
		logDebug("Iteration-10kpublish #%d", i)
		err = node2.VerifyMessageReceived(msg, hash)
		require.NoError(t, err, "verification failed @%d", i)

	}

	captureMemory(t.Name(), "at end")
}

func TestStressConnectDisconnect1kIteration(t *testing.T) {
	requiresNode(t)
	captureMemory(t.Name(), "at start")

	node0Cfg := DefaultWakuConfig
	node0Cfg.Relay = true
	node0, err := StartWakuNode(&node0Cfg)
	require.NoError(t, err)
	node1Cfg := DefaultWakuConfig
	node1Cfg.Relay = true
	node1, err := StartWakuNode(&node1Cfg)
	require.NoError(t, err)
	defer func() {
		node0.Close()
		node1.Close()
	}()

	iterations := 1000
	for i := 1; i <= iterations; i++ {
		err := node0.Peers().ConnectTo(context.Background(), node1)
		require.NoError(t, err, "Iteration %d: node0 failed to connect to node1", i)
		time.Sleep(150 * time.Millisecond)
		count, err := node0.Peers().NumConnected()
		require.NoError(t, err, "Iteration %d: failed to get peers for node0", i)
		logDebug("Iteration %d: node0 sees %d connected peers", i, count)
		if count == 1 {
			msg := node0.CreateMessage()
			msg.Payload = []byte(fmt.Sprintf("Iteration %d: message from node0", i))
			msgHash, err := node0.Relay().Publish(context.Background(), DefaultPubsubTopic, msg)
			require.NoError(t, err, "Iteration %d: node0 failed to publish message", i)
			logDebug("Iteration %d: node0 published message with hash %s", i, msgHash.String())
		}
		err = node0.Peers().DisconnectFrom(node1)
		require.NoError(t, err, "Iteration %d: node0 failed to disconnect from node1", i)
		logDebug("Iteration %d: node0 disconnected from node1", i)
		time.Sleep(250 * time.Millisecond)
	}
	captureMemory(t.Name(), "at end")
}

func TestStressRandomNodesInMesh(t *testing.T) {
	requiresNode(t)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	minNodes := 5
	maxNodes := 15
	nodes := make([]*Node, 0, maxNodes)

	for i := 0; i < minNodes; i++ {
		cfg := DefaultWakuConfig
		cfg.Relay = true
		n, err := StartWakuNode(&cfg)
		require.NoError(t, err, "Failed to start initial node %d", i+1)
		nodes = append(nodes, n)
	}

	err := ConnectAllPeers(nodes)
	time.Sleep(1 * time.Second)
	require.NoError(t, err, "Failed to connect initial nodes with ConnectAllPeers")

	captureMemory(t.Name(), "at start")

	testDuration := 10 * time.Minute
	endTime := time.Now().Add(testDuration)

	for time.Now().Before(endTime) {
		action := r.Intn(2)

		if action == 0 && len(nodes) < maxNodes {
			i := len(nodes)
			cfg := DefaultWakuConfig
			cfg.Relay = true
			newNode, err := StartWakuNode(&cfg)
			if err == nil {
				nodes = append(nodes, newNode)
				err := ConnectAllPeers(nodes)
				if err == nil {
					logDebug("Added node%d, now connecting all peers", i+1)
				} else {
					logDebug("Failed to reconnect all peers after adding node%d: %v", i+1, err)
				}
			} else {
				logDebug("Failed to start new node: %v", err)
			}
		} else if action == 1 && len(nodes) > minNodes {
			removeIndex := r.Intn(len(nodes))
			toRemove := nodes[removeIndex]
			nodes = append(nodes[:removeIndex], nodes[removeIndex+1:]...)
			toRemove.Close()
			logDebug("Removed node  %d from mesh", removeIndex)
			if len(nodes) > 1 {
				err := ConnectAllPeers(nodes)
				if err == nil {
					logDebug("Reconnected all peers  node  %d", removeIndex)
				} else {
					logDebug("Failed to reconnect all peers when removing node  %d: %v", removeIndex, err)
				}
			}
		}

		time.Sleep(5 * time.Second)

		for j, n := range nodes {
			count, err := n.Peers().NumConnected()
			if err != nil {
				logDebug("Node%d: error getting connected peers: %v", j+1, err)
			} else {
				logDebug("Node%d sees %d connected peers", j+1, count)
			}
		}

		time.Sleep(3 * time.Second)
	}

	for _, n := range nodes {
		n.Close()
	}

	captureMemory(t.Name(), "at end")
}

func TestStressLargePayloadEphemeralMessagesEndurance(t *testing.T) {
	requiresNode(t)
	nodePubCfg := DefaultWakuConfig
	nodePubCfg.Relay = true
	publisher, err := StartWakuNode(&nodePubCfg)
	require.NoError(t, err)

	nodeRecvCfg := DefaultWakuConfig
	nodeRecvCfg.Relay = true
	receiver, err := StartWakuNode(&nodeRecvCfg)
	require.NoError(t, err)

	err = receiver.Relay().Subscribe(DefaultPubsubTopic)
	require.NoError(t, err)

	defer func() {
		publisher.Close()
		time.Sleep(30 * time.Second)
		receiver.Close()

	}()
	err = publisher.Peers().ConnectTo(context.Background(), receiver)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	captureMemory(t.Name(), "at start")

	maxIterations := 5000
	payloadSize := 100 * 1024
	largePayload := make([]byte, payloadSize)
	for i := range largePayload {
		largePayload[i] = 'a'
	}

	var publishedMessages int
	for i := 0; i < maxIterations; i++ {
		msg := publisher.CreateMessage()
		msg.Payload = largePayload
		ephemeral := true
		msg.Ephemeral = &ephemeral

		_, err := publisher.Relay().Publish(context.Background(), DefaultPubsubTopic, msg)
		if err == nil {
			publishedMessages++
		} else {
			logError("Error publishing ephemeral message: %v", err)
		}

		time.Sleep(1 * time.Second)
		logDebug("###Iteration number %d", i+1)
	}

	captureMemory(t.Name(), "at end")

}

func TestStress2Nodes2kIterationTearDown(t *testing.T) {
	requiresNode(t)

	captureMemory(t.Name(), "at start")
	var err error
	totalIterations := 2000
	for i := 1; i <= totalIterations; i++ {
		var nodes []*Node
		for n := 1; n <= 2; n++ {
			cfg := DefaultWakuConfig
			cfg.Relay = true
			cfg.Discv5Discovery = false
			require.NoError(t, err, "Failed to get free ports for node%d", n)
			node, err := NewFromWakuConfig(&cfg, fmt.Sprintf("node%d", n))
			require.NoError(t, err, "Failed to create node%d", n)
			err = node.Start()
			require.NoError(t, err, "Failed to start node%d", n)
			nodes = append(nodes, node)
		}
		err = ConnectAllPeers(nodes)
		require.NoError(t, err)
		message := nodes[0].CreateMessage()
		msgHash, err := nodes[0].Relay().Publish(context.Background(), DefaultPubsubTopic, message)
		require.NoError(t, err)
		time.Sleep(500 * time.Millisecond)
		err = nodes[1].VerifyMessageReceived(message, msgHash, 500*time.Millisecond)
		require.NoError(t, err, "Node1 did not receive message from node1")
		for _, node := range nodes {
			node.Close()
			time.Sleep(50 * time.Millisecond)
		}
		runtime.GC()
		time.Sleep(250 * time.Millisecond)
		runtime.GC()
		logDebug("Iteration numberrrrrr  %d", i)
	}
	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	runtime.GC()
	captureMemory(t.Name(), "at end")
	//require.LessOrEqual(t, finalRSS, initialRSS*3, "OS-level RSS soared above threshold after %d cycles", totalIterations)
}

func TestPeerExchangePXLoad(t *testing.T) {
	requiresNode(t)
	pxServerCfg := DefaultWakuConfig
	pxServerCfg.PeerExchange = true
	pxServerCfg.Relay = true
	pxServer, err := StartWakuNode(&pxServerCfg)
	require.NoError(t, err, "Failed to start PX server")
	defer pxServer.Close()

	relayA, err := StartWakuNode(&DefaultWakuConfig)
	require.NoError(t, err, "Failed to start RelayA")
	defer relayA.Close()

	relayB, err := StartWakuNode(&DefaultWakuConfig)
	require.NoError(t, err, "Failed to start RelayB")
	defer relayB.Close()

	err = pxServer.Peers().ConnectTo(context.Background(), relayA)
	require.NoError(t, err, "PXServer failed to connect RelayA")
	err = pxServer.Peers().ConnectTo(context.Background(), relayB)
	require.NoError(t, err, "PXServer failed to connect RelayB")

	time.Sleep(2 * time.Second)

	captureMemory(t.Name(), "at start")

	testDuration := 30 * time.Minute
	endTime := time.Now().Add(testDuration)

	lastPublishTime := time.Now().Add(-5 * time.Second) // so first publish is immediate
	for time.Now().Before(endTime) {
		// Publish a message from the PX server every 5 seconds
		if time.Since(lastPublishTime) >= 5*time.Second {
			msg := pxServer.CreateMessage()
			msg.Payload = []byte("PX server message stream")
			_, _ = pxServer.Relay().Publish(context.Background(), DefaultPubsubTopic, msg)
			lastPublishTime = time.Now()
		}

		// Create a light node that relies on PX, run for 3s
		lightCfg := DefaultWakuConfig
		lightCfg.Relay = false
		lightCfg.Store = false
		lightCfg.PeerExchange = true
		lightNode, err := StartWakuNode(&lightCfg)
		if err == nil {
			errPX := lightNode.Peers().ConnectTo(context.Background(), pxServer)
			if errPX == nil {
				// Request peers from PX server
				_, _ = lightNode.PeerExchange().Request(2)
			}
			time.Sleep(3 * time.Second)
			lightNode.Close()
		} else {
			logDebug("Failed to start light node: %v", err)
		}

		time.Sleep(1 * time.Second)
	}

	captureMemory(t.Name(), "at end")
}
