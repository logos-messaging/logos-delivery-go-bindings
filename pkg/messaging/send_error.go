package messaging

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SendErrorKind tells why the node refused a message. A refused message gets
// no event.
type SendErrorKind string

const (
	// SendNotStarted: the node is not started.
	SendNotStarted SendErrorKind = "NotStarted"
	// SendNotConnected: the node has no connected or dialable peer.
	SendNotConnected SendErrorKind = "NotConnected"
	// SendNoShardMapsContentTopic: the content topic maps to no shard of the node.
	SendNoShardMapsContentTopic SendErrorKind = "NoShardMapsContentTopic"
	// SendQueueFull: the node has its maximum of sends with no last event.
	SendQueueFull SendErrorKind = "QueueFull"
	// SendRateLimited: the rate-limit budget of the epoch is spent. RetryAfter
	// says when the next epoch starts.
	SendRateLimited SendErrorKind = "RateLimited"
	// SendTooLarge: the message, with its RLN proof, exceeds the maximum
	// message size.
	SendTooLarge SendErrorKind = "TooLarge"
)

// SendError is a message the node refused.
type SendError struct {
	Kind SendErrorKind
	// Message is the node's description of the refusal.
	Message string
	// RetryAfter is, for SendRateLimited, the time until the next epoch.
	RetryAfter time.Duration

	err error
}

func (e *SendError) Error() string {
	return fmt.Sprintf("messaging: send refused (%s): %s", e.Kind, e.Message)
}

// Unwrap returns the error the library answered with.
func (e *SendError) Unwrap() error { return e.err }

const sendRefusalPrefix = "Send failed: "

var (
	refusalKind = regexp.MustCompile(` \(([A-Za-z]+)\)$`)
	retryAfter  = regexp.MustCompile(`retry after: (\d+) s$`)
)

// sendError reads a refusal from the library's answer, which is
// "Send failed: <description> (<kind>)".
func sendError(err error) error {
	text := strings.TrimSpace(err.Error())
	kind := refusalKind.FindStringSubmatchIndex(text)
	if !strings.HasPrefix(text, sendRefusalPrefix) || kind == nil {
		return fmt.Errorf("messaging: send: %w", err)
	}
	se := &SendError{
		Kind:    SendErrorKind(text[kind[2]:kind[3]]),
		Message: text[len(sendRefusalPrefix):kind[0]],
		err:     err,
	}
	if m := retryAfter.FindStringSubmatch(se.Message); m != nil {
		if s, convErr := strconv.Atoi(m[1]); convErr == nil {
			se.RetryAfter = time.Duration(s) * time.Second
		}
	}
	return se
}
