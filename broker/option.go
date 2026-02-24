package broker

import (
	"github.com/prometheus/client_golang/prometheus"
)

type config struct {
	metrics *brokerMetrics
	workers int
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

// WithMetrics sets up Prometheus metrics using the provided Registerer and applies them to the configuration.
func WithMetrics(reg prometheus.Registerer) Option {
	return func(c *config) {
		c.metrics = newMetrics(reg)
	}
}
