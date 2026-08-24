package common

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
)

// wireBytes decodes a byte field as the library serialises it, which is not
// base64. A seq[byte] is rendered by Nim's std/json, whose default is an array
// of integers, so "hello" arrives as [104,101,108,108,111].
//
// Base64 strings and null decode too, so the day the library normalises its
// encodings this keeps working instead of silently dropping every payload.
type wireBytes []byte

func (b *wireBytes) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*b = nil
		return nil
	}

	if data[0] == '[' {
		// Not []byte: encoding/json reads that from a base64 string only.
		var nums []int
		if err := json.Unmarshal(data, &nums); err != nil {
			return err
		}
		out := make([]byte, len(nums))
		for i, n := range nums {
			out[i] = byte(n)
		}
		*b = out
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	*b = decoded
	return nil
}

// optHasField is the discriminant Nim's std/json emits for a results.Opt[T]:
// the variant object's two private fields, rather than the value or null.
const optHasField = `"oResultPrivate"`

// unwrapOpt reduces an Opt[T] document to the value it holds, reporting false
// when the Opt is empty. A document that is already the bare value passes
// through unchanged, so this keeps working if the library ever serialises Opt
// as the value itself.
func unwrapOpt(data []byte) (json.RawMessage, bool) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, false
	}
	if trimmed[0] != '{' || !bytes.Contains(trimmed, []byte(optHasField)) {
		return trimmed, true
	}

	var opt struct {
		Has   bool            `json:"oResultPrivate"`
		Value json.RawMessage `json:"vResultPrivate"`
	}
	if err := json.Unmarshal(trimmed, &opt); err != nil {
		return nil, false
	}
	if !opt.Has || len(opt.Value) == 0 {
		return nil, false
	}
	return opt.Value, true
}
