package broker

type config struct {
	workers int
}

// Option is a function type used to modify the configuration of a component by applying changes to a config instance.
type Option func(*config)

// WithWorkers returns an Option to configure the number of workers in the config. It sets the value if n is greater than 0.
func WithWorkers(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.workers = n
		}
	}
}
