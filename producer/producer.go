package producer

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

/*
Producer wraps NATS publishing capabilities.
*/
type Producer struct {
	nc  *nats.Conn
	js  jetstream.JetStream
	cfg config
}

/*
New creates a new Producer instance.
*/
func New(nc *nats.Conn, subject string, opts ...Option) (*Producer, error) {
	if subject == "" {
		return nil, errors.New("subject is required")
	}

	cfg := config{
		subject: subject,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	js, _ := jetstream.New(nc)

	return &Producer{
		nc:  nc,
		js:  js,
		cfg: cfg,
	}, nil
}

/*
Publish sends a message using JetStream if available, otherwise Core NATS.
*/
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

	if p.js != nil {
		var jsOpts []jetstream.PublishOpt

		if pubCfg.msgID != "" {
			jsOpts = append(jsOpts, jetstream.WithMsgID(pubCfg.msgID))
		}

		_, err := p.js.PublishMsg(ctx, msg, jsOpts...)
		return err
	}

	return p.nc.PublishMsg(msg)
}

/*
Request performs a request-reply interaction.
*/
func (p *Producer) Request(ctx context.Context, data []byte) ([]byte, error) {
	msg, err := p.nc.RequestWithContext(ctx, p.cfg.subject, data)
	if err != nil {
		return nil, err
	}
	return msg.Data, nil
}
