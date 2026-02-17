package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/silviolleite/loafer-natsx/broker"
	"github.com/silviolleite/loafer-natsx/conn"
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

	// Connect to NATS
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("orders-broker"),
		conn.WithMaxReconnects(-1),
	)
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer nc.Drain()

	// Create routes
	createdRoute, err := router.New(
		router.TypeJetStream,
		"orders.created",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-created-durable"),
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
		router.WithDurable("orders-cancelled-durable"),
		router.WithDeliveryPolicy(router.DeliverNewPolicy),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	// Create broker
	br := broker.New(
		nc,
		log,
		broker.WithWorkers(2),
	)

	// Register routes

	r1, _ := broker.NewRouteRegistration(
		createdRoute,
		func(ctx context.Context, data []byte) (any, error) {
			slog.Info("order created",
				"payload", string(data),
			)
			time.Sleep(200 * time.Millisecond)
			return nil, nil
		},
	)

	r2, _ := broker.NewRouteRegistration(
		cancelledRoute,
		func(ctx context.Context, data []byte) (any, error) {
			slog.Info("order cancelled",
				"payload", string(data),
			)
			time.Sleep(150 * time.Millisecond)
			return nil, nil
		},
	)

	slog.Info("broker started with 2 routes")

	err = br.Run(ctx, r1, r2)
	if err != nil {
		slog.Error("broker stopped due to error", "error", err)
	}

	slog.Info("broker shutdown complete")
}
