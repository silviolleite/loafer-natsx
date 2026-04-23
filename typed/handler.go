package typed

import (
	"context"
	"fmt"

	"github.com/silviolleite/loafer-natsx/consumer"
)

// HandlerFunc is a type-safe handler that receives a decoded message of type T
// and returns a response of type R.
//
// The context passed to the handler carries message metadata (subject, headers,
// reply subject, and JetStream-specific fields such as stream name, sequence,
// and delivery count). Use [consumer.MetadataFromContext] to access it:
//
//	func(ctx context.Context, msg Order) (Response, error) {
//	    meta, ok := consumer.MetadataFromContext(ctx)
//	    if ok {
//	        correlationID := meta.Headers.Get("X-Correlation-ID")
//	        subject := meta.Subject // e.g. "orders.created.user-123"
//	    }
//	    // process msg...
//	}
type HandlerFunc[T any, R any] func(ctx context.Context, msg T) (R, error)

// WrapHandler adapts a typed HandlerFunc into a consumer.HandlerFunc by
// decoding the raw bytes with the provided codec before invoking fn.
//
// The context forwarded to fn already carries [consumer.Metadata] populated
// by the consumer layer. Handlers can retrieve it with
// [consumer.MetadataFromContext] to access subject, headers, reply subject,
// and JetStream-specific fields (stream, sequence, delivery count, timestamp).
func WrapHandler[T any, R any](codec Codec[T], fn HandlerFunc[T, R]) consumer.HandlerFunc {
	return func(ctx context.Context, data []byte) (any, error) {
		msg, err := codec.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("typed: decode: %w", err)
		}

		return fn(ctx, msg)
	}
}
