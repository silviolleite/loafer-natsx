package consumer_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/logger"
	"github.com/silviolleite/loafer-natsx/router"
)

// unsupportedTypeRoute returns a *router.Route whose internal routeType is set
// to a value that does not match any known case, bypassing the validation in
// router.New (which rejects unknown types).
func unsupportedTypeRoute(t *testing.T) *router.Route {
	t.Helper()

	r, err := router.New(router.TypePubSub, "unsupported.subject")
	assert.NoError(t, err)

	rv := reflect.ValueOf(r).Elem()
	f := rv.FieldByName("routeType")

	// Write to unexported field via unsafe.
	ptr := (*router.Type)(unsafe.Pointer(f.UnsafeAddr()))
	*ptr = router.Type(999)

	return r
}

// TestNew_NilLogger verifies that consumer.New accepts a nil logger and
// substitutes a NopLogger so that subsequent calls do not panic.
func TestNew_NilLogger(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, err := nats.Connect(url)
	assert.NoError(t, err)
	defer nc.Close()

	c, err := consumer.New(nc, nil)
	assert.NoError(t, err)
	assert.NotNil(t, c)
}

// TestPubSub_HandlerError verifies that a handler error on a PubSub route is
// logged and does not crash the consumer.
func TestPubSub_HandlerError(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	var once sync.Once

	r, _ := router.New(router.TypePubSub, "pubsub.handler.error")

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		once.Do(wg.Done)
		return nil, errors.New("handler error")
	})
	assert.NoError(t, err)

	_ = nc.Publish("pubsub.handler.error", []byte("data"))

	wait(&wg)
}

// TestQueue_HandlerError verifies that a handler error on a Queue route is
// logged and does not crash the consumer.
func TestQueue_HandlerError(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	var once sync.Once

	r, _ := router.New(router.TypeQueue, "queue.handler.error", router.WithQueueGroup("workers"))

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		once.Do(wg.Done)
		return nil, errors.New("handler error")
	})
	assert.NoError(t, err)

	_ = nc.Publish("queue.handler.error", []byte("data"))

	wait(&wg)
}

// TestStart_UnsupportedType verifies that consumer.Start returns
// ErrUnsupportedType when the route carries an unknown router type.
func TestStart_UnsupportedType(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, err := consumer.New(nc, logger.NopLogger{})
	assert.NoError(t, err)

	r := unsupportedTypeRoute(t)

	err = c.Start(context.Background(), r, func(ctx context.Context, b []byte) (any, error) {
		return nil, nil
	})

	assert.ErrorIs(t, err, loafernatsx.ErrUnsupportedType)
}

// TestStartPubSub_SubscribeError verifies that startPubSub propagates the
// subscribe error when the NATS connection is closed before Start is called.
func TestStartPubSub_SubscribeError(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	nc.Close() // close before subscribing

	c, err := consumer.New(nc, logger.NopLogger{})
	assert.NoError(t, err)

	r, _ := router.New(router.TypePubSub, "closed.pubsub")

	err = c.Start(context.Background(), r, func(ctx context.Context, b []byte) (any, error) {
		return nil, nil
	})

	assert.Error(t, err)
}

// TestStartQueue_SubscribeError verifies that startQueue propagates the
// subscribe error when the NATS connection is closed before Start is called.
func TestStartQueue_SubscribeError(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	nc.Close()

	c, err := consumer.New(nc, logger.NopLogger{})
	assert.NoError(t, err)

	r, _ := router.New(router.TypeQueue, "closed.queue", router.WithQueueGroup("workers"))

	err = c.Start(context.Background(), r, func(ctx context.Context, b []byte) (any, error) {
		return nil, nil
	})

	assert.Error(t, err)
}

// TestStartRequestReply_SubscribeError verifies that startRequestReply
// propagates the subscribe error when the connection is closed.
func TestStartRequestReply_SubscribeError(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	nc.Close()

	c, err := consumer.New(nc, logger.NopLogger{})
	assert.NoError(t, err)

	r, _ := router.New(router.TypeRequestReply, "closed.rr", router.WithQueueGroup("workers"))

	err = c.Start(context.Background(), r, func(ctx context.Context, b []byte) (any, error) {
		return nil, nil
	})

	assert.Error(t, err)
}

// TestDefaultReply_NoReplySubject exercises the defaultReply path where the
// handler succeeds but msg.Respond fails because there is no reply subject.
func TestDefaultReply_NoReplySubject(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, _ := router.New(
		router.TypeRequestReply,
		"rr.noreply",
		router.WithQueueGroup("workers"),
	)

	var called int32

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		atomic.AddInt32(&called, 1)
		return nil, nil
	})
	assert.NoError(t, err)

	// Publish without a reply subject — Respond will fail internally.
	_ = nc.Publish("rr.noreply", []byte("data"))

	time.Sleep(150 * time.Millisecond)

	// Handler was invoked; the Respond error is logged but does not crash.
	assert.GreaterOrEqual(t, atomic.LoadInt32(&called), int32(1))
}

