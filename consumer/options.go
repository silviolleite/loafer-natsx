package consumer

import "time"

/*
Option configures a Route during creation.
*/
type Option func(*Config)

/*
WithQueueGroup sets the queue group for a queue consumer.
*/
func WithQueueGroup(group string) Option {
	return func(c *Config) {
		c.QueueGroup = group
	}
}

/*
WithStream sets the stream name for a JetStream consumer.
*/
func WithStream(stream string) Option {
	return func(c *Config) {
		c.Stream = stream
	}
}

/*
WithDurable sets the durable name for a JetStream consumer.
*/
func WithDurable(durable string) Option {
	return func(c *Config) {
		c.Durable = durable
	}
}

/*
WithAckWait sets the acknowledgment wait duration.
*/
func WithAckWait(d time.Duration) Option {
	return func(c *Config) {
		c.AckWait = d
	}
}

/*
WithMaxDeliver sets the maximum delivery attempts.
*/
func WithMaxDeliver(max int) Option {
	return func(c *Config) {
		c.MaxDeliver = max
	}
}

/*
WithHandlerTimeout sets the handler execution timeout.
*/
func WithHandlerTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.HandlerTimeout = d
	}
}

/*
WithReply sets the reply function for request-reply consumers.
*/
func WithReply(r ReplyFunc) Option {
	return func(c *Config) {
		c.Reply = r
	}
}
