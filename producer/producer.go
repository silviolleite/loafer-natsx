package producer

import (
	"context"

	"github.com/nats-io/nats.go"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/logger"
)

// Producer represents a message producer capable of publishing messages to a specific subject with customizable options.
type Producer struct {
	log       logger.Logger
	publisher Publisher
	subject   string
}

// New initializes and returns a new Producer instance configured with the given Publisher, subject, and optional options.
// Returns an error if the subject is empty or the provided Publisher is nil.
func New(
	publisher Publisher,
	subject string,
	opts ...Option,
) (*Producer, error) {
	if subject == "" {
		return nil, loafernatsx.ErrMissingSubject
	}

	if publisher == nil {
		return nil, loafernatsx.ErrUnsupportedType
	}

	cfg := config{
		subject: subject,
		log:     logger.NopLogger{},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Producer{
		subject:   cfg.subject,
		log:       cfg.log,
		publisher: publisher,
	}, nil
}

// Publish sends a message to the configured subject using the provided data and optional publish options.
func (p *Producer) Publish(
	ctx context.Context,
	data []byte,
	opts ...PublishOption,
) error {
	pubCfg := PublishOptions{}

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

	return p.publisher.Publish(ctx, msg, pubCfg)
}

// Request sends a request to the configured subject with the provided data and waits for a response.
// Only Core NATS producers support request operations.
// Returns an error if the configured Publisher does not support request operations.
func (p *Producer) Request(
	ctx context.Context,
	data []byte,
) ([]byte, error) {
	r, ok := p.publisher.(Requester)
	if !ok {
		return nil, loafernatsx.ErrRequestNotSupported
	}

	return r.Request(ctx, p.subject, data)
}
