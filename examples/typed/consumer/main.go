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

	// Connect to NATS
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("typed-consumer"),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Close()

	// Create consumer
	cons, err := consumer.New(nc, logger)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	// Create Pub/Sub route
	route, err := router.New(
		router.TypePubSub,
		"orders.created",
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	codec := typed.JSONCodec[Order]{}

	// Start subscription with typed handler
	err = cons.Start(ctx, route, typed.WrapHandler(codec, func(ctx context.Context, msg Order) (any, error) {
		fmt.Printf("received order: %s (%.2f)\n", msg.OrderID, msg.Amount)
		return nil, nil
	}))
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	// Create typed producer to publish some messages
	strategy := coreprod.NewCoreStrategy(nc)
	prod, err := typed.NewProducer[Order](strategy, "orders.created", codec)
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	// Publish typed messages
	for i := 1; i <= 5; i++ {
		err = prod.Publish(ctx, Order{
			OrderID: fmt.Sprintf("%d", i),
			Amount:  float64(i) * 10.50,
		})
		if err != nil {
			slog.Error("publish failed", "error", err)
			continue
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Keep process alive to receive messages
	time.Sleep(3 * time.Second)

	slog.Info("typed consumer example finished")

	// output:
	// received order: 1 (10.50)
	// received order: 2 (21.00)
	// received order: 3 (31.50)
	// received order: 4 (42.00)
	// received order: 5 (52.50)
	// 2026/02/14 10:00:00 INFO typed consumer example finished
}
