package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/router"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("orders-consumer"),
		conn.WithReconnectWait(2*time.Second),
		conn.WithMaxReconnects(-1),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Drain()

	cons, err := consumer.New(nc, log)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	route, err := router.New(
		router.TypeJetStream,
		"orders.new",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-consumer-durable"),
		router.WithDeliveryPolicy(router.DeliverNewPolicy),
		router.WithAckWait(30*time.Second),
		router.WithMaxDeliver(5),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	slog.Info("jetstream consumer started and listening...")

	err = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		// Access message metadata (subject, headers, stream, sequence, etc.)
		// without coupling the handler to the underlying NATS message type.
		if meta, ok := consumer.MetadataFromContext(ctx); ok {
			slog.Info("message received",
				"subject", meta.Subject,
				"stream", meta.Stream,
				"sequence", meta.Sequence,
				"delivery", meta.NumDelivered,
				"correlation_id", meta.Headers.Get(consumer.HeaderCorrelationIDKey),
				"payload", string(data),
			)
		}

		// Simulate a transient dependency failure (e.g. downstream service
		// temporarily unavailable). NakWithDelayError requests redelivery
		// after a specific backoff duration instead of an immediate retry,
		// reducing pressure on the failing dependency.
		//
		// The error can be returned directly or wrapped:
		//   return nil, loafernatsx.NakWithDelayError{Delay: 10 * time.Second}
		//   return nil, fmt.Errorf("db unavailable: %w", loafernatsx.NakWithDelayError{Delay: 30 * time.Second})
		//
		// For non-retryable failures use ErrPermanentFailure (acks the message):
		//   return nil, fmt.Errorf("bad payload: %w", loafernatsx.ErrPermanentFailure)
		//
		// To skip retries and route directly to DLQ use ErrSendToDLQ:
		//   return nil, fmt.Errorf("unrecoverable: %w", loafernatsx.ErrSendToDLQ)

		_ = loafernatsx.NakWithDelayError{} // imported for documentation purposes

		time.Sleep(200 * time.Millisecond)

		return nil, nil
	})
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	<-ctx.Done()

	slog.Info("shutting down consumer...")
}
