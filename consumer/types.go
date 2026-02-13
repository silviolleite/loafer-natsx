package consumer

/*
Type represents the type of NATS consumer.
*/
type Type int

const (

	// RouteTypePubSub represents a consumer type for handling standard publish-subscribe messaging.
	RouteTypePubSub Type = iota

	// RouteTypeQueue represents a consumer type for processing messages in a work queue model with load balancing.
	RouteTypeQueue

	// RouteTypeRequestReply represents a consumer type for handling messaging in a request-reply pattern.
	RouteTypeRequestReply

	// RouteTypeJetStream represents a consumer type for handling messaging using NATS JetStream.
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
