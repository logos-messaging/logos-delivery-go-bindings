package kernel

import (
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/pb"
	"github.com/stretchr/testify/require"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"google.golang.org/protobuf/proto"
)

func TestStoreQuery3Nodes(t *testing.T) {
	Debug("Starting test to verify store query from a peer using direct peer connections")
	queryTimestamp := proto.Int64(time.Now().UnixNano())
	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 (Relay enabled)")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2 (Relay & Store enabled)")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = false

	Debug("Creating Node3 (Peer connected to Node2)")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
	}()

	Debug("Connecting Node1 to Node2")
	err = node1.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node1 to Node2")

	Debug("Connecting Node3 to Node2")
	err = node3.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node3 to Node2")

	Debug("Waiting for peer connections to stabilize")
	err = WaitForAutoConnection([]*WakuNode{node1, node2, node3})
	require.NoError(t, err, "Nodes did not connect within timeout")
	Debug("Publishing message from Node1 using RelayPublish")
	message := node1.CreateMessage(&pb.WakuMessage{
		Payload:      []byte("test-message"),
		ContentTopic: "test-content-topic",
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	})

	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err, "Failed to publish message from Node1")

	Debug("Waiting for message delivery to Node2")
	time.Sleep(2 * time.Second)

	Debug("Verifying that Node2 received the message")
	err = node2.VerifyMessageReceived(message, msgHash)
	require.NoError(t, err, "Node2 should have received the message")

	Debug("Node3 querying stored messages from Node2")
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart:   queryTimestamp,
		IncludeData: true,
	}
	res, err := node3.GetStoredMessages(node2, storeQueryRequest)
	var storedMessages = (*res.Messages)[0]
	require.NoError(t, err, "Failed to retrieve stored messages from Node2")
	require.NotEmpty(t, storedMessages.WakuMessage, "Expected at least one stored message")
	Debug("Verifying stored message matches the published message")
	require.Equal(t, string(message.Payload), string(storedMessages.WakuMessage.Payload), "Stored message payload does not match")
	Debug("Test successfully verified store query from a peer using direct peer connections")
}

func TestStoreQueryMultipleMessages(t *testing.T) {
	Debug("Starting test to verify store query with multiple messages")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 (Relay enabled)")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2 (Relay & Store enabled)")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = false

	Debug("Creating Node3 (Peer connected to Node2)")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
	}()
	var timestamp = proto.Int64(time.Now().UnixNano())
	Debug("Connecting Node1 to Node2")
	err = node1.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node1 to Node2")

	Debug("Connecting Node3 to Node2")
	err = node3.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node3 to Node2")

	Debug("Waiting for peer connections to stabilize")
	time.Sleep(2 * time.Second)

	numMessages := 50
	var sentHashes []common.MessageHash
	defaultPubsubTopic := DefaultPubsubTopic

	Debug("Publishing %d messages from Node1 using RelayPublish", numMessages)
	for i := 0; i < numMessages; i++ {
		message := node1.CreateMessage(&pb.WakuMessage{
			Payload:      []byte(fmt.Sprintf("message-%d", i)),
			ContentTopic: "test-content-topic",
			Timestamp:    proto.Int64(time.Now().UnixNano()),
		})

		msgHash, err := node1.RelayPublishNoCTX(defaultPubsubTopic, message)
		require.NoError(t, err, "Failed to publish message from Node1")

		sentHashes = append(sentHashes, msgHash)
	}

	Debug("Waiting for message delivery to Node2")
	time.Sleep(1 * time.Second)

	Debug("Node3 querying stored messages from Node2")
	storeRequest := &common.StoreQueryRequest{
		IncludeData:       true,
		ContentTopics:     &[]string{"test-content-topic"},
		PaginationLimit:   proto.Uint64(uint64(50)),
		PaginationForward: true,
		TimeStart:         timestamp,
	}
	res, err := node3.GetStoredMessages(node2, storeRequest)
	require.NoError(t, err, "Failed to retrieve stored messages from Node2")
	require.NotNil(t, res.Messages, "Expected stored messages but received nil")

	storedMessages := *res.Messages

	require.Len(t, storedMessages, numMessages, "Expected to retrieve exactly 50 messages")

	Debug("Verifying stored message hashes match sent message hashes")
	var receivedHashes []common.MessageHash
	for _, storedMsg := range storedMessages {
		receivedHashes = append(receivedHashes, storedMsg.MessageHash)
	}

	require.ElementsMatch(t, sentHashes, receivedHashes, "Sent and received message hashes do not match")

	Debug("Test successfully verified store query with multiple messages")
}

