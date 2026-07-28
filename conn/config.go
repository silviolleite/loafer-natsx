package conn

import (
	"crypto/tls"
	"time"
)

type config struct {
	tls           *tls.Config
	name          string
	reconnectWait time.Duration
	maxReconnects int
	timeout       time.Duration
}
