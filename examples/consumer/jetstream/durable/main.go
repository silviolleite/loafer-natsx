package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/consumer"
	jsprod "github.com/silviolleite/loafer-natsx/producer"
	"github.com/silviolleite/loafer-natsx/router"
)

func main() {
	ctx := context.Background()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("jetstream-durable-example"),
	)
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer nc.Drain()

	js, err := jetstream.New(nc)
	if err != nil {
		slog.Error("failed to create jetstream", "error", err)
		return
	}

	cons, err := consumer.New(nc, logger)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	// Route subject uses a wildcard token: orders.created.<region>
	// The handler extracts the region from the subject via Metadata.
	route, err := router.New(
		router.TypeJetStream,
		"orders.created",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-durable"),
		router.WithDeliveryPolicy(router.DeliverNewPolicy),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(5)

	err = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		// consumer.MetadataFromContext gives access to subject, headers,
		// stream name, sequence, delivery count, and timestamp — without
		// exposing the underlying jetstream.Msg to the handler.
		if meta, ok := consumer.MetadataFromContext(ctx); ok {
			// Parse a dynamic token from the subject.
			// e.g. "orders.created.us-east" → region = "us-east"
			parts := strings.Split(meta.Subject, ".")
			region := "unknown"
			if len(parts) == 3 {
				region = parts[2]
			}

			correlationID := meta.Headers.Get(consumer.HeaderCorrelationIDKey)

			fmt.Printf("durable consumer received: %s | stream=%s seq=%d region=%s correlation_id=%s\n",
				string(data), meta.Stream, meta.Sequence, region, correlationID)
		} else {
			fmt.Println("durable consumer received:", string(data))
		}

		wg.Done()
		return nil, nil
	})
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	strategy := jsprod.NewJetStreamStrategy(js, logger)
	prod, err := jsprod.New(strategy, "orders.created")
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	regions := []string{"us-east", "us-west", "eu-central", "ap-south", "sa-east"}

	for i, region := range regions {
		msg := fmt.Sprintf(`{"order_id":"%d","region":"%s"}`, i+1, region)

		h := nats.Header{}
		h.Set(consumer.HeaderCorrelationIDKey, fmt.Sprintf("cid-%d", i+1))

		_, err = prod.Publish(ctx, []byte(msg), jsprod.PublishWithHeaders(h))
		if err != nil {
			slog.Error("publish failed", "error", err)
			continue
		}
	}

	wg.Wait()

	fmt.Println("all messages processed, exiting")

	// output:
	// durable consumer received: {"order_id":"1","region":"us-east"} | stream=ORDERS seq=1 region=us-east correlation_id=cid-1
	// durable consumer received: {"order_id":"2","region":"us-west"} | stream=ORDERS seq=2 region=us-west correlation_id=cid-2
	// durable consumer received: {"order_id":"3","region":"eu-central"} | stream=ORDERS seq=3 region=eu-central correlation_id=cid-3
	// durable consumer received: {"order_id":"4","region":"ap-south"} | stream=ORDERS seq=4 region=ap-south correlation_id=cid-4
	// durable consumer received: {"order_id":"5","region":"sa-east"} | stream=ORDERS seq=5 region=sa-east correlation_id=cid-5
	// all messages processed, exiting
}
