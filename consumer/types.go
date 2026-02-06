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
