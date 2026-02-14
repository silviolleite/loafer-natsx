package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/router"
)

func main() {
	// Graceful shutdown context
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
		conn.WithName("orders-consumer"),
		conn.WithReconnectWait(2*time.Second),
		conn.WithMaxReconnects(-1),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Drain()

	// Create consumer engine
	cons, err := consumer.New(nc, log)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	// Create durable JetStream route
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
		slog.Info("message received",
			"payload", string(data),
		)

		// // Simulate processing
		// time.Sleep(200 * time.Millisecond)

		return nil, nil
	})

	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	// Block forever until signal
	<-ctx.Done()

	slog.Info("shutting down consumer...")
}
