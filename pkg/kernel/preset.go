package kernel

import (
	"encoding/json"
	"fmt"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
)

// PresetFields describes a network preset, keyed by logos-delivery's preset
// field names: "clusterId", "entryNodes", "storeNodes", "discv5BootstrapNodes",
// "maxMessageSize", "numShards", "shards" and so on. A "base" names a preset
// whose values fill the fields left out; a field that is set replaces the
// base's value, lists included. Without a base, "clusterId" is required.
type PresetFields map[string]any

// AddPreset adds the network preset name, or replaces it, so that a Config can
// select it by name. It applies to every node created afterwards in this
// process; nodes that already exist keep the preset they were created with.
// A registered preset takes precedence over a built-in one of the same name.
func AddPreset(name string, fields PresetFields) error {
	presetJSON, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("kernel: add preset %q: marshal: %w", name, err)
	}
	if err := ffi.AddPreset(name, string(presetJSON)); err != nil {
		return fmt.Errorf("kernel: add preset %q: %w", name, err)
	}
	return nil
}
