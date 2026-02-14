package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	coreprod "github.com/silviolleite/loafer-natsx/producer/core"
	"github.com/silviolleite/loafer-natsx/router"
)

func main() {
	ctx := context.Background()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("request-reply-example"),
	)
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer nc.Drain()

	// Create consumer
	cons, err := consumer.New(nc, logger)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	// Create request-reply route
	route, err := router.New(
		router.TypeRequestReply,
		"orders.process",
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	// Start consumer
	err = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		fmt.Println("received request:", string(data))

		// Simulate processing
		time.Sleep(500 * time.Millisecond)

		response := fmt.Sprintf(`{"status":"processed","original":%s}`, string(data))
		return []byte(response), nil
	})
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	// Create producer
	prod, err := coreprod.New(nc, "orders.process")
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	// Perform request
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	requestPayload := []byte(`{"order_id":"123"}`)

	reply, err := prod.Request(reqCtx, requestPayload)
	if err != nil {
		slog.Error("request failed", "error", err)
		return
	}

	fmt.Println("received reply:", string(reply))

	// output:
	// received request: {"order_id":"123"}
	// received reply: ok
}
