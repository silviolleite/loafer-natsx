package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/conn"
	coreProducer "github.com/silviolleite/loafer-natsx/producer/core"
)

func main() {
	ctx := context.Background()
	// Set the default logger for the entire application (optional)
	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(nats.DefaultURL,
		conn.WithName("orders-coreProducer"),
		conn.WithReconnectWait(1*time.Second),
		conn.WithMaxReconnects(3),
		conn.WithTimeout(5*time.Second),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Close()

	// Create Core Producer
	prod, err := coreProducer.New(
		nc, "orders.new",
		coreProducer.WithLogger(logger),
	)
	if err != nil {
		slog.Error("failed to create coreProducer", "error", err)
		return
	}

	nMin := 1.0
	nMax := 1000.0
	for i := 1; i <= 20; i++ {
		data := fmt.Sprintf(`{"order_id": "%d", "amount": %.2f}`, i, nMin+rand.Float64()*(nMax-nMin))
		err = prod.Publish(ctx, []byte(data), coreProducer.PublishWithHeaders(nats.Header{
			"order-type": []string{"new"},
		}))
		if err != nil {
			slog.Error("publish failed", "error", err)
			continue
		}
	}

	slog.Info("done")

	// output:
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=34 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=34 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=34 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=36 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=36 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=36 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=36 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=36 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=36 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=36 headers_count=1
	// 2026/02/13 11:01:44 DEBUG publishing message subject=orders.new nats_core=true payload_bytes=35 headers_count=1
	// 2026/02/13 11:01:44 INFO done
}
