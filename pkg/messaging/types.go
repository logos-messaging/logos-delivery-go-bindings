package messaging

import "github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel"

type (
	// ContentTopic names the application-level channel a message belongs to,
	// by convention "/<app>/<version>/<name>/<encoding>".
	ContentTopic = kernel.ContentTopic
	// RequestID correlates a send with the delivery events it produces.
	RequestID = kernel.RequestID
)
