package core

import "github.com/silviolleite/loafer-natsx/logger"

// Option defines a function type used to modify the configuration of a Producer.
type Option func(*config)

// WithLogger sets a custom logger for the configuration.
func WithLogger(logger logger.Logger) Option {
	return func(c *config) {
		c.log = logger
	}
}
