package consumer_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/logger"
	"github.com/silviolleite/loafer-natsx/router"
)

func runServer(js bool) (*server.Server, string) {
	opts := natstest.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = js
	s := natstest.RunServer(&opts)
	return s, s.ClientURL()
}

func TestPubSub(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	r, _ := router.New(router.TypePubSub, "test.pubsub")

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		wg.Done()
		return nil, nil
	})

	assert.NoError(t, err)

	_ = nc.Publish("test.pubsub", []byte("data"))

	wait(&wg)
}

func TestQueue(t *testing.T) {
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
		"test.queue",
		router.WithQueueGroup("workers"),
	)

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		wg.Done()
		return nil, nil
	})

	assert.NoError(t, err)

	_ = nc.Publish("test.queue", []byte("data"))

	wait(&wg)
}

func TestRequestReply_Default(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, _ := router.New(
		router.TypeRequestReply,
		"test.req",
		router.WithQueueGroup("workers"),
	)

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		return nil, nil
	})

	assert.NoError(t, err)

	msg, err := nc.Request("test.req", []byte("data"), time.Second)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), msg.Data)
}

func TestRequestReply_CustomReply(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reply := func(ctx context.Context, result any, err error) ([]byte, nats.Header, error) {
		return []byte("custom"), nil, nil
	}

	r, _ := router.New(
		router.TypeRequestReply,
		"test.req.custom",
		router.WithQueueGroup("workers"),
		router.WithReply(reply),
	)

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		return "result", nil
	})

	assert.NoError(t, err)

	msg, err := nc.Request("test.req.custom", []byte("data"), time.Second)
	assert.NoError(t, err)
	assert.Equal(t, []byte("custom"), msg.Data)
}

func TestJetStream_Ack(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	_, _ = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "TEST",
		Subjects: []string{"test.js"},
	})

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	r, _ := router.New(
		router.TypeJetStream,
		"test.js",
		router.WithStream("TEST"),
		router.WithDurable("d1"),
	)

	var once sync.Once

	handler := func(ctx context.Context, data []byte) (any, error) {
		once.Do(func() {
			wg.Done()
		})
		return nil, nil
	}

	err := c.Start(ctx, r, handler)

	assert.NoError(t, err)

	_, _ = js.Publish(context.Background(), "test.js", []byte("data"))

	wait(&wg)
}

func TestJetStream_DLQ(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	_, _ = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "TESTQ",
		Subjects: []string{"test.dlq"},
	})

	_, _ = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "DLQ",
		Subjects: []string{"dlq.test.dlq"},
	})

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	r, _ := router.New(
		router.TypeJetStream,
		"test.dlq",
		router.WithStream("TESTQ"),
		router.WithDurable("d2"),
		router.WithAckWait(1*time.Second),
		router.WithMaxDeliver(1),
		router.WithEnableDLQ(),
	)

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		return nil, errors.New("fail")
	})
	assert.NoError(t, err)
	_, _ = nc.Subscribe("dlq.test.dlq", func(msg *nats.Msg) {
		wg.Done()
	})

	_, _ = js.Publish(context.Background(), "test.dlq", []byte("data"))

	wait(&wg)
}

func wait(wg *sync.WaitGroup) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		panic("timeout")
	}
}
