// Package messaging is the high-level, idiomatic Go binding for the
// logos-delivery Messaging API (an opinionated layer over the kernel protocols
// that owns reliability, re-subscriptions, store-based catch-up and the
// Messaging event surface).
//
// It will expose a Node type (create/start/stop, send/subscribe/unsubscribe)
// and a unified Events channel, backed by cgo calls into liblogosdelivery via
// the internal/ffi package. Scaffolding only for now; the implementation lands
// in a follow-up.
package messaging
