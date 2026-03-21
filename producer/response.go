package producer

import "github.com/nats-io/nats.go"

// Response encapsulates the data and headers from a NATS reply message.
type Response struct {
	Header nats.Header
	Data   []byte
}
