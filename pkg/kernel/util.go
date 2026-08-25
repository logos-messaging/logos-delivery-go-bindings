package kernel

import (
	"context"
	"fmt"
	"time"
)

// requestTimeout is the deadline applied to a context that carries none of its
// own. It has to be a real number: the library passes the value straight to
// chronos' withTimeout, where zero milliseconds expires immediately.
const requestTimeout = 30 * time.Second

// timeoutMillis renders a context's remaining time as the millisecond timeout
// the library expects, falling back to def when the context has no deadline.
func timeoutMillis(ctx context.Context, def time.Duration) int {
	deadline, ok := ctx.Deadline()
	if !ok {
		return int(def.Milliseconds())
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Milliseconds())
}

// FormatWakuRelayTopic renders the pubsub topic for a cluster's shard.
func FormatWakuRelayTopic(clusterID uint16, shard uint16) string {
	return fmt.Sprintf("/waku/2/rs/%d/%d", clusterID, shard)
}
