// This file holds the low-level Kernel (waku_*) tier of the single
// liblogosdelivery library (declared in liblogosdelivery_kernel.h). The shared
// plumbing — Handle, RetOK, EventHandler, the node lifecycle, the response/event
// callbacks and the call helper — lives in liblogosdelivery.go (same package).
package ffi

/*
#include <liblogosdelivery_kernel.h>
#include <stdlib.h>

// logosGoCallback (the synchronous response callback) is defined in
// liblogosdelivery.go; the kernel wrappers below reuse it.
extern void logosGoCallback(int ret, char* msg, size_t len, void* resp);
static void cGoWakuStartDiscV5(void* ctx, void* resp) {
	waku_start_discv5(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuStopDiscV5(void* ctx, void* resp) {
	waku_stop_discv5(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuVersion(void* ctx, void* resp) {
	waku_version(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuRelayPublish(void* ctx, const char* pubSubTopic, const char* jsonWakuMessage, int timeoutMs, void* resp) {
	waku_relay_publish(ctx, (FFICallBack) logosGoCallback, resp, pubSubTopic, jsonWakuMessage, timeoutMs);
}
static void cGoWakuRelaySubscribe(void* ctx, const char* pubSubTopic, void* resp) {
	waku_relay_subscribe(ctx, (FFICallBack) logosGoCallback, resp, pubSubTopic);
}
static void cGoWakuRelayAddProtectedShard(void* ctx, int clusterId, int shardId, char* publicKey, void* resp) {
	waku_relay_add_protected_shard(ctx, (FFICallBack) logosGoCallback, resp, clusterId, shardId, publicKey);
}
static void cGoWakuRelayUnsubscribe(void* ctx, const char* pubSubTopic, void* resp) {
	waku_relay_unsubscribe(ctx, (FFICallBack) logosGoCallback, resp, pubSubTopic);
}
static void cGoWakuConnect(void* ctx, const char* peerMultiAddr, int timeoutMs, void* resp) {
	waku_connect(ctx, (FFICallBack) logosGoCallback, resp, peerMultiAddr, timeoutMs);
}
static void cGoWakuDialPeer(void* ctx, const char* peerMultiAddr, const char* protocol, int timeoutMs, void* resp) {
	waku_dial_peer(ctx, (FFICallBack) logosGoCallback, resp, peerMultiAddr, protocol, timeoutMs);
}
static void cGoWakuDialPeerById(void* ctx, const char* peerId, const char* protocol, int timeoutMs, void* resp) {
	waku_dial_peer_by_id(ctx, (FFICallBack) logosGoCallback, resp, peerId, protocol, timeoutMs);
}
static void cGoWakuDisconnectPeerById(void* ctx, const char* peerId, void* resp) {
	waku_disconnect_peer_by_id(ctx, (FFICallBack) logosGoCallback, resp, peerId);
}
static void cGoWakuDisconnectAllPeers(void* ctx, void* resp) {
	waku_disconnect_all_peers(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuListenAddresses(void* ctx, void* resp) {
	waku_listen_addresses(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuGetMyENR(void* ctx, void* resp) {
	waku_get_my_enr(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuGetMyPeerId(void* ctx, void* resp) {
	waku_get_my_peerid(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuPingPeer(void* ctx, const char* peerAddr, int timeoutMs, void* resp) {
	waku_ping_peer(ctx, (FFICallBack) logosGoCallback, resp, peerAddr, timeoutMs);
}
static void cGoWakuGetPeersInMesh(void* ctx, const char* pubSubTopic, void* resp) {
	waku_relay_get_peers_in_mesh(ctx, (FFICallBack) logosGoCallback, resp, pubSubTopic);
}
static void cGoWakuGetNumPeersInMesh(void* ctx, const char* pubSubTopic, void* resp) {
	waku_relay_get_num_peers_in_mesh(ctx, (FFICallBack) logosGoCallback, resp, pubSubTopic);
}
static void cGoWakuGetNumConnectedRelayPeers(void* ctx, const char* pubSubTopic, void* resp) {
	waku_relay_get_num_connected_peers(ctx, (FFICallBack) logosGoCallback, resp, pubSubTopic);
}
static void cGoWakuGetConnectedRelayPeers(void* ctx, const char* pubSubTopic, void* resp) {
	waku_relay_get_connected_peers(ctx, (FFICallBack) logosGoCallback, resp, pubSubTopic);
}
static void cGoWakuGetConnectedPeers(void* ctx, void* resp) {
	waku_get_connected_peers(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuGetPeerIdsFromPeerStore(void* ctx, void* resp) {
	waku_get_peerids_from_peerstore(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuGetConnectedPeersInfo(void* ctx, void* resp) {
	waku_get_connected_peers_info(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuStoreQuery(void* ctx, const char* jsonQuery, const char* peerAddr, int timeoutMs, void* resp) {
	waku_store_query(ctx, (FFICallBack) logosGoCallback, resp, jsonQuery, peerAddr, timeoutMs);
}
static void cGoWakuPeerExchangeQuery(void* ctx, uint64_t numPeers, void* resp) {
	waku_peer_exchange_request(ctx, (FFICallBack) logosGoCallback, resp, numPeers);
}
static void cGoWakuGetPeerIdsByProtocol(void* ctx, const char* protocol, void* resp) {
	waku_get_peerids_by_protocol(ctx, (FFICallBack) logosGoCallback, resp, protocol);
}
static void cGoWakuDnsDiscovery(void* ctx, const char* entTreeUrl, const char* nameDnsServer, int timeoutMs, void* resp) {
	waku_dns_discovery(ctx, (FFICallBack) logosGoCallback, resp, entTreeUrl, nameDnsServer, timeoutMs);
}
static void cGoWakuIsOnline(void* ctx, void* resp) {
	waku_is_online(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoWakuGetMetrics(void* ctx, void* resp) {
	waku_get_metrics(ctx, (FFICallBack) logosGoCallback, resp);
}
*/
import "C"

