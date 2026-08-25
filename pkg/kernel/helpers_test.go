package kernel

import (
	"testing"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
)

// requiresNode marks a test that starts a real node and talks to a network, so
// that `go test -short` runs only the tests needing neither.
func requiresNode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping: needs a live node")
	}
}

// StartWakuNode creates a node from a legacy flat configuration and starts it.
// A nil configuration uses DefaultWakuConfig.
func StartWakuNode(customCfg *common.WakuConfig) (*Node, error) {
	nodeCfg := DefaultWakuConfig
	if customCfg != nil {
		nodeCfg = *customCfg
	}

	node, err := NewFromWakuConfig(&nodeCfg)
	if err != nil {
		return nil, err
	}

	if err := node.Start(); err != nil {
		_ = node.Close()
		return nil, err
	}
	return node, nil
}
