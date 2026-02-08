package consumer

/*
Type represents the type of NATS consumer.
*/
type Type int

const (
	RouteTypePubSub Type = iota
	RouteTypeQueue
	RouteTypeRequestReply
	RouteTypeJetStream
)

// DeliverPolicy defines the delivery behavior for consuming messages from a stream.
type DeliverPolicy int

const (
	// DeliverAllPolicy starts delivering messages from the very beginning of a
	// stream. This is the default.
	DeliverAllPolicy DeliverPolicy = iota

	// DeliverLastPolicy will start the consumer with the last sequence
	// received.
	DeliverLastPolicy

	// DeliverNewPolicy will only deliver new messages that are sent after the
	// consumer is created.
	DeliverNewPolicy

	// DeliverByStartSequencePolicy will deliver messages starting from a given
	// sequence configured with OptStartSeq in ConsumerConfig.
	DeliverByStartSequencePolicy

	// DeliverByStartTimePolicy will deliver messages starting from a given time
	// configured with OptStartTime in ConsumerConfig.
	DeliverByStartTimePolicy

	// DeliverLastPerSubjectPolicy will start the consumer with the last message
	// for all subjects received.
	DeliverLastPerSubjectPolicy
)
