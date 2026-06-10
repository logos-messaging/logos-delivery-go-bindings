// Package ffi holds the cgo bridge over the logos-delivery C libraries: the
// synchronous request/callback plumbing, the global event callback, and the
// handle registry. It exposes Go-typed primitives so the public packages
// (e.g. messaging) stay pure Go.
//
// Scaffolding only for now; the Messaging API bindings land here (over
// liblogosdelivery) in a follow-up.
package ffi
