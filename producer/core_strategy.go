package producer

import (
	"context"

	"github.com/nats-io/nats.go"
)

type coreStrategy struct {
	nc *nats.Conn
}

var (
	_ Publisher    = (*coreStrategy)(nil)
	_ Requester    = (*coreStrategy)(nil)
	_ RequestMsger = (*coreStrategy)(nil)
)

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
	reply, err := c.nc.RequestWithContext(ctx, subject, data)
	if err != nil {
		return nil, err
	}
	return &Response{Data: reply.Data, Header: reply.Header}, nil
}

// RequestMsg sends the provided message as a request and waits for a response within the provided context.
// The message headers are forwarded to the responder, allowing callers to attach correlation IDs,
// tracing metadata, and other application-specific context to the request.
func (c *coreStrategy) RequestMsg(
	ctx context.Context,
	msg *nats.Msg,
) (*Response, error) {
	reply, err := c.nc.RequestMsgWithContext(ctx, msg)
	if err != nil {
		return nil, err
	}
	return &Response{Data: reply.Data, Header: reply.Header}, nil
}
