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
	"github.com/silviolleite/loafer-natsx/reply"
	"github.com/silviolleite/loafer-natsx/router"
	"github.com/silviolleite/loafer-natsx/typed"
)

// Order represents a typed request message.
type Order struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

// ProcessedOrder represents the response after processing.
type ProcessedOrder struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
}

func main() {
	ctx := context.Background()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("typed-request-reply"),
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

	// Create request-reply route with typed reply function
	replyFn := typed.WrapReply(func(ctx context.Context, result ProcessedOrder, handlerErr error) ([]byte, nats.Header, error) {
		return reply.JSON(ctx, result, handlerErr)
	})

	route, err := router.New(
		router.TypeRequestReply,
		"orders.process",
		router.WithReply(replyFn),
		router.WithQueueGroup("orders-processor"),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	codec := typed.JSONCodec[Order]{}

	// Start consumer with typed handler
	err = cons.Start(ctx, route, typed.WrapHandler(codec, func(ctx context.Context, msg Order) (any, error) {
		fmt.Printf("received request: order %s (%.2f)\n", msg.OrderID, msg.Amount)

		return ProcessedOrder{
			Status:  "processed",
			OrderID: msg.OrderID,
		}, nil
	}))
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	// Create typed requester with both request and response codecs
	strategy := coreprod.NewCoreStrategy(nc)
	reqCodec := typed.JSONCodec[Order]{}
	resCodec := typed.JSONCodec[ProcessedOrder]{}
	requester, err := typed.NewRequester[Order, ProcessedOrder](strategy, "orders.process", reqCodec, resCodec)
	if err != nil {
		slog.Error("failed to create requester", "error", err)
		return
	}

	// Perform typed request — response is automatically decoded
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	result, err := requester.Request(reqCtx, Order{OrderID: "123", Amount: 99.90})
	if err != nil {
		slog.Error("request failed", "error", err)
		return
	}

	fmt.Printf("received reply: %s (order %s)\n", result.Status, result.OrderID)

	// output:
	// received request: order 123 (99.90)
	// received reply: processed (order 123)
}
