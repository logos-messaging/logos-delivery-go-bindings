package kernel

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/timesource"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pproto "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/pb"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/utils"
	"github.com/multiformats/go-multiaddr"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
)

const requestTimeout = 30 * time.Second
const MsgChanBufferSize = 1024
const TopicHealthChanBufferSize = 1024
const ConnectionChangeChanBufferSize = 1024

// WakuNode represents an instance of an nwaku node
type WakuNode struct {
	wakuCtx              ffi.Handle
	config               *common.WakuConfig
	MsgChan              chan common.Envelope
	TopicHealthChan      chan topicHealth
	ConnectionChangeChan chan connectionChange
	nodeName             string
	_                    timesource.Timesource
}

func NewWakuNode(config *common.WakuConfig, nodeName string) (*WakuNode, error) {
	Debug("Creating new WakuNode: %v", nodeName)
	n := &WakuNode{
		config:   config,
		nodeName: nodeName,
	}

	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	n.wakuCtx, err = ffi.WakuNew(string(jsonConfig))
	if err != nil {
		Error("error wakuNew for %s: %v", nodeName, err)
		return nil, err
	}

	n.MsgChan = make(chan common.Envelope, MsgChanBufferSize)
	n.TopicHealthChan = make(chan topicHealth, TopicHealthChanBufferSize)
	n.ConnectionChangeChan = make(chan connectionChange, ConnectionChangeChanBufferSize)

	ffi.SetWakuEventHandler(n.wakuCtx, n.onRawEvent)

	Debug("Successfully created WakuNode: %s", nodeName)
	return n, nil
}

// onRawEvent receives every libwaku event for this node from the ffi bridge.
func (n *WakuNode) onRawEvent(ret int, msg string) {
	if ret == ffi.RetOK {
		n.OnEvent(msg)
		return
	}
	if msg != "" {
		Error("wakuEventCallback retCode not ok, retCode: %v: %v", ret, msg)
	} else {
		Error("wakuEventCallback retCode not ok, retCode: %v", ret)
	}
}

type jsonEvent struct {
	EventType string `json:"eventType"`
}

type topicHealth struct {
	PubsubTopic string `json:"pubsubTopic"`
	TopicHealth string `json:"topicHealth"`
}

type connectionChange struct {
	PeerId    peer.ID `json:"peerId"`
	PeerEvent string  `json:"peerEvent"`
}

func (n *WakuNode) OnEvent(eventStr string) {
	jsonEvent := jsonEvent{}
	err := json.Unmarshal([]byte(eventStr), &jsonEvent)
	if err != nil {
		Error("could not unmarshal nwaku event string: %v", err)

		return
	}

	switch jsonEvent.EventType {
	case "message":
		n.parseMessageEvent(eventStr)
	case "relay_topic_health_change":
		n.parseTopicHealthChangeEvent(eventStr)
	case "connection_change":
		n.parseConnectionChangeEvent(eventStr)
	}
}

func (n *WakuNode) parseMessageEvent(eventStr string) {
	var envelope common.Envelope
	err := json.Unmarshal([]byte(eventStr), &envelope)
	if err != nil {
		Error("could not parse message %v", err)
		return
	}
	select {
	case n.MsgChan <- envelope:
	default:
		Warn("Can't deliver message to subscription, MsgChan is full")
	}
}

func (n *WakuNode) parseTopicHealthChangeEvent(eventStr string) {

	topicHealth := topicHealth{}
	err := json.Unmarshal([]byte(eventStr), &topicHealth)
	if err != nil {
		Error("could not parse topic health change %v", err)
	}

	select {
	case n.TopicHealthChan <- topicHealth:
	default:
		Warn("Can't deliver topic health event, TopicHealthChan is full")
	}
}

