package messaging

// Mode selects how much of the stack a node runs.
type Mode string

const (
	// ModeCore runs a full node: relay plus the service protocols.
	ModeCore Mode = "Core"
	// ModeEdge runs a light client: lightpush, filter and store clients.
	ModeEdge Mode = "Edge"
)

// Network presets. A preset fixes the cluster id, sharding, entry nodes and
// RLN settings for a known network, so a Config usually needs nothing else.
const (
	PresetTWN        = "twn"         // The Waku Network
	PresetLogosDev   = "logos.dev"   // Logos Dev Network
	PresetLogosTest  = "logos.test"  // Logos Test Network
	PresetStatusProd = "status.prod" // Status Production Network
)

// Overrides is a bag of per-field configuration overrides. Keys are node
// configuration field names or their CLI switch equivalents ("clusterId" or
// "cluster-id"); the library rejects keys it does not recognise.
type Overrides map[string]any

// Config is a MessagingClient's node configuration. It marshals to the layered
// configuration JSON that logosdelivery_create_node consumes: a network
// preset plus, optionally, overrides for the messaging and reliable-channel
// layers.
//
// Every field is omitted when empty, and that matters: the library still
// accepts a legacy flat configuration blob, and it decides between the two
// shapes by looking for bare top-level keys. A Config that emitted zero values
// would be read as a flat blob and silently ignore the layered defaults, so do
// not add fields here without `omitempty`.
type Config struct {
	// Mode is the node role. Defaults to ModeCore when empty.
	Mode Mode `json:"mode,omitempty"`
	// Preset names the network to join, e.g. PresetLogosDev.
	Preset string `json:"preset,omitempty"`
	// MessagingOverrides overrides individual node configuration fields, e.g.
	// {"tcp-port": 60000, "listen-address": "0.0.0.0"}.
	MessagingOverrides Overrides `json:"messagingOverrides,omitempty"`
	// ChannelsOverrides overrides reliable-channel configuration fields.
	ChannelsOverrides Overrides `json:"channelsOverrides,omitempty"`
}
