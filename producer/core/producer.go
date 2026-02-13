package core

import (
	"context"

	"github.com/nats-io/nats.go"

	loafernastx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/logger"
)

// Producer is a type that enables publishing and request-reply messaging using NATS and JetStream.
type Producer struct {
	nc  *nats.Conn
	cfg config
}

// New creates a Core Producer using Core NATS.
func New(nc *nats.Conn, subject string, opts ...Option) (*Producer, error) {
	if subject == "" {
		return nil, loafernastx.ErrMissingSubject
	}

	cfg := config{
		subject: subject,
		log:     logger.NopLogger{},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Producer{
		nc:  nc,
		cfg: cfg,
	}, nil
}

// Publish sends a message using JetStream if available, otherwise Core NATS.
func (p *Producer) Publish(ctx context.Context, data []byte, opts ...PublishOption) error {
	pubCfg := publishOptions{}

	for _, opt := range opts {
		opt(&pubCfg)
	}

	msg := &nats.Msg{
		Subject: p.cfg.subject,
		Data:    data,
	}

	if pubCfg.headers != nil {
		msg.Header = pubCfg.headers
	}

	p.cfg.log.Debug(
		"publishing message",
		"subject", p.cfg.subject,
		"nats_core", true,
		"payload_bytes", len(msg.Data),
		"headers_count", len(msg.Header),
	)
	return p.nc.PublishMsg(msg)
}

// Request sends a request with the provided data and context to the subject and waits for a response.
// Returns the response data or an error if the request fails.
func (p *Producer) Request(ctx context.Context, data []byte) ([]byte, error) {
	msg, err := p.nc.RequestWithContext(ctx, p.cfg.subject, data)
	if err != nil {
		return nil, err
	}
	return msg.Data, nil
}
