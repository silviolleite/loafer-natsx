package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/logger"
)

// Producer represents a message producer capable of publishing messages to a specific subject with customizable options.
type Producer struct {
	log            logger.Logger
	publisher      Publisher
	subject        string
	requestTimeout time.Duration
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
		subject:        subject,
		log:            logger.NopLogger{},
		requestTimeout: defaultRequestTimeout,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Producer{
		subject:        cfg.subject,
		log:            cfg.log,
		publisher:      publisher,
		requestTimeout: cfg.requestTimeout,
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
// When a request timeout is configured via WithRequestTimeout, the context is
// wrapped with a deadline so the call does not block indefinitely if the
// consumer becomes unavailable.
// Returns a *Response containing the reply data and headers, or an error if the configured Publisher
// does not support request operations.
func (p *Producer) Request(
	ctx context.Context,
	data []byte,
) (*Response, error) {
	r, ok := p.publisher.(Requester)
	if !ok {
		return nil, loafernatsx.ErrRequestNotSupported
	}

	if p.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.requestTimeout)
		defer cancel()
	}

	resp, err := r.Request(ctx, p.subject, data)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", loafernatsx.ErrRequestTimeout, err)
		}

		return nil, err
	}

	return resp, nil
}
