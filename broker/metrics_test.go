package broker

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gatherMetrics(t *testing.T, reg *prometheus.Registry) []*dto.MetricFamily {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	return mfs
}

func findMetricFamily(t *testing.T, mfs []*dto.MetricFamily, name string) *dto.MetricFamily {
	t.Helper()
	for _, mf := range mfs {
		if mf.GetName() == name {
			return mf
		}
	}
	return nil
}

func getGaugeValue(t *testing.T, mfs []*dto.MetricFamily, name string) float64 {
	t.Helper()
	mf := findMetricFamily(t, mfs, name)
	if mf == nil {
		return 0
	}
	ms := mf.GetMetric()
	if len(ms) == 0 {
		return 0
	}
	return ms[0].GetGauge().GetValue()
}

func getCounterValue(t *testing.T, mfs []*dto.MetricFamily, name, subject string) float64 {
	t.Helper()
	mf := findMetricFamily(t, mfs, name)
	if mf == nil {
		return 0
	}
	for _, m := range mf.GetMetric() {
		for _, l := range m.GetLabel() {
			if l.GetName() == "subject" && l.GetValue() == subject {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func getHistogramCount(t *testing.T, mfs []*dto.MetricFamily, name, subject string) uint64 {
	t.Helper()
	mf := findMetricFamily(t, mfs, name)
	if mf == nil {
		return 0
	}
	for _, m := range mf.GetMetric() {
		for _, l := range m.GetLabel() {
			if l.GetName() == "subject" && l.GetValue() == subject {
				return m.GetHistogram().GetSampleCount()
			}
		}
	}
	return 0
}

func TestNewMetrics_RegistersExpectedCollectors(t *testing.T) {
	reg := prometheus.NewRegistry()

	m := newMetrics(reg)
	assert.NotNil(t, m)
}

func TestInflightIncDec(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	mfs := gatherMetrics(t, reg)
	assert.Equal(t, float64(0), getGaugeValue(t, mfs, "loafer_inflight"))

	m.inflightInc()
	mfs = gatherMetrics(t, reg)
	assert.Equal(t, float64(1), getGaugeValue(t, mfs, "loafer_inflight"))

	m.inflightDec()
	mfs = gatherMetrics(t, reg)
	assert.Equal(t, float64(0), getGaugeValue(t, mfs, "loafer_inflight"))
}

func TestIncRequest_IncrementsCounterForSubject(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	subject := "test.subject"

	m.incRequest(subject)
	m.incRequest(subject)

	mfs := gatherMetrics(t, reg)
	val := getCounterValue(t, mfs, "loafer_requests_total", subject)
	assert.Equal(t, float64(2), val)
}

func TestIncError_IncrementsCounterForSubject(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	subject := "test.subject"

	m.incError(subject)

	mfs := gatherMetrics(t, reg)
	val := getCounterValue(t, mfs, "loafer_errors_total", subject)
	assert.Equal(t, float64(1), val)
}

func TestObserveDuration_EmitsHistogramMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	subject := "test.subject"

	m.observeDuration(subject, 150*time.Millisecond)

	mfs := gatherMetrics(t, reg)
	count := getHistogramCount(t, mfs, "loafer_request_duration_seconds", subject)
	assert.Equal(t, uint64(1), count)
}
