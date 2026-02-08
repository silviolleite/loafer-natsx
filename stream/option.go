package stream

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type Option func(*config)

/*
WithSubjects sets the subjects for the stream.
*/
func WithSubjects(subjects ...string) Option {
	return func(c *config) {
		c.subjects = subjects
	}
}

/*
WithRetention sets the retention policy.
*/
func WithRetention(r jetstream.RetentionPolicy) Option {
	return func(c *config) {
		c.retention = r
	}
}

/*
WithMaxAge sets the maximum message age.
*/
func WithMaxAge(d time.Duration) Option {
	return func(c *config) {
		c.maxAge = d
	}
}

/*
WithStorage sets the storage type.
*/
func WithStorage(s jetstream.StorageType) Option {
	return func(c *config) {
		c.storage = s
	}
}

/*
WithReplicas sets the number of replicas.
*/
func WithReplicas(n int) Option {
	return func(c *config) {
		c.replicas = n
	}
}

/*
WithDuplicateWindow sets the deduplication window.
*/
func WithDuplicateWindow(d time.Duration) Option {
	return func(c *config) {
		c.duplicateWindow = d
	}
}
