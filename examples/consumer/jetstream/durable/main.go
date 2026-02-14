package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	jsprod "github.com/silviolleite/loafer-natsx/producer/jetstream"
	"github.com/silviolleite/loafer-natsx/router"
	"github.com/silviolleite/loafer-natsx/stream"
)

func main() {
	ctx := context.Background()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("jetstream-durable-example"),
	)
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer nc.Drain()

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		slog.Error("failed to create jetstream", "error", err)
		return
	}

	// Ensure stream exists
	err = stream.Ensure(
		ctx,
		js,
		"ORDERS",
		stream.WithSubjects("orders.created"),
		stream.WithMaxAge(24*time.Hour),
	)
	if err != nil {
		slog.Error("failed to ensure stream", "error", err)
		return
	}

	// Create consumer engine
	cons, err := consumer.New(nc, logger)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	// Create durable JetStream route
	route, err := router.New(
		router.TypeJetStream,
		"orders.created",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-durable"),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(5)

	// Start consumer
	err = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		fmt.Println("durable consumer received:", string(data))
		wg.Done()
		return nil, nil
	})
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	// Create JetStream producer
	prod, err := jsprod.New(js, "orders.created")
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	// Publish messages
	for i := 1; i <= 5; i++ {
		msg := fmt.Sprintf(`{"order_id":"%d"}`, i)
		err = prod.Publish(ctx, []byte(msg))
		if err != nil {
			slog.Error("publish failed", "error", err)
			continue
		}
	}

	// Wait for all messages to be processed
	wg.Wait()

	fmt.Println("all messages processed, exiting")

	// output:
	// durable consumer received: {"order_id":"1"}
	// durable consumer received: {"order_id":"2"}
	// durable consumer received: {"order_id":"3"}
	// durable consumer received: {"order_id":"4"}
	// durable consumer received: {"order_id":"5"}
	// all messages processed, exiting
}
