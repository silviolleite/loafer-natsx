package broker

import (
	"context"
	"time"

	"github.com/silviolleite/loafer-natsx/consumer"
)

func (b *Broker) instrument(
	reg *RouteRegistration,
) consumer.HandlerFunc {
	handler := reg.Handler()
	subject := reg.Route().Subject()

	// No metrics configured → zero overhead wrapper
	if b.metrics == nil {
		return handler
	}

	return func(ctx context.Context, data []byte) (any, error) {
		start := time.Now()

		b.metrics.inflightInc()
		defer b.metrics.inflightDec()

		res, err := handler(ctx, data)

		b.metrics.observeDuration(subject, time.Since(start))

		if err != nil {
			b.metrics.incError(subject)
			return nil, err
		}

		b.metrics.incRequest(subject)
		return res, nil
	}
}
