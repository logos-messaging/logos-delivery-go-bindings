// Package kernel is the Go wrapper over a logos-delivery node.
//
// Node owns the library context, and is the only place the underlying FFI
// handle lives. The kernel protocols are reached through facades taken from a
// Node, one per protocol:
//
//	node, err := kernel.New(kernel.Config{Preset: kernel.PresetLogosDev})
//	defer node.Close()
//	node.Start()
//
//	node.Relay().Subscribe(pubsubTopic)
//	node.Peers().Connect(ctx, addr)
//	resp, err := node.Store().Query(ctx, request, peerInfo)
//	id, err := node.Debug().PeerID()
//
// The grouping mirrors the C library's own modules: Relay, Store, Peers,
// DiscV5, PeerExchange, DNSDiscovery and Debug map onto
// kernel_api/protocols/relay_api.nim, store_api.nim, peer_manager_api.nim,
// discovery_api.nim and debug_node_api.nim. Node itself carries only the
// lifecycle.
//
// This package knows nothing of the Messaging API. That tier lives in
// pkg/messaging, whose MessagingClient drives a Node and exposes it, so both
// tiers run against one node.
package kernel
