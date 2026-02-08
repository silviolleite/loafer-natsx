package producer

import (
	"github.com/nats-io/nats.go"
)

type publishOptions struct {
	msgID   string
	headers nats.Header
}

type PublishOption func(*publishOptions)

/*
WithMsgID sets the JetStream message ID for deduplication.
*/
func WithMsgID(id string) PublishOption {
	return func(p *publishOptions) {
		p.msgID = id
	}
}

/*
WithHeaders sets custom headers for the message.
*/
func WithHeaders(h nats.Header) PublishOption {
	return func(p *publishOptions) {
		p.headers = h
	}
}
