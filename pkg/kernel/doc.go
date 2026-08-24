// Package kernel is the Go wrapper over a logos-delivery node.
//
// Node owns the library context that both API tiers share, and is the only
// place the underlying FFI handle lives. The Kernel API (the waku_* tier of
// liblogosdelivery) is reached through protocol facades taken from a Node:
//
//	node, err := kernel.New(kernel.Config{Preset: kernel.PresetLogosDev})
//	defer node.Close()
//	node.Start()
//
//	node.Relay().Subscribe(pubsubTopic)
//	node.Peers().Connect(ctx, addr)
//	resp, err := node.Store().Query(ctx, request, peerInfo)
//
// The facade grouping mirrors the C library's own modules, so Relay, Store,
// Peers and Discovery map onto kernel_api/protocols/relay_api.nim,
// store_api.nim, peer_manager_api.nim and discovery_api.nim respectively.
// Node itself carries the lifecycle and the node's own identity and health.
//
// The Messaging API is a separate layer over the same node: see pkg/messaging,
// whose MessagingClient exposes its Node so both tiers drive one node.
package kernel
