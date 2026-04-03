package producer

import (
	"context"

	"github.com/nats-io/nats.go"
)

type coreStrategy struct {
	nc *nats.Conn
}

// NewCoreStrategy creates and returns a new Publisher instance backed by a coreStrategy using the provided nats.Conn.
func NewCoreStrategy(nc *nats.Conn) Publisher {
	return &coreStrategy{nc: nc}
}

// Publish sends a message to a specified NATS subject using the configured connection in coreStrategy.
// Core NATS publish is fire-and-forget, so PublishResult fields remain at zero values.
func (c *coreStrategy) Publish(
	ctx context.Context,
	msg *nats.Msg,
	_ PublishOptions,
) (*PublishResult, error) {
	if err := c.nc.PublishMsg(msg); err != nil {
		return nil, err
	}
	return &PublishResult{}, nil
}

// Request sends a request to a specified NATS subject and waits for a response within the provided context.
func (c *coreStrategy) Request(
	ctx context.Context,
	subject string,
	data []byte,
) (*Response, error) {
	msg, err := c.nc.RequestWithContext(ctx, subject, data)
	if err != nil {
		return nil, err
	}
	return &Response{Data: msg.Data, Header: msg.Header}, nil
}