func (n *WakuNode) parseConnectionChangeEvent(eventStr string) {

	connectionChange := connectionChange{}
	err := json.Unmarshal([]byte(eventStr), &connectionChange)
	if err != nil {
		Error("could not parse connection change %v", err)
	}

	select {
	case n.ConnectionChangeChan <- connectionChange:
	default:
		Warn("Can't deliver connection change event, ConnectionChangeChan is full")
	}
}

func (n *WakuNode) GetNumConnectedRelayPeers(optPubsubTopic ...string) (int, error) {

	Debug("Fetching number of connected relay peers for %s", n.nodeName)
	pubsubTopic := ""
	if len(optPubsubTopic) > 0 {
		pubsubTopic = optPubsubTopic[0]
	}

	numPeersStr, err := ffi.WakuGetNumConnectedRelayPeers(n.wakuCtx, pubsubTopic)
	if err != nil {
		errMsg := "error GetNumConnectedRelayPeers: " + err.Error()
		Error("Failed to get number of connected relay peers for %s: %s", n.nodeName, errMsg)
		return 0, errors.New(errMsg)
	}

	numPeers, err := strconv.Atoi(numPeersStr)
	if err != nil {
		Error("Failed to convert relay peer count for %s: %v", n.nodeName, err)
		return 0, err
	}
	Debug("Successfully fetched number of connected relay peers for %s: %d", n.nodeName, numPeers)
	return numPeers, nil
}

func (n *WakuNode) GetConnectedRelayPeers(optPubsubTopic ...string) (peer.IDSlice, error) {

	pubsubTopic := ""
	if len(optPubsubTopic) > 0 {
		pubsubTopic = optPubsubTopic[0]
	}

	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to get connected relay peers: %v", err)
		return nil, err
	}

	Debug("Fetching connected relay peers for pubsubTopic: %v, node: %v", pubsubTopic, n.nodeName)

	peersStr, err := ffi.WakuGetConnectedRelayPeers(n.wakuCtx, pubsubTopic)
	if err != nil {
		errMsg := "error GetConnectedRelayPeers: " + err.Error()
		Error("Failed to get connected relay peers for pubsubTopic: %v:, node: %v. %v", pubsubTopic, n.nodeName, errMsg)
		return nil, errors.New(errMsg)
	}

	if peersStr == "" {
		Debug("No connected relay peers found for pubsubTopic: %v, node: %v", pubsubTopic, n.nodeName)
		return nil, nil
	}

	peerIDs := strings.Split(peersStr, ",")
	var peers peer.IDSlice
	for _, peerID := range peerIDs {
		id, err := peer.Decode(peerID)
		if err != nil {
			Error("Failed to decode peer ID for %v: %v", n.nodeName, err)
			return nil, err
		}
		peers = append(peers, id)
	}

	Debug("Successfully fetched connected relay peers for pubsubTopic: %v, node: %v count: %v", pubsubTopic, n.nodeName, len(peers))
	return peers, nil
}

func (n *WakuNode) DisconnectPeerByID(peerID peer.ID) error {
	if err := ffi.WakuDisconnectPeerByID(n.wakuCtx, peerID.String()); err != nil {
		return fmt.Errorf("error DisconnectPeerById: %w", err)
	}
	return nil
}

func (n *WakuNode) DisconnectAllPeers() error {
	if err := ffi.WakuDisconnectAllPeers(n.wakuCtx); err != nil {
		return fmt.Errorf("error DisconnectAllPeers: %w", err)
	}
	return nil
}

func (n *WakuNode) GetConnectedPeers() (peer.IDSlice, error) {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to get connected peers %v", err)
		return nil, err
	}

	Debug("Fetching connected peers for %v", n.nodeName)

	peersStr, err := ffi.WakuGetConnectedPeers(n.wakuCtx)
	if err != nil {
		errMsg := "error GetConnectedPeers: " + err.Error()
		Error("Failed to get connected peers for %v: %v", n.nodeName, errMsg)
		return nil, errors.New(errMsg)
	}

	if peersStr == "" {
		Debug("No connected peers found for %v", n.nodeName)
		return nil, nil
	}

	peerIDs := strings.Split(peersStr, ",")
	var peers peer.IDSlice
	for _, peerID := range peerIDs {
		id, err := peer.Decode(peerID)
		if err != nil {
			Error("Failed to decode peer ID for %v: %v", n.nodeName, err)
			return nil, err
		}
		peers = append(peers, id)
	}

	Debug("Successfully fetched connected peers for %v, count: %v", n.nodeName, len(peers))
	return peers, nil
}

