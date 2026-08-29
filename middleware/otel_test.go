package middleware_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/middleware"
)

func newRecorder() (*tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	return rec, tp
}

func TestOTel_SuccessSpan(t *testing.T) {
	rec, tp := newRecorder()

	ctx := consumer.WithMetadata(context.Background(), &consumer.Metadata{Subject: "orders.created"})

	handler := middleware.OTel(middleware.WithTracerProvider(tp))(
		func(ctx context.Context, data []byte) (any, error) {
			return nil, nil
		},
	)

	_, err := handler(ctx, nil)
	require.NoError(t, err)

	spans := rec.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "loafer.process/orders.created", span.Name())
	assert.Equal(t, trace.SpanKindConsumer, span.SpanKind())
	assert.Equal(t, codes.Ok, span.Status().Code)
	assert.Contains(t, attrMap(span), "messaging.system")
	assert.Equal(t, "nats", attrMap(span)["messaging.system"])
	assert.Equal(t, "orders.created", attrMap(span)["messaging.destination.name"])
}

func TestOTel_ErrorSpan(t *testing.T) {
	rec, tp := newRecorder()

	ctx := consumer.WithMetadata(context.Background(), &consumer.Metadata{Subject: "orders.failed"})
	wantErr := errors.New("boom")

	handler := middleware.OTel(middleware.WithTracerProvider(tp))(
		func(ctx context.Context, data []byte) (any, error) {
			return nil, wantErr
		},
	)

	_, err := handler(ctx, nil)
	require.ErrorIs(t, err, wantErr)

	spans := rec.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.NotEmpty(t, span.Events())
}

func TestOTel_JetStreamAttributes(t *testing.T) {
	rec, tp := newRecorder()

	ctx := consumer.WithMetadata(context.Background(), &consumer.Metadata{
		Subject:  "orders.created",
		Stream:   "ORDERS",
		Consumer: "orders-durable",
		Sequence: 42,
	})

	handler := middleware.OTel(middleware.WithTracerProvider(tp))(
		func(ctx context.Context, data []byte) (any, error) {
			return nil, nil
		},
	)

	_, err := handler(ctx, nil)
	require.NoError(t, err)

	attrs := attrMap(rec.Ended()[0])
	assert.Equal(t, "42", attrs["messaging.message.id"])
	assert.Equal(t, "ORDERS", attrs["messaging.nats.stream"])
	assert.Equal(t, "orders-durable", attrs["messaging.nats.consumer"])
}

func TestOTel_ContinuesTraceFromHeaders(t *testing.T) {
	rec, tp := newRecorder()
	propagator := propagation.TraceContext{}

	parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), "producer")
	headers := nats.Header{}
	propagator.Inject(parentCtx, propagation.HeaderCarrier(headers))
	parentSpan.End()

	ctx := consumer.WithMetadata(context.Background(), &consumer.Metadata{
		Subject: "orders.created",
		Headers: headers,
	})

	handler := middleware.OTel(
		middleware.WithTracerProvider(tp),
		middleware.WithPropagator(propagator),
	)(func(ctx context.Context, data []byte) (any, error) {
		return nil, nil
	})

	_, err := handler(ctx, nil)
	require.NoError(t, err)

	var child sdktrace.ReadOnlySpan
	for _, s := range rec.Ended() {
		if s.Name() == "loafer.process/orders.created" {
			child = s
		}
	}

	require.NotNil(t, child)
	assert.Equal(t, parentSpan.SpanContext().TraceID(), child.SpanContext().TraceID())
	assert.Equal(t, parentSpan.SpanContext().SpanID(), child.Parent().SpanID())
}

func TestOTel_LinkFromContext(t *testing.T) {
	rec, tp := newRecorder()
	propagator := propagation.TraceContext{}

	parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), "producer")
	headers := nats.Header{}
	propagator.Inject(parentCtx, propagation.HeaderCarrier(headers))
	parentSpan.End()

	ctx := consumer.WithMetadata(context.Background(), &consumer.Metadata{
		Subject: "orders.created",
		Headers: headers,
	})

	handler := middleware.OTel(
		middleware.WithTracerProvider(tp),
		middleware.WithPropagator(propagator),
		middleware.WithLinkFromContext(),
	)(func(ctx context.Context, data []byte) (any, error) {
		return nil, nil
	})

	_, err := handler(ctx, nil)
	require.NoError(t, err)

	var child sdktrace.ReadOnlySpan
	for _, s := range rec.Ended() {
		if s.Name() == "loafer.process/orders.created" {
			child = s
		}
	}

	require.NotNil(t, child)
	assert.NotEqual(t, parentSpan.SpanContext().TraceID(), child.SpanContext().TraceID())
	require.Len(t, child.Links(), 1)
	assert.Equal(t, parentSpan.SpanContext().TraceID(), child.Links()[0].SpanContext.TraceID())
}

func attrMap(span sdktrace.ReadOnlySpan) map[string]string {
	out := make(map[string]string)
	for _, kv := range span.Attributes() {
		out[string(kv.Key)] = kv.Value.AsString()
	}
	return out
}
