package router

import (
	"time"

	loafernastx "github.com/silviolleite/loafer-natsx"
)

const (
	defaultMaxDeliveries = 10
	defaultAckWait       = 30 * time.Second
)

// Route represents a message consumption route definition.
type Route struct {
	cfg       *config
	routeType Type
}

// Type returns the router type.
func (r *Route) Type() Type {
	return r.routeType
}

// Subject returns the route subject.
func (r *Route) Subject() string {
	return r.cfg.subject
}

// QueueGroup returns the queue group if configured.
func (r *Route) QueueGroup() string {
	return r.cfg.queueGroup
}

// Stream returns the JetStream stream name.
func (r *Route) Stream() string {
	return r.cfg.stream
}

// Durable returns the JetStream durable name.
func (r *Route) Durable() string {
	return r.cfg.durable
}

// AckWait returns the JetStream acknowledgment wait duration.
func (r *Route) AckWait() time.Duration {
	return r.cfg.ackWait
}

// MaxDeliver returns the JetStream max delivery attempts.
func (r *Route) MaxDeliver() int {
	return r.cfg.maxDeliver
}

// DLQEnabled indicates whether DLQ is enabled.
func (r *Route) DLQEnabled() bool {
	return r.cfg.enableDLQ
}

// ReplyFunc returns the reply function if configured.
func (r *Route) ReplyFunc() ReplyFunc {
	return r.cfg.reply
}

// HandlerTimeout returns the configured handler timeout.
func (r *Route) HandlerTimeout() time.Duration {
	return r.cfg.handlerTimeout
}

// DeliveryPolicy returns the configured delivery policy for consuming messages from the stream.
func (r *Route) DeliveryPolicy() DeliverPolicy {
	return r.cfg.deliveryPolicy
}

// New creates a validated Route definition applying default values when necessary.
func New(routeType Type, subject string, opts ...Option) (*Route, error) {
	cfg := &config{
		subject: subject,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.subject == "" {
		return nil, loafernastx.ErrMissingSubject
	}

	switch routeType {
	case RouteTypeQueue:
		if cfg.queueGroup == "" {
			return nil, loafernastx.ErrMissingQueueGroup
		}

	case RouteTypeJetStream:
		if cfg.stream == "" {
			return nil, loafernastx.ErrMissingStream
		}
		if cfg.durable == "" {
			return nil, loafernastx.ErrMissingDurable
		}
		if cfg.maxDeliver == 0 {
			cfg.maxDeliver = defaultMaxDeliveries
		}
		if cfg.ackWait == 0 {
			cfg.ackWait = defaultAckWait
		}

	case RouteTypeRequestReply:
		// nothing required beyond subject

	case RouteTypePubSub:
		// nothing required beyond subject

	default:
		return nil, loafernastx.ErrUnsupportedType
	}

	return &Route{
		routeType: routeType,
		cfg:       cfg,
	}, nil
}
