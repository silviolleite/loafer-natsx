package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/producer"
	"github.com/silviolleite/loafer-natsx/stream"
)

func main() {
	ctx := context.Background()
	// Set the default logger for the entire application (optional)
	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Connect to NATS
	nc, err := conn.Connect(nats.DefaultURL,
		conn.WithName("orders-jetstream-producer"),
		conn.WithReconnectWait(1*time.Second),
		conn.WithMaxReconnects(3),
		conn.WithTimeout(5*time.Second),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Close()

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		slog.Error("failed to create jetstream", "error", err)
		return
	}

	err = stream.Ensure(
		ctx, js, "ORDERS",
		stream.WithSubjects("orders.new"),
		stream.WithRetention(jetstream.LimitsPolicy),
		stream.WithMaxAge(7*24*time.Hour),
		// Set a custom deduplication window of 1 minute
		// Default is 2 minutes
		// stream.WithDuplicateWindow(1 * time.Minute),
	)
	if err != nil {
		slog.Error("failed to ensure stream", "error", err)
		return
	}

	// Create producer
	prod, err := producer.NewJetStream(
		js, "orders.new",
		producer.WithLogger(logger),
	)
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	nMin := 1.0
	nMax := 1000.0
	for i := 1; i <= 5; i++ {
		data := fmt.Sprintf(`{"order_id": "%d", "amount": %.2f}`, i, nMin+rand.Float64()*(nMax-nMin))
		// // Publish without deduplication and custom headers
		// err = prod.Publish(ctx, []byte(data))
		// Publish with message ID (deduplication) and custom headers
		err = prod.Publish(
			ctx, []byte(data),
			// WithMsgID sets the JetStream deduplication ID for the message
			// It will use the Deduplication window configured in the stream or default to 2 minutes
			producer.PublishWithMsgID(fmt.Sprintf("%d", i)),
			producer.PublishWithHeaders(nats.Header{
				"MY-CUSTOM-READER": []string{"my-custom-value"},
			}),
		)
		if err != nil {
			slog.Error("publish failed", "error", err)
			continue
		}
	}

	slog.Info("done")
	// output:
	// 2026/02/16 10:57:34 DEBUG publishing message subject=orders.new payload_bytes=35 headers_count=1
	// 2026/02/16 10:57:34 INFO jetstream deduplication enabled subject=orders.new msg_id=1 note="duplicate window defined by stream (default 2m)"
	// 2026/02/16 10:57:34 DEBUG jetstream publish acknowledged stream=ORDERS sequence=1 duplicate=false
	// 2026/02/16 10:57:34 DEBUG publishing message subject=orders.new payload_bytes=34 headers_count=1
	// 2026/02/16 10:57:34 INFO jetstream deduplication enabled subject=orders.new msg_id=2 note="duplicate window defined by stream (default 2m)"
	// 2026/02/16 10:57:34 DEBUG jetstream publish acknowledged stream=ORDERS sequence=2 duplicate=false
	// 2026/02/16 10:57:34 DEBUG publishing message subject=orders.new payload_bytes=35 headers_count=1
	// 2026/02/16 10:57:34 INFO jetstream deduplication enabled subject=orders.new msg_id=3 note="duplicate window defined by stream (default 2m)"
	// 2026/02/16 10:57:34 DEBUG jetstream publish acknowledged stream=ORDERS sequence=3 duplicate=false
	// 2026/02/16 10:57:34 DEBUG publishing message subject=orders.new payload_bytes=35 headers_count=1
	// 2026/02/16 10:57:34 INFO jetstream deduplication enabled subject=orders.new msg_id=4 note="duplicate window defined by stream (default 2m)"
	// 2026/02/16 10:57:34 DEBUG jetstream publish acknowledged stream=ORDERS sequence=4 duplicate=false
	// 2026/02/16 10:57:34 DEBUG publishing message subject=orders.new payload_bytes=35 headers_count=1
	// 2026/02/16 10:57:34 INFO jetstream deduplication enabled subject=orders.new msg_id=5 note="duplicate window defined by stream (default 2m)"
	// 2026/02/16 10:57:34 DEBUG jetstream publish acknowledged stream=ORDERS sequence=5 duplicate=false
	// 2026/02/16 10:57:34 INFO done
}
