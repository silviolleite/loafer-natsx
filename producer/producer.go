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
// Returns a *PublishResult with publish metadata and an error if the publish failed.
func (p *Producer) Publish(
	ctx context.Context,
	data []byte,
	opts ...PublishOption,
) (*PublishResult, error) {
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
// It is a convenience wrapper around RequestMsg that builds a *nats.Msg from the configured subject
// and the provided data. Only Core NATS producers support request operations.
// Returns a *Response containing the reply data and headers, or an error if the configured Publisher
// does not support request operations.
func (p *Producer) Request(
	ctx context.Context,
	data []byte,
) (*Response, error) {
	return p.RequestMsg(ctx, &nats.Msg{
		Subject: p.subject,
		Data:    data,
	})
}

// RequestMsg sends the provided message as a request to the configured subject and waits for a response.
// Unlike Request, it accepts a full *nats.Msg so callers can attach headers (correlation IDs, tracing
// metadata, etc.) to the outgoing request. When the message subject is empty, the producer subject is used.
// Only Core NATS producers support request operations.
// When a request timeout is configured via WithRequestTimeout, the context is wrapped with a deadline so
// the call does not block indefinitely if the consumer becomes unavailable.
// Returns a *Response containing the reply data and headers, or an error if the message is nil or the
// configured Publisher does not support request operations.
func (p *Producer) RequestMsg(
	ctx context.Context,
	msg *nats.Msg,
) (*Response, error) {
	if msg == nil {
		return nil, loafernatsx.ErrMissingMessage
	}

	if msg.Subject == "" {
		msg.Subject = p.subject
	}

	if p.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.requestTimeout)
		defer cancel()
	}

	p.log.Debug(
		"sending request",
		"subject", msg.Subject,
		"payload_bytes", len(msg.Data),
		"headers_count", len(msg.Header),
	)

	resp, err := p.request(ctx, msg)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", loafernatsx.ErrRequestTimeout, err)
		}

		return nil, err
	}

	return resp, nil
}

// request dispatches to the most specific request capability the underlying Publisher supports,
// preferring RequestMsger (which preserves headers) over the legacy Requester. It returns
// ErrRequestNotSupported when the Publisher supports neither.
func (p *Producer) request(ctx context.Context, msg *nats.Msg) (*Response, error) {
	switch r := p.publisher.(type) {
	case RequestMsger:
		return r.RequestMsg(ctx, msg)
	case Requester:
		return r.Request(ctx, msg.Subject, msg.Data)
	default:
		return nil, loafernatsx.ErrRequestNotSupported
	}
}
