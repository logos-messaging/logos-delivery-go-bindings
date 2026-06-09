// Package kernel is the low-level Go wrapper over the logos-delivery Kernel API
// (libwaku): relay, store, lightpush, filter, peer management, discovery.
//
// It mirrors go-waku's shape and predates the Messaging API. It is considered
// legacy: once logos-delivery#3851 consolidates libwaku and liblogosdelivery
// into a single tiered library, the kernel surface will be re-pointed at that
// library and exposed as accessors on the messaging Node rather than as a
// standalone package. New consumers should prefer the messaging package.
package kernel
