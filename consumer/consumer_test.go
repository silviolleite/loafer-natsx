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

func TestRequestReply_PropagateHeaders(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	reply := func(ctx context.Context, result any, err error) ([]byte, nats.Header, error) {
		return []byte("custom"), nil, nil
	}

	r, _ := router.New(
		router.TypeRequestReply,
		"test.req.headers",
		router.WithQueueGroup("workers"),
		router.WithReply(reply),
	)

	err := c.Start(context.Background(), r, func(ctx context.Context, b []byte) (any, error) {
		return "result", nil
	})
	assert.NoError(t, err)

	req := &nats.Msg{
		Subject: "test.req.headers",
		Data:    []byte("data"),
		Header:  nats.Header{},
	}
	req.Header.Add(consumer.HeaderCorrelationIDKey, "cid-1")
	req.Header.Add(consumer.HeaderCorrelationIDKey, "cid-2")
	req.Header.Add(consumer.HeaderTraceParentKey, "tp-1")
	req.Header.Add(consumer.HeaderTraceStateKey, "ts-1")
	req.Header.Add(consumer.HeaderBaggageKey, "bg-1")
	req.Header.Add(consumer.HeaderErrorKey, "should-not-propagate")
	req.Header.Add("X-Unrelated", "should-not-propagate")

	resp, err := nc.RequestMsg(req, time.Second)
	assert.NoError(t, err)
	assert.Equal(t, []byte("custom"), resp.Data)

	assert.NotNil(t, resp.Header)
	assert.ElementsMatch(t, []string{"cid-1", "cid-2"}, resp.Header[consumer.HeaderCorrelationIDKey])
	assert.Equal(t, "tp-1", resp.Header.Get(consumer.HeaderTraceParentKey))
	assert.Equal(t, "ts-1", resp.Header.Get(consumer.HeaderTraceStateKey))
	assert.Equal(t, "bg-1", resp.Header.Get(consumer.HeaderBaggageKey))
	assert.Empty(t, resp.Header.Get(consumer.HeaderErrorKey))
	assert.Empty(t, resp.Header.Get("X-Unrelated"))
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

func TestPubSub_PanicRecovered(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, _ := router.New(router.TypePubSub, "panic.pubsub")

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		panic("boom")
	})

	assert.NoError(t, err)

	_ = nc.Publish("panic.pubsub", []byte("data"))

	time.Sleep(100 * time.Millisecond)
}

func TestRequestReply_Default_Error(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx := context.Background()

	r, _ := router.New(
		router.TypeRequestReply,
		"test.req.err",
		router.WithQueueGroup("workers"),
	)

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		return nil, errors.New("fail")
	})

	assert.NoError(t, err)

	_, err = nc.Request("test.req.err", []byte("data"), time.Second)
	assert.Error(t, err)
}

func TestRequestReply_ReplyBuilderError(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	reply := func(ctx context.Context, result any, err error) ([]byte, nats.Header, error) {
		return nil, nil, errors.New("builder fail")
	}

	r, _ := router.New(
		router.TypeRequestReply,
		"test.req.builder",
		router.WithQueueGroup("workers"),
		router.WithReply(reply),
	)

	err := c.Start(context.Background(), r,
		func(ctx context.Context, b []byte) (any, error) {
			return "ok", nil
		})

	assert.NoError(t, err)

	_, err = nc.Request("test.req.builder", []byte("data"), time.Second)
	assert.Error(t, err)
}

func TestJetStream_NakPath(t *testing.T) {
	s, url := runServer(true)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	js, _ := jetstream.New(nc)

	_, _ = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "TESTNAK",
		Subjects: []string{"test.nak"},
	})

	c, _ := consumer.New(nc, logger.NopLogger{})

	r, _ := router.New(
		router.TypeJetStream,
		"test.nak",
		router.WithStream("TESTNAK"),
		router.WithDurable("dnak"),
	)

	err := c.Start(context.Background(), r,
		func(ctx context.Context, b []byte) (any, error) {
			return nil, errors.New("fail")
		})

	assert.NoError(t, err)

	_, _ = js.Publish(context.Background(), "test.nak", []byte("data"))

	time.Sleep(200 * time.Millisecond)
}

func TestDrainOnCancel(t *testing.T) {
	s, url := runServer(false)
	defer s.Shutdown()

	nc, _ := nats.Connect(url)
	defer nc.Close()

	c, _ := consumer.New(nc, logger.NopLogger{})

	ctx, cancel := context.WithCancel(context.Background())

	r, _ := router.New(router.TypePubSub, "test.drain")

	err := c.Start(ctx, r, func(ctx context.Context, b []byte) (any, error) {
		return nil, nil
	})

	assert.NoError(t, err)

	cancel()

	time.Sleep(100 * time.Millisecond)
}
