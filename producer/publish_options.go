package producer

import (
	"github.com/nats-io/nats.go"
)

// PublishOptions holds configuration for customizing the publishing behavior, including headers and message ID.
type PublishOptions struct {
	headers nats.Header
	msgID   string
}

// PublishOption represents a functional option for configuring the behavior of a publish operation.
type PublishOption func(*PublishOptions)

// PublishWithMsgID sets a custom message ID for the publish operation by modifying the PublishOptions.
func PublishWithMsgID(id string) PublishOption {
	return func(p *PublishOptions) {
		p.msgID = id
	}
}

// PublishWithHeaders sets the headers for the message by updating the PublishOptions with the provided nats.Header.
func PublishWithHeaders(h nats.Header) PublishOption {
	return func(p *PublishOptions) {
		p.headers = h
	}
}
