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
)

func main() {
	ctx := context.Background()

	// Configure structured logger
	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS server
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("pubsub-example"),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Close()

	// Create Consumer
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

	// Start subscription
	err = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		fmt.Println("received message:", string(data))
		return nil, nil
	})
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	// Create Core Producer
	prod, err := coreprod.NewCore(nc, "orders.created")
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	// Publish some messages
	for i := 1; i <= 5; i++ {
		msg := fmt.Sprintf(`{"order_id": "%d"}`, i)

		err = prod.Publish(ctx, []byte(msg))
		if err != nil {
			slog.Error("publish failed", "error", err)
			continue
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Keep process alive to receive messages
	time.Sleep(3 * time.Second)

	slog.Info("pub/sub example finished")

	// output:
	// received message: {"order_id": "1"}
	// received message: {"order_id": "2"}
	// received message: {"order_id": "3"}
	// received message: {"order_id": "4"}
	// received message: {"order_id": "5"}
	// 2026/02/14 09:44:31 INFO pub/sub example finished
}
