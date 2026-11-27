package messaging

import "github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel"

// The node configuration lives with the node itself, in pkg/kernel: a client
// and the kernel protocols configure one and the same node. These aliases keep
// it spellable from here.
type (
	// Mode selects how much of the stack a node runs.
	Mode = kernel.Mode
	// Overrides is a bag of per-field configuration overrides.
	Overrides = kernel.Overrides
	// Config is a node's configuration.
	Config = kernel.Config
	// PresetFields describes a network preset for AddPreset.
	PresetFields = kernel.PresetFields
)

const (
	// ModeCore runs a full node: relay plus the service protocols.
	ModeCore = kernel.ModeCore
	// ModeEdge runs a light client: lightpush, filter and store clients.
	ModeEdge = kernel.ModeEdge
)

// Network presets. A preset fixes the cluster id, sharding, entry nodes and
// RLN settings for a known network, so a Config usually needs nothing else.
const (
	PresetTWN           = kernel.PresetTWN
	PresetLogosDev      = kernel.PresetLogosDev
	PresetLogosTest     = kernel.PresetLogosTest
	PresetStatusProd    = kernel.PresetStatusProd
	PresetStatusStaging = kernel.PresetStatusStaging
)

// AddPreset adds the network preset name, or replaces it, for the nodes
// created afterwards in this process. See kernel.AddPreset.
func AddPreset(name string, fields PresetFields) error {
	return kernel.AddPreset(name, fields)
}
