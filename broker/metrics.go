package broker

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type brokerMetrics struct {
	inflight      *prometheus.GaugeVec
	requestsTotal *prometheus.CounterVec
	errorsTotal   *prometheus.CounterVec
	duration      *prometheus.HistogramVec
}

func newMetrics(reg prometheus.Registerer) *brokerMetrics {
	m := &brokerMetrics{
		inflight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "loafer_inflight",
				Help: "Number of inflight handler executions",
			},
			[]string{"subject"},
		),
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "loafer_requests_total",
				Help: "Total processed messages",
			},
			[]string{"subject"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "loafer_errors_total",
				Help: "Total handler errors",
			},
			[]string{"subject"},
		),
		duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "loafer_request_duration_seconds",
				Help:    "Handler duration",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"subject"},
		),
	}

	reg.MustRegister(
		m.inflight,
		m.requestsTotal,
		m.errorsTotal,
		m.duration,
	)

	return m
}

func (m *brokerMetrics) inflightInc(subject string) {
	m.inflight.WithLabelValues(subject).Inc()
}

func (m *brokerMetrics) inflightDec(subject string) {
	m.inflight.WithLabelValues(subject).Dec()
}

func (m *brokerMetrics) incRequest(subject string) {
	m.requestsTotal.WithLabelValues(subject).Inc()
}

func (m *brokerMetrics) incError(subject string) {
	m.errorsTotal.WithLabelValues(subject).Inc()
}

func (m *brokerMetrics) observeDuration(subject string, d time.Duration) {
	m.duration.WithLabelValues(subject).Observe(d.Seconds())
}
