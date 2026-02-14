package broker

import (
	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/router"
)

// RouteRegistration binds a router.Route with its corresponding handler.
// It is validated at creation time to prevent invalid broker configuration.
type RouteRegistration struct {
	route   *router.Route
	handler consumer.HandlerFunc
}

// NewRouteRegistration creates a validated RouteRegistration.
func NewRouteRegistration(
	r *router.Route,
	h consumer.HandlerFunc,
) (*RouteRegistration, error) {
	if r == nil {
		return nil, loafernatsx.ErrNilRoute
	}

	if h == nil {
		return nil, loafernatsx.ErrNilHandler
	}

	return &RouteRegistration{
		route:   r,
		handler: h,
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