func (n *WakuNode) GetPeersInMesh(pubsubTopic string) (peer.IDSlice, error) {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to get peers in mesh: %v", err)
		return nil, err
	}

	Debug("Fetching peers in mesh peers for pubsubTopic: %v, node: %v", pubsubTopic, n.nodeName)

	peersStr, err := ffi.WakuGetPeersInMesh(n.wakuCtx, pubsubTopic)
	if err != nil {
		errMsg := "error GetPeersInMesh: " + err.Error()
		Error("Failed to get peers in mesh for pubsubTopic: %v:, node: %v. %v", pubsubTopic, n.nodeName, errMsg)
		return nil, errors.New(errMsg)
	}

	if peersStr == "" {
		Debug("No peers in mesh found for pubsubTopic: %v, node: %v", pubsubTopic, n.nodeName)
		return nil, nil
	}

	peerIDs := strings.Split(peersStr, ",")
	var peers peer.IDSlice
	for _, peerID := range peerIDs {
		id, err := peer.Decode(peerID)
		if err != nil {
			Error("Failed to decode peer ID for %v: %v", n.nodeName, err)
			return nil, err
		}
		peers = append(peers, id)
	}

	Debug("Successfully fetched mesh peers for pubsubTopic: %v, node: %v count: %v", pubsubTopic, n.nodeName, len(peers))
	return peers, nil
}

func (n *WakuNode) RelaySubscribe(pubsubTopic string) error {
	if pubsubTopic == "" {
		return errors.New("pubsub topic is empty")
	}

	if n.wakuCtx == nil {
		return errors.New("wakuCtx is nil")
	}

	if err := ffi.WakuRelaySubscribe(n.wakuCtx, pubsubTopic); err != nil {
		Error("Failed to subscribe to relay on node %s, pubsubTopic: %s, error: %v", n.nodeName, pubsubTopic, err)
		return fmt.Errorf("error WakuRelaySubscribe: %w", err)
	}

	Debug("Successfully subscribed to relay on node %s, pubsubTopic: %s", n.nodeName, pubsubTopic)
	return nil
}

func (n *WakuNode) RelayAddProtectedShard(clusterId uint16, shardId uint16, pubkey *ecdsa.PublicKey) error {
	if pubkey == nil {
		return errors.New("error WakuRelayAddProtectedShard: pubkey can't be nil")
	}

	if n.wakuCtx == nil {
		return errors.New("wakuCtx is nil")
	}

	keyHexStr := hex.EncodeToString(crypto.FromECDSAPub(pubkey))

	if err := ffi.WakuRelayAddProtectedShard(n.wakuCtx, int(clusterId), int(shardId), keyHexStr); err != nil {
		return fmt.Errorf("error WakuRelayAddProtectedShard: %w", err)
	}
	return nil
}

func (n *WakuNode) RelayUnsubscribe(pubsubTopic string) error {
	if pubsubTopic == "" {
		err := errors.New("pubsub topic is empty")
		Error("Failed to unsubscribe from relay: %v", err)
		return err
	}

	if n.wakuCtx == nil {
		return errors.New("wakuCtx is nil")
	}

	Debug("Attempting to unsubscribe from relay on node %s, pubsubTopic: %s", n.nodeName, pubsubTopic)
	if err := ffi.WakuRelayUnsubscribe(n.wakuCtx, pubsubTopic); err != nil {
		Error("Failed to unsubscribe from relay on node %s, pubsubTopic: %s, error: %v", n.nodeName, pubsubTopic, err)
		return fmt.Errorf("error WakuRelayUnsubscribe: %w", err)
	}

	Debug("Successfully unsubscribed from relay on node %s, pubsubTopic: %s", n.nodeName, pubsubTopic)
	return nil
}

