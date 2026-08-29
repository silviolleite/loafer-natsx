package conn

import (
	"crypto/tls"
	"time"
)

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

// WithSecure enables a secure (TLS) connection to the NATS server using the
// provided *tls.Config. When set, the client requires the server to support
// TLS and will fail to connect to a server that does not offer it.
func WithSecure(tlsConfig *tls.Config) Option {
	return func(c *config) {
		c.tlsConfig = tlsConfig
	}
}
