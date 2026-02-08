package producer

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
