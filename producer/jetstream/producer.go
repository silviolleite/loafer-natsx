package jetstream

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	loafernastx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/logger"
)

// Producer is a type that enables publishing and request-reply messaging using NATS and JetStream.
type Producer struct {
	nc  *nats.Conn
	js  jetstream.JetStream
	cfg config
}

// New creates a Producer using JetStream.
func New(js jetstream.JetStream, subject string, opts ...Option) (*Producer, error) {
	err := validateStream(js, subject)
	if err != nil {
		return nil, err
	}

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
		js:  js,
		cfg: cfg,
	}, nil
}

func validateStream(js jetstream.JetStream, stream string) error {
	if stream == "" {
		return loafernastx.ErrMissingStream
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := js.AccountInfo(ctx); err != nil {
		return fmt.Errorf("jetstream not operational: %w", err)
	}

	return nil
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

	var jsOpts []jetstream.PublishOpt

	if pubCfg.msgID != "" {
		jsOpts = append(jsOpts, jetstream.WithMsgID(pubCfg.msgID))
		p.cfg.log.Info(
			"publishing with msg_id (JetStream deduplication enabled)",
			"msg_id", pubCfg.msgID,
			"dedup_window_default", "2m (if not configured in stream)",
		)
	}

	p.cfg.log.Debug(
		"publishing message",
		"subject", p.cfg.subject,
		"jetstream", true,
		"msg_id", pubCfg.msgID,
		"payload_bytes", len(msg.Data),
		"headers_count", len(msg.Header),
	)
	ack, err := p.js.PublishMsg(ctx, msg, jsOpts...)
	if err != nil {
		return err
	}

	p.cfg.log.Debug("published message",
		"ack_sequence", ack.Sequence,
		"ack_stream", ack.Stream,
	)

	return nil
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
