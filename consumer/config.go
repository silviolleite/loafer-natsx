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
	subject        string
	queueGroup     string
	stream         string
	durable        string
	ackWait        time.Duration
	maxDeliver     int
	fifo           bool
	dedupWindow    time.Duration
	enableDLQ      bool
	reply          ReplyFunc
	handlerTimeout time.Duration
}
