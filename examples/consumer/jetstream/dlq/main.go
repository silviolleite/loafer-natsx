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
	jsprod "github.com/silviolleite/loafer-natsx/producer"
	"github.com/silviolleite/loafer-natsx/router"
	"github.com/silviolleite/loafer-natsx/stream"
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

	// Ensure stream
	err = stream.Ensure(
		ctx,
		js,
		"ORDERS",
		stream.WithSubjects("orders.failed"),
		stream.WithMaxAge(24*time.Hour),
	)
	if err != nil {
		slog.Error("failed to ensure stream", "error", err)
		return
	}

	// Consumer engine
	cons, _ := consumer.New(nc, logger)

	route, err := router.New(
		router.TypeJetStream,
		"orders.failed",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-dlq-durable"),
		router.WithMaxDeliver(3),
		router.WithAckWait(2*time.Second),
		router.WithEnableDLQ(true),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe to DLQ subject
	_, _ = nc.Subscribe("dlq.orders.failed", func(msg *nats.Msg) {
		fmt.Println("DLQ received:", string(msg.Data))
		fmt.Println("DLQ headers:", msg.Header)
		wg.Done()
	})

	// Start failing consumer
	_ = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		fmt.Println("processing message:", string(data))
		return nil, fmt.Errorf("simulated processing failure")
	})

	// Create JetStream producer
	prod, _ := jsprod.NewJetStream(js, "orders.failed")

	// Publish a single message
	_ = prod.Publish(ctx, []byte(`{"order_id":"999"}`))

	// Wait for DLQ message
	wg.Wait()

	fmt.Println("DLQ flow completed")

	// output:
	// processing message: {"order_id":"999"}
	// 2026/02/14 10:17:23 ERROR handler error subject=orders.failed error="simulated processing failure"
	// processing message: {"order_id":"999"}
	// 2026/02/14 10:17:23 ERROR handler error subject=orders.failed error="simulated processing failure"
	// processing message: {"order_id":"999"}
	// 2026/02/14 10:17:23 ERROR handler error subject=orders.failed error="simulated processing failure"
	// DLQ received: {"order_id":"999"}
	// DLQ headers: map[X-Error:[simulated processing failure] X-Retry-Count:[3]]
	// DLQ flow completed
}
