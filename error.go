package loafernatsx

const (
	// ErrUnsupportedType indicates an error when an unsupported router type is encountered.
	ErrUnsupportedType = Err("unsupported router type")

	// ErrMissingURL indicates that a connect URL is required but was not provided.
	ErrMissingURL = Err("connect URL is required")

	// ErrMissingSubject indicates an error when the required subject is not provided.
	ErrMissingSubject = Err("subject is required")

	// ErrMissingQueueGroup indicates an error when a queue group is required but not provided for a queue router.
	ErrMissingQueueGroup = Err("queue group is required for queue router")

	// ErrMissingStream indicates an error when a stream is required but not provided for a jetstream router.
	ErrMissingStream = Err("stream is required for jetstream router")

	// ErrMissingDurable indicates an error when a durable name is required but not provided for a jetstream router.
	ErrMissingDurable = Err("durable name is required for jetstream router")

	// ErrMissingName indicates an error when a stream name is required but not provided.
	ErrMissingName = Err("stream name is required")

	// ErrMissingSubjects indicates that at least one subject must be specified.
	ErrMissingSubjects = Err("at least one subject is required")

	// ErrNilJetStream indicates that a JetStream instance is required but was not provided.
	ErrNilJetStream = Err("jetstream instance is required")

	// ErrNilRoute indicates that the provided route instance is nil, which is invalid for route registration.
	ErrNilRoute = Err("route cannot be nil")

	// ErrNilHandler indicates that the provided handler instance is nil, which is invalid for route registration.
	ErrNilHandler = Err("handler cannot be nil")

	// ErrNoRoutes indicates that no routes were provided when attempting to configure or run the broker.
	ErrNoRoutes = Err("no routes provided")

	// ErrNilRouteRegistration indicates that a route registration provided to the broker is nil, which is not allowed.
	ErrNilRouteRegistration = Err("route registration cannot be nil")
)

// Err represents an error as a string type and implements the error interface.
type Err string

// Error returns an error message as a string
func (e Err) Error() string {
	return string(e)
}