func TestStoreQueryWith5Pagination(t *testing.T) {
	Debug("Starting test to verify store query with pagination")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 (Relay enabled)")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2 (Relay & Store enabled)")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = false

	Debug("Creating Node3 (Peer connected to Node2)")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
	}()

	Debug("Connecting Node1 to Node2")
	err = node1.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node1 to Node2")

	Debug("Connecting Node3 to Node2")
	err = node3.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node3 to Node2")

	Debug("Waiting for peer connections to stabilize")
	time.Sleep(2 * time.Second)

	numMessages := 10
	defaultPubsubTopic := DefaultPubsubTopic

	Debug("Publishing %d messages from Node1 using RelayPublish", numMessages)
	for i := 0; i < numMessages; i++ {
		message := node1.CreateMessage(&pb.WakuMessage{
			Payload:      []byte(fmt.Sprintf("message-%d", i)),
			ContentTopic: "test-content-topic",
			Timestamp:    proto.Int64(time.Now().UnixNano()),
		})

		_, err := node1.RelayPublishNoCTX(defaultPubsubTopic, message)
		require.NoError(t, err, "Failed to publish message from Node1")

	}

	Debug("Waiting for message delivery to Node2")
	time.Sleep(1 * time.Second)

	Debug("Node3 querying stored messages from Node2 with PaginationLimit = 5")
	storeRequest := common.StoreQueryRequest{
		IncludeData:       true,
		ContentTopics:     &[]string{"test-content-topic"},
		PaginationLimit:   proto.Uint64(uint64(5)),
		PaginationForward: true,
	}

	res, err := node3.GetStoredMessages(node2, &storeRequest)
	require.NoError(t, err, "Failed to retrieve stored messages from Node2")
	require.NotNil(t, res.Messages, "Expected stored messages but received nil")

	storedMessages := *res.Messages

	require.Len(t, storedMessages, 5, "Expected to retrieve exactly 5 messages due to pagination limit")

	Debug("Test successfully verified store query with pagination limit")
}

func TestStoreQueryWithPaginationMultiplePages(t *testing.T) {
	Debug("Starting test to verify store query with pagination across multiple pages")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 (Relay enabled)")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2 (Relay & Store enabled)")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = false

	Debug("Creating Node3 (Peer connected to Node2)")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
	}()
	var timestamp = proto.Int64(time.Now().UnixNano())
	Debug("Connecting Node1 to Node2")
	err = node1.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node1 to Node2")

	Debug("Connecting Node3 to Node2")
	err = node3.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node3 to Node2")

	Debug("Waiting for peer connections to stabilize")
	time.Sleep(2 * time.Second)

	numMessages := 8
	var sentHashes []common.MessageHash
	defaultPubsubTopic := DefaultPubsubTopic

	Debug("Publishing %d messages from Node1 using RelayPublish", numMessages)
	for i := 0; i < numMessages; i++ {
		message := node1.CreateMessage(&pb.WakuMessage{
			Payload:      []byte(fmt.Sprintf("message-%d", i)),
			ContentTopic: "test-content-topic",
			Timestamp:    proto.Int64(time.Now().UnixNano()),
		})

		msgHash, err := node1.RelayPublishNoCTX(defaultPubsubTopic, message)
		require.NoError(t, err, "Failed to publish message from Node1")

		sentHashes = append(sentHashes, msgHash)
	}

	Debug("Waiting for message delivery to Node2")
	time.Sleep(1 * time.Second)

	Debug("Node3 querying first page of stored messages from Node2")
	storeRequest1 := common.StoreQueryRequest{
		IncludeData:       true,
		ContentTopics:     &[]string{"test-content-topic"},
		PaginationLimit:   proto.Uint64(5),
		PaginationForward: true,
		TimeStart:         timestamp,
	}

	res1, err := node3.GetStoredMessages(node2, &storeRequest1)
	require.NoError(t, err, "Failed to retrieve first page of stored messages from Node2")
	require.NotNil(t, res1.Messages, "Expected stored messages but received nil")

	storedMessages1 := *res1.Messages
	require.Len(t, storedMessages1, 5, "Expected to retrieve exactly 5 messages from first query")

	for i := 0; i < 5; i++ {
		require.Equal(t, sentHashes[i], storedMessages1[i].MessageHash, "Message order mismatch in first query")
	}

	Debug("Node3 querying second page of stored messages from Node2")
	storeRequest2 := common.StoreQueryRequest{
		IncludeData:       true,
		ContentTopics:     &[]string{"test-content-topic"},
		PaginationLimit:   proto.Uint64(5),
		PaginationForward: true,
		PaginationCursor:  &res1.PaginationCursor,
	}

	res2, err := node3.GetStoredMessages(node2, &storeRequest2)
	require.NoError(t, err, "Failed to retrieve second page of stored messages from Node2")
	require.NotNil(t, res2.Messages, "Expected stored messages but received nil")

	storedMessages2 := *res2.Messages
	require.Len(t, storedMessages2, 3, "Expected to retrieve exactly 3 messages from second query")

	for i := 0; i < 3; i++ {
		require.Equal(t, sentHashes[i+5], storedMessages2[i].MessageHash, "Message order mismatch in second query")
	}

	Debug("Test successfully verified store query pagination across multiple pages")
}

