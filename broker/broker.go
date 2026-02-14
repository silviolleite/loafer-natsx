package broker

import (
	"context"
	"sync"

	"github.com/nats-io/nats.go"
	loafernatsx "github.com/silviolleite/loafer-natsx"

	"github.com/silviolleite/loafer-natsx/consumer"

	"github.com/silviolleite/loafer-natsx/logger"
)

const defaultWorkers = 5

// Broker represents a message broker that coordinates message routing and processing using NATS and configurable workers.
type Broker struct {
	nc      *nats.Conn
	log     logger.Logger
	workers int
}

// New creates a new Broker instance with the given NATS connection, logger, and optional configuration options.
func New(nc *nats.Conn, log logger.Logger, opts ...Option) *Broker {
	cfg := config{
		workers: defaultWorkers,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if log == nil {
		log = logger.NopLogger{}
	}

	return &Broker{
		nc:      nc,
		log:     log,
		workers: cfg.workers,
	}
}

// Run starts routing and processing messages using the provided handlers and waits for completion or error detection.
func (b *Broker) Run(ctx context.Context, regs ...*RouteRegistration) error {
	if len(regs) == 0 {
		return loafernatsx.ErrNoRoutes
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)

	for _, reg := range regs {
		if reg == nil {
			return loafernatsx.ErrNilRouteRegistration
		}

		go func(r *RouteRegistration) {
			if err := b.runRoute(ctx, r); err != nil {
				select {
				case errCh <- err:
				default:
				}
				cancel()
			}
		}(reg)
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

func (b *Broker) runRoute(
	ctx context.Context,
	reg *RouteRegistration,
) error {
	cons, err := consumer.New(b.nc, b.log)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for i := 0; i < b.workers; i++ {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			sErr := cons.Start(ctx, reg.Route(), reg.Handler())
			if sErr != nil {
				b.log.Error(
					"route worker failed",
					"subject", reg.Route().Subject(),
					"worker", workerID,
					"error", sErr,
				)
			}
		}(i)
	}

	wg.Wait()
	return nil
}
