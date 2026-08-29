package middleware_test

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/middleware"
)

func ctxWithSubject(subject string) context.Context {
	return consumer.WithMetadata(context.Background(), &consumer.Metadata{Subject: subject})
}

func counterValue(t *testing.T, reg *prometheus.Registry, name, subject string) float64 {
	t.Helper()

	mfs, err := reg.Gather()
	require.NoError(t, err)

	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}

		for _, m := range mf.GetMetric() {
			if matchSubject(m, subject) {
				return m.GetCounter().GetValue()
			}
		}
	}

	return 0
}

func gaugeValue(t *testing.T, reg *prometheus.Registry, name, subject string) float64 {
	t.Helper()

	mfs, err := reg.Gather()
	require.NoError(t, err)

	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}

		for _, m := range mf.GetMetric() {
			if matchSubject(m, subject) {
				return m.GetGauge().GetValue()
			}
		}
	}

	return 0
}

func histogramCount(t *testing.T, reg *prometheus.Registry, name, subject string) uint64 {
	t.Helper()

	mfs, err := reg.Gather()
	require.NoError(t, err)

	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}

		for _, m := range mf.GetMetric() {
			if matchSubject(m, subject) {
				return m.GetHistogram().GetSampleCount()
			}
		}
	}

	return 0
}

func matchSubject(m *dto.Metric, subject string) bool {
	for _, l := range m.GetLabel() {
		if l.GetName() == "subject" && l.GetValue() == subject {
			return true
		}
	}

	return false
}

func TestMetrics_SuccessPath(t *testing.T) {
	reg := prometheus.NewRegistry()
	subject := "orders.created"

	handler := middleware.Metrics(middleware.WithMetricsRegisterer(reg))(
		func(ctx context.Context, data []byte) (any, error) {
			return "ok", nil
		},
	)

	res, err := handler(ctxWithSubject(subject), nil)

	assert.NoError(t, err)
	assert.Equal(t, "ok", res)
	assert.Equal(t, float64(1), counterValue(t, reg, "loafer_requests_total", subject))
	assert.Equal(t, float64(0), counterValue(t, reg, "loafer_errors_total", subject))
	assert.Equal(t, float64(0), gaugeValue(t, reg, "loafer_inflight", subject))
	assert.Equal(t, uint64(1), histogramCount(t, reg, "loafer_request_duration_seconds", subject))
}

func TestMetrics_ErrorPath(t *testing.T) {
	reg := prometheus.NewRegistry()
	subject := "orders.failed"
	wantErr := errors.New("boom")

	handler := middleware.Metrics(middleware.WithMetricsRegisterer(reg))(
		func(ctx context.Context, data []byte) (any, error) {
			return nil, wantErr
		},
	)

	_, err := handler(ctxWithSubject(subject), nil)

	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, float64(1), counterValue(t, reg, "loafer_requests_total", subject))
	assert.Equal(t, float64(1), counterValue(t, reg, "loafer_errors_total", subject))
	assert.Equal(t, uint64(1), histogramCount(t, reg, "loafer_request_duration_seconds", subject))
}

func TestMetrics_UnknownSubjectWithoutMetadata(t *testing.T) {
	reg := prometheus.NewRegistry()

	handler := middleware.Metrics(middleware.WithMetricsRegisterer(reg))(
		func(ctx context.Context, data []byte) (any, error) {
			return nil, nil
		},
	)

	_, err := handler(context.Background(), nil)

	assert.NoError(t, err)
	assert.Equal(t, float64(1), counterValue(t, reg, "loafer_requests_total", "unknown"))
}

func TestMetrics_IdempotentRegistration(t *testing.T) {
	reg := prometheus.NewRegistry()
	subject := "orders.shared"

	first := middleware.Metrics(middleware.WithMetricsRegisterer(reg))
	second := middleware.Metrics(middleware.WithMetricsRegisterer(reg))

	base := func(ctx context.Context, data []byte) (any, error) {
		return nil, nil
	}

	_, err := first(base)(ctxWithSubject(subject), nil)
	assert.NoError(t, err)

	_, err = second(base)(ctxWithSubject(subject), nil)
	assert.NoError(t, err)

	assert.Equal(t, float64(2), counterValue(t, reg, "loafer_requests_total", subject))
}

func TestMetrics_CustomBuckets(t *testing.T) {
	reg := prometheus.NewRegistry()
	subject := "orders.buckets"

	handler := middleware.Metrics(
		middleware.WithMetricsRegisterer(reg),
		middleware.WithMetricsBuckets([]float64{0.1, 0.5, 1}),
	)(func(ctx context.Context, data []byte) (any, error) {
		return nil, nil
	})

	_, err := handler(ctxWithSubject(subject), nil)

	assert.NoError(t, err)
	assert.Equal(t, uint64(1), histogramCount(t, reg, "loafer_request_duration_seconds", subject))
}
