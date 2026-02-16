package producer

import (
	"github.com/nats-io/nats.go"
)

type publishOptions struct {
	headers nats.Header
	msgID   string
}

// PublishOption represents a functional option for configuring the behavior of a publish operation.
type PublishOption func(*publishOptions)

// PublishWithMsgID sets a custom message ID for the publish operation by modifying the publishOptions.
func PublishWithMsgID(id string) PublishOption {
	return func(p *publishOptions) {
		p.msgID = id
	}
}

// PublishWithHeaders sets the headers for the message by updating the publishOptions with the provided nats.Header.
func PublishWithHeaders(h nats.Header) PublishOption {
	return func(p *publishOptions) {
		p.headers = h
	}
}
