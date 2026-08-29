package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/silviolleite/loafer-natsx/broker"
	"github.com/silviolleite/loafer-natsx/conn"
	"github.com/silviolleite/loafer-natsx/middleware"
	"github.com/silviolleite/loafer-natsx/router"
)

// This example shows how to compose observability middlewares on the broker:
//
//   - middleware.Metrics installs the Prometheus collectors (exposed on :9090).
//   - middleware.OTel creates a tracing span per message and continues any
//     trace propagated through the NATS headers.
//
// Both are wired through broker.WithGlobalMiddleware, so they apply to every
// registered route. Additional custom middlewares can be added the same way.
func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Configure the OpenTelemetry tracer provider with a stdout exporter so
	// spans are printed to the console. Replace stdouttrace with an OTLP
	// exporter to ship spans to a real collector.
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		slog.Error("failed to create trace exporter", "error", err)
		return
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(shutdownCtx)
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Expose Prometheus metrics.
	metricsServer := &http.Server{
		Addr:              ":9090",
		Handler:           promhttp.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("starting metrics server", "port", 9090)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "error", err)
		}
	}()

	nc, err := conn.Connect(
		nats.DefaultURL,
		conn.WithName("orders-broker"),
		conn.WithMaxReconnects(-1),
	)
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return
	}
	defer nc.Drain()

	createdRoute, err := router.New(
		router.TypeJetStream,
		"orders.created",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-created-durable"),
		router.WithDeliveryPolicy(router.DeliverNewPolicy),
	)
	if err != nil {
		slog.Error("failed to create route", "error", err)
		return
	}

	// Compose the observability middlewares globally. They run outermost around
	// every route handler: OTel starts the span first, then Metrics measures the
	// handler duration inside it.
	br := broker.New(
		nc,
		log,
		broker.WithWorkers(2),
		broker.WithGlobalMiddleware(
			middleware.OTel(),
			middleware.Metrics(middleware.WithMetricsRegisterer(prometheus.DefaultRegisterer)),
		),
	)

	reg, _ := broker.NewRouteRegistration(
		createdRoute,
		func(ctx context.Context, data []byte) (any, error) {
			slog.Info("order created", "payload", string(data))
			time.Sleep(200 * time.Millisecond)
			return nil, nil
		},
	)

	slog.Info("broker started with metrics + tracing middleware")

	if err := br.Run(ctx, reg); err != nil {
		slog.Error("broker stopped due to error", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("metrics server shutdown error", "error", err)
	}

	slog.Info("broker shutdown complete")
}
