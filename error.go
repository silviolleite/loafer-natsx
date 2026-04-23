package loafernatsx

import "time"

const (
	// ErrUnsupportedType indicates an error when an unsupported router type is encountered.
	ErrUnsupportedType = Err("unsupported router type")

	// ErrMissingURL indicates that a connection URL is required but was not provided.
	ErrMissingURL = Err("connection URL is required")

	// ErrMissingSubject indicates an error when the required subject is not provided.
	ErrMissingSubject = Err("subject is required")

	// ErrMissingQueueGroup indicates an error when a queue group is required but not provided for the router.
	ErrMissingQueueGroup = Err("queue group is required for the router")

	// ErrMissingStream indicates an error when a stream is required but not provided for a jetstream router.
	ErrMissingStream = Err("stream is required for jetstream router")

	// ErrMissingDurable indicates an error when a durable name is required but not provided for a jetstream router.
	ErrMissingDurable = Err("durable name is required for jetstream router")

	// ErrNilRoute indicates that the provided route instance is nil, which is invalid for route registration.
	ErrNilRoute = Err("route cannot be nil")

	// ErrNilHandler indicates that the provided handler instance is nil, which is invalid for route registration.
	ErrNilHandler = Err("handler cannot be nil")

	// ErrNoRoutes indicates that no routes were provided when attempting to configure or run the broker.
	ErrNoRoutes = Err("no routes provided")

	// ErrNilRouteRegistration indicates that a route registration provided to the broker is nil, which is not allowed.
	ErrNilRouteRegistration = Err("route registration cannot be nil")

	// ErrRequestNotSupported indicates that request-reply routes are not supported for JetStream producers.
	ErrRequestNotSupported = Err("request-reply routes are not supported for JetStream producers")

	// ErrRequestTimeout indicates that a request-reply operation exceeded its deadline.
	ErrRequestTimeout = Err("request timeout: consumer did not reply in time")

	// ErrPermanentFailure indicates that the handler failed with a permanent (non-retryable) error.
	// When a handler returns an error wrapping ErrPermanentFailure, the message is acknowledged
	// (Ack) to prevent further redelivery attempts. Use this when the failure cannot be fixed
	// by retrying (e.g. malformed payload, business rule violation, missing precondition).
	ErrPermanentFailure = Err("permanent failure: message acknowledged without retry")

	// ErrSendToDLQ indicates that the handler explicitly requests the message to be routed
	// directly to the Dead Letter Queue, bypassing the normal retry flow. This only has effect
	// on JetStream routes with DLQ enabled; on other route types it behaves like a regular error.
	ErrSendToDLQ = Err("send to dead letter queue: skip retries")
)

// Err represents an error as a string type and implements the error interface.
type Err string

// Error returns an error message as a string
func (e Err) Error() string {
	return string(e)
}

// NakWithDelayError instructs the JetStream consumer to negatively acknowledge
// the message and request redelivery after the specified delay. This enables
// progressive backoff strategies without exposing the underlying jetstream.Msg
// to the handler.
//
// Usage:
//
//	return nil, loafernatsx.NakWithDelayError{Delay: 30 * time.Second}
//	return nil, fmt.Errorf("context: %w", loafernatsx.NakWithDelayError{Delay: 5 * time.Minute})
//
// On non-JetStream routes this error is treated as a regular error (logged,
// no special ack behavior).
type NakWithDelayError struct {
	// Delay is the duration the server should wait before redelivering the message.
	Delay time.Duration
}

// Error implements the error interface.
func (e NakWithDelayError) Error() string {
	return "nak with delay: redelivery in " + e.Delay.String()
}
