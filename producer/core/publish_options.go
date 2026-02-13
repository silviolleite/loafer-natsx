package core

import (
	"github.com/nats-io/nats.go"
)

type publishOptions struct {
	msgID   string
	headers nats.Header
}

// PublishOption represents a functional option for configuring the behavior of a publish operation.
type PublishOption func(*publishOptions)

// PublishWithHeaders sets the headers for the message by updating the publishOptions with the provided nats.Header.
func PublishWithHeaders(h nats.Header) PublishOption {
	return func(p *publishOptions) {
		p.headers = h
	}
}
