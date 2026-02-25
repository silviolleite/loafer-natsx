package typed

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/router"
)

// ReplyFunc is a type-safe version of router.ReplyFunc that receives the
// handler result already asserted to type R.
type ReplyFunc[R any] func(ctx context.Context, result R, handlerErr error) ([]byte, nats.Header, error)

// WrapReply adapts a typed ReplyFunc[R] into a router.ReplyFunc by performing
// a type assertion on the result returned by the handler.
func WrapReply[R any](fn ReplyFunc[R]) router.ReplyFunc {
	return func(ctx context.Context, result any, handlerErr error) ([]byte, nats.Header, error) {
		if handlerErr != nil {
			var zero R
			return fn(ctx, zero, handlerErr)
		}

		v, ok := result.(R)
		if !ok {
			return nil, nil, fmt.Errorf("typed: reply: expected %T, got %T", *new(R), result)
		}

		return fn(ctx, v, nil)
	}
}
