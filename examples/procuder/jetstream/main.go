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
	producer "github.com/silviolleite/loafer-natsx/producer/jetstream"
	"github.com/silviolleite/loafer-natsx/stream"
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
	prod, err := producer.New(
		js, "orders.new",
		producer.WithLogger(logger),
	)
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	nMin := 1.0
	nMax := 1000.0
	for i := 1; i <= 20; i++ {
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
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=1 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=1 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=21 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=2 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=2 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=22 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=3 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=3 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=23 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=4 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=4 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=24 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=5 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=5 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=25 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=6 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=6 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=26 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=7 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=7 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=27 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=8 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=8 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=28 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=9 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=9 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=29 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=10 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=10 payload_bytes=36 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=30 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=11 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=11 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=31 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=12 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=12 payload_bytes=36 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=32 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=13 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=13 payload_bytes=36 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=33 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=14 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=14 payload_bytes=36 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=34 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=15 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=15 payload_bytes=36 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=35 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=16 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=16 payload_bytes=36 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=36 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=17 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=17 payload_bytes=36 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=37 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=18 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=18 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=38 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=19 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=19 payload_bytes=36 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=39 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO publishing with msg_id (JetStream deduplication enabled) msg_id=20 dedup_window_default="2m (if not configured in stream)"
	// 2026/02/13 14:12:26 DEBUG publishing message subject=orders.new jetstream=true msg_id=20 payload_bytes=35 headers_count=1
	// 2026/02/13 14:12:26 DEBUG published message ack_sequence=40 ack_stream=ORDERS
	// 2026/02/13 14:12:26 INFO done
}