func TestStoreQueryWithPaginationReverseOrder(t *testing.T) {
	Debug("Starting test to verify store query with pagination in reverse order")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 (Relay enabled)")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2 (Relay & Store enabled)")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = false

	Debug("Creating Node3 (Peer connected to Node2)")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
	}()

	Debug("Connecting Node1 to Node2")
	err = node1.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node1 to Node2")

	Debug("Connecting Node3 to Node2")
	err = node3.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node3 to Node2")

	Debug("Waiting for peer connections to stabilize")
	time.Sleep(2 * time.Second)

	numMessages := 8
	var sentHashes []common.MessageHash
	defaultPubsubTopic := DefaultPubsubTopic

	queryTimestamp := proto.Int64(time.Now().UnixNano())
	Debug("Publishing %d messages from Node1 using RelayPublish", numMessages)
	for i := 0; i < numMessages; i++ {
		message := node1.CreateMessage(&pb.WakuMessage{
			Payload:      []byte(fmt.Sprintf("message-%d", i)),
			ContentTopic: "test-content-topic",
			Timestamp:    proto.Int64(time.Now().UnixNano()),
		})

		msgHash, err := node1.RelayPublishNoCTX(defaultPubsubTopic, message)
		require.NoError(t, err, "Failed to publish message from Node1")

		sentHashes = append(sentHashes, msgHash)
		Debug("sent hash number %i is %s", i, sentHashes[i])
	}

	Debug("Waiting for message delivery to Node2")
	time.Sleep(1 * time.Second)

	Debug("Node3 querying first page of stored messages from Node2 (Newest first)")
	storeRequest1 := common.StoreQueryRequest{
		IncludeData:       true,
		ContentTopics:     &[]string{"test-content-topic"},
		PaginationLimit:   proto.Uint64(5),
		PaginationForward: false,
		TimeStart:         queryTimestamp,
	}

	res1, err := node3.GetStoredMessages(node2, &storeRequest1)
	require.NoError(t, err, "Failed to retrieve first page of stored messages from Node2")
	require.NotNil(t, res1.Messages, "Expected stored messages but received nil")

	storedMessages1 := *res1.Messages
	require.Len(t, storedMessages1, 5, "Expected to retrieve exactly 5 messages from first query")
	for i := 0; i < 5; i++ {
		Debug("stored hashes round 2 iteration %i is %s", i, storedMessages1[i].MessageHash)
	}

	for i := 0; i < 5; i++ {
		require.Equal(t, sentHashes[i+3], storedMessages1[i].MessageHash, "Message order mismatch in first query")
	}

	Debug("Node3 querying second page of stored messages from Node2")
	storeRequest2 := common.StoreQueryRequest{
		IncludeData:       true,
		ContentTopics:     &[]string{"test-content-topic"},
		PaginationLimit:   proto.Uint64(3),
		PaginationForward: false,
		PaginationCursor:  &res1.PaginationCursor,
		TimeStart:         queryTimestamp,
	}

	res2, err := node3.GetStoredMessages(node2, &storeRequest2)
	require.NoError(t, err, "Failed to retrieve second page of stored messages from Node2")
	require.NotNil(t, res2.Messages, "Expected stored messages but received nil")

	storedMessages2 := *res2.Messages
	require.Len(t, storedMessages2, 3, "Expected to retrieve exactly 3 messages from second query")

	for i := 0; i < 3; i++ {
		require.Equal(t, sentHashes[i], storedMessages2[i].MessageHash, "Message order mismatch in second query")

	}

	Debug("Test successfully verified store query pagination in reverse order")
}

