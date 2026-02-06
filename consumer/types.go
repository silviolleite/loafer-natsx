package consumer

/*
Type represents the type of NATS consumer.
*/
type Type int

const (
	TypePubSub Type = iota
	TypeQueue
	TypeRequestReply
	TypeJetStream
)
