package conn

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx"
)

const (
	defaultReconnectWait = 2 * time.Second
	defaultMaxReconnects = -1
	defaultTimeout       = 5 * time.Second
)

// Connect establishes a connection to a NATS server and returns a *nats.Conn.
func Connect(url string, opts ...Option) (*nats.Conn, error) {
	if url == "" {
		return nil, loafernastx.ErrMissingURL
	}

	cfg := config{
		name:          "loafer-natsx-" + randomSuffix(),
		reconnectWait: defaultReconnectWait,
		maxReconnects: defaultMaxReconnects,
		timeout:       defaultTimeout,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	options := []nats.Option{
		nats.Name(cfg.name),
		nats.Timeout(cfg.timeout),
		nats.MaxReconnects(cfg.maxReconnects),
		nats.ReconnectWait(cfg.reconnectWait),
	}

	nc, err := nats.Connect(url, options...)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

// randomSuffix generates a short random hex string.
func randomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
