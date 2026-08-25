// This file holds the low-level Kernel (waku_*) tier of the single
// liblogosdelivery library. The shared plumbing — Handle, RetOK, EventHandler,
// the node lifecycle, the reply callbacks and the await helper — lives in
// liblogosdelivery.go (same package).
//
// The kernel entry points follow the same two generated shapes as the Messaging
// ones: no-argument calls take a raw LogosDeliveryScalarRawFn, everything else
// takes a per-call reply callback plus a <Name>Req argument struct.
package ffi

/*
#include <liblogosdelivery.h>
#include <stdint.h>
#include <stdlib.h>

// logosScalarReply and logosReply are defined in liblogosdelivery.go; the
// kernel wrappers below reuse them.
extern void logosScalarReply(int callerRet, char* msg, size_t len, void* userData);
extern void logosReply(int errCode, char* reply, char* errMsg, void* userData);

static int cGoWakuStartDiscV5(void* ctx, uintptr_t ud) {
	return waku_start_discv5(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuStopDiscV5(void* ctx, uintptr_t ud) {
	return waku_stop_discv5(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuVersion(void* ctx, uintptr_t ud) {
	return waku_version(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuRelayPublish(void* ctx, const char* pubSubTopic, const char* jsonWakuMessage, uint32_t timeoutMs, uintptr_t ud) {
	WakuRelayPublishReq req;
	req.pubSubTopic = pubSubTopic;
	req.jsonWakuMessage = jsonWakuMessage;
	req.timeoutMs = timeoutMs;
	return waku_relay_publish(ctx, (LogosDeliveryWakuRelayPublishReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuRelaySubscribe(void* ctx, const char* pubSubTopic, uintptr_t ud) {
	WakuRelaySubscribeReq req;
	req.pubSubTopic = pubSubTopic;
	return waku_relay_subscribe(ctx, (LogosDeliveryWakuRelaySubscribeReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuRelayAddProtectedShard(void* ctx, uint16_t clusterId, uint16_t shardId, const char* publicKey, uintptr_t ud) {
	WakuRelayAddProtectedShardReq req;
	req.clusterId = clusterId;
	req.shardId = shardId;
	req.publicKey = publicKey;
	return waku_relay_add_protected_shard(ctx, (LogosDeliveryWakuRelayAddProtectedShardReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuRelayUnsubscribe(void* ctx, const char* pubSubTopic, uintptr_t ud) {
	WakuRelayUnsubscribeReq req;
	req.pubSubTopic = pubSubTopic;
	return waku_relay_unsubscribe(ctx, (LogosDeliveryWakuRelayUnsubscribeReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuConnect(void* ctx, const char* peerMultiAddr, uint32_t timeoutMs, uintptr_t ud) {
	WakuConnectReq req;
	req.peerMultiAddr = peerMultiAddr;
	req.timeoutMs = timeoutMs;
	return waku_connect(ctx, (LogosDeliveryWakuConnectReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuDialPeer(void* ctx, const char* peerMultiAddr, const char* protocol, uint32_t timeoutMs, uintptr_t ud) {
	WakuDialPeerReq req;
	req.peerMultiAddr = peerMultiAddr;
	req.protocol = protocol;
	req.timeoutMs = timeoutMs;
	return waku_dial_peer(ctx, (LogosDeliveryWakuDialPeerReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuDialPeerById(void* ctx, const char* peerId, const char* protocol, uint32_t timeoutMs, uintptr_t ud) {
	WakuDialPeerByIdReq req;
	req.peerId = peerId;
	req.protocol = protocol;
	req.timeoutMs = timeoutMs;
	return waku_dial_peer_by_id(ctx, (LogosDeliveryWakuDialPeerByIdReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuDisconnectPeerById(void* ctx, const char* peerId, uintptr_t ud) {
	WakuDisconnectPeerByIdReq req;
	req.peerId = peerId;
	return waku_disconnect_peer_by_id(ctx, (LogosDeliveryWakuDisconnectPeerByIdReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuDisconnectAllPeers(void* ctx, uintptr_t ud) {
	return waku_disconnect_all_peers(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuListenAddresses(void* ctx, uintptr_t ud) {
	return waku_listen_addresses(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuGetMyENR(void* ctx, uintptr_t ud) {
	return waku_get_my_enr(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuGetMyPeerId(void* ctx, uintptr_t ud) {
	return waku_get_my_peerid(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuPingPeer(void* ctx, const char* peerAddr, uint32_t timeoutMs, uintptr_t ud) {
	WakuPingPeerReq req;
	req.peerAddr = peerAddr;
	req.timeoutMs = timeoutMs;
	return waku_ping_peer(ctx, (LogosDeliveryWakuPingPeerReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuGetPeersInMesh(void* ctx, const char* pubSubTopic, uintptr_t ud) {
	WakuRelayGetPeersInMeshReq req;
	req.pubSubTopic = pubSubTopic;
	return waku_relay_get_peers_in_mesh(ctx, (LogosDeliveryWakuRelayGetPeersInMeshReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuGetNumPeersInMesh(void* ctx, const char* pubSubTopic, uintptr_t ud) {
	WakuRelayGetNumPeersInMeshReq req;
	req.pubSubTopic = pubSubTopic;
	return waku_relay_get_num_peers_in_mesh(ctx, (LogosDeliveryWakuRelayGetNumPeersInMeshReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuGetNumConnectedRelayPeers(void* ctx, const char* pubSubTopic, uintptr_t ud) {
	WakuRelayGetNumConnectedPeersReq req;
	req.pubSubTopic = pubSubTopic;
	return waku_relay_get_num_connected_peers(ctx, (LogosDeliveryWakuRelayGetNumConnectedPeersReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuGetConnectedRelayPeers(void* ctx, const char* pubSubTopic, uintptr_t ud) {
	WakuRelayGetConnectedPeersReq req;
	req.pubSubTopic = pubSubTopic;
	return waku_relay_get_connected_peers(ctx, (LogosDeliveryWakuRelayGetConnectedPeersReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuGetConnectedPeers(void* ctx, uintptr_t ud) {
	return waku_get_connected_peers(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuGetPeerIdsFromPeerStore(void* ctx, uintptr_t ud) {
	return waku_get_peerids_from_peerstore(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuGetConnectedPeersInfo(void* ctx, uintptr_t ud) {
	return waku_get_connected_peers_info(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuStoreQuery(void* ctx, const char* jsonQuery, const char* peerAddr, int32_t timeoutMs, uintptr_t ud) {
	WakuStoreQueryReq req;
	req.jsonQuery = jsonQuery;
	req.peerAddr = peerAddr;
	req.timeoutMs = timeoutMs;
	return waku_store_query(ctx, (LogosDeliveryWakuStoreQueryReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuPeerExchangeQuery(void* ctx, uint64_t numPeers, uintptr_t ud) {
	return waku_peer_exchange_request(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud, numPeers);
}
static int cGoWakuGetPeerIdsByProtocol(void* ctx, const char* protocol, uintptr_t ud) {
	WakuGetPeeridsByProtocolReq req;
	req.protocol = protocol;
	return waku_get_peerids_by_protocol(ctx, (LogosDeliveryWakuGetPeeridsByProtocolReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuDnsDiscovery(void* ctx, const char* enrTreeUrl, const char* nameDnsServer, int32_t timeoutMs, uintptr_t ud) {
	WakuDnsDiscoveryReq req;
	req.enrTreeUrl = enrTreeUrl;
	req.nameDnsServer = nameDnsServer;
	req.timeoutMs = timeoutMs;
	return waku_dns_discovery(ctx, (LogosDeliveryWakuDnsDiscoveryReplyFn) logosReply, (void*) ud, &req);
}
static int cGoWakuIsOnline(void* ctx, uintptr_t ud) {
	return waku_is_online(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
static int cGoWakuGetMetrics(void* ctx, uintptr_t ud) {
	return waku_get_metrics(ctx, (LogosDeliveryScalarRawFn) logosScalarReply, (void*) ud);
}
*/
import "C"

