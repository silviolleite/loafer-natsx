package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	coreprod "github.com/silviolleite/loafer-natsx/producer"
	"github.com/silviolleite/loafer-natsx/router"
	"github.com/silviolleite/loafer-natsx/typed"
)

// Order represents a typed message contract.
type Order struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

func main() {
	ctx := context.Background()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("typed-consumer"),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Close()

	cons, err := consumer.New(nc, logger)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	route, err := router.New(
		router.TypePubSub,
		"orders.created",
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	codec := typed.JSONCodec[Order]{}

	// WrapHandler decodes the raw bytes into Order before calling the handler.
	// The context carries consumer.Metadata populated by the consumer layer —
	// use consumer.MetadataFromContext to access subject, headers, reply
	// subject, and JetStream-specific fields (stream, sequence, delivery
	// count, timestamp) without coupling the handler to any NATS message type.
	err = cons.Start(ctx, route, typed.WrapHandler(codec, func(ctx context.Context, msg Order) (any, error) {
		if meta, ok := consumer.MetadataFromContext(ctx); ok {
			correlationID := meta.Headers.Get(consumer.HeaderCorrelationIDKey)
			fmt.Printf("received order: %s (%.2f) | subject=%s correlation_id=%s\n",
				msg.OrderID, msg.Amount, meta.Subject, correlationID)
		} else {
			fmt.Printf("received order: %s (%.2f)\n", msg.OrderID, msg.Amount)
		}

		return nil, nil
	}))
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	strategy := coreprod.NewCoreStrategy(nc)
	prod, err := typed.NewProducer[Order](strategy, "orders.created", codec)
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	for i := 1; i <= 5; i++ {
		h := nats.Header{}
		h.Set(consumer.HeaderCorrelationIDKey, fmt.Sprintf("cid-%d", i))

		_, err = prod.Publish(ctx, Order{
			OrderID: fmt.Sprintf("%d", i),
			Amount:  float64(i) * 10.50,
		}, coreprod.PublishWithHeaders(h))
		if err != nil {
			slog.Error("publish failed", "error", err)
			continue
		}

		time.Sleep(500 * time.Millisecond)
	}

	time.Sleep(3 * time.Second)

	slog.Info("typed consumer example finished")

	// output:
	// received order: 1 (10.50) | subject=orders.created correlation_id=cid-1
	// received order: 2 (21.00) | subject=orders.created correlation_id=cid-2
	// received order: 3 (31.50) | subject=orders.created correlation_id=cid-3
	// received order: 4 (42.00) | subject=orders.created correlation_id=cid-4
	// received order: 5 (52.50) | subject=orders.created correlation_id=cid-5
	// 2026/02/14 10:00:00 INFO typed consumer example finished
}