func (n *WakuNode) PeerExchangeRequest(numPeers uint64) (uint64, error) {
	numRecvPeersStr, err := ffi.WakuPeerExchangeRequest(n.wakuCtx, numPeers)
	if err != nil {
		Error("PeerExchangeRequest failed: %v", err)
		return 0, err
	}

	numRecvPeers, err := strconv.ParseUint(numRecvPeersStr, 10, 64)
	if err != nil {
		Error("Failed to parse number of received peers: %v", err)
		return 0, err
	}
	return numRecvPeers, nil
}

func (n *WakuNode) StartDiscV5() error {

	Debug("Starting DiscV5 for node: %s", n.nodeName)
	if err := ffi.WakuStartDiscV5(n.wakuCtx); err != nil {
		errMsg := "error WakuStartDiscV5: " + err.Error()
		Error("Failed to start DiscV5 for node %s: %v", n.nodeName, errMsg)
		return errors.New(errMsg)
	}
	Debug("Successfully started DiscV5 for node: %s", n.nodeName)
	return nil
}

func (n *WakuNode) StopDiscV5() error {
	if err := ffi.WakuStopDiscV5(n.wakuCtx); err != nil {
		errMsg := "error WakuStopDiscV5: " + err.Error()
		Error("Failed to stop DiscV5 for node %s: %v", n.nodeName, errMsg)
		return errors.New(errMsg)
	}
	Debug("Successfully stopped DiscV5 for node: %s", n.nodeName)
	return nil
}

func (n *WakuNode) Version() (string, error) {
	version, err := ffi.WakuVersion(n.wakuCtx)
	if err != nil {
		errMsg := "error WakuVersion: " + err.Error()
		Error("Failed to fetch Waku version for node %s: %v", n.nodeName, errMsg)
		return "", errors.New(errMsg)
	}

	Debug("Successfully fetched Waku version for node %s: %s", n.nodeName, version)
	return version, nil
}

func (n *WakuNode) StoreQuery(ctx context.Context, storeRequest *common.StoreQueryRequest, peerInfo peer.AddrInfo) (*common.StoreQueryResponse, error) {
	timeoutMs := getContextTimeoutMilliseconds(ctx)

	b, err := json.Marshal(storeRequest)
	if err != nil {
		return nil, err
	}

	addrs := make([]string, len(peerInfo.Addrs))
	for i, addr := range utils.EncapsulatePeerID(peerInfo.ID, peerInfo.Addrs...) {
		addrs[i] = addr.String()
	}

	jsonResponseStr, err := ffi.WakuStoreQuery(n.wakuCtx, string(b), strings.Join(addrs, ","), timeoutMs)
	if err != nil {
		return nil, fmt.Errorf("error WakuStoreQuery: %w", err)
	}

	storeQueryResponse := common.StoreQueryResponse{}
	err = json.Unmarshal([]byte(jsonResponseStr), &storeQueryResponse)
	if err != nil {
		return nil, err
	}
	return &storeQueryResponse, nil
}

func (n *WakuNode) RelayPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string) (common.MessageHash, error) {
	timeoutMs := getContextTimeoutMilliseconds(ctx)

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return common.MessageHash(""), err
	}

	msgHash, err := ffi.WakuRelayPublish(n.wakuCtx, pubsubTopic, string(jsonMsg), timeoutMs)
	if err != nil {
		return common.MessageHash(""), fmt.Errorf("WakuRelayPublish: %w", err)
	}

	parsedMsgHash, err := common.ToMessageHash(msgHash)
	if err != nil {
		return common.MessageHash(""), err
	}
	return parsedMsgHash, nil
}

