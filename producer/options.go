package producer

import "time"

/*
Option configures a Producer.
*/
type Option func(*config)

/*
WithFIFO enables FIFO publishing semantics.
*/
func WithFIFO() Option {
	return func(c *config) {
		c.fifo = true
	}
}

/*
WithDeduplication sets a message deduplication ID.
*/
func WithDeduplication(id string) Option {
	return func(c *config) {
		c.dedupID = id
	}
}

/*
WithDedupWindow sets the deduplication window.
*/
func WithDedupWindow(d time.Duration) Option {
	return func(c *config) {
		c.dedupWindow = d
	}
}
