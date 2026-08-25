package ffi

// Handle is a node context owned by the C library.
//
// The library hands out a token rather than an address, so the handle holds an
// integer; Go must never keep the token in a pointer. It is a defined type in
// an internal package, because pkg/kernel exposes a node's Handle so the tiers
// built on top of one can reach it, and such a value cannot be built or read
// outside this module.
type Handle struct {
	ctx uintptr
}

// Valid reports whether the handle refers to a context. A zero Handle does
// not, and no entry point may be called with one.
func (h Handle) Valid() bool { return h.ctx != 0 }