func (n *WakuNode) RelayPublishNoCTX(pubsubTopic string, message *pb.WakuMessage) (common.MessageHash, error) {
	if n == nil {
		err := errors.New("cannot publish message; node is nil")
		Error("Failed to publish message via relay: %v", err)
		return "", err
	}

	// Handling context internally with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	Debug("Attempting to publish message via relay on node %s", n.nodeName)

	msgHash, err := n.RelayPublish(ctx, message, pubsubTopic)
	if err != nil {
		Error("Failed to publish message via relay on node %s: %v", n.nodeName, err)
		return "", err
	}

	Debug("Successfully published message via relay on node %s, messageHash: %s", n.nodeName, msgHash.String())
	return msgHash, nil
}

func (n *WakuNode) DnsDiscovery(ctx context.Context, enrTreeUrl string, nameDnsServer string) ([]multiaddr.Multiaddr, error) {
	timeoutMs := getContextTimeoutMilliseconds(ctx)

	nodeAddresses, err := ffi.WakuDnsDiscovery(n.wakuCtx, enrTreeUrl, nameDnsServer, timeoutMs)
	if err != nil {
		return nil, fmt.Errorf("error WakuDnsDiscovery: %w", err)
	}

	var addrsRet []multiaddr.Multiaddr
	addrss := strings.Split(nodeAddresses, ",")
	for _, addr := range addrss {
		addr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		addrsRet = append(addrsRet, addr)
	}
	return addrsRet, nil
}

func (n *WakuNode) PingPeer(ctx context.Context, peerInfo peer.AddrInfo) (time.Duration, error) {
	addrs := make([]string, len(peerInfo.Addrs))
	for i, addr := range utils.EncapsulatePeerID(peerInfo.ID, peerInfo.Addrs...) {
		addrs[i] = addr.String()
	}

	timeoutMs := getContextTimeoutMilliseconds(ctx)

	rttStr, err := ffi.WakuPingPeer(n.wakuCtx, strings.Join(addrs, ","), timeoutMs)
	if err != nil {
		return 0, fmt.Errorf("PingPeer: %w", err)
	}

	rttInt, err := strconv.ParseInt(rttStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(rttInt), nil
}

func (n *WakuNode) Start() error {
	Debug("Starting %s", n.nodeName)

	if err := ffi.WakuStart(n.wakuCtx); err != nil {
		errMsg := "error WakuStart: " + err.Error()
		Error("Failed to start %s: %s", n.nodeName, errMsg)
		return errors.New(errMsg)
	}

	Debug("Successfully started %s", n.nodeName)
	return nil
}

func (n *WakuNode) Stop() error {

	Debug("Stopping %s", n.nodeName)
	if err := ffi.WakuStop(n.wakuCtx); err != nil {
		errMsg := "error WakuStop: " + err.Error()
		Error("Failed to stop %s: %s", n.nodeName, errMsg)
		return errors.New(errMsg)
	}

	Debug("Successfully stopped %s", n.nodeName)
	return nil
}

func (n *WakuNode) Destroy() error {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to destroy %v", err)
		return err
	}

	Debug("Destroying %v", n.nodeName)

	if err := ffi.WakuDestroy(n.wakuCtx); err != nil {
		errMsg := "error WakuDestroy: " + err.Error()
		Error("Failed to destroy %v: %v", n.nodeName, errMsg)
		return errors.New(errMsg)
	}

	Debug("Successfully destroyed %s", n.nodeName)
	return nil
}

func (n *WakuNode) PeerID() (peer.ID, error) {
	peerIdStr, err := ffi.WakuGetMyPeerID(n.wakuCtx)
	if err != nil {
		return "", err
	}

	id, err := peer.Decode(peerIdStr)
	if err != nil {
		errMsg := "WakuGetMyPeerId - decoding peerId: %w"
		return "", fmt.Errorf(errMsg, err)
	}
	return id, nil
}

