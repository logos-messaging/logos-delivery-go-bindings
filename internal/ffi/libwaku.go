// This file holds the low-level Kernel (waku_*) tier of the single
// liblogosdelivery library. The shared plumbing — Handle, RetOK, EventHandler,
// the node lifecycle, the reply callback and the call helper — lives in
// liblogosdelivery.go (same package).
package ffi

/*
#include <stdint.h>
#include <stdlib.h>

typedef void (*FFICallback)(int ret, const char* msg, size_t len, void* userData);
typedef int (*logosRawFn)(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);

int waku_start_discv5(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_stop_discv5(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_version(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_relay_publish(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_relay_subscribe(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_relay_add_protected_shard(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_relay_unsubscribe(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_connect(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_dial_peer(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_dial_peer_by_id(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_disconnect_peer_by_id(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_disconnect_all_peers(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_listen_addresses(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_get_my_enr(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_get_my_peerid(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_ping_peer(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_relay_get_peers_in_mesh(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_relay_get_num_peers_in_mesh(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_relay_get_num_connected_peers(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_relay_get_connected_peers(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_get_connected_peers(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_get_peerids_from_peerstore(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_get_connected_peers_info(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_store_query(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_peer_exchange_request(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_get_peerids_by_protocol(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_dns_discovery(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_is_online(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int waku_get_metrics(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
*/
import "C"

// StartDiscV5 starts DiscV5 peer discovery.
func StartDiscV5(h Handle) error {
	_, err := call(C.logosRawFn(C.waku_start_discv5), h, noArgs{})
	return err
}

// StopDiscV5 stops DiscV5 peer discovery.
func StopDiscV5(h Handle) error {
	_, err := call(C.logosRawFn(C.waku_stop_discv5), h, noArgs{})
	return err
}

// Version returns the library version string.
func Version(h Handle) (string, error) {
	return call(C.logosRawFn(C.waku_version), h, noArgs{})
}

// RelayPublish publishes a WakuMessage JSON on a pubsub topic and returns
// the message hash.
func RelayPublish(h Handle, pubsubTopic, messageJSON string, timeoutMs int) (string, error) {
	req := struct {
		PubSubTopic     string `cbor:"pubSubTopic"`
		JsonWakuMessage string `cbor:"jsonWakuMessage"`
		TimeoutMs       uint32 `cbor:"timeoutMs"`
	}{pubsubTopic, messageJSON, uint32(timeoutMs)}
	return call(C.logosRawFn(C.waku_relay_publish), h, req)
}

// RelaySubscribe subscribes the node to a pubsub topic.
func RelaySubscribe(h Handle, pubsubTopic string) error {
	req := struct {
		PubSubTopic string `cbor:"pubSubTopic"`
	}{pubsubTopic}
	_, err := call(C.logosRawFn(C.waku_relay_subscribe), h, req)
	return err
}

// RelayAddProtectedShard registers the hex-encoded public key allowed to
// sign messages on a protected shard.
func RelayAddProtectedShard(h Handle, clusterID, shardID int, publicKeyHex string) error {
	req := struct {
		ClusterId uint16 `cbor:"clusterId"`
		ShardId   uint16 `cbor:"shardId"`
		PublicKey string `cbor:"publicKey"`
	}{uint16(clusterID), uint16(shardID), publicKeyHex}
	_, err := call(C.logosRawFn(C.waku_relay_add_protected_shard), h, req)
	return err
}

// RelayUnsubscribe unsubscribes the node from a pubsub topic.
func RelayUnsubscribe(h Handle, pubsubTopic string) error {
	req := struct {
		PubSubTopic string `cbor:"pubSubTopic"`
	}{pubsubTopic}
	_, err := call(C.logosRawFn(C.waku_relay_unsubscribe), h, req)
	return err
}

// Connect dials a peer multiaddress.
func Connect(h Handle, peerMultiAddr string, timeoutMs int) error {
	req := struct {
		PeerMultiAddr string `cbor:"peerMultiAddr"`
		TimeoutMs     uint32 `cbor:"timeoutMs"`
	}{peerMultiAddr, uint32(timeoutMs)}
	_, err := call(C.logosRawFn(C.waku_connect), h, req)
	return err
}

// DialPeer dials a peer multiaddress over a specific protocol.
func DialPeer(h Handle, peerMultiAddr, protocol string, timeoutMs int) error {
	req := struct {
		PeerMultiAddr string `cbor:"peerMultiAddr"`
		Protocol      string `cbor:"protocol"`
		TimeoutMs     uint32 `cbor:"timeoutMs"`
	}{peerMultiAddr, protocol, uint32(timeoutMs)}
	_, err := call(C.logosRawFn(C.waku_dial_peer), h, req)
	return err
}

// DialPeerByID dials a known peer id over a specific protocol.
func DialPeerByID(h Handle, peerID, protocol string, timeoutMs int) error {
	req := struct {
		PeerId    string `cbor:"peerId"`
		Protocol  string `cbor:"protocol"`
		TimeoutMs uint32 `cbor:"timeoutMs"`
	}{peerID, protocol, uint32(timeoutMs)}
	_, err := call(C.logosRawFn(C.waku_dial_peer_by_id), h, req)
	return err
}

// DisconnectPeerByID drops the connection to a peer.
func DisconnectPeerByID(h Handle, peerID string) error {
	req := struct {
		PeerId string `cbor:"peerId"`
	}{peerID}
	_, err := call(C.logosRawFn(C.waku_disconnect_peer_by_id), h, req)
	return err
}

// DisconnectAllPeers drops all peer connections.
func DisconnectAllPeers(h Handle) error {
	_, err := call(C.logosRawFn(C.waku_disconnect_all_peers), h, noArgs{})
	return err
}

