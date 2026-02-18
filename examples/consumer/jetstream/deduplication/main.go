package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	jsprod "github.com/silviolleite/loafer-natsx/producer"
	"github.com/silviolleite/loafer-natsx/router"
)

func main() {
	ctx := context.Background()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("jetstream-dedup-example"),
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

	// Create JetStream producer
	strategy := jsprod.NewJetStreamStrategy(js, logger)
	prod, err := jsprod.New(strategy, "orders.dedup")
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	msgID := "order-123"

	fmt.Println("Publishing first message...")

	err = prod.Publish(
		ctx,
		[]byte(`{"order_id":"123"}`),
		jsprod.PublishWithMsgID(msgID),
	)
	if err != nil {
		slog.Error("publish failed", "error", err)
		return
	}

	fmt.Println("Publishing duplicate message with same MsgID...")

	err = prod.Publish(
		ctx,
		[]byte(`{"order_id":"123"}`),
		jsprod.PublishWithMsgID(msgID),
	)
	if err != nil {
		slog.Error("publish failed", "error", err)
		return
	}

	// Create consumer to verify how many messages exist
	cons, err := consumer.New(nc, logger)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	route, err := router.New(
		router.TypeJetStream,
		"orders.dedup",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-dedup-durable"),
		router.WithDeliveryPolicy(router.DeliverAllPolicy),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	count := 0

	err = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		count++
		fmt.Println("Consumer received:", string(data))
		wg.Done()
		return nil, nil
	})
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	wg.Wait()

	fmt.Println("Total messages stored in stream:", count)
	fmt.Println("Deduplication demonstration completed")

	// output:
	// Publishing first message...
	// Publishing duplicate message with same MsgID...
	// Consumer received: {"order_id":"123"}
	// Total messages stored in stream: 1
	// Deduplication demonstration completed
}