import "unsafe"

// StartDiscV5 starts DiscV5 peer discovery.
func StartDiscV5(h Handle) error {
	_, err := call(func(resp unsafe.Pointer) { C.cGoWakuStartDiscV5(h, resp) })
	return err
}

// StopDiscV5 stops DiscV5 peer discovery.
func StopDiscV5(h Handle) error {
	_, err := call(func(resp unsafe.Pointer) { C.cGoWakuStopDiscV5(h, resp) })
	return err
}

// Version returns the libwaku version string.
func Version(h Handle) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuVersion(h, resp) })
}

// RelayPublish publishes a WakuMessage JSON on a pubsub topic and returns
// the message hash.
func RelayPublish(h Handle, pubsubTopic, messageJSON string, timeoutMs int) (string, error) {
	cTopic := C.CString(pubsubTopic)
	cMsg := C.CString(messageJSON)
	defer C.free(unsafe.Pointer(cTopic))
	defer C.free(unsafe.Pointer(cMsg))
	return call(func(resp unsafe.Pointer) { C.cGoWakuRelayPublish(h, cTopic, cMsg, C.int(timeoutMs), resp) })
}

// RelaySubscribe subscribes the node to a pubsub topic.
func RelaySubscribe(h Handle, pubsubTopic string) error {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	_, err := call(func(resp unsafe.Pointer) { C.cGoWakuRelaySubscribe(h, cTopic, resp) })
	return err
}

// RelayAddProtectedShard registers the hex-encoded public key allowed to
// sign messages on a protected shard.
func RelayAddProtectedShard(h Handle, clusterID, shardID int, publicKeyHex string) error {
	cPublicKey := C.CString(publicKeyHex)
	defer C.free(unsafe.Pointer(cPublicKey))
	_, err := call(func(resp unsafe.Pointer) {
		C.cGoWakuRelayAddProtectedShard(h, C.int(clusterID), C.int(shardID), cPublicKey, resp)
	})
	return err
}

// RelayUnsubscribe unsubscribes the node from a pubsub topic.
func RelayUnsubscribe(h Handle, pubsubTopic string) error {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	_, err := call(func(resp unsafe.Pointer) { C.cGoWakuRelayUnsubscribe(h, cTopic, resp) })
	return err
}

// Connect dials a peer multiaddress.
func Connect(h Handle, peerMultiAddr string, timeoutMs int) error {
	cAddr := C.CString(peerMultiAddr)
	defer C.free(unsafe.Pointer(cAddr))
	_, err := call(func(resp unsafe.Pointer) { C.cGoWakuConnect(h, cAddr, C.int(timeoutMs), resp) })
	return err
}

// DialPeer dials a peer multiaddress over a specific protocol.
func DialPeer(h Handle, peerMultiAddr, protocol string, timeoutMs int) error {
	cAddr := C.CString(peerMultiAddr)
	cProtocol := C.CString(protocol)
	defer C.free(unsafe.Pointer(cAddr))
	defer C.free(unsafe.Pointer(cProtocol))
	_, err := call(func(resp unsafe.Pointer) { C.cGoWakuDialPeer(h, cAddr, cProtocol, C.int(timeoutMs), resp) })
	return err
}

// DialPeerByID dials a known peer id over a specific protocol.
func DialPeerByID(h Handle, peerID, protocol string, timeoutMs int) error {
	cPeerID := C.CString(peerID)
	cProtocol := C.CString(protocol)
	defer C.free(unsafe.Pointer(cPeerID))
	defer C.free(unsafe.Pointer(cProtocol))
	_, err := call(func(resp unsafe.Pointer) { C.cGoWakuDialPeerById(h, cPeerID, cProtocol, C.int(timeoutMs), resp) })
	return err
}

// DisconnectPeerByID drops the connection to a peer.
func DisconnectPeerByID(h Handle, peerID string) error {
	cPeerID := C.CString(peerID)
	defer C.free(unsafe.Pointer(cPeerID))
	_, err := call(func(resp unsafe.Pointer) { C.cGoWakuDisconnectPeerById(h, cPeerID, resp) })
	return err
}