func (n *WakuNode) Connect(ctx context.Context, addr multiaddr.Multiaddr) error {
	timeoutMs := getContextTimeoutMilliseconds(ctx)

	if err := ffi.WakuConnect(n.wakuCtx, addr.String(), timeoutMs); err != nil {
		return fmt.Errorf("error WakuConnect: %w", err)
	}
	return nil
}

func (n *WakuNode) DialPeerByID(ctx context.Context, peerID peer.ID, protocol libp2pproto.ID) error {
	timeoutMs := getContextTimeoutMilliseconds(ctx)

	if err := ffi.WakuDialPeerByID(n.wakuCtx, peerID.String(), string(protocol), timeoutMs); err != nil {
		return fmt.Errorf("error DialPeerById: %w", err)
	}
	return nil
}

func (n *WakuNode) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	listenAddresses, err := ffi.WakuListenAddresses(n.wakuCtx)
	if err != nil {
		return nil, fmt.Errorf("error WakuListenAddresses: %w", err)
	}

	var addrsRet []multiaddr.Multiaddr
	addrss := strings.Split(listenAddresses, ",")
	for _, addr := range addrss {
		addr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		addrsRet = append(addrsRet, addr)
	}
	return addrsRet, nil
}

func (n *WakuNode) ENR() (*enode.Node, error) {
	enrStr, err := ffi.WakuGetMyENR(n.wakuCtx)
	if err != nil {
		return nil, fmt.Errorf("error WakuGetMyENR: %w", err)
	}

	node, err := enode.Parse(enode.ValidSchemes, enrStr)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (n *WakuNode) GetNumPeersInMesh(pubsubTopic string) (int, error) {
	numPeersStr, err := ffi.WakuGetNumPeersInMesh(n.wakuCtx, pubsubTopic)
	if err != nil {
		return 0, fmt.Errorf("error GetNumPeersInMesh: %w", err)
	}

	numPeers, err := strconv.Atoi(numPeersStr)
	if err != nil {
		errMsg := "GetNumPeersInMesh - error converting string to int: " + err.Error()
		return 0, errors.New(errMsg)
	}
	return numPeers, nil
}

func (n *WakuNode) GetPeerIDsFromPeerStore() (peer.IDSlice, error) {
	peersStr, err := ffi.WakuGetPeerIDsFromPeerStore(n.wakuCtx)
	if err != nil {
		return nil, fmt.Errorf("GetPeerIdsFromPeerStore: %s", err.Error())
	}

	if peersStr == "" {
		return nil, nil
	}
	// peersStr contains a comma-separated list of peer ids
	itemsPeerIds := strings.Split(peersStr, ",")

	var peers peer.IDSlice
	for _, peerId := range itemsPeerIds {
		id, err := peer.Decode(peerId)
		if err != nil {
			return nil, fmt.Errorf("GetPeerIdsFromPeerStore - decoding peerId: %w", err)
		}
		peers = append(peers, id)
	}
	return peers, nil
}

func (n *WakuNode) GetConnectedPeersInfo() (common.PeersData, error) {
	jsonStr, err := ffi.WakuGetConnectedPeersInfo(n.wakuCtx)
	if err != nil {
		return nil, fmt.Errorf("GetConnectedPeersInfo: %s", err.Error())
	}

	if jsonStr == "" {
		return nil, nil
	}

	peerData, err := common.ParsePeerInfoFromJSON(jsonStr)

	if err != nil {
		return nil, fmt.Errorf("GetConnectedPeersInfo - failed parsing JSON: %w", err)
	}

	return peerData, nil
}

func (n *WakuNode) GetPeerIDsByProtocol(protocol libp2pproto.ID) (peer.IDSlice, error) {
	peersStr, err := ffi.WakuGetPeerIDsByProtocol(n.wakuCtx, string(protocol))
	if err != nil {
		return nil, fmt.Errorf("GetPeerIdsByProtocol: error GetPeerIdsByProtocol: %s", err.Error())
	}

	if peersStr == "" {
		return nil, nil
	}
	// peersStr contains a comma-separated list of peer ids
	itemsPeerIds := strings.Split(peersStr, ",")

	var peers peer.IDSlice
	for _, p := range itemsPeerIds {
		id, err := peer.Decode(p)
		if err != nil {
			return nil, fmt.Errorf("GetPeerIdsByProtocol - decoding peerId: %w", err)
		}
		peers = append(peers, id)
	}
	return peers, nil
}

func (n *WakuNode) DialPeer(ctx context.Context, peerAddr multiaddr.Multiaddr, protocol libp2pproto.ID) error {
	timeoutMs := getContextTimeoutMilliseconds(ctx)

	if err := ffi.WakuDialPeer(n.wakuCtx, peerAddr.String(), string(protocol), timeoutMs); err != nil {
		return fmt.Errorf("error DialPeer: %w", err)
	}
	return nil
}

func (n *WakuNode) GetNumConnectedPeers() (int, error) {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to get number of connected peers %v", err)
		return 0, err
	}

	Debug("Fetching number of connected peers for %v", n.nodeName)

	peers, err := n.GetConnectedPeers()
	if err != nil {
		Error("Failed to fetch connected peers for %v: %v ", n.nodeName, err)
		return 0, err
	}

	numPeers := len(peers)
	Debug("Successfully fetched number of connected peers for %v, count: %v", n.nodeName, numPeers)

	return numPeers, nil
}

