package broker

import (
	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/middleware"
	"github.com/silviolleite/loafer-natsx/router"
)

// RouteRegistration binds a router.Route with its corresponding handler and any
// per-registration middleware. It is validated at creation time to prevent
// invalid broker configuration.
type RouteRegistration struct {
	route       *router.Route
	handler     consumer.HandlerFunc
	middlewares []middleware.Middleware
}

// NewRouteRegistration creates a validated RouteRegistration.
//
// Optional middlewares are applied to this registration only, inside the
// broker's global middlewares. They run in the order provided and nil entries
// are ignored.
func NewRouteRegistration(
	r *router.Route,
	h consumer.HandlerFunc,
	mws ...middleware.Middleware,
) (*RouteRegistration, error) {
	if r == nil {
		return nil, loafernatsx.ErrNilRoute
	}

	if h == nil {
		return nil, loafernatsx.ErrNilHandler
	}

	cleaned := make([]middleware.Middleware, 0, len(mws))
	for _, mw := range mws {
		if mw != nil {
			cleaned = append(cleaned, mw)
		}
	}

	return &RouteRegistration{
		route:       r,
		handler:     h,
		middlewares: cleaned,
	}, nil
}

// Route returns the associated router.Route.
func (rr *RouteRegistration) Route() *router.Route {
	return rr.route
}

// Handler returns the associated handler.
func (rr *RouteRegistration) Handler() consumer.HandlerFunc {
	return rr.handler
}

// Middlewares returns the per-registration middlewares applied to this route.
func (rr *RouteRegistration) Middlewares() []middleware.Middleware {
	return rr.middlewares
}
