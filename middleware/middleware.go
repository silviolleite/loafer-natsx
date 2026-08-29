package middleware

import "context"

// Handler is the function signature for NATS message processing. It receives a
// context carrying cancellation, deadlines, and the message Metadata attached
// by the consumer, together with the raw message payload. It returns the
// handler result and a non-nil error when processing fails.
//
// It mirrors consumer.HandlerFunc so a composed middleware chain can be used
// directly as a consumer handler.
type Handler func(ctx context.Context, data []byte) (any, error)

// Middleware wraps a Handler with additional behavior and returns the wrapped
// Handler.
type Middleware func(Handler) Handler

// Chain composes multiple middlewares into a single Middleware using
// first-is-outermost semantics: the first middleware in the list becomes the
// outermost layer and therefore runs first on the way in and last on the way
// out.
//
//	Chain(A, B, C)(h) == A(B(C(h)))
//
// The resulting execution order on the way in is A → B → C → h. Nil
// middlewares are skipped, and calling Chain with no effective middlewares
// returns a Middleware that leaves the handler unchanged.
func Chain(mws ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			if mws[i] == nil {
				continue
			}
			next = mws[i](next)
		}
		return next
	}
}
