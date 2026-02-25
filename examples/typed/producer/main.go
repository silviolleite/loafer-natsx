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
	"github.com/silviolleite/loafer-natsx/typed"
)

// Order represents a typed message contract.
type Order struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

func main() {
	ctx := context.Background()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(nats.DefaultURL,
		conn.WithName("typed-producer"),
		conn.WithReconnectWait(1*time.Second),
		conn.WithMaxReconnects(3),
		conn.WithTimeout(5*time.Second),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Close()

	// Create typed producer
	strategy := producer.NewCoreStrategy(nc)
	codec := typed.JSONCodec[Order]{}

	prod, err := typed.NewProducer[Order](
		strategy, "orders.new", codec,
		producer.WithLogger(slog.Default()),
	)
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	// Publish typed messages
	for i := 1; i <= 5; i++ {
		err = prod.Publish(ctx, Order{
			OrderID: fmt.Sprintf("%d", i),
			Amount:  1.0 + rand.Float64()*999.0,
		}, producer.PublishWithHeaders(nats.Header{
			"order-type": []string{"new"},
		}))
		if err != nil {
			slog.Error("publish failed", "error", err)
			continue
		}
	}

	slog.Info("done")

	// output:
	// 2026/02/16 10:58:48 DEBUG publishing message subject=orders.new payload_bytes=38 headers_count=1
	// 2026/02/16 10:58:48 DEBUG publishing message subject=orders.new payload_bytes=38 headers_count=1
	// 2026/02/16 10:58:48 DEBUG publishing message subject=orders.new payload_bytes=38 headers_count=1
	// 2026/02/16 10:58:48 DEBUG publishing message subject=orders.new payload_bytes=38 headers_count=1
	// 2026/02/16 10:58:48 DEBUG publishing message subject=orders.new payload_bytes=38 headers_count=1
	// 2026/02/16 10:58:48 INFO done
}
