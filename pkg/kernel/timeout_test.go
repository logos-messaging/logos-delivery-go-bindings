package kernel

import (
	"context"
	"testing"
	"time"
)

func TestTimeoutMillisFallsBackToRequestTimeout(t *testing.T) {
	if got, want := timeoutMillis(context.Background(), requestTimeout),
		int(requestTimeout.Milliseconds()); got != want {
		t.Errorf("timeoutMillis(Background) = %d, want %d", got, want)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if got := timeoutMillis(ctx, requestTimeout); got <= 0 || got > 60_000 {
		t.Errorf("timeoutMillis(1m) = %d, want (0, 60000]", got)
	}

	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpired()
	if got := timeoutMillis(expired, requestTimeout); got != 0 {
		t.Errorf("timeoutMillis(expired, requestTimeout) = %d, want 0", got)
	}
}
