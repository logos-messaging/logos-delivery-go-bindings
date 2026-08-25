package ffi

import "unsafe"

// Handle is a node context owned by the C library.
//
// It wraps the pointer in a defined type rather than aliasing unsafe.Pointer,
// because pkg/kernel exposes a node's Handle so the tiers built on top of one
// can reach it. An alias would put a plain unsafe.Pointer in that exported
// signature, which any consumer could take and dereference; a defined type in
// an internal package cannot be named outside this module, so the value is
// inert there.
type Handle struct {
	ctx unsafe.Pointer
}

// Valid reports whether the handle refers to a context. A zero Handle does
// not, and no entry point may be called with one.
func (h Handle) Valid() bool { return h.ctx != nil }
