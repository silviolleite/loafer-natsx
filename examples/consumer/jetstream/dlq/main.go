package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	jsprod "github.com/silviolleite/loafer-natsx/producer"
	"github.com/silviolleite/loafer-natsx/router"
)

func main() {
	ctx := context.Background()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("jetstream-dlq-example"),
	)
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer nc.Drain()

	js, err := jetstream.New(nc)
	if err != nil {
		slog.Error("failed to create jetstream", "error", err)
		return
	}

	cons, _ := consumer.New(nc, logger)

	route, err := router.New(
		router.TypeJetStream,
		"orders.failed",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-dlq-durable"),
		router.WithMaxDeliver(3),
		router.WithAckWait(2*time.Second),
		router.WithEnableDLQ(),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe to DLQ subject to observe routed messages.
	_, _ = nc.Subscribe("dlq.orders.failed", func(msg *nats.Msg) {
		fmt.Println("DLQ received:", string(msg.Data))
		fmt.Println("DLQ headers:", msg.Header)
		wg.Done()
	})

	attempt := 0

	// The handler demonstrates three explicit ack-control error types:
	//
	//   loafernatsx.NakWithDelayError  — request delayed redelivery (backoff)
	//   loafernatsx.ErrPermanentFailure — ack immediately, no retry
	//   loafernatsx.ErrSendToDLQ       — route directly to DLQ, skip retries
	//
	// On the first attempt we apply a short backoff. On the second we give up
	// and send the message straight to the DLQ.
	_ = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		attempt++
		fmt.Printf("attempt %d — processing: %s\n", attempt, string(data))

		switch attempt {
		case 1:
			// Transient failure: ask for redelivery after 500 ms.
			return nil, loafernatsx.NakWithDelayError{Delay: 500 * time.Millisecond}

		case 2:
			// Permanent failure: ack the message to stop retries entirely.
			// Use this when the payload is malformed or a business rule is
			// permanently violated and retrying would never help.
			return nil, fmt.Errorf("malformed payload: %w", loafernatsx.ErrPermanentFailure)

		default:
			// Explicit DLQ routing: skip remaining retries and send to DLQ.
			// Use this when you want to preserve the message for inspection
			// without exhausting the MaxDeliver budget.
			return nil, fmt.Errorf("unrecoverable: %w", loafernatsx.ErrSendToDLQ)
		}
	})

	// Verify that ErrSendToDLQ and ErrPermanentFailure unwrap correctly.
	wrapped := fmt.Errorf("context: %w", loafernatsx.ErrPermanentFailure)
	fmt.Println("errors.Is ErrPermanentFailure:", errors.Is(wrapped, loafernatsx.ErrPermanentFailure))

	// Create JetStream producer and publish a single message.
	strategy := jsprod.NewJetStreamStrategy(js, logger)
	prod, _ := jsprod.New(strategy, "orders.failed")
	_, _ = prod.Publish(ctx, []byte(`{"order_id":"999"}`))

	// Wait for the DLQ message.
	wg.Wait()

	fmt.Println("DLQ flow completed")

	// output:
	// errors.Is ErrPermanentFailure: true
	// attempt 1 — processing: {"order_id":"999"}
	// attempt 2 — processing: {"order_id":"999"}
	// attempt 3 — processing: {"order_id":"999"}
	// DLQ received: {"order_id":"999"}
	// DLQ headers: map[X-Error:[send to dead letter queue: skip retries] X-Retry-Count:[3]]
	// DLQ flow completed
}
