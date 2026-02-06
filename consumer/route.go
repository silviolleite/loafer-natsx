package consumer

import (
	"time"

	"github.com/silviolleite/loafer-natsx"
)

const (
	defaultMaxDeliveries = 10
	defaultAckWait       = 30 * time.Second
)

/*
Route represents a message consumption route definition.
*/
type Route struct {
	Type   Type
	Config *Config
}

/*
NewRoute creates a validated Route definition applying default values when necessary.
*/
func NewRoute(t Type, subject string, opts ...Option) (*Route, error) {
	cfg := &Config{
		Subject: subject,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.Subject == "" {
		return nil, loafernastx.ErrMissingSubject
	}

	switch t {

	case TypeQueue:
		if cfg.QueueGroup == "" {
			return nil, loafernastx.ErrMissingQueueGroup
		}

	case TypeJetStream:
		if cfg.Stream == "" {
			return nil, loafernastx.ErrMissingStream
		}
		if cfg.Durable == "" {
			return nil, loafernastx.ErrMissingDurable
		}
		if cfg.MaxDeliver == 0 {
			cfg.MaxDeliver = defaultMaxDeliveries
		}
		if cfg.AckWait == 0 {
			cfg.AckWait = defaultAckWait
		}

	case TypeRequestReply:
		// nothing required beyond subject

	case TypePubSub:
		// nothing required beyond subject

	default:
		return nil, loafernastx.ErrUnsupportedType
	}

	return &Route{
		Type:   t,
		Config: cfg,
	}, nil
}