func getContextTimeoutMilliseconds(ctx context.Context) int {
	deadline, ok := ctx.Deadline()
	if ok {
		return int(time.Until(deadline).Milliseconds())
	}
	return 0
}

func FormatWakuRelayTopic(clusterId uint16, shard uint16) string {
	return fmt.Sprintf("/waku/2/rs/%d/%d", clusterId, shard)
}

func GetFreePortIfNeeded(tcpPort int, discV5UDPPort int) (int, int, error) {
	if tcpPort == 0 {
		for i := 0; i < 10; i++ {
			tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("localhost", "0"))
			if err != nil {
				Warn("unable to resolve tcp addr: %v", err)
				continue
			}
			tcpListener, err := net.ListenTCP("tcp", tcpAddr)
			if err != nil {
				Warn("unable to listen on addr: addr=%v, error=%v", tcpAddr, err)

				continue
			}
			tcpPort = tcpListener.Addr().(*net.TCPAddr).Port
			tcpListener.Close()
			break
		}
		if tcpPort == 0 {
			return -1, -1, errors.New("could not obtain a free TCP port")
		}
	}

	if discV5UDPPort == 0 {
		for i := 0; i < 10; i++ {
			udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("localhost", "0"))
			if err != nil {
				Warn("unable to resolve udp addr: %v", err)
				continue
			}

			udpListener, err := net.ListenUDP("udp", udpAddr)
			if err != nil {
				Warn("unable to listen on addr: addr=%v, error=%v", udpAddr, err)

				continue
			}

			discV5UDPPort = udpListener.LocalAddr().(*net.UDPAddr).Port
			udpListener.Close()
			break
		}
		if discV5UDPPort == 0 {
			return -1, -1, errors.New("could not obtain a free UDP port")
		}
	}

	return tcpPort, discV5UDPPort, nil
}

