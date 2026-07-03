// Package ffi groups the cgo bridge over the logos-delivery C library.
//
// The single liblogosdelivery library exposes the full API — the unified node
// lifecycle, the Messaging API, Reliable Channels, and the low-level Kernel
// (waku_*) tier — so a single bridge subpackage, liblogosdelivery, serves both
// pkg/messaging and pkg/kernel.
package ffi
