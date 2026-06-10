// Package ffi holds the cgo bridge over the logos-delivery C libraries: the
// synchronous request/callback plumbing, the global event callback, and the
// handle registry. It exposes Go-typed primitives so the public packages
// (e.g. messaging) stay pure Go.
//
// Currently holds the Kernel API bridge (libwaku); the Messaging API bindings
// (over liblogosdelivery) land here in a follow-up.
package ffi
