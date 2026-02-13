package conn

import "time"

// Option is a functional option type used to configure connection settings dynamically.
type Option func(*config)

// WithName sets the connection name.
func WithName(name string) Option {
	return func(c *config) {
		c.name = name
	}
}

// WithReconnectWait sets the reconnection wait duration.
func WithReconnectWait(d time.Duration) Option {
	return func(c *config) {
		c.reconnectWait = d
	}
}

// WithMaxReconnects sets the maximum reconnection attempts.
func WithMaxReconnects(n int) Option {
	return func(c *config) {
		c.maxReconnects = n
	}
}

// WithTimeout sets the connection timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		c.timeout = d
	}
}
