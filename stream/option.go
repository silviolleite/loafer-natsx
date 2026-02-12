package stream

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Option represents a function that modifies a stream configuration.
type Option func(*config)

// WithSubjects sets the subjects for the stream configuration and returns an Option to apply this change.
func WithSubjects(subjects ...string) Option {
	return func(c *config) {
		c.subjects = subjects
	}
}

// WithRetention sets the retention policy for the stream configuration and returns an Option to apply this change.
func WithRetention(r jetstream.RetentionPolicy) Option {
	return func(c *config) {
		c.retention = r
	}
}

// WithMaxAge sets the maximum age for messages in the stream configuration and returns an Option to apply this change.
func WithMaxAge(d time.Duration) Option {
	return func(c *config) {
		c.maxAge = d
	}
}

// WithStorage sets the storage type for the stream configuration.
func WithStorage(s jetstream.StorageType) Option {
	return func(c *config) {
		c.storage = s
	}
}

// WithReplicas sets the number of replicas for the stream configuration.
func WithReplicas(n int) Option {
	return func(c *config) {
		c.replicas = n
	}
}

// WithDuplicateWindow sets the duration for the duplicate window in the stream configuration.
func WithDuplicateWindow(d time.Duration) Option {
	return func(c *config) {
		c.duplicateWindow = d
	}
}