func TestQueryFailWhenNoStorePeer(t *testing.T) {
	Debug("Starting test to verify store query failure when node2 has no store")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 with Relay enabled")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = false

	Debug("Creating Node2 with Relay enabled but Store disabled")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = true

	Debug("Creating Node3")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	Debug("Connecting Node3 to Node2")
	err = node3.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node3 to Node2")

	Debug("Sender Node1 is publishing a message")
	message := node1.CreateMessage()
	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	Debug("Verifying that Node3 fails to retrieve stored messages since Node2 has store disabled")
	storedMessages, err := node3.GetStoredMessages(node2, nil)
	require.Error(t, err, "Expected Node3's store query to fail because Node2 has store disabled")
	require.Empty(t, storedMessages, "Expected no messages in store for Node3")

	Debug("Test successfully verified that store query fails when Node2 does not store messages")
}

func TestQueryFailWithIncorrectStaticNode(t *testing.T) {
	Debug("Starting test to verify store query failure when Node3 has an incorrect static node address")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 with Relay enabled")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node1Address, err := node1.ListenAddresses()
	require.NoError(t, err, "Failed to get listening address for Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true
	node2Config.Staticnodes = []string{node1Address[0].String()}

	Debug("Creating Node2 with Store enabled")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node2Address, err := node2.ListenAddresses()
	require.NoError(t, err, "Failed to get listening address for Node2")

	var incorrectAddress = node2Address[0].String()[:len(node2Address[0].String())-10]
	node3Config := DefaultWakuConfig
	node3Config.Relay = true
	node3Config.Staticnodes = []string{incorrectAddress}

	Debug("Original Node2 Address: %s", node2Address[0].String())
	Debug("Modified Node2 Address: %s", incorrectAddress)

	Debug("Creating Node3 with an incorrect static node address")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
	}()

	Debug("Sender Node1 is publishing a message")
	queryTimestamp := proto.Int64(time.Now().UnixNano())
	message := node1.CreateMessage()
	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	Debug("Verifying that Node3 fails to retrieve stored messages due to incorrect static node")
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart: queryTimestamp,
	}
	storedmsgs, err := node3.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Expected Node3's store query to fail due to incorrect static node")
	require.Nil(t, (*storedmsgs.Messages)[0].WakuMessage, "Expected no messages in store for Node3")

	Debug("Test successfully verified store query failure due to incorrect static node configuration")
}

func TestStoreQueryWithoutData(t *testing.T) {
	Debug("Starting test to verify store query returns only message hashes when IncludeData is false")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 with Relay enabled")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true // Enable store on Node2

	Debug("Creating Node2 with Store enabled")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = true

	Debug("Creating Node3")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	Debug("Connecting Node3 to Node2")
	err = node3.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node3 to Node2")

	Debug("Sender Node1 is publishing a message")
	message := node1.CreateMessage()
	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	Debug("Querying stored messages from Node3 with IncludeData = false")
	storeQueryRequest := &common.StoreQueryRequest{
		IncludeData: false,
	}

	storedmsgs, err := node3.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Failed to query store messages from Node2")
	require.NotNil(t, storedmsgs.Messages, "Expected store response to contain message hashes")

	firstMessage := (*storedmsgs.Messages)[0]
	require.Nil(t, firstMessage.WakuMessage, "Expected message payload to be empty when IncludeData is false")
	require.NotEmpty(t, (*storedmsgs.Messages)[0].MessageHash, "Expected message hash to be present")
	Debug("Queried message hash: %s", (*storedmsgs.Messages)[0].MessageHash)

	Debug("Test successfully verified that store query returns only message hashes when IncludeData is false")
}

