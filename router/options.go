package router

import "time"

// Option configures a Route during creation.
type Option func(*config)

// WithQueueGroup sets the queue group for a queue router.
func WithQueueGroup(group string) Option {
	return func(c *config) {
		c.queueGroup = group
	}
}

// WithStream sets the stream name for a JetStream router.
func WithStream(stream string) Option {
	return func(c *config) {
		c.stream = stream
	}
}

// WithDurable sets the durable name for a JetStream router.
func WithDurable(durable string) Option {
	return func(c *config) {
		c.durable = durable
	}
}

// WithAckWait sets the acknowledgment wait duration.
func WithAckWait(d time.Duration) Option {
	return func(c *config) {
		c.ackWait = d
	}
}

// WithMaxDeliver sets the maximum delivery attempts.
func WithMaxDeliver(n int) Option {
	return func(c *config) {
		c.maxDeliver = n
	}
}

// WithHandlerTimeout sets the handler execution timeout.
func WithHandlerTimeout(d time.Duration) Option {
	return func(c *config) {
		c.handlerTimeout = d
	}
}

// WithReply sets the reply function for request-reply consumers.
func WithReply(r ReplyFunc) Option {
	return func(c *config) {
		c.reply = r
	}
}

// WithEnableDLQ enables the Dead Letter Queue (DLQ) for the router by setting the enableDLQ configuration to true.
func WithEnableDLQ() Option {
	return func(c *config) {
		c.enableDLQ = true
	}
}

// WithDeliveryPolicy sets the message delivery policy for a JetStream consumer and returns an Option to apply this change.
func WithDeliveryPolicy(policy DeliverPolicy) Option {
	return func(c *config) {
		c.deliveryPolicy = policy
	}
}
