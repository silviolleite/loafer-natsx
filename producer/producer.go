package producer

import (
	"context"

	"github.com/nats-io/nats.go"
)

/*
Producer wraps NATS publishing capabilities.
*/
type Producer struct {
	nc  *nats.Conn
	js  nats.JetStreamContext
	cfg Config
}

/*
New creates a new Producer instance.
*/
func New(nc *nats.Conn, cfg Config) (*Producer, error) {
	js, _ := nc.JetStream()
	return &Producer{nc: nc, js: js, cfg: cfg}, nil
}

/*
Publish sends a message using the configured producer behavior.
*/
func (p *Producer) Publish(ctx context.Context, data []byte) error {
	if p.js != nil {
		_, err := p.js.Publish(p.cfg.Subject, data)
		return err
	}

	return p.nc.Publish(p.cfg.Subject, data)
}

/*
Request performs a request-reply interaction.
*/
func (p *Producer) Request(ctx context.Context, data []byte) ([]byte, error) {
	msg, err := p.nc.RequestWithContext(ctx, p.cfg.Subject, data)
	if err != nil {
		return nil, err
	}
	return msg.Data, nil
}