// Create & start node
func StartWakuNode(nodeName string, customCfg *common.WakuConfig) (*WakuNode, error) {

	Debug("Initializing %s", nodeName)

	var nodeCfg common.WakuConfig
	if customCfg == nil {
		nodeCfg = DefaultWakuConfig
	} else {
		nodeCfg = *customCfg
	}

	tcpPort, udpPort, err := GetFreePortIfNeeded(nodeCfg.TcpPort, nodeCfg.Discv5UdpPort)
	if err != nil {
		Error("Failed to allocate unique ports: %v", err)
		tcpPort, udpPort = 0, 0
	}

	if nodeCfg.TcpPort == 0 {
		nodeCfg.TcpPort = tcpPort
	}
	if nodeCfg.Discv5UdpPort == 0 {
		nodeCfg.Discv5UdpPort = udpPort
	}

	Debug("Creating %s", nodeName)
	node, err := NewWakuNode(&nodeCfg, nodeName)
	if err != nil {
		Error("Failed to create %s: %v", nodeName, err)
		return nil, err
	}

	Debug("Starting %s", nodeName)
	if err := node.Start(); err != nil {
		Error("Failed to start %s: %v", nodeName, err)
		return nil, err
	}

	Debug("Successfully started %s", nodeName)
	return node, nil
}

func (n *WakuNode) StopAndDestroy() error {
	Debug("Stopping and destroying Node")
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to stop and destroy: %v", err)
		return err
	}

	Debug("Stopping %s", n.nodeName)

	err := n.Stop()
	if err != nil {
		Error("Failed to stop %s: %v", n.nodeName, err)
		return err
	}

	Debug("Destroying %s", n.nodeName)

	err = n.Destroy()
	if err != nil {
		Error("Failed to destroy %s: %v", n.nodeName, err)
		return err
	}

	Debug("Successfully stopped and destroyed %s", n.nodeName)
	return nil
}

func (n *WakuNode) ConnectPeer(targetNode *WakuNode) error {

	Debug("Connecting %s to %s", n.nodeName, targetNode.nodeName)

	targetPeerID, err := targetNode.PeerID()
	if err != nil {
		Error("Failed to get PeerID of target node %s: %v", targetNode.nodeName, err)
		return err
	}

	targetAddr, err := targetNode.ListenAddresses()
	if err != nil || len(targetAddr) == 0 {
		Error("Failed to get listen addresses for target node %s: %v", targetNode.nodeName, err)
		return errors.New("target node has no listen addresses")
	}

	Debug("Attempting connection to peer %s", targetPeerID.String())

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	err = n.Connect(ctx, targetAddr[0])
	if err != nil {
		Error("Failed to connect to peer %s: %v", targetPeerID.String(), err)
		return err
	}

	Debug("Successfully connected %s to %s", n.nodeName, targetNode.nodeName)
	return nil
}

func (n *WakuNode) DisconnectPeer(target *WakuNode) error {
	Debug("Disconnecting %s from %s", n.nodeName, target.nodeName)

	targetPeerID, err := target.PeerID()
	if err != nil {
		Error("Failed to get PeerID of target node %s: %v", target.nodeName, err)
		return err
	}

	err = n.DisconnectPeerByID(targetPeerID)
	if err != nil {
		Error("Failed to disconnect peer %s: %v", targetPeerID.String(), err)
		return err
	}

	Debug("Successfully disconnected %s from %s", n.nodeName, target.nodeName)
	return nil
}

func (n *WakuNode) IsOnline() (bool, error) {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to get online state %v", err)
		return false, err
	}

	Debug("Querying online state for %v", n.nodeName)

	onlineStr, err := ffi.WakuIsOnline(n.wakuCtx)
	if err != nil {
		errMsg := "error IsOnline: " + err.Error()
		Error("Failed to query online state for %v: %v", n.nodeName, errMsg)
		return false, errors.New(errMsg)
	}

	return onlineStr == "true", nil
}

func (n *WakuNode) GetMetrics() (string, error) {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to get metrics %v", err)
		return "", err
	}

	Debug("Querying metrics for %v", n.nodeName)

	metricsStr, err := ffi.WakuGetMetrics(n.wakuCtx)
	if err != nil {
		errMsg := "error GetMetrics: " + err.Error()
		Error("Failed to query metrics for %v: %v", n.nodeName, errMsg)
		return "", errors.New(errMsg)
	}

	if metricsStr == "" {
		errMsg := "received empty metrics response"
		Error(errMsg)
		return "", errors.New(errMsg)
	}

	return metricsStr, nil
}
