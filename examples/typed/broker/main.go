package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/broker"
	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/router"
	"github.com/silviolleite/loafer-natsx/typed"
)

// Order represents a typed message contract.
type Order struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("typed-broker"),
		conn.WithMaxReconnects(-1),
	)
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer nc.Drain()

	codec := typed.JSONCodec[Order]{}

	// Create routes
	createdRoute, err := router.New(
		router.TypeJetStream,
		"orders.created",
		router.WithStream("ORDERS"),
		router.WithDurable("typed-orders-created"),
		router.WithDeliveryPolicy(router.DeliverNewPolicy),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	cancelledRoute, err := router.New(
		router.TypeJetStream,
		"orders.cancelled",
		router.WithStream("ORDERS"),
		router.WithDurable("typed-orders-cancelled"),
		router.WithDeliveryPolicy(router.DeliverNewPolicy),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	// Create broker
	br := broker.New(nc, log, broker.WithWorkers(2))

	// Register routes with typed handlers
	r1, _ := broker.NewRouteRegistration(
		createdRoute,
		typed.WrapHandler(codec, func(ctx context.Context, msg Order) (any, error) {
			fmt.Printf("order created: %s (%.2f)\n", msg.OrderID, msg.Amount)
			return nil, nil
		}),
	)

	r2, _ := broker.NewRouteRegistration(
		cancelledRoute,
		typed.WrapHandler(codec, func(ctx context.Context, msg Order) (any, error) {
			fmt.Printf("order cancelled: %s\n", msg.OrderID)
			return nil, nil
		}),
	)

	slog.Info("typed broker started with 2 routes")

	if err = br.Run(ctx, r1, r2); err != nil {
		slog.Error("broker stopped due to error", "error", err)
	}

	slog.Info("typed broker shutdown complete")

	// output:
	// 2026/02/14 10:00:00 INFO typed broker started with 2 routes
	// order created: 1 (99.90)
	// order cancelled: 2
	// 2026/02/14 10:00:05 INFO typed broker shutdown complete
}
