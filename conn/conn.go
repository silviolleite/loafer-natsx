package conn

import (
	"time"

	"github.com/nats-io/nats.go"
)

/*
Config holds the configuration for establishing a NATS connection.
*/
type Config struct {
	URL           string
	Name          string
	ReconnectWait time.Duration
	MaxReconnects int
	Timeout       time.Duration
}

/*
Connect establishes a connection to a NATS server and returns a *nats.Conn.
*/
func Connect(cfg Config) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name(cfg.Name),
		nats.Timeout(cfg.Timeout),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, err
	}

	return nc, nil
}
