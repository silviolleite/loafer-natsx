package producer

import (
	"context"

	"github.com/nats-io/nats.go"
)

// Publisher defines an interface for publishing messages with support for context and configurable options.
type Publisher interface {

	// Publish sends a message using the provided context, message, and configurable publish options.
	// Returns a *PublishResult with publish metadata (populated for JetStream, empty for Core NATS)
	// and an error if the publish failed.
	Publish(ctx context.Context, msg *nats.Msg, opts PublishOptions) (*PublishResult, error)
}

// Requester defines an interface for sending requests with a subject and data and receiving a response.
type Requester interface {

	// Request sends a request with the specified subject and data, waits for a response,
	// and returns a *Response containing the reply data and headers, or an error.
	Request(ctx context.Context, subject string, data []byte) (*Response, error)
}