// DisconnectAllPeers drops all peer connections.
func DisconnectAllPeers(h Handle) error {
	_, err := call(func(resp unsafe.Pointer) { C.cGoWakuDisconnectAllPeers(h, resp) })
	return err
}

// ListenAddresses returns the node's listen multiaddresses as a
// comma-separated list.
func ListenAddresses(h Handle) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuListenAddresses(h, resp) })
}

// GetMyENR returns the node's ENR record.
func GetMyENR(h Handle) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetMyENR(h, resp) })
}

// GetMyPeerID returns the node's peer id.
func GetMyPeerID(h Handle) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetMyPeerId(h, resp) })
}

// PingPeer pings a peer (comma-separated multiaddresses) and returns the
// round-trip time in nanoseconds.
func PingPeer(h Handle, peerAddrs string, timeoutMs int) (string, error) {
	cAddr := C.CString(peerAddrs)
	defer C.free(unsafe.Pointer(cAddr))
	return call(func(resp unsafe.Pointer) { C.cGoWakuPingPeer(h, cAddr, C.int(timeoutMs), resp) })
}

// GetPeersInMesh returns the relay mesh peer ids for a pubsub topic as a
// comma-separated list.
func GetPeersInMesh(h Handle, pubsubTopic string) (string, error) {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetPeersInMesh(h, cTopic, resp) })
}

// GetNumPeersInMesh returns the relay mesh peer count for a pubsub topic.
func GetNumPeersInMesh(h Handle, pubsubTopic string) (string, error) {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetNumPeersInMesh(h, cTopic, resp) })
}

// GetNumConnectedRelayPeers returns the connected relay peer count for a
// pubsub topic.
func GetNumConnectedRelayPeers(h Handle, pubsubTopic string) (string, error) {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetNumConnectedRelayPeers(h, cTopic, resp) })
}

// GetConnectedRelayPeers returns the connected relay peer ids for a pubsub
// topic as a comma-separated list.
func GetConnectedRelayPeers(h Handle, pubsubTopic string) (string, error) {
	cTopic := C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cTopic))
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetConnectedRelayPeers(h, cTopic, resp) })
}

// GetConnectedPeers returns the connected peer ids as a comma-separated
// list.
func GetConnectedPeers(h Handle) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetConnectedPeers(h, resp) })
}

// GetPeerIDsFromPeerStore returns the peer-store peer ids as a
// comma-separated list.
func GetPeerIDsFromPeerStore(h Handle) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetPeerIdsFromPeerStore(h, resp) })
}

// GetConnectedPeersInfo returns the connected peers' info as JSON.
func GetConnectedPeersInfo(h Handle) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetConnectedPeersInfo(h, resp) })
}

// StoreQuery runs a store query (JSON) against a peer (comma-separated
// multiaddresses) and returns the response JSON.
func StoreQuery(h Handle, queryJSON, peerAddrs string, timeoutMs int) (string, error) {
	cQuery := C.CString(queryJSON)
	cAddr := C.CString(peerAddrs)
	defer C.free(unsafe.Pointer(cQuery))
	defer C.free(unsafe.Pointer(cAddr))
	return call(func(resp unsafe.Pointer) { C.cGoWakuStoreQuery(h, cQuery, cAddr, C.int(timeoutMs), resp) })
}

// PeerExchangeRequest asks peer exchange for numPeers peers and returns
// the number of received peers.
func PeerExchangeRequest(h Handle, numPeers uint64) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuPeerExchangeQuery(h, C.uint64_t(numPeers), resp) })
}

// GetPeerIDsByProtocol returns the peer ids supporting a protocol as a
// comma-separated list.
func GetPeerIDsByProtocol(h Handle, protocol string) (string, error) {
	cProtocol := C.CString(protocol)
	defer C.free(unsafe.Pointer(cProtocol))
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetPeerIdsByProtocol(h, cProtocol, resp) })
}

// DnsDiscovery resolves an ENR tree URL via DNS discovery and returns the
// discovered multiaddresses as a comma-separated list.
func DnsDiscovery(h Handle, enrTreeURL, nameDNSServer string, timeoutMs int) (string, error) {
	cEnrTree := C.CString(enrTreeURL)
	cDNSServer := C.CString(nameDNSServer)
	defer C.free(unsafe.Pointer(cEnrTree))
	defer C.free(unsafe.Pointer(cDNSServer))
	return call(func(resp unsafe.Pointer) { C.cGoWakuDnsDiscovery(h, cEnrTree, cDNSServer, C.int(timeoutMs), resp) })
}

// IsOnline reports the node's online state ("true"/"false").
func IsOnline(h Handle) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuIsOnline(h, resp) })
}

// GetMetrics returns the node's metrics in Prometheus text format.
func GetMetrics(h Handle) (string, error) {
	return call(func(resp unsafe.Pointer) { C.cGoWakuGetMetrics(h, resp) })
}
