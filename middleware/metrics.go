package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metric names exported by the Metrics middleware. They preserve the historical
// broker metric names so existing dashboards and alerts keep working.
const (
	metricInflight = "loafer_inflight"
	metricRequests = "loafer_requests_total"
	metricErrors   = "loafer_errors_total"
	metricDuration = "loafer_request_duration_seconds"
)

// labelSubject is the Prometheus label used to partition metrics by subject.
const labelSubject = "subject"

// metricsConfig holds the resolved configuration for the Metrics middleware.
type metricsConfig struct {
	registerer prometheus.Registerer
	buckets    []float64
}

// MetricsOption configures the Metrics middleware.
type MetricsOption func(*metricsConfig)

// WithMetricsRegisterer sets a custom prometheus.Registerer for the Metrics
// middleware. When this option is not supplied, or is supplied with a nil
// registerer, the middleware falls back to prometheus.DefaultRegisterer.
func WithMetricsRegisterer(r prometheus.Registerer) MetricsOption {
	return func(cfg *metricsConfig) {
		if r != nil {
			cfg.registerer = r
		}
	}
}

// WithMetricsBuckets overrides the histogram buckets used by the processing
// duration histogram. A nil or empty slice is ignored so the middleware keeps
// prometheus.DefBuckets.
func WithMetricsBuckets(buckets []float64) MetricsOption {
	return func(cfg *metricsConfig) {
		if len(buckets) > 0 {
			cfg.buckets = buckets
		}
	}
}

// loadMetricsConfig builds a metricsConfig from the supplied options,
// defaulting the registerer to prometheus.DefaultRegisterer and the buckets to
// prometheus.DefBuckets when no valid override is given.
func loadMetricsConfig(opts ...MetricsOption) metricsConfig {
	cfg := metricsConfig{
		registerer: prometheus.DefaultRegisterer,
		buckets:    prometheus.DefBuckets,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.registerer == nil {
		cfg.registerer = prometheus.DefaultRegisterer
	}

	if len(cfg.buckets) == 0 {
		cfg.buckets = prometheus.DefBuckets
	}

	return cfg
}

// Metrics returns a Middleware that instruments message processing with
// Prometheus collectors, all labeled by the message subject read from the
// context Metadata:
//
//   - loafer_inflight (gauge): incremented when processing begins and
//     decremented when it completes.
//   - loafer_requests_total (counter): incremented once processing completes,
//     for both success and error outcomes.
//   - loafer_errors_total (counter): incremented when the wrapped handler
//     returns an error.
//   - loafer_request_duration_seconds (histogram): observes the elapsed handler
//     processing time in seconds.
//
// Collectors are registered on the configured registerer using a safe,
// idempotent registration that reuses any previously registered collector, so
// constructing the middleware more than once for the same registerer never
// panics on duplicate registration. The wrapped handler's result and error are
// always propagated unchanged, keeping the middleware transparent to callers.
func Metrics(opts ...MetricsOption) Middleware {
	cfg := loadMetricsConfig(opts...)

	inflight := registerGaugeVec(cfg.registerer, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricInflight,
		Help: "Number of inflight handler executions",
	}, []string{labelSubject}))

	requests := registerCounterVec(cfg.registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: metricRequests,
		Help: "Total processed messages",
	}, []string{labelSubject}))

	errorsTotal := registerCounterVec(cfg.registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: metricErrors,
		Help: "Total handler errors",
	}, []string{labelSubject}))

	duration := registerHistogramVec(cfg.registerer, prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    metricDuration,
		Help:    "Handler duration",
		Buckets: cfg.buckets,
	}, []string{labelSubject}))

	return func(next Handler) Handler {
		return func(ctx context.Context, data []byte) (any, error) {
			subject := subjectFromContext(ctx)

			inflight.WithLabelValues(subject).Inc()
			defer inflight.WithLabelValues(subject).Dec()

			start := time.Now()
			res, err := next(ctx, data)
			duration.WithLabelValues(subject).Observe(time.Since(start).Seconds())

			requests.WithLabelValues(subject).Inc()

			if err != nil {
				errorsTotal.WithLabelValues(subject).Inc()
				return res, err
			}

			return res, nil
		}
	}
}

// registerCounterVec registers c with the registerer and returns the usable
// collector. When an equivalent collector is already registered, the existing
// one is returned instead, avoiding a panic on duplicate registration.
func registerCounterVec(r prometheus.Registerer, c *prometheus.CounterVec) *prometheus.CounterVec {
	if err := r.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			if existing, ok := are.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
	}

	return c
}

// registerHistogramVec registers h with the registerer and returns the usable
// collector, reusing an already registered equivalent collector when present.
func registerHistogramVec(r prometheus.Registerer, h *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := r.Register(h); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			if existing, ok := are.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
		}
	}

	return h
}

// registerGaugeVec registers g with the registerer and returns the usable
// collector, reusing an already registered equivalent collector when present.
func registerGaugeVec(r prometheus.Registerer, g *prometheus.GaugeVec) *prometheus.GaugeVec {
	if err := r.Register(g); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			if existing, ok := are.ExistingCollector.(*prometheus.GaugeVec); ok {
				return existing
			}
		}
	}

	return g
}
