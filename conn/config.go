package conn

import "time"

type config struct {
	name          string
	reconnectWait time.Duration
	maxReconnects int
	timeout       time.Duration
}
