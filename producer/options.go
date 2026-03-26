package producer

import (
	"time"

	"github.com/silviolleite/loafer-natsx/logger"
)

// Option defines a function type used to modify the configuration of a Producer.
type Option func(*config)

// WithLogger sets a custom logger for the configuration.
func WithLogger(logger logger.Logger) Option {
	return func(c *config) {
		c.log = logger
	}
}

// WithRequestTimeout sets a maximum duration for request-reply operations.
// When set, the Producer wraps the caller's context with this deadline before
// sending the request. This prevents the producer from waiting indefinitely
// when the consumer dies mid-handler. A zero or negative value is ignored.
func WithRequestTimeout(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.requestTimeout = d
		}
	}
}

// WithoutRequestTimeout disables the default request timeout. When applied,
// request-reply operations rely solely on the caller's context for
// cancellation. Use this when the caller manages its own deadlines.
func WithoutRequestTimeout() Option {
	return func(c *config) {
		c.requestTimeout = 0
	}
}