func TestStoreQueryWithWrongContentTopic(t *testing.T) {
	Debug("Starting test to verify store query fails when using an incorrect content topic and an old timestamp")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 with Relay enabled")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2 with Store enabled")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = true

	Debug("Creating Node3")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	Debug("Connecting Node3 to Node2")
	err = node3.ConnectPeer(node2)
	require.NoError(t, err, "Failed to connect Node3 to Node2")

	Debug("Recording timestamp before message publication")
	queryTimestamp := proto.Int64(time.Now().UnixNano())

	Debug("Sender Node1 is publishing a message with a correct content topic")
	message := node1.CreateMessage()
	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	Debug("Querying stored messages from Node3 with an incorrect content topic and an old timestamp")
	storeQueryRequest := &common.StoreQueryRequest{
		ContentTopics: &[]string{"incorrect-content-topic"},
		TimeStart:     queryTimestamp,
	}

	storedmsgs, _ := node3.GetStoredMessages(node2, storeQueryRequest)
	require.Empty(t, (*storedmsgs.Messages), "Expected no messages to be returned for incorrect content topic and timestamp")
	Debug("Test successfully verified that store query fails when using an incorrect content topic and an old timestamp")
}

func TestCheckStoredMSGsEphemeralTrue(t *testing.T) {
	Debug("Starting test to verify ephemeral messages are not stored")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 with Relay enabled")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2 with Store enabled")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	Debug("Recording timestamp before message publication")
	queryTimestamp := proto.Int64(time.Now().UnixNano())

	Debug("Sender Node1 is publishing an ephemeral message")
	message := node1.CreateMessage()
	ephemeralTrue := true
	message.Ephemeral = &ephemeralTrue

	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	Debug("Querying stored messages from Node2")
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart: queryTimestamp,
	}

	storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Failed to query store messages from Node2")
	require.Equal(t, 0, len(*storedmsgs.Messages), "Expected no stored messages for ephemeral messages")

	Debug("Test successfully verified that ephemeral messages are not stored")
}

func TestCheckStoredMSGsEphemeralFalse(t *testing.T) {
	Debug("Starting test to verify non-ephemeral messages are stored")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 with Relay enabled")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2 with Store enabled")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	Debug("Recording timestamp before message publication")
	queryTimestamp := proto.Int64(time.Now().UnixNano())

	Debug("Sender Node1 is publishing a non-ephemeral message")
	message := node1.CreateMessage()
	ephemeralFalse := false
	message.Ephemeral = &ephemeralFalse

	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	Debug("Querying stored messages from Node2")
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart: queryTimestamp,
	}

	storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Failed to query store messages from Node2")
	require.Equal(t, 1, len(*storedmsgs.Messages), "Expected exactly one stored message")

	Debug("Test finished successfully ")
}

func TestCheckLegacyStore(t *testing.T) {
	t.Skip("Skipping test as legacy store is about to be removed and no longer maintained")
	Debug("Starting test ")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1 ")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true
	node2Config.LegacyStore = true

	Debug("Creating Node2")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")
	queryTimestamp := proto.Int64(time.Now().UnixNano())

	Debug("Sender Node1 is publishing a message")
	message := node1.CreateMessage()
	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err)
	require.NotEmpty(t, msgHash)

	Debug("Querying stored messages from Node2 using Node1")
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart: queryTimestamp,
	}

	storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Failed to query store messages from Node2")
	require.Equal(t, 1, len(*storedmsgs.Messages), "Expected exactly one stored message")

	Debug("Test finished successfully ")

}

func TestStoredMessagesWithVDifferentPayloads(t *testing.T) {
	Debug("Starting test ")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2 ")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	for _, pLoad := range SAMPLE_INPUTS {
		queryTimestamp := proto.Int64(time.Now().UnixNano())

		Debug("Sender Node1 is publishing message with payload: %s", pLoad.Value)
		message := node1.CreateMessage()
		message.Payload = []byte(pLoad.Value)

		msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
		require.NoError(t, err, "Failed to publish message")
		require.NotEmpty(t, msgHash, "Message hash is empty")

		Debug("Querying stored messages from Node2 using Node1")
		storeQueryRequest := &common.StoreQueryRequest{
			TimeStart:   queryTimestamp,
			IncludeData: true,
		}

		storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
		require.NoError(t, err, "Failed to query store messages from Node2")
		retrievedMessage := (*storedmsgs.Messages)[0]
		require.Equal(t, pLoad.Value, string(retrievedMessage.WakuMessage.Payload), "Expected WakuMessage but got nil")
		Debug("Payload matches expected %s", string(retrievedMessage.WakuMessage.Payload))
	}

	Debug("Test finished successfully ")
}