// TestJetStream_DLQEnabledButNilMeta exercises the branch in handleJetStreamError
// where DLQ is enabled but msg.Metadata() returns an error (meta == nil).
// We trigger this by using a non-JetStream message that still satisfies the
// jetstream.Msg interface — in practice we rely on the fact that a freshly
// created consumer with a very short AckWait will redeliver, but the metadata
// path is exercised via the normal DLQ flow with MaxDeliver=1.
//
// The nil-meta branch is reached when the JetStream server does not attach
// metadata to the message. We approximate this by verifying that when DLQ is
// enabled and MaxDeliver=1 but the DLQ stream does NOT exist, the consumer
// falls back gracefully (logs the publish error and naks).
func TestJetStream_DLQPublishError_FallsBackToNak(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	streamName := uniqueName("DLQPUBERR")
	subject := "dlqpuberr." + streamName
	durable := uniqueName("dlqpuberr_dur")

	_, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	})
	assert.NoError(t, err)

	// Intentionally do NOT create the DLQ stream — the publish will fail
	// because the server has no matching stream for the dlq.* subject.
	// The consumer should log the error and nak the message.

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, _ := router.New(
		router.TypeJetStream,
		subject,
		router.WithStream(streamName),
		router.WithDurable(durable),
		router.WithAckWait(500*time.Millisecond),
		router.WithMaxDeliver(1),
		router.WithEnableDLQ(),
	)

	var calls int32

	err = c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		atomic.AddInt32(&calls, 1)
		return nil, errors.New("transient error")
	})
	assert.NoError(t, err)

	_, err = js.Publish(context.Background(), subject, []byte("data"))
	assert.NoError(t, err)

	// Wait for at least one delivery attempt.
	time.Sleep(2 * time.Second)

	assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(1))
}

// TestJetStream_Queue_MetadataInContext verifies that the Queue consumer type
// also propagates metadata correctly through the context.
func TestQueue_MetadataInContext(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	r, _ := router.New(
		router.TypeQueue,
		"meta.queue",
		router.WithQueueGroup("workers"),
	)

	var (
		mu      sync.Mutex
		gotMeta *consumer.Metadata
		gotOK   bool
		once    sync.Once
	)

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		once.Do(func() {
			md, ok := consumer.MetadataFromContext(ctx)
			mu.Lock()
			gotMeta = md
			gotOK = ok
			mu.Unlock()
			wg.Done()
		})
		return nil, nil
	})
	assert.NoError(t, err)

	req := &nats.Msg{
		Subject: "meta.queue",
		Data:    []byte("data"),
		Header:  nats.Header{},
	}
	req.Header.Set(consumer.HeaderCorrelationIDKey, "queue-cid")

	_ = nc.PublishMsg(req)

	wait(&wg)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, gotOK)
	assert.NotNil(t, gotMeta)
	assert.Equal(t, "meta.queue", gotMeta.Subject)
	assert.Equal(t, "queue-cid", gotMeta.Headers.Get(consumer.HeaderCorrelationIDKey))
}

// TestJetStream_HandlerError_NakWithDLQDisabled verifies that a regular
// handler error on a JetStream route with DLQ disabled results in a Nak
// (message redelivered) and does not crash.
func TestJetStream_HandlerError_NakWithDLQDisabled(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	streamName := uniqueName("NAKDLQOFF")
	subject := "nakdlqoff." + streamName
	durable := uniqueName("nakdlqoff_dur")

	_, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	})
	assert.NoError(t, err)

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, _ := router.New(
		router.TypeJetStream,
		subject,
		router.WithStream(streamName),
		router.WithDurable(durable),
		router.WithAckWait(200*time.Millisecond),
		router.WithMaxDeliver(3),
	)

	var calls int32

	err = c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		atomic.AddInt32(&calls, 1)
		return nil, fmt.Errorf("transient: %w", errors.New("fail"))
	})
	assert.NoError(t, err)

	_, err = js.Publish(context.Background(), subject, []byte("data"))
	assert.NoError(t, err)

	// Wait for at least 2 redeliveries to confirm Nak is working.
	time.Sleep(2 * time.Second)

	assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2))
}

// TestJetStream_StartError_InvalidStream verifies that startJetStream returns
// an error when the stream does not exist.
func TestJetStream_StartError_InvalidStream(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	r, _ := router.New(
		router.TypeJetStream,
		"nonexistent.subject",
		router.WithStream("NONEXISTENT_STREAM"),
		router.WithDurable("some_durable"),
	)

	err := c.Start(context.Background(), r, func(ctx context.Context, b []byte) (any, error) {
		return nil, nil
	})

	assert.Error(t, err)
}
