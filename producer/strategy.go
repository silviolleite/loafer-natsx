package producer

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/silviolleite/loafer-natsx/logger"
)

// publishStrategy defines how a message is published.
type publishStrategy interface {
	Publish(ctx context.Context, msg *nats.Msg, opts publishOptions) error
}

// coreStrategy implements publishing using Core NATS.
type coreStrategy struct {
	nc *nats.Conn
}

// newCoreStrategy creates a new Core NATS publishing strategy.
func newCoreStrategy(nc *nats.Conn) publishStrategy {
	return &coreStrategy{nc: nc}
}

// Publish sends a message using Core NATS.
func (c *coreStrategy) Publish(
	ctx context.Context,
	msg *nats.Msg,
	_ publishOptions,
) error {
	return c.nc.PublishMsg(msg)
}

// jetStreamStrategy implements publishing using JetStream.
type jetStreamStrategy struct {
	js  jetstream.JetStream
	log logger.Logger
}

// newJetStreamStrategy creates a new JetStream publishing strategy.
func newJetStreamStrategy(js jetstream.JetStream, log logger.Logger) publishStrategy {
	return &jetStreamStrategy{
		js:  js,
		log: log,
	}
}

// Publish sends a message using JetStream with optional deduplication.
func (j *jetStreamStrategy) Publish(
	ctx context.Context,
	msg *nats.Msg,
	opts publishOptions,
) error {
	var jsOpts []jetstream.PublishOpt

	if opts.msgID != "" {
		jsOpts = append(jsOpts, jetstream.WithMsgID(opts.msgID))

		j.log.Info(
			"jetstream deduplication enabled",
			"subject", msg.Subject,
			"msg_id", opts.msgID,
			"note", "duplicate window defined by stream (default 2m)",
		)
	}

	ack, err := j.js.PublishMsg(ctx, msg, jsOpts...)
	if err != nil {
		return err
	}

	j.log.Debug(
		"jetstream publish acknowledged",
		"stream", ack.Stream,
		"sequence", ack.Sequence,
		"duplicate", ack.Duplicate,
	)

	return nil
}
