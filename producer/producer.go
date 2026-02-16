package producer

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/logger"
)

// Producer publishes messages using a configured strategy.
type Producer struct {
	log      logger.Logger
	strategy publishStrategy
	nc       *nats.Conn
	subject  string
}

// NewCore creates a Producer configured for Core NATS publishing.
func NewCore(
	nc *nats.Conn,
	subject string,
	opts ...Option,
) (*Producer, error) {
	if subject == "" {
		return nil, loafernatsx.ErrMissingSubject
	}

	cfg := config{
		subject: subject,
		log:     logger.NopLogger{},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Producer{
		subject:  cfg.subject,
		log:      cfg.log,
		nc:       nc,
		strategy: newCoreStrategy(nc),
	}, nil
}

// NewJetStream creates a Producer configured for JetStream publishing.
func NewJetStream(
	js jetstream.JetStream,
	subject string,
	opts ...Option,
) (*Producer, error) {
	if subject == "" {
		return nil, loafernatsx.ErrMissingSubject
	}

	cfg := config{
		subject: subject,
		log:     logger.NopLogger{},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Producer{
		subject:  cfg.subject,
		log:      cfg.log,
		strategy: newJetStreamStrategy(js, cfg.log),
	}, nil
}

// Publish sends a message using the configured publishing strategy.
func (p *Producer) Publish(
	ctx context.Context,
	data []byte,
	opts ...PublishOption,
) error {
	pubCfg := publishOptions{}

	for _, opt := range opts {
		opt(&pubCfg)
	}

	msg := &nats.Msg{
		Subject: p.subject,
		Data:    data,
	}

	if pubCfg.headers != nil {
		msg.Header = pubCfg.headers
	}

	p.log.Debug(
		"publishing message",
		"subject", p.subject,
		"payload_bytes", len(data),
		"headers_count", len(msg.Header),
	)

	return p.strategy.Publish(ctx, msg, pubCfg)
}

// Request sends a request message and waits for a reply.
// Only available for Core NATS producers.
func (p *Producer) Request(
	ctx context.Context,
	data []byte,
) ([]byte, error) {
	if p.nc == nil {
		return nil, loafernatsx.ErrRequestNotSupported
	}

	msg, err := p.nc.RequestWithContext(ctx, p.subject, data)
	if err != nil {
		return nil, err
	}

	return msg.Data, nil
}
