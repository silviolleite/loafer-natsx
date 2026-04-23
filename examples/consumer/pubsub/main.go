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
	"github.com/silviolleite/loafer-natsx/router"
)

func main() {
	ctx := context.Background()

	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("pubsub-example"),
	)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		return
	}
	defer nc.Close()

	cons, err := consumer.New(nc, logger)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		return
	}

	route, err := router.New(
		router.TypePubSub,
		"orders.created",
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	err = cons.Start(ctx, route, func(ctx context.Context, data []byte) (any, error) {
		// consumer.MetadataFromContext provides access to subject, headers,
		// and reply subject without coupling the handler to a specific NATS
		// message type. It works identically across PubSub, Queue,
		// RequestReply, and JetStream routes.
		if meta, ok := consumer.MetadataFromContext(ctx); ok {
			correlationID := meta.Headers.Get(consumer.HeaderCorrelationIDKey)
			traceParent := meta.Headers.Get(consumer.HeaderTraceParentKey)

			fmt.Printf("received on %s | correlation_id=%s traceparent=%s | payload=%s\n",
				meta.Subject, correlationID, traceParent, string(data))
		} else {
			fmt.Println("received message:", string(data))
		}

		return nil, nil
	})
	if err != nil {
		slog.Error("failed to start consumer", "error", err)
		return
	}

	strategy := coreprod.NewCoreStrategy(nc)
	prod, err := coreprod.New(strategy, "orders.created")
	if err != nil {
		slog.Error("failed to create producer", "error", err)
		return
	}

	for i := 1; i <= 5; i++ {
		msg := fmt.Sprintf(`{"order_id": "%d"}`, i)

		h := nats.Header{}
		h.Set(consumer.HeaderCorrelationIDKey, fmt.Sprintf("cid-%d", i))
		h.Set(consumer.HeaderTraceParentKey, fmt.Sprintf("00-trace%d-span%d-01", i, i))

		_, err = prod.Publish(ctx, []byte(msg), coreprod.PublishWithHeaders(h))
		if err != nil {
			slog.Error("publish failed", "error", err)
			continue
		}

		time.Sleep(500 * time.Millisecond)
	}

	time.Sleep(3 * time.Second)

	slog.Info("pub/sub example finished")

	// output:
	// received on orders.created | correlation_id=cid-1 traceparent=00-trace1-span1-01 | payload={"order_id": "1"}
	// received on orders.created | correlation_id=cid-2 traceparent=00-trace2-span2-01 | payload={"order_id": "2"}
	// received on orders.created | correlation_id=cid-3 traceparent=00-trace3-span3-01 | payload={"order_id": "3"}
	// received on orders.created | correlation_id=cid-4 traceparent=00-trace4-span4-01 | payload={"order_id": "4"}
	// received on orders.created | correlation_id=cid-5 traceparent=00-trace5-span5-01 | payload={"order_id": "5"}
	// 2026/02/14 09:44:31 INFO pub/sub example finished
}
