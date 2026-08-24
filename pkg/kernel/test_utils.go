package kernel

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/pb"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/utils"
	"google.golang.org/protobuf/proto"
)

type NwakuInfo struct {
	ListenAddresses []string `json:"listenAddresses"`
	EnrUri          string   `json:"enrUri"`
}

func GetNwakuInfo(host *string, port *int) (NwakuInfo, error) {
	nwakuRestPort := 8645
	if port != nil {
		nwakuRestPort = *port
	}
	envNwakuRestPort := os.Getenv("NWAKU_REST_PORT")
	if envNwakuRestPort != "" {
		v, err := strconv.Atoi(envNwakuRestPort)
		if err != nil {
			return NwakuInfo{}, err
		}
		nwakuRestPort = v
	}

	nwakuRestHost := "localhost"
	if host != nil {
		nwakuRestHost = *host
	}
	envNwakuRestHost := os.Getenv("NWAKU_REST_HOST")
	if envNwakuRestHost != "" {
		nwakuRestHost = envNwakuRestHost
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:%d/debug/v1/info", nwakuRestHost, nwakuRestPort))
	if err != nil {
		return NwakuInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NwakuInfo{}, err
	}

	var data NwakuInfo
	err = json.Unmarshal(body, &data)
	if err != nil {
		return NwakuInfo{}, err
	}

	return data, nil
}

type BackOffOption func(*backoff.ExponentialBackOff)

func RetryWithBackOff(o func() error, options ...BackOffOption) error {
	b := backoff.ExponentialBackOff{
		InitialInterval:     time.Millisecond * 100,
		RandomizationFactor: 0.1,
		Multiplier:          1,
		MaxInterval:         time.Second,
		MaxElapsedTime:      time.Second * 10,
		Clock:               backoff.SystemClock,
	}
	for _, option := range options {
		option(&b)
	}
	b.Reset()
	return backoff.Retry(o, &b)
}

func (n *Node) CreateMessage(customMessage ...*pb.WakuMessage) *pb.WakuMessage {
	logDebug("Creating a WakuMessage")

	if len(customMessage) > 0 && customMessage[0] != nil {
		logDebug("Using provided custom message")
		return customMessage[0]
	}

	logDebug("Using default message format")
	defaultMessage := &pb.WakuMessage{
		Payload:      []byte("This is a default Waku message payload"),
		ContentTopic: DefaultContentTopic,
		Version:      proto.Uint32(0),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	}

	logDebug("Successfully created a default WakuMessage")
	return defaultMessage
}

func WaitForAutoConnection(nodeList []*Node) error {
	logDebug("Waiting for auto-connection of nodes...")

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	err := RetryWithBackOff(func() error {
		for _, node := range nodeList {
			peers, err := node.Peers().Connected()
			if err != nil {
				return err
			}

			if len(peers) < 1 {
				return errors.New("expected at least one connected peer") // Retry
			}

			logDebug("node has %d connected peers", len(peers))
		}

		return nil
	}, options)

	if err != nil {
		logError("Auto-connection failed after retries: %v", err)
		return err
	}

	logDebug("Auto-connection check completed successfully")
	return nil
}

func (n *Node) VerifyMessageReceived(expectedMessage *pb.WakuMessage, expectedHash common.MessageHash, timeout ...time.Duration) error {

	var verifyTimeout time.Duration
	if len(timeout) > 0 {
		verifyTimeout = timeout[0]
	} else {
		verifyTimeout = DefaultTimeOut
	}

	logDebug("Verifying if the message was received on node %s, timeout: %v", verifyTimeout)

	ctx, cancel := context.WithTimeout(context.Background(), verifyTimeout)
	defer cancel()

	select {
	case envelope := <-n.Messages():
		if string(expectedMessage.Payload) != string(envelope.Message().Payload) {
			logError("Payload does not match")
			return errors.New("payload does not match")
		}
		if expectedMessage.ContentTopic != envelope.Message().ContentTopic {
			logError("Content topic does not match")
			return errors.New("content topic does not match")
		}
		if expectedHash != envelope.Hash() {
			logError("Message hash does not match")
			return errors.New("message hash does not match")
		}
		logDebug("message received and verified: %s", string(envelope.Message().Payload))
		return nil
	case <-ctx.Done():
		logError("Timeout: message not received within %v", verifyTimeout)
		return errors.New("timeout: message not received within the given duration")
	}
}

func ConnectAllPeers(nodes []*Node) error {
	if len(nodes) == 0 {
		logError("Cannot connect peers: node list is empty")
		return errors.New("node list is empty")
	}

	timeout := time.Duration(len(nodes)*2) * time.Second
	logDebug("Connecting nodes in a relay chain with timeout: %v", timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for i := 0; i < len(nodes)-1; i++ {
		logDebug("Connecting node %d to node %d", i, i+1)
		err := nodes[i].Peers().ConnectTo(context.Background(), nodes[i+1])
		if err != nil {
			logError("Failed to connect node %d to node %d: %v", i, i+1, err)
			return err
		}
	}

	<-ctx.Done()
	logDebug("Connections stabilized")
	return nil
}

func SubscribeNodesToTopic(nodes []*Node, topic string) error {
	for _, node := range nodes {
		logDebug("Subscribing node %s to topic %s", topic)
		err := node.Relay().Subscribe(topic)

		if err != nil {
			logError("Failed to subscribe node %s to topic %s: %v", topic, err)
			return err
		}
		logDebug("Node %s successfully subscribed to topic %s", topic)
	}
	return nil
}

func (n *Node) GetStoredMessages(storeNode *Node, storeRequest *common.StoreQueryRequest) (*common.StoreQueryResponse, error) {
	logDebug("Starting store query request")

	if storeRequest == nil {
		logDebug("Using DefaultStoreQueryRequest")
		storeRequest = &DefaultStoreQueryRequest
	}

	storeMultiaddr, err := storeNode.Debug().ListenAddresses()
	if err != nil {
		logError("Failed to retrieve listen addresses for store node: %v", err)
		return nil, err
	}

	if len(storeMultiaddr) == 0 {
		logError("Store node has no available listen addresses")
		return nil, errors.New("store node has no available listen addresses")
	}

	storeNodeAddrInfo, err := peer.AddrInfoFromString(storeMultiaddr[0].String())
	if err != nil {
		logError("Failed to convert store node address to AddrInfo: %v", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logDebug("Querying store node for messages")
	res, err := n.Store().Query(ctx, storeRequest, *storeNodeAddrInfo)
	if err != nil {
		logError("StoreQuery failed: %v", err)
		return nil, err
	}

	logDebug("Store query successful, retrieved %d messages", len(*res.Messages))
	return res, nil
}

func recordMemoryMetricsPX(testName, phase string, heapAllocKB, rssKB uint64) error {
	staticMu := sync.Mutex{}
	staticMu.Lock()
	defer staticMu.Unlock()

	file, err := os.OpenFile("px_load_metrics.csv", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if stat.Size() == 0 {
		header := []string{"TestName", "Phase", "HeapAlloc(KB)", "RSS(KB)", "Timestamp"}
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	row := []string{
		testName,
		phase,
		strconv.FormatUint(heapAllocKB, 10),
		strconv.FormatUint(rssKB, 10),
		time.Now().Format(time.RFC3339),
	}
	return writer.Write(row)
}

func captureMemory(testName, phase string) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	heapKB := ms.HeapAlloc / 1024
	rssKB, _ := utils.GetRSSKB()

	logDebug("[%s] Memory usage  (%s): %d KB (RSS %d KB)", testName, phase, heapKB, rssKB)

	_ = recordMemoryMetricsPX(testName, phase, heapKB, rssKB)
}
