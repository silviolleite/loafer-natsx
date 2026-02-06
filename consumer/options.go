package consumer

import "time"

/*
Option configures a Route during creation.
*/
type Option func(*config)

/*
WithQueueGroup sets the queue group for a queue consumer.
*/
func WithQueueGroup(group string) Option {
	return func(c *config) {
		c.queueGroup = group
	}
}

/*
WithStream sets the stream name for a JetStream consumer.
*/
func WithStream(stream string) Option {
	return func(c *config) {
		c.stream = stream
	}
}

/*
WithDurable sets the durable name for a JetStream consumer.
*/
func WithDurable(durable string) Option {
	return func(c *config) {
		c.durable = durable
	}
}

/*
WithAckWait sets the acknowledgment wait duration.
*/
func WithAckWait(d time.Duration) Option {
	return func(c *config) {
		c.ackWait = d
	}
}

/*
WithMaxDeliver sets the maximum delivery attempts.
*/
func WithMaxDeliver(max int) Option {
	return func(c *config) {
		c.maxDeliver = max
	}
}

/*
WithHandlerTimeout sets the handler execution timeout.
*/
func WithHandlerTimeout(d time.Duration) Option {
	return func(c *config) {
		c.handlerTimeout = d
	}
}

/*
WithReply sets the reply function for request-reply consumers.
*/
func WithReply(r ReplyFunc) Option {
	return func(c *config) {
		c.reply = r
	}
}

// WithEnableDLQ specifies whether the Dead Letter Queue (DLQ) is enabled for the configuration.
func WithEnableDLQ(enable bool) Option {
	return func(c *config) {
		c.enableDLQ = enable
	}
}

// WithFifo sets the FIFO (First-In-First-Out) mode in the config based on the provided enable flag.
func WithFifo(enable bool) Option {
	return func(c *config) {
		c.fifo = enable
	}
}

// WithDedupWindow sets the duration of the deduplication window for message processing.
func WithDedupWindow(d time.Duration) Option {
	return func(c *config) {
		c.dedupWindow = d
	}
}
