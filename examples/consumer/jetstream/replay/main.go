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

	nc, err := conn.Connect(nats.DefaultURL, conn.WithName("jetstream-replay-example"))
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

	// Create producer
	prod, _ := jsprod.NewJetStream(js, "orders.replay")

	// Publish messages BEFORE consumer starts
	for i := 1; i <= 5; i++ {
		msg := fmt.Sprintf(`{"order_id":"%d"}`, i)
		_ = prod.Publish(ctx, []byte(msg))
	}

	fmt.Println("messages published before consumer start")

	// Create consumer
	cons, _ := consumer.New(nc, logger)

	route, err := router.New(
		router.TypeJetStream,
		"orders.replay",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-replay-durable"),
		router.WithDeliveryPolicy(router.DeliverAllPolicy),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(5)

	// Start consumer AFTER messages were published
	_ = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		fmt.Println("replayed message:", string(data))
		wg.Done()
		return nil, nil
	})

	wg.Wait()

	fmt.Println("historical replay completed")

	// output:
	// messages published before consumer start
	// replayed message: {"order_id":"1"}
	// replayed message: {"order_id":"2"}
	// replayed message: {"order_id":"3"}
	// replayed message: {"order_id":"4"}
	// replayed message: {"order_id":"5"}
	// historical replay completed
}
