package router

// Type represents the type of NATS router.
type Type int

const (

	// TypePubSub represents a router type for handling standard publish-subscribe messaging.
	TypePubSub Type = iota

	// TypeQueue represents a router type for processing messages in a work queue model with load balancing.
	TypeQueue

	// TypeRequestReply represents a router type for handling messaging in a request-reply pattern.
	// It works similarly to TypeQueue, but the reply subject is used to send the response message.
	TypeRequestReply

	// TypeJetStream represents a router type for handling messaging using NATS JetStream.
	TypeJetStream
)

// DeliverPolicy defines the delivery behavior for consuming messages from a stream.
type DeliverPolicy int

const (
	// DeliverAllPolicy starts delivering messages from the very beginning of a
	// stream. This is the default.
	DeliverAllPolicy DeliverPolicy = iota

	// DeliverLastPolicy will start the router with the last sequence
	// received.
	DeliverLastPolicy

	// DeliverNewPolicy will only deliver new messages that are sent after the
	// router is created.
	DeliverNewPolicy

	// DeliverByStartSequencePolicy will deliver messages starting from a given
	// sequence configured with OptStartSeq in ConsumerConfig.
	DeliverByStartSequencePolicy

	// DeliverByStartTimePolicy will deliver messages starting from a given time
	// configured with OptStartTime in ConsumerConfig.
	DeliverByStartTimePolicy

	// DeliverLastPerSubjectPolicy will start the router with the last message
	// for all subjects received.
	DeliverLastPerSubjectPolicy
)