func TestStoredMessagesWithDifferentContentTopics(t *testing.T) {
	Debug("Starting test for different content topics")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	for _, contentTopic := range CONTENT_TOPICS_DIFFERENT_SHARDS {

		Debug("Node1 is publishing message with content topic: %s", contentTopic)
		queryTimestamp := proto.Int64(time.Now().UnixNano())
		message := node1.CreateMessage()
		message.ContentTopic = contentTopic

		msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
		require.NoError(t, err, "Failed to publish message")
		require.NotEmpty(t, msgHash, "Message hash is empty")

		Debug("Querying stored messages from Node2 using Node1")
		storeQueryRequest := &common.StoreQueryRequest{
			TimeStart:     queryTimestamp,
			IncludeData:   true,
			ContentTopics: &[]string{contentTopic},
		}

		storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
		require.NoError(t, err, "Failed to query store messages from Node2")
		require.Greater(t, len(*storedmsgs.Messages), 0, "Expected at least one stored message")
		require.Equal(t, contentTopic, (*storedmsgs.Messages)[0].WakuMessage.ContentTopic, "Stored message content topic does not match expected")
		Debug("Veified content topic %s ", (*storedmsgs.Messages)[0].WakuMessage.ContentTopic)
	}

	Debug("Test finished successfully")
}

func TestStoredMessagesWithDifferentPubsubTopics(t *testing.T) {
	Debug("Starting test for different pubsub topics")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	for _, pubsubTopic := range PUBSUB_TOPICS_STORE {

		Debug("Node1 is publishing message on pubsub topic: %s", pubsubTopic)
		node1.RelaySubscribe(pubsubTopic)
		node2.RelaySubscribe(pubsubTopic)
		time.Sleep(time.Second * 2)
		queryTimestamp := proto.Int64(time.Now().UnixNano())
		var msg = node1.CreateMessage()
		msgHash, err := node1.RelayPublishNoCTX(pubsubTopic, msg)
		require.NoError(t, err, "Failed to publish message")
		require.NotEmpty(t, msgHash, "Message hash is empty")

		Debug("Querying stored messages from Node2 using Node1")
		storeQueryRequest := &common.StoreQueryRequest{
			TimeStart:   queryTimestamp,
			IncludeData: true,
			PubsubTopic: pubsubTopic,
		}

		storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
		require.NoError(t, err, "Failed to query store messages from Node2")
		require.Greater(t, len(*storedmsgs.Messages), 0, "Expected at least one stored message")
		require.Equal(t, pubsubTopic, (*storedmsgs.Messages)[0].PubsubTopic, "Stored message pubsub topic does not match expected")
	}

	Debug("Test finished successfully")
}

func TestStoredMessagesWithMetaField(t *testing.T) {
	Debug("Starting test ")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")
	queryTimestamp := proto.Int64(time.Now().UnixNano())

	Debug("Node1 is publishing a message with meta field set")
	message := node1.CreateMessage()
	message.Payload = []byte("payload")
	message.Meta = []byte([]byte("hello"))

	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err, "Failed to publish message")
	require.NotEmpty(t, msgHash, "Message hash is empty")

	Debug("Querying stored messages from Node2 using Node1")
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart:   queryTimestamp,
		IncludeData: true,
	}

	storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Failed to query store messages from Node2")
	require.Greater(t, len(*storedmsgs.Messages), 0, "Expected at least one stored message")

	retrievedMessage := (*storedmsgs.Messages)[0].WakuMessage
	require.Equal(t, string(message.Payload), string(retrievedMessage.Payload), "Payload does not match")
	require.Equal(t, string(message.Meta), string(retrievedMessage.Meta), "Meta field does not match expected Base64-encoded payload")

	Debug("Test finished successfully ")
}

