package producer

// PublishResult encapsulates metadata returned after a successful publish.
// For Core NATS, all fields remain at their zero values since core publish
// is fire-and-forget. For JetStream, the fields are populated from the
// server acknowledgement.
type PublishResult struct {
	// Stream is the name of the JetStream stream that accepted the message.
	Stream string

	// Sequence is the stream sequence number assigned to the message.
	Sequence uint64

	// Duplicate indicates whether the server detected this message as a
	// duplicate based on the deduplication window.
	Duplicate bool
}
