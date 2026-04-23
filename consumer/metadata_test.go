package consumer_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/logger"
	"github.com/silviolleite/loafer-natsx/router"
)

// uniqueName returns a fresh identifier on every call so that JetStream
// streams/durables do not collide across tests or repeated runs.
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func TestMetadataFromContext_Missing(t *testing.T) {
	md, ok := consumer.MetadataFromContext(context.Background())
	assert.False(t, ok)
	assert.Nil(t, md)
}

func TestMetadataFromContext_NilContext(t *testing.T) {
	//nolint:staticcheck // intentional nil ctx to cover defensive path
	md, ok := consumer.MetadataFromContext(nil)
	assert.False(t, ok)
	assert.Nil(t, md)
}

func TestWithMetadata_RoundTrip(t *testing.T) {
	headers := nats.Header{}
	headers.Set("X-Correlation-ID", "abc-123")

	md := &consumer.Metadata{
		Subject: "orders.created",
		Reply:   "_INBOX.42",
		Headers: headers,
	}

	ctx := consumer.WithMetadata(context.Background(), md)

	got, ok := consumer.MetadataFromContext(ctx)
	assert.True(t, ok)
	assert.NotNil(t, got)
	assert.Equal(t, "orders.created", got.Subject)
	assert.Equal(t, "_INBOX.42", got.Reply)
	assert.Equal(t, "abc-123", got.Headers.Get("X-Correlation-ID"))
}

func TestWithMetadata_OverridesPreviousValue(t *testing.T) {
	ctx := consumer.WithMetadata(context.Background(), &consumer.Metadata{Subject: "a"})
	ctx = consumer.WithMetadata(ctx, &consumer.Metadata{Subject: "b"})

	got, ok := consumer.MetadataFromContext(ctx)
	assert.True(t, ok)
	assert.NotNil(t, got)
	assert.Equal(t, "b", got.Subject)
}

func TestPubSub_MetadataInContext(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	var (
		mu      sync.Mutex
		gotMeta *consumer.Metadata
		gotOK   bool
		once    sync.Once
	)

	r, _ := router.New(router.TypePubSub, "meta.pubsub")

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
		Subject: "meta.pubsub",
		Data:    []byte("data"),
		Header:  nats.Header{},
	}
	req.Header.Set(consumer.HeaderCorrelationIDKey, "cid-42")

	_ = nc.PublishMsg(req)

	wait(&wg)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, gotOK)
	assert.NotNil(t, gotMeta)
	assert.Equal(t, "meta.pubsub", gotMeta.Subject)
	assert.Equal(t, "cid-42", gotMeta.Headers.Get(consumer.HeaderCorrelationIDKey))
}

func TestRequestReply_MetadataInContext(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, _ := router.New(
		router.TypeRequestReply,
		"meta.req",
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
		})
		return nil, nil
	})
	assert.NoError(t, err)

	_, err = nc.Request("meta.req", []byte("data"), time.Second)
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, gotOK)
	assert.NotNil(t, gotMeta)
	assert.Equal(t, "meta.req", gotMeta.Subject)
	assert.NotEmpty(t, gotMeta.Reply)
}

func TestJetStream_MetadataInContext(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	streamName := uniqueName("METAJS")
	subject := "meta.js." + streamName
	durable := uniqueName("meta_dur")

	_, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	})
	assert.NoError(t, err)

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	r, _ := router.New(
		router.TypeJetStream,
		subject,
		router.WithStream(streamName),
		router.WithDurable(durable),
		router.WithAckWait(5*time.Second),
	)

	var (
		mu      sync.Mutex
		gotMeta *consumer.Metadata
		gotOK   bool
		once    sync.Once
	)

	err = c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
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

	_, err = js.Publish(context.Background(), subject, []byte("data"))
	assert.NoError(t, err)

	wait(&wg)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, gotOK)
	assert.NotNil(t, gotMeta)
	assert.Equal(t, subject, gotMeta.Subject)
	assert.Equal(t, streamName, gotMeta.Stream)
	// The message may be redelivered before the handler acks in rare timing
	// scenarios; stream sequence is deterministic (first message) but
	// NumDelivered and redelivery-sensitive fields only need to be >= 1.
	assert.Equal(t, uint64(1), gotMeta.Sequence)
	assert.GreaterOrEqual(t, gotMeta.NumDelivered, uint64(1))
	assert.False(t, gotMeta.Timestamp.IsZero())
}

