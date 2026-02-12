package producer

import "github.com/silviolleite/loafer-natsx/logger"

// Option defines a function type used to modify the configuration of a Producer.
type Option func(*config)

// WithJetStream enables JetStream publishing.
func WithJetStream() Option {
	return func(c *config) {
		c.useJS = true
	}
}

// WithLogger sets a custom logger for the configuration.
func WithLogger(logger logger.Logger) Option {
	return func(c *config) {
		c.log = logger
	}
}