import "unsafe"

// StartDiscV5 starts DiscV5 peer discovery.
func StartDiscV5(h Handle) error {
	_, err := await(func(ud C.uintptr_t) C.int { return C.cGoWakuStartDiscV5(h.ctx, ud) })
	return err
}

// StopDiscV5 stops DiscV5 peer discovery.
func StopDiscV5(h Handle) error {
	_, err := await(func(ud C.uintptr_t) C.int { return C.cGoWakuStopDiscV5(h.ctx, ud) })
	return err
}

// Version returns the library version string.
func Version(h Handle) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuVersion(h.ctx, ud) })
}

// RelayPublish publishes a WakuMessage JSON on a pubsub topic and returns
// the message hash.
func RelayPublish(h Handle, pubsubTopic, messageJSON string, timeoutMs int) (string, error) {
	cTopic := C.CString(pubsubTopic)
	cMsg := C.CString(messageJSON)
	defer C.free(unsafe.Pointer(cTopic))
	defer C.free(unsafe.Pointer(cMsg))
	return await(func(ud C.uintptr_t) C.int {
		return C.cGoWakuRelayPublish(h.ctx, cTopic, cMsg, C.uint32_t(timeoutMs), ud)
	})
}

// RelaySubscribe subscribes the node to a pubsub topic.
func RelaySubscribe(h Handle, pubsubTopic string) error {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	_, err := await(func(ud C.uintptr_t) C.int { return C.cGoWakuRelaySubscribe(h.ctx, cTopic, ud) })
	return err
}

