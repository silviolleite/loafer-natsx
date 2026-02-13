package consumer

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
)

/*
ReplyFunc allows customizing the reply payload for RequestReply routes.
handlerErr is the error returned by the handler (if any).
*/
type ReplyFunc func(ctx context.Context, result any, handlerErr error) ([]byte, nats.Header, error)

type config struct {
	reply          ReplyFunc
	subject        string
	queueGroup     string
	stream         string
	durable        string
	ackWait        time.Duration
	maxDeliver     int
	handlerTimeout time.Duration
	deliveryPolicy DeliverPolicy
	fifo           bool
	enableDLQ      bool
}
