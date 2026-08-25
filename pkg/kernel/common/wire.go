package common

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// The kernel library serialises Nim values with Nim's own JSON encoder, which
// renders two shapes that Go's encoding/json cannot read directly:
//
//   - results.Opt[T] becomes {"oResultPrivate":bool,"vResultPrivate":T} rather
//     than the bare value, with oResultPrivate false standing for "no value".
//   - seq[byte] becomes an array of integers rather than a base64 string.
//
// Both decoders below accept the plain encoding as well, so normalising the
// library's output later relaxes them rather than breaking them.

// optWrapper mirrors the private fields of Nim's results.Opt variant object.
type optWrapper struct {
	HasValue *bool           `json:"oResultPrivate"`
	Value    json.RawMessage `json:"vResultPrivate"`
}

// unwrapOpt returns the value inside an Opt[T] wrapper. Input that is not a
// wrapper is returned unchanged; a wrapper holding no value returns nil.
func unwrapOpt(data []byte) json.RawMessage {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return data
	}

	var wrapper optWrapper
	if err := json.Unmarshal(trimmed, &wrapper); err != nil || wrapper.HasValue == nil {
		return data
	}

	if !*wrapper.HasValue {
		return nil
	}
	return wrapper.Value
}

// unmarshalOpt decodes an Opt[T]-wrapped value into target, leaving target
// untouched when the wrapper carries no value.
func unmarshalOpt(data []byte, target any) error {
	inner := unwrapOpt(data)
	if inner == nil {
		return nil
	}
	return json.Unmarshal(inner, target)
}

// wireBytes decodes a byte string from either the integer array the library
// renders seq[byte] as, or the base64 string encoding/json emits.
type wireBytes []byte

func (b *wireBytes) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*b = nil
		return nil
	}

	if trimmed[0] != '[' {
		return json.Unmarshal(trimmed, (*[]byte)(b))
	}

	var numbers []int
	if err := json.Unmarshal(trimmed, &numbers); err != nil {
		return err
	}

	decoded := make([]byte, len(numbers))
	for i, n := range numbers {
		if n < 0 || n > 255 {
			return fmt.Errorf("byte out of range at index %d: %d", i, n)
		}
		decoded[i] = byte(n)
	}
	*b = decoded

	return nil
}
