package broker

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/silviolleite/loafer-natsx/middleware"
)

type config struct {
	globalMiddlewares []middleware.Middleware
	workers           int
}

// Option is a function type used to modify the configuration of a component by applying changes to a config instance.
type Option func(*config)

// WithWorkers returns an Option to configure the number of workers in the config. It sets the value if n is greater than 0.
func WithWorkers(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.workers = n
		}
	}
}

// WithMetrics enables Prometheus metrics on the given Registerer by installing
// the built-in metrics middleware as a global middleware.
//
// It is convenience sugar over WithGlobalMiddleware(middleware.Metrics(...)):
// every message processed by the broker is instrumented with the loafer_*
// collectors labeled by subject. A nil Registerer falls back to
// prometheus.DefaultRegisterer.
func WithMetrics(reg prometheus.Registerer) Option {
	return func(c *config) {
		c.globalMiddlewares = append(
			c.globalMiddlewares,
			middleware.Metrics(middleware.WithMetricsRegisterer(reg)),
		)
	}
}

// WithGlobalMiddleware appends middlewares applied outermost to every message
// across all route registrations, ahead of any per-registration middleware.
// They run first on the way in and last on the way out. Multiple calls
// accumulate in order and nil middlewares are ignored.
//
// It is the extension point for plugging in observability backends such as the
// built-in middleware.Metrics (Prometheus) and middleware.OTel (OpenTelemetry),
// as well as any custom middleware.Middleware implementation.
func WithGlobalMiddleware(mws ...middleware.Middleware) Option {
	return func(c *config) {
		for _, mw := range mws {
			if mw != nil {
				c.globalMiddlewares = append(c.globalMiddlewares, mw)
			}
		}
	}
}
