package typed

import (
	"context"
	"fmt"

	"github.com/silviolleite/loafer-natsx/consumer"
)

// HandlerFunc is a type-safe handler that receives a decoded message of type T
// and returns a response of type R.
type HandlerFunc[T any, R any] func(ctx context.Context, msg T) (R, error)

// WrapHandler adapts a typed HandlerFunc into a consumer.HandlerFunc by
// decoding the raw bytes with the provided codec before invoking fn.
func WrapHandler[T any, R any](codec Codec[T], fn HandlerFunc[T, R]) consumer.HandlerFunc {
	return func(ctx context.Context, data []byte) (any, error) {
		msg, err := codec.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("typed: decode: %w", err)
		}

		return fn(ctx, msg)
	}
}
