package producer

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/silviolleite/loafer-natsx/logger"
)

type jetStreamStrategy struct {
	js  jetstream.JetStream
	log logger.Logger
}

// NewJetStreamStrategy creates a new Publisher implementation using JetStream and a provided logger.
func NewJetStreamStrategy(js jetstream.JetStream, log logger.Logger) Publisher {
	return &jetStreamStrategy{
		js:  js,
		log: log,
	}
}

// Publish sends a message to a JetStream subject with optional deduplication and logging based on the provided options.
func (j *jetStreamStrategy) Publish(
	ctx context.Context,
	msg *nats.Msg,
	opts PublishOptions,
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
