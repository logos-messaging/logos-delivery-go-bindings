package kernel

import "testing"

// requiresNode marks a test that starts a real node and talks to a network, so
// that `go test -short` runs only the tests needing neither.
func requiresNode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping: needs a live node")
	}
}
