package kernel

import (
	"encoding/json"
	"fmt"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
)

// PresetFields is a complete network preset, keyed by the field names of
// logos-delivery's NetworkPresetConf: "clusterId", "maxMessageSize",
// "shardingConf", "entryNodes", "storeNodes" and so on. "maxMessageSize" and
// "shardingConf" are required; "shardingConf" is a list of shard ids for
// auto-sharding, or "static". A field left out is zero or empty.
type PresetFields map[string]any

// AddPreset adds the network preset name, or replaces one added under that
// name, so that a Config can select it by name. It applies to every node
// created afterwards in this process; nodes that already exist keep the preset
// they were created with. A built-in preset cannot be replaced.
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
