// Package ffi groups the cgo bridges over liblogosdelivery, the unified
// logos-delivery C library.
//
// Each ABI gets its own subpackage — libwaku for the legacy waku_* Kernel API,
// liblogosdelivery for the logosdelivery_* Messaging API. Since
// logos-delivery#3949 merged the two libraries into one, both bridges link the
// same liblogosdelivery and may coexist in a single binary; the split is kept
// purely so callers import only the ABI they use.
package ffi