// ListenAddresses returns the node's listen multiaddresses as a
// comma-separated list.
func ListenAddresses(h Handle) (string, error) {
	return call(C.logosRawFn(C.waku_listen_addresses), h, noArgs{})
}

// GetMyENR returns the node's ENR record.
func GetMyENR(h Handle) (string, error) {
	return call(C.logosRawFn(C.waku_get_my_enr), h, noArgs{})
}

// GetMyPeerID returns the node's peer id.
func GetMyPeerID(h Handle) (string, error) {
	return call(C.logosRawFn(C.waku_get_my_peerid), h, noArgs{})
}

// PingPeer pings a peer (comma-separated multiaddresses) and returns the
// round-trip time in nanoseconds.
func PingPeer(h Handle, peerAddrs string, timeoutMs int) (string, error) {
	req := struct {
		PeerAddr  string `cbor:"peerAddr"`
		TimeoutMs uint32 `cbor:"timeoutMs"`
	}{peerAddrs, uint32(timeoutMs)}
	return call(C.logosRawFn(C.waku_ping_peer), h, req)
}

// GetPeersInMesh returns the relay mesh peer ids for a pubsub topic as a
// comma-separated list.
func GetPeersInMesh(h Handle, pubsubTopic string) (string, error) {
	req := struct {
		PubSubTopic string `cbor:"pubSubTopic"`
	}{pubsubTopic}
	return call(C.logosRawFn(C.waku_relay_get_peers_in_mesh), h, req)
}

// GetNumPeersInMesh returns the relay mesh peer count for a pubsub topic.
func GetNumPeersInMesh(h Handle, pubsubTopic string) (string, error) {
	req := struct {
		PubSubTopic string `cbor:"pubSubTopic"`
	}{pubsubTopic}
	return call(C.logosRawFn(C.waku_relay_get_num_peers_in_mesh), h, req)
}

// GetNumConnectedRelayPeers returns the connected relay peer count for a
// pubsub topic.
func GetNumConnectedRelayPeers(h Handle, pubsubTopic string) (string, error) {
	req := struct {
		PubSubTopic string `cbor:"pubSubTopic"`
	}{pubsubTopic}
	return call(C.logosRawFn(C.waku_relay_get_num_connected_peers), h, req)
}

// GetConnectedRelayPeers returns the connected relay peer ids for a pubsub
// topic as a comma-separated list.
func GetConnectedRelayPeers(h Handle, pubsubTopic string) (string, error) {
	req := struct {
		PubSubTopic string `cbor:"pubSubTopic"`
	}{pubsubTopic}
	return call(C.logosRawFn(C.waku_relay_get_connected_peers), h, req)
}

// GetConnectedPeers returns the connected peer ids as a comma-separated
// list.
func GetConnectedPeers(h Handle) (string, error) {
	return call(C.logosRawFn(C.waku_get_connected_peers), h, noArgs{})
}

// GetPeerIDsFromPeerStore returns the peer-store peer ids as a
// comma-separated list.
func GetPeerIDsFromPeerStore(h Handle) (string, error) {
	return call(C.logosRawFn(C.waku_get_peerids_from_peerstore), h, noArgs{})
}

// GetConnectedPeersInfo returns the connected peers' info as JSON.
func GetConnectedPeersInfo(h Handle) (string, error) {
	return call(C.logosRawFn(C.waku_get_connected_peers_info), h, noArgs{})
}

// StoreQuery runs a store query (JSON) against a peer (comma-separated
// multiaddresses) and returns the response JSON.
func StoreQuery(h Handle, queryJSON, peerAddrs string, timeoutMs int) (string, error) {
	req := struct {
		JsonQuery string `cbor:"jsonQuery"`
		PeerAddr  string `cbor:"peerAddr"`
		TimeoutMs int32  `cbor:"timeoutMs"`
	}{queryJSON, peerAddrs, int32(timeoutMs)}
	return call(C.logosRawFn(C.waku_store_query), h, req)
}

// PeerExchangeRequest asks peer exchange for numPeers peers and returns
// the number of received peers.
func PeerExchangeRequest(h Handle, numPeers uint64) (string, error) {
	req := struct {
		NumPeers uint64 `cbor:"numPeers"`
	}{numPeers}
	return call(C.logosRawFn(C.waku_peer_exchange_request), h, req)
}

// GetPeerIDsByProtocol returns the peer ids supporting a protocol as a
// comma-separated list.
func GetPeerIDsByProtocol(h Handle, protocol string) (string, error) {
	req := struct {
		Protocol string `cbor:"protocol"`
	}{protocol}
	return call(C.logosRawFn(C.waku_get_peerids_by_protocol), h, req)
}

// DnsDiscovery resolves an ENR tree URL via DNS discovery and returns the
// discovered multiaddresses as a comma-separated list.
func DnsDiscovery(h Handle, enrTreeURL, nameDNSServer string, timeoutMs int) (string, error) {
	req := struct {
		EnrTreeUrl    string `cbor:"enrTreeUrl"`
		NameDnsServer string `cbor:"nameDnsServer"`
		TimeoutMs     int32  `cbor:"timeoutMs"`
	}{enrTreeURL, nameDNSServer, int32(timeoutMs)}
	return call(C.logosRawFn(C.waku_dns_discovery), h, req)
}

// IsOnline reports the node's online state ("true"/"false").
func IsOnline(h Handle) (string, error) {
	return call(C.logosRawFn(C.waku_is_online), h, noArgs{})
}

// GetMetrics returns the node's metrics in Prometheus text format.
func GetMetrics(h Handle) (string, error) {
	return call(C.logosRawFn(C.waku_get_metrics), h, noArgs{})
}
