package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/producer"
)

func main() {
	ctx := context.Background()
	// Set the default logger for the entire application (optional)
	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(nats.DefaultURL,
		conn.WithName("orders-producer"),
		conn.WithReconnectWait(1*time.Second),
		conn.WithMaxReconnects(3),
		conn.WithTimeout(5*time.Second),
	)
	if err != nil {
		slog.Error("failed to connect to nats: %v", err)
		return
	}
	defer nc.Close()

	// Create producer
	prod, err := producer.NewCore(
		nc, "orders.new",
		producer.WithLogger(logger),
	)
	if err != nil {
		slog.Error("failed to create producer: %v", err)
		return
	}

	nMin := 1.0
	nMax := 1000.0
	for i := 1; i <= 20; i++ {
		data := fmt.Sprintf(`{"order_id": "%d", "amount": %.2f}`, i, nMin+rand.Float64()*(nMax-nMin))
		err = prod.Publish(ctx, []byte(data))
		if err != nil {
			slog.Error("publish failed: %v", err)
			continue
		}
	}

	slog.Info("done")
}
