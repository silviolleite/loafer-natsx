package broker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/broker"
	"github.com/silviolleite/loafer-natsx/logger"
	"github.com/silviolleite/loafer-natsx/router"
)

func newRoute(t *testing.T) *router.Route {
	r, err := router.New(router.TypePubSub, "test.subject")
	assert.NoError(t, err)
	return r
}

func TestRun_NoRoutes(t *testing.T) {
	b := broker.New(nil, logger.NopLogger{})

	err := b.Run(context.Background())

	assert.ErrorIs(t, err, loafernatsx.ErrNoRoutes)
}

func TestRun_NilRegistration(t *testing.T) {
	b := broker.New(nil, logger.NopLogger{})

	err := b.Run(context.Background(), nil)

	assert.ErrorIs(t, err, loafernatsx.ErrNilRouteRegistration)
}

func TestRouteRegistrationValidation(t *testing.T) {
	r := newRoute(t)

	_, err := broker.NewRouteRegistration(nil, func(context.Context, []byte) (any, error) {
		return nil, nil
	})
	assert.ErrorIs(t, err, loafernatsx.ErrNilRoute)

	_, err = broker.NewRouteRegistration(r, nil)
	assert.ErrorIs(t, err, loafernatsx.ErrNilHandler)
}

func TestRouteRegistration_Getters(t *testing.T) {
	r := newRoute(t)

	h := func(context.Context, []byte) (any, error) {
		return nil, nil
	}

	reg, err := broker.NewRouteRegistration(r, h)
	assert.NoError(t, err)

	assert.Equal(t, r, reg.Route())
	assert.NotNil(t, reg.Handler())
}

func TestBroker_Run_ContextCancel(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	b := broker.New(nc, logger.NopLogger{}, broker.WithWorkers(2))

	r := newRoute(t)

	reg, _ := broker.NewRouteRegistration(
		r,
		func(ctx context.Context, _ []byte) (any, error) {
			<-ctx.Done()
			return nil, nil
		},
	)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := b.Run(ctx, reg)

	assert.NoError(t, err)
}

func TestBroker_Run_WorkersConcurrency(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	var mu sync.Mutex
	calls := 0

	b := broker.New(nc, logger.NopLogger{}, broker.WithWorkers(5))

	r := newRoute(t)

	reg, _ := broker.NewRouteRegistration(
		r,
		func(ctx context.Context, _ []byte) (any, error) {
			mu.Lock()
			calls++
			mu.Unlock()

			return nil, nil
		},
	)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = nc.Publish("test.subject", []byte("msg"))
	}()

	go func() {
		time.Sleep(120 * time.Millisecond)
		cancel()
	}()

	err := b.Run(ctx, reg)

	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, calls, 1)
}

func TestWithWorkersOption(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	b := broker.New(nc, nil, broker.WithWorkers(10))

	assert.NotNil(t, b)
}

func runServer() (*server.Server, string) {
	opts := &server.Options{
		Port: -1,
	}

	s, _ := server.NewServer(opts)
	go s.Start()

	if !s.ReadyForConnections(10 * 1e9) {
		panic("nats not ready")
	}

	return s, s.ClientURL()
}

func TestBroker_CancelOnWorkerError(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	b := broker.New(nc, logger.NopLogger{}, broker.WithWorkers(2))

	// valid route (should be cancelled)
	okRoute, _ := router.New(
		router.TypePubSub,
		"ok.subject",
	)

	okReg, _ := broker.NewRouteRegistration(
		okRoute,
		func(ctx context.Context, _ []byte) (any, error) {
			<-ctx.Done()
			return nil, nil
		},
	)

	// invalid route → JetStream stream does not exist
	failRoute, _ := router.New(
		router.TypeJetStream,
		"fail.subject",
		router.WithStream("DOES_NOT_EXIST"),
		router.WithDurable("d1"),
	)

	failReg, _ := broker.NewRouteRegistration(
		failRoute,
		func(ctx context.Context, _ []byte) (any, error) {
			return nil, nil
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()

	err := b.Run(ctx, okReg, failReg)

	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Less(t, elapsed, 4*time.Second)
}

func TestBroker_PanicRecovery(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	b := broker.New(nc, logger.NopLogger{})

	r, _ := router.New(router.TypePubSub, "panic.subject")

	reg, _ := broker.NewRouteRegistration(
		r,
		func(ctx context.Context, _ []byte) (any, error) {
			panic("boom")
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = nc.Publish("panic.subject", []byte("msg"))
	}()

	err := b.Run(ctx, reg)

	assert.NoError(t, err)
}

func counterValueBySubject(mfs []*dto.MetricFamily, metricName string, subject string) float64 {
	for _, mf := range mfs {
		if mf.GetName() != metricName {
			continue
		}

		for _, m := range mf.GetMetric() {
			lbls := m.GetLabel()

			for _, l := range lbls {
				if l.GetName() == "subject" && l.GetValue() == subject {
					if m.GetCounter() != nil {
						return m.GetCounter().GetValue()
					}
				}
			}
		}
	}
	return 0
}

func TestWithMetricsOption(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	reg := prometheus.NewRegistry()

	b := broker.New(
		nc,
		logger.NopLogger{},
		broker.WithWorkers(1),
		broker.WithMetrics(reg),
	)
	assert.NotNil(t, b)

	subject := "metrics.subject"
	r, _ := router.New(router.TypePubSub, subject)

	registration, _ := broker.NewRouteRegistration(
		r,
		func(ctx context.Context, _ []byte) (any, error) {
			return nil, nil
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = nc.Publish(subject, []byte("msg"))
				_ = nc.Flush()
			}
		}
	}()

	err := b.Run(ctx, registration)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		mfs, gErr := reg.Gather()
		if gErr != nil {
			return false
		}
		return counterValueBySubject(mfs, "loafer_requests_total", subject) >= 1.0
	}, 2*time.Second, 25*time.Millisecond)
}
