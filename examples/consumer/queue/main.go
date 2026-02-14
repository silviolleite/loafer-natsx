package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	coreprod "github.com/silviolleite/loafer-natsx/producer/core"
	"github.com/silviolleite/loafer-natsx/router"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(nats.DefaultURL, conn.WithName("queue-example"))
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer nc.Drain()

	// Create two consumers in the same queue group
	cons1, _ := consumer.New(nc, logger)
	cons2, _ := consumer.New(nc, logger)

	route, err := router.New(
		router.TypeQueue,
		"orders.queue",
		router.WithQueueGroup("workers"),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(10)

	handler := func(name string) consumer.HandlerFunc {
		return func(ctx context.Context, data []byte) (any, error) {
			fmt.Printf("[%s] processed: %s\n", name, string(data))
			wg.Done()
			return nil, nil
		}
	}

	// Start both consumers
	_ = cons1.Start(ctx, route, handler("consumer-1"))
	_ = cons2.Start(ctx, route, handler("consumer-2"))

	// Create producer
	prod, err := coreprod.New(nc, "orders.queue")
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	// Publish messages
	for i := 1; i <= 10; i++ {
		msg := fmt.Sprintf(`{"order_id": "%d"}`, i)
		_ = prod.Publish(ctx, []byte(msg))
	}

	// Wait until all messages are processed
	wg.Wait()

	fmt.Println("all messages processed, exiting")

	// output:
	// [consumer-2] processed: {"order_id": "1"}
	// [consumer-2] processed: {"order_id": "2"}
	// [consumer-2] processed: {"order_id": "7"}
	// [consumer-1] processed: {"order_id": "3"}
	// [consumer-1] processed: {"order_id": "4"}
	// [consumer-1] processed: {"order_id": "5"}
	// [consumer-1] processed: {"order_id": "6"}
	// [consumer-1] processed: {"order_id": "9"}
	// [consumer-1] processed: {"order_id": "10"}
	// [consumer-2] processed: {"order_id": "8"}
	// all messages processed, exiting
}