// RelayAddProtectedShard registers the hex-encoded public key allowed to
// sign messages on a protected shard.
func RelayAddProtectedShard(h Handle, clusterID, shardID int, publicKeyHex string) error {
	cPublicKey := C.CString(publicKeyHex)
	defer C.free(unsafe.Pointer(cPublicKey))
	_, err := await(func(ud C.uintptr_t) C.int {
		return C.cGoWakuRelayAddProtectedShard(h.ctx, C.uint16_t(clusterID), C.uint16_t(shardID), cPublicKey, ud)
	})
	return err
}

// RelayUnsubscribe unsubscribes the node from a pubsub topic.
func RelayUnsubscribe(h Handle, pubsubTopic string) error {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	_, err := await(func(ud C.uintptr_t) C.int { return C.cGoWakuRelayUnsubscribe(h.ctx, cTopic, ud) })
	return err
}

// Connect dials a peer multiaddress.
func Connect(h Handle, peerMultiAddr string, timeoutMs int) error {
	cAddr := C.CString(peerMultiAddr)
	defer C.free(unsafe.Pointer(cAddr))
	_, err := await(func(ud C.uintptr_t) C.int { return C.cGoWakuConnect(h.ctx, cAddr, C.uint32_t(timeoutMs), ud) })
	return err
}

// DialPeer dials a peer multiaddress over a specific protocol.
func DialPeer(h Handle, peerMultiAddr, protocol string, timeoutMs int) error {
	cAddr := C.CString(peerMultiAddr)
	cProtocol := C.CString(protocol)
	defer C.free(unsafe.Pointer(cAddr))
	defer C.free(unsafe.Pointer(cProtocol))
	_, err := await(func(ud C.uintptr_t) C.int {
		return C.cGoWakuDialPeer(h.ctx, cAddr, cProtocol, C.uint32_t(timeoutMs), ud)
	})
	return err
}

// DialPeerByID dials a known peer id over a specific protocol.
func DialPeerByID(h Handle, peerID, protocol string, timeoutMs int) error {
	cPeerID := C.CString(peerID)
	cProtocol := C.CString(protocol)
	defer C.free(unsafe.Pointer(cPeerID))
	defer C.free(unsafe.Pointer(cProtocol))
	_, err := await(func(ud C.uintptr_t) C.int {
		return C.cGoWakuDialPeerById(h.ctx, cPeerID, cProtocol, C.uint32_t(timeoutMs), ud)
	})
	return err
}

// DisconnectPeerByID drops the connection to a peer.
func DisconnectPeerByID(h Handle, peerID string) error {
	cPeerID := C.CString(peerID)
	defer C.free(unsafe.Pointer(cPeerID))
	_, err := await(func(ud C.uintptr_t) C.int { return C.cGoWakuDisconnectPeerById(h.ctx, cPeerID, ud) })
	return err
}

// DisconnectAllPeers drops all peer connections.
func DisconnectAllPeers(h Handle) error {
	_, err := await(func(ud C.uintptr_t) C.int { return C.cGoWakuDisconnectAllPeers(h.ctx, ud) })
	return err
}

// ListenAddresses returns the node's listen multiaddresses as a
// comma-separated list.
func ListenAddresses(h Handle) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuListenAddresses(h.ctx, ud) })
}

// GetMyENR returns the node's ENR record.
func GetMyENR(h Handle) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetMyENR(h.ctx, ud) })
}

// GetMyPeerID returns the node's peer id.
func GetMyPeerID(h Handle) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetMyPeerId(h.ctx, ud) })
}

