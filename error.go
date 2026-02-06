package loafernastx

const (
	// ErrUnsupportedType indicates an error when an unsupported consumer type is encountered.
	ErrUnsupportedType = Err("unsupported consumer type")

	// ErrNilConfig represents an error indicating that the consumer configuration cannot be nil.
	ErrNilConfig = Err("consumer config cannot be nil")

	// ErrMissingSubject indicates an error when the required subject is not provided.
	ErrMissingSubject = Err("subject is required")

	// ErrMissingQueueGroup indicates an error when a queue group is required but not provided for a queue consumer.
	ErrMissingQueueGroup = Err("queue group is required for queue consumer")

	// ErrMissingStream indicates an error when a stream is required but not provided for a jetstream consumer.
	ErrMissingStream = Err("stream is required for jetstream consumer")

	// ErrMissingDurable indicates an error when a durable name is required but not provided for a jetstream consumer.
	ErrMissingDurable = Err("durable name is required for jetstream consumer")
)

// Err represents an error as a string type and implements the error interface.
type Err string

// Error returns an error message as a string
func (e Err) Error() string {
	return string(e)
}