func TestStoredMessagesWithVersionField(t *testing.T) {
	Debug("Starting test")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	version := uint32(2)

	queryTimestamp := proto.Int64(time.Now().UnixNano())

	Debug("Node1 is publishing a message with version field set")
	message := node1.CreateMessage()
	message.Version = &version

	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err, "Failed to publish message")
	require.NotEmpty(t, msgHash, "Message hash is empty")

	Debug("Querying stored messages from Node2 using Node1")
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart:   queryTimestamp,
		IncludeData: true,
	}

	storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Failed to query store messages from Node2")
	require.Greater(t, len(*storedmsgs.Messages), 0, "Expected at least one stored message")

	retrievedMessage := (*storedmsgs.Messages)[0].WakuMessage
	require.Equal(t, version, *retrievedMessage.Version, "Version field does not match expected value")

	Debug("Test finished successfully ")
}

func TestStoredDuplicateMessage(t *testing.T) {
	Debug("Starting test")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	Debug("Waiting for peer connections to stabilize")
	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 10 * time.Second
	}

	require.NoError(t, RetryWithBackOff(func() error {
		numPeersNode1, err := node1.GetNumConnectedRelayPeers(DefaultPubsubTopic)
		if err != nil {
			return err
		}
		if numPeersNode1 == 0 {
			return fmt.Errorf("node1 has 0 relay peers, expected at least 1")
		}

		numPeersNode2, err := node2.GetNumConnectedRelayPeers(DefaultPubsubTopic)
		if err != nil {
			return err
		}
		if numPeersNode2 == 0 {
			return fmt.Errorf("node2 has 0 relay peers, expected at least 1")
		}

		return nil
	}, options), "Peers did not stabilize in time")

	queryTimestamp := proto.Int64(time.Now().UnixNano())
	var msg = node1.CreateMessage()
	Debug("Node1 is publishing two identical messages")
	_, err = node1.RelayPublishNoCTX(DefaultPubsubTopic, msg)
	require.NoError(t, err, "Failed to publish first message")

	_, err = node2.RelayPublishNoCTX(DefaultPubsubTopic, msg)
	require.NoError(t, err, "Failed to publish second message")

	Debug("Querying stored messages from Node2 using Node1")
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart:   queryTimestamp,
		IncludeData: true,
	}

	storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Failed to query store messages from Node2")

	require.Equal(t, 1, len(*storedmsgs.Messages), "Expected only one stored message since identical messages should be deduplicated")

	Debug("Test finished successfully")
}

func TestQueryStoredMessagesWithoutPublishing(t *testing.T) {
	Debug("Starting test: Querying stored messages without publishing any")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	queryTimestamp := proto.Int64(time.Now().UnixNano())

	Debug("Querying stored messages from Node2 without publishing any message")
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart:   queryTimestamp,
		IncludeData: true,
	}

	storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Failed to query store messages from Node2")
	require.Empty(t, *storedmsgs.Messages, "Expected no stored messages")

	Debug("Test finished successfully")
}

func TestQueryStoredMessagesWithWrongHash(t *testing.T) {
	Debug("Starting test: Querying stored messages with a slightly modified message hash")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Store = true

	Debug("Creating Node2")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying both nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Connecting Node2 to Node1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node2 to Node1")

	queryTimestamp := proto.Int64(time.Now().UnixNano())

	Debug("Node1 is publishing a message")
	message := node1.CreateMessage()
	message.Payload = []byte("Test message for hash modification")

	msgHash, err := node1.RelayPublishNoCTX(DefaultPubsubTopic, message)
	require.NoError(t, err, "Failed to publish message")
	require.NotEmpty(t, msgHash, "Message hash is empty")

	Debug("MOdify the original message hash: %s", msgHash)
	modifiedHash := common.MessageHash(msgHash[:len(msgHash)-2] + "da")

	Debug("Querying stored messages from Node2 using a modified hash: %s", modifiedHash)
	storeQueryRequest := &common.StoreQueryRequest{
		TimeStart:     queryTimestamp,
		IncludeData:   true,
		MessageHashes: &[]common.MessageHash{modifiedHash},
	}

	storedmsgs, err := node1.GetStoredMessages(node2, storeQueryRequest)
	require.NoError(t, err, "Failed to query store messages from Node2 with modified hash")
	require.Empty(t, *storedmsgs.Messages, "Expected no stored messages with a modified hash")

	Debug("Test finished successfully: Query with a modified hash returned 0 messages as expected")
}
