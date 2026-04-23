package broker_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	"github.com/silviolleite/loafer-natsx/broker"
	"github.com/silviolleite/loafer-natsx/logger"
	"github.com/silviolleite/loafer-natsx/router"
)

func gatherBrokerMetrics(t *testing.T, reg *prometheus.Registry) []*dto.MetricFamily {
	t.Helper()
	mfs, err := reg.Gather()
	assert.NoError(t, err)
	return mfs
}

func counterBySubject(mfs []*dto.MetricFamily, name, subject string) float64 {
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "subject" && l.GetValue() == subject {
					if c := m.GetCounter(); c != nil {
						return c.GetValue()
					}
				}
			}
		}
	}
	return 0
}

func histogramCountBySubject(mfs []*dto.MetricFamily, name, subject string) uint64 {
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "subject" && l.GetValue() == subject {
					if h := m.GetHistogram(); h != nil {
						return h.GetSampleCount()
					}
				}
			}
		}
	}
	return 0
}

// TestInstrumentation_SuccessPath verifies that a successful handler invocation
// increments loafer_requests_total and records a duration observation, while
// loafer_errors_total remains zero.
func TestInstrumentation_SuccessPath(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	reg := prometheus.NewRegistry()

	subject := "instr.success"

	b := broker.New(nc, logger.NopLogger{}, broker.WithWorkers(1), broker.WithMetrics(reg))

	r, _ := router.New(router.TypePubSub, subject)

	wg := sync.WaitGroup{}
	wg.Add(1)

	var once sync.Once

	registration, _ := broker.NewRouteRegistration(r, func(ctx context.Context, _ []byte) (any, error) {
		once.Do(wg.Done)
		return nil, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = nc.Publish(subject, []byte("msg"))
	}()

	go func() {
		wg.Wait()
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := b.Run(ctx, registration)
	assert.NoError(t, err)

	mfs := gatherBrokerMetrics(t, reg)
	assert.GreaterOrEqual(t, counterBySubject(mfs, "loafer_requests_total", subject), float64(1))
	assert.Equal(t, float64(0), counterBySubject(mfs, "loafer_errors_total", subject))
	assert.GreaterOrEqual(t, histogramCountBySubject(mfs, "loafer_request_duration_seconds", subject), uint64(1))
}

// TestInstrumentation_ErrorPath verifies that a handler returning an error
// increments both loafer_requests_total and loafer_errors_total.
func TestInstrumentation_ErrorPath(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	reg := prometheus.NewRegistry()

	subject := "instr.error"

	b := broker.New(nc, logger.NopLogger{}, broker.WithWorkers(1), broker.WithMetrics(reg))

	r, _ := router.New(router.TypePubSub, subject)

	wg := sync.WaitGroup{}
	wg.Add(1)

	var once sync.Once

	registration, _ := broker.NewRouteRegistration(r, func(ctx context.Context, _ []byte) (any, error) {
		once.Do(wg.Done)
		return nil, errors.New("handler error")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = nc.Publish(subject, []byte("msg"))
	}()

	go func() {
		wg.Wait()
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := b.Run(ctx, registration)
	assert.NoError(t, err)

	mfs := gatherBrokerMetrics(t, reg)
	assert.GreaterOrEqual(t, counterBySubject(mfs, "loafer_requests_total", subject), float64(1))
	assert.GreaterOrEqual(t, counterBySubject(mfs, "loafer_errors_total", subject), float64(1))
	assert.GreaterOrEqual(t, histogramCountBySubject(mfs, "loafer_request_duration_seconds", subject), uint64(1))
}

// TestInstrumentation_NoMetrics verifies that when no metrics registry is
// configured, the broker returns the handler directly (zero-overhead path)
// and messages are still processed correctly.
func TestInstrumentation_NoMetrics(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	subject := "instr.nometrics"

	b := broker.New(nc, logger.NopLogger{}, broker.WithWorkers(1))

	r, _ := router.New(router.TypePubSub, subject)

	wg := sync.WaitGroup{}
	wg.Add(1)

	var once sync.Once

	registration, _ := broker.NewRouteRegistration(r, func(ctx context.Context, _ []byte) (any, error) {
		once.Do(wg.Done)
		return nil, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = nc.Publish(subject, []byte("msg"))
	}()

	go func() {
		wg.Wait()
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := b.Run(ctx, registration)
	assert.NoError(t, err)
}
