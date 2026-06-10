// Package ffi groups the cgo bridges over the logos-delivery C libraries.
//
// Each C library gets its own subpackage (libwaku now; liblogosdelivery in a
// follow-up) so that a binary links exactly the libraries it imports — the
// two .so files carry overlapping symbols and must never be linked together
// (until logos-delivery#3851 consolidates them).
package ffi