// PingPeer pings a peer (comma-separated multiaddresses) and returns the
// round-trip time in nanoseconds.
func PingPeer(h Handle, peerAddrs string, timeoutMs int) (string, error) {
	cAddr := C.CString(peerAddrs)
	defer C.free(unsafe.Pointer(cAddr))
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuPingPeer(h.ctx, cAddr, C.uint32_t(timeoutMs), ud) })
}

// GetPeersInMesh returns the relay mesh peer ids for a pubsub topic as a
// comma-separated list.
func GetPeersInMesh(h Handle, pubsubTopic string) (string, error) {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetPeersInMesh(h.ctx, cTopic, ud) })
}

// GetNumPeersInMesh returns the relay mesh peer count for a pubsub topic.
func GetNumPeersInMesh(h Handle, pubsubTopic string) (string, error) {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetNumPeersInMesh(h.ctx, cTopic, ud) })
}

// GetNumConnectedRelayPeers returns the connected relay peer count for a
// pubsub topic.
func GetNumConnectedRelayPeers(h Handle, pubsubTopic string) (string, error) {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetNumConnectedRelayPeers(h.ctx, cTopic, ud) })
}

// GetConnectedRelayPeers returns the connected relay peer ids for a pubsub
// topic as a comma-separated list.
func GetConnectedRelayPeers(h Handle, pubsubTopic string) (string, error) {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetConnectedRelayPeers(h.ctx, cTopic, ud) })
}

// GetConnectedPeers returns the connected peer ids as a comma-separated
// list.
func GetConnectedPeers(h Handle) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetConnectedPeers(h.ctx, ud) })
}

// GetPeerIDsFromPeerStore returns the peer-store peer ids as a
// comma-separated list.
func GetPeerIDsFromPeerStore(h Handle) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetPeerIdsFromPeerStore(h.ctx, ud) })
}

// GetConnectedPeersInfo returns the connected peers' info as JSON.
func GetConnectedPeersInfo(h Handle) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetConnectedPeersInfo(h.ctx, ud) })
}

// StoreQuery runs a store query (JSON) against a peer (comma-separated
// multiaddresses) and returns the response JSON.
func StoreQuery(h Handle, queryJSON, peerAddrs string, timeoutMs int) (string, error) {
	cQuery := C.CString(queryJSON)
	cAddr := C.CString(peerAddrs)
	defer C.free(unsafe.Pointer(cQuery))
	defer C.free(unsafe.Pointer(cAddr))
	return await(func(ud C.uintptr_t) C.int {
		return C.cGoWakuStoreQuery(h.ctx, cQuery, cAddr, C.int32_t(timeoutMs), ud)
	})
}

// PeerExchangeRequest asks peer exchange for numPeers peers and returns
// the number of received peers.
func PeerExchangeRequest(h Handle, numPeers uint64) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuPeerExchangeQuery(h.ctx, C.uint64_t(numPeers), ud) })
}

// GetPeerIDsByProtocol returns the peer ids supporting a protocol as a
// comma-separated list.
func GetPeerIDsByProtocol(h Handle, protocol string) (string, error) {
	cProtocol := C.CString(protocol)
	defer C.free(unsafe.Pointer(cProtocol))
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetPeerIdsByProtocol(h.ctx, cProtocol, ud) })
}

// DnsDiscovery resolves an ENR tree URL via DNS discovery and returns the
// discovered multiaddresses as a comma-separated list.
func DnsDiscovery(h Handle, enrTreeURL, nameDNSServer string, timeoutMs int) (string, error) {
	cEnrTree := C.CString(enrTreeURL)
	cDNSServer := C.CString(nameDNSServer)
	defer C.free(unsafe.Pointer(cEnrTree))
	defer C.free(unsafe.Pointer(cDNSServer))
	return await(func(ud C.uintptr_t) C.int {
		return C.cGoWakuDnsDiscovery(h.ctx, cEnrTree, cDNSServer, C.int32_t(timeoutMs), ud)
	})
}

// IsOnline reports the node's online state ("true"/"false").
func IsOnline(h Handle) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuIsOnline(h.ctx, ud) })
}

// GetMetrics returns the node's metrics in Prometheus text format.
func GetMetrics(h Handle) (string, error) {
	return await(func(ud C.uintptr_t) C.int { return C.cGoWakuGetMetrics(h.ctx, ud) })
}
