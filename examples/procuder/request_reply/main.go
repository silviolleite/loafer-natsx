package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/producer"
)

const (
	totalRequests = 100
	concurrency   = 20
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)

	// Connect
	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("rr-client-concurrent"),
		conn.WithMaxReconnects(-1),
	)
	if err != nil {
		slog.Error("connect failed", "error", err)
		return
	}
	defer nc.Drain()

	prod, err := producer.NewCore(nc, "orders.calculate")
	if err != nil {
		slog.Error("producer error", "error", err)
		return
	}

	start := time.Now()

	jobs := make(chan int)
	var wg sync.WaitGroup

	// Worker pool
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for id := range jobs {
				sendRequest(prod, id)
			}
		}(w)
	}

	// Send jobs
	for i := 1; i <= totalRequests; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("\nFinished %d requests in %s\n", totalRequests, elapsed)
}

func sendRequest(prod *producer.Producer, id int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	payload := fmt.Sprintf(`{"order_id":%d}`, id)

	start := time.Now()

	resp, err := prod.Request(ctx, []byte(payload))
	if err != nil {
		slog.Error("request failed",
			"id", id,
			"error", err,
		)
		return
	}

	latency := time.Since(start)

	slog.Info("response received",
		"id", id,
		"latency_ms", latency.Milliseconds(),
		"response", string(resp),
	)
}