func TestJetStream_PermanentFailure_Acks(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	streamName := uniqueName("PERM")
	subject := "perm.fail." + streamName
	durable := uniqueName("perm_dur")

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
		router.WithAckWait(500*time.Millisecond),
		router.WithMaxDeliver(5),
	)

	var calls int32

	err = c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		atomic.AddInt32(&calls, 1)
		return nil, fmt.Errorf("%w: bad payload", loafernatsx.ErrPermanentFailure)
	})
	assert.NoError(t, err)

	_, err = js.Publish(context.Background(), subject, []byte("data"))
	assert.NoError(t, err)

	// Wait longer than (MaxDeliver × AckWait) so that any redelivery would
	// have triggered extra handler calls.
	time.Sleep(3 * time.Second)

	// Exactly one delivery because permanent failure triggers Ack.
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestJetStream_SendToDLQ_SkipsRetry(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	streamName := uniqueName("DLQSKIP")
	subject := "dlqskip.fail." + streamName
	dlqSubject := "dlq." + subject
	dlqStream := uniqueName("DLQSKIP_OUT")
	durable := uniqueName("dlqskip_dur")

	_, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	})
	assert.NoError(t, err)

	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     dlqStream,
		Subjects: []string{dlqSubject},
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
		router.WithAckWait(2*time.Second),
		router.WithMaxDeliver(10),
		router.WithEnableDLQ(),
	)

	var calls int32

	err = c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		atomic.AddInt32(&calls, 1)
		return nil, fmt.Errorf("%w: cannot process", loafernatsx.ErrSendToDLQ)
	})
	assert.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)

	var (
		mu         sync.Mutex
		dlqHeaders nats.Header
		once       sync.Once
	)

	_, _ = nc.Subscribe(dlqSubject, func(msg *nats.Msg) {
		once.Do(func() {
			mu.Lock()
			dlqHeaders = msg.Header
			mu.Unlock()
			wg.Done()
		})
	})

	_, err = js.Publish(context.Background(), subject, []byte("data"))
	assert.NoError(t, err)

	wait(&wg)

	// Only one delivery attempt → message went straight to DLQ.
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))

	mu.Lock()
	defer mu.Unlock()
	assert.NotNil(t, dlqHeaders)
	assert.Contains(t, dlqHeaders.Get(consumer.HeaderErrorKey), "send to dead letter queue")
}

func TestJetStream_SendToDLQ_WithoutDLQEnabled_FallsBackToAck(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	streamName := uniqueName("DLQOFF")
	subject := "dlqoff.fail." + streamName
	durable := uniqueName("dlqoff_dur")

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
		router.WithAckWait(500*time.Millisecond),
		router.WithMaxDeliver(5),
	)

	var calls int32

	err = c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		atomic.AddInt32(&calls, 1)
		return nil, fmt.Errorf("%w: no dlq configured", loafernatsx.ErrSendToDLQ)
	})
	assert.NoError(t, err)

	_, err = js.Publish(context.Background(), subject, []byte("data"))
	assert.NoError(t, err)

	// Wait enough time for potential redeliveries. Since DLQ is not enabled,
	// the handler acks to avoid infinite retries.
	time.Sleep(3 * time.Second)

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestJetStream_NakWithDelay_Redelivers(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	streamName := uniqueName("NAKDELAY")
	subject := "nakdelay." + streamName
	durable := uniqueName("nakdelay_dur")

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
		router.WithAckWait(10*time.Second),
		router.WithMaxDeliver(5),
	)

	var calls int32

	err = c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			return nil, loafernatsx.NakWithDelayError{Delay: 100 * time.Millisecond}
		}
		return nil, nil
	})
	assert.NoError(t, err)

	_, err = js.Publish(context.Background(), subject, []byte("data"))
	assert.NoError(t, err)

	time.Sleep(2 * time.Second)

	assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2),
		"message should be redelivered after NakWithDelay")
}

func TestJetStream_NakWithDelay_Wrapped(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	streamName := uniqueName("NAKWRAP")
	subject := "nakwrap." + streamName
	durable := uniqueName("nakwrap_dur")

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
		router.WithAckWait(10*time.Second),
		router.WithMaxDeliver(5),
	)

	var calls int32

	err = c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			return nil, fmt.Errorf("precondition not met: %w",
				loafernatsx.NakWithDelayError{Delay: 100 * time.Millisecond})
		}
		return nil, nil
	})
	assert.NoError(t, err)

	_, err = js.Publish(context.Background(), subject, []byte("data"))
	assert.NoError(t, err)

	time.Sleep(2 * time.Second)

	assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2),
		"wrapped NakWithDelayError should also trigger delayed redelivery")
}
