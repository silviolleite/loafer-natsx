package middleware

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/silviolleite/loafer-natsx/consumer"
)

const (
	// tracerName is the instrumentation scope name used when acquiring a Tracer
	// from the configured TracerProvider.
	tracerName = "loafer-natsx"

	// spanNamePrefix is prepended to the subject to form the span name
	// following the "loafer.process/<subject>" pattern.
	spanNamePrefix = "loafer.process/"

	// messagingSystem is the value reported for the messaging.system span
	// attribute, identifying NATS as the messaging backend.
	messagingSystem = "nats"
)

// otelConfig holds the resolved configuration for the OTel middleware.
type otelConfig struct {
	tracerProvider  trace.TracerProvider
	propagator      propagation.TextMapPropagator
	linkFromContext bool
}

// OTelOption configures the OpenTelemetry middleware.
type OTelOption func(*otelConfig)

// WithTracerProvider sets a custom trace.TracerProvider for the OTel
// middleware. When this option is not supplied, the middleware falls back to
// the globally registered provider returned by otel.GetTracerProvider.
func WithTracerProvider(tp trace.TracerProvider) OTelOption {
	return func(cfg *otelConfig) {
		if tp != nil {
			cfg.tracerProvider = tp
		}
	}
}

// WithPropagator sets a custom propagation.TextMapPropagator used to extract the
// incoming trace context from the NATS message headers. When this option is not
// supplied, the middleware falls back to the globally registered propagator
// returned by otel.GetTextMapPropagator.
func WithPropagator(p propagation.TextMapPropagator) OTelOption {
	return func(cfg *otelConfig) {
		if p != nil {
			cfg.propagator = p
		}
	}
}

// WithLinkFromContext changes how the processing span relates to the trace
// context carried by the incoming message headers.
//
// By default the middleware starts the processing span as a child of the
// extracted producer span, continuing the same trace. When this option is
// enabled, the middleware instead starts the processing span as the root of a
// new trace and attaches the incoming span context as a span link. This is
// useful for long-lived consumers where inheriting the producer's trace would
// otherwise create unbounded or misleading traces, while still preserving the
// causal relationship through the link.
//
// When the incoming headers carry no valid span context, the span is started as
// a new root without any link.
func WithLinkFromContext() OTelOption {
	return func(cfg *otelConfig) {
		cfg.linkFromContext = true
	}
}

// loadOTelConfig builds an otelConfig from the supplied options, defaulting the
// TracerProvider and propagator to the global instances when no override is
// given.
func loadOTelConfig(opts ...OTelOption) otelConfig {
	cfg := otelConfig{
		tracerProvider: otel.GetTracerProvider(),
		propagator:     otel.GetTextMapPropagator(),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.tracerProvider == nil {
		cfg.tracerProvider = otel.GetTracerProvider()
	}

	if cfg.propagator == nil {
		cfg.propagator = otel.GetTextMapPropagator()
	}

	return cfg
}

// OTel returns a Middleware that creates a distributed tracing span for each
// message processing operation.
//
// For every message it extracts any trace context propagated through the NATS
// message headers and starts a span named "loafer.process/<subject>" with
// SpanKind Consumer, recording the following attributes when available:
//
//   - messaging.system ("nats")
//   - messaging.operation ("process")
//   - messaging.destination.name (the subject)
//   - messaging.message.id (the JetStream stream sequence, when present)
//   - messaging.nats.stream and messaging.nats.consumer (JetStream only)
//
// By default the span continues the extracted producer trace as a child span.
// Supplying WithLinkFromContext changes this behavior so the span becomes the
// root of a new trace and the incoming span context is attached as a span link
// instead.
//
// The span context is propagated to the wrapped handler through the returned
// context. When the handler returns an error, the error is recorded as a span
// event and the span status is set to Error; otherwise the status is set to Ok.
// The handler's result and error are always propagated unchanged, keeping the
// middleware transparent to callers.
func OTel(opts ...OTelOption) Middleware {
	cfg := loadOTelConfig(opts...)

	return func(next Handler) Handler {
		return func(ctx context.Context, data []byte) (any, error) {
			md, _ := consumer.MetadataFromContext(ctx)
			subject := subjectFromContext(ctx)

			ctx = extractContext(ctx, cfg.propagator, md)

			tracer := cfg.tracerProvider.Tracer(tracerName)
			startOpts := spanStartOptions(ctx, md, subject, cfg.linkFromContext)

			ctx, span := tracer.Start(ctx, spanNamePrefix+subject, startOpts...)
			defer span.End()

			res, err := next(ctx, data)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return res, err
			}

			span.SetStatus(codes.Ok, "")

			return res, nil
		}
	}
}

// extractContext returns a context populated with the trace context propagated
// through the message headers, or the original context unchanged when no
// headers are available.
func extractContext(
	ctx context.Context,
	propagator propagation.TextMapPropagator,
	md *consumer.Metadata,
) context.Context {
	if md == nil || md.Headers == nil {
		return ctx
	}

	return propagator.Extract(ctx, propagation.HeaderCarrier(md.Headers))
}

// spanStartOptions builds the span start options, including messaging
// attributes and, when link mode is enabled, a new root with a link to the
// incoming span context.
func spanStartOptions(
	ctx context.Context,
	md *consumer.Metadata,
	subject string,
	linkFromContext bool,
) []trace.SpanStartOption {
	attrs := []attribute.KeyValue{
		attribute.String("messaging.system", messagingSystem),
		attribute.String("messaging.operation", "process"),
		attribute.String("messaging.destination.name", subject),
	}

	if md != nil && md.Stream != "" {
		attrs = append(attrs,
			attribute.String("messaging.message.id", strconv.FormatUint(md.Sequence, 10)),
			attribute.String("messaging.nats.stream", md.Stream),
			attribute.String("messaging.nats.consumer", md.Consumer),
		)
	}

	startOpts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindConsumer),
	}

	if linkFromContext {
		startOpts = append(startOpts, trace.WithNewRoot())
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			startOpts = append(startOpts, trace.WithLinks(trace.Link{SpanContext: sc}))
		}
	}

	return startOpts
}
