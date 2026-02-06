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

/*
Config defines how a consumer route should behave.
*/
type Config struct {
	Subject    string
	QueueGroup string

	// JetStream-specific
	Stream     string
	Durable    string
	AckWait    time.Duration
	MaxDeliver int

	FIFO        bool
	DedupWindow time.Duration

	// DLQ
	EnableDLQ  bool
	DLQSubject string

	// RequestReply-specific
	Reply          ReplyFunc
	HandlerTimeout time.Duration
}
