package broker

import (
	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/middleware"
)

// compose builds the effective handler for a route registration by wrapping its
// handler with the broker's global middlewares followed by the registration's
// own middlewares.
//
// Composition uses first-is-outermost semantics: global middlewares are the
// outermost layer and run first on the way in and last on the way out, then the
// per-registration middlewares, then the user handler. When no middleware is
// configured the raw handler is returned unchanged for a zero-overhead path.
func (b *Broker) compose(reg *RouteRegistration) consumer.HandlerFunc {
	handler := reg.Handler()

	regMiddlewares := reg.Middlewares()
	if len(b.middlewares) == 0 && len(regMiddlewares) == 0 {
		return handler
	}

	mws := make([]middleware.Middleware, 0, len(b.middlewares)+len(regMiddlewares))
	mws = append(mws, b.middlewares...)
	mws = append(mws, regMiddlewares...)

	wrapped := middleware.Chain(mws...)(middleware.Handler(handler))

	return consumer.HandlerFunc(wrapped)
}
