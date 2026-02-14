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

	slog.SetLogLoggerLevel(slog.LevelDebug)
	logger := slog.Default()

	// Connect to NATS
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("broker-example-producer"),
		conn.WithReconnectWait(1*time.Second),
		conn.WithMaxReconnects(-1),
		conn.WithTimeout(5*time.Second),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Drain()

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		slog.Error("failed to create jetstream", "error", err)
		return
	}

	// Ensure stream with both subjects
	err = stream.Ensure(
		ctx,
		js,
		"ORDERS",
		stream.WithSubjects("orders.created", "orders.cancelled"),
		stream.WithRetention(jetstream.LimitsPolicy),
		stream.WithMaxAge(7*24*time.Hour),
	)
	if err != nil {
		slog.Error("failed to ensure stream", "error", err)
		return
	}

	// Create producers for both subjects
	createdProducer, err := producer.New(
		js,
		"orders.created",
		producer.WithLogger(logger),
	)
	if err != nil {
		slog.Error("failed to create created producer", "error", err)
		return
	}

	cancelledProducer, err := producer.New(
		js,
		"orders.cancelled",
		producer.WithLogger(logger),
	)
	if err != nil {
		slog.Error("failed to create cancelled producer", "error", err)
		return
	}

	slog.Info("broker example producer started")

	nMin := 10.0
	nMax := 1000.0
	counter := 1

	for {
		orderID := counter
		amount := nMin + rand.Float64()*(nMax-nMin)

		payload := fmt.Sprintf(
			`{"order_id":"%d","amount":%.2f}`,
			orderID,
			amount,
		)

		msgID := fmt.Sprintf("order-%d", orderID)

		if orderID%2 == 0 {
			err = createdProducer.Publish(
				ctx,
				[]byte(payload),
				producer.PublishWithMsgID(msgID),
			)
			if err == nil {
				slog.Info("published order created",
					"order_id", orderID,
				)
			}
		} else {
			err = cancelledProducer.Publish(
				ctx,
				[]byte(payload),
				producer.PublishWithMsgID(msgID),
			)
			if err == nil {
				slog.Info("published order cancelled",
					"order_id", orderID,
				)
			}
		}

		if err != nil {
			slog.Error("publish failed", "error", err)
		}

		counter++
		time.Sleep(500 * time.Millisecond)
	}
}

