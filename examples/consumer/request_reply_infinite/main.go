package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/reply"
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
		conn.WithName("orders-reply-consumer"),
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

	// Create request reply route
	route, err := router.New(
		router.TypeRequestReply,
		"orders.calculate",
		router.WithReply(reply.JSON),
		router.WithQueueGroup("orders-processor"),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	slog.Info("request reply consumer started and listening...")

	err = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		slog.Info("received", "request", string(data))

		// Simulate processing
		time.Sleep(500 * time.Millisecond)

		response := json.RawMessage(fmt.Sprintf(`{"status":"processed","original":%s}`, string(data)))
		return response, nil
	})

	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	// Block forever until signal
	<-ctx.Done()

	slog.Info("shutting down consumer...")
}
