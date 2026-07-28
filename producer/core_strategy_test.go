package producer_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/producer"
)

func runServer() (*server.Server, string) {
	opts := natstest.DefaultTestOptions
	opts.Port = -1
	s := natstest.RunServer(&opts)
	return s, s.ClientURL()
}

func TestCoreStrategy_Publish(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, err := nats.Connect(url)
	assert.NoError(t, err)
	defer nc.Close()

	strategy := producer.NewCoreStrategy(nc)

	received := make(chan []byte, 1)

	_, err = nc.Subscribe("test.core", func(msg *nats.Msg) {
		received <- msg.Data
	})
	assert.NoError(t, err)

	result, err := strategy.Publish(
		context.Background(),
		&nats.Msg{
			Subject: "test.core",
			Data:    []byte("data"),
		},
		producer.PublishOptions{},
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	select {
	case data := <-received:
		assert.Equal(t, []byte("data"), data)
	case <-time.After(2 * time.Second):
		t.Fatal("message not received")
	}
}

func TestCoreStrategy_Request(t *testing.T) {
	t.Run("returns Response with Data and Header", func(t *testing.T) {
		s, url := runServer()
		defer s.Shutdown()

		nc, err := nats.Connect(url)
		require.NoError(t, err)
		defer nc.Close()

		_, err = nc.Subscribe("test.req", func(msg *nats.Msg) {
			reply := &nats.Msg{
				Data:   []byte("ok"),
				Header: nats.Header{},
			}
			reply.Header.Set("X-Status", "success")
			reply.Header.Set("X-Custom", "value")
			_ = msg.RespondMsg(reply)
		})
		require.NoError(t, err)

		strategy := producer.NewCoreStrategy(nc)
		req := strategy.(producer.Requester)

		resp, err := req.Request(context.Background(), "test.req", []byte("ping"))
		require.NoError(t, err)
		assert.Equal(t, []byte("ok"), resp.Data)
		assert.Equal(t, "success", resp.Header.Get("X-Status"))
		assert.Equal(t, "value", resp.Header.Get("X-Custom"))
	})

	t.Run("returns nil and error when no responders", func(t *testing.T) {
		s, url := runServer()
		defer s.Shutdown()

		nc, err := nats.Connect(url, nats.NoEcho())
		require.NoError(t, err)
		defer nc.Close()

		strategy := producer.NewCoreStrategy(nc)
		req := strategy.(producer.Requester)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		resp, err := req.Request(ctx, "test.no.responders", []byte("ping"))
		assert.Nil(t, resp)
		assert.Error(t, err)
	})
}

func TestProducer_RequestTimeout_Integration(t *testing.T) {
	t.Run("times out and returns ErrRequestTimeout when consumer never replies", func(t *testing.T) {
		s, url := runServer()
		defer s.Shutdown()

		nc, err := nats.Connect(url)
		require.NoError(t, err)
		defer nc.Close()

		_, err = nc.Subscribe("test.slow", func(msg *nats.Msg) {
			time.Sleep(10 * time.Second)
		})
		require.NoError(t, err)

		strategy := producer.NewCoreStrategy(nc)
		p, err := producer.New(strategy, "test.slow", producer.WithRequestTimeout(200*time.Millisecond))
		require.NoError(t, err)

		start := time.Now()
		resp, err := p.Request(context.Background(), []byte("ping"))
		elapsed := time.Since(start)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, loafernatsx.ErrRequestTimeout)
		assert.Less(t, elapsed, 2*time.Second)
	})

	t.Run("succeeds when consumer replies within timeout", func(t *testing.T) {
		s, url := runServer()
		defer s.Shutdown()

		nc, err := nats.Connect(url)
		require.NoError(t, err)
		defer nc.Close()

		_, err = nc.Subscribe("test.fast", func(msg *nats.Msg) {
			_ = msg.Respond([]byte("pong"))
		})
		require.NoError(t, err)

		strategy := producer.NewCoreStrategy(nc)
		p, err := producer.New(strategy, "test.fast", producer.WithRequestTimeout(5*time.Second))
		require.NoError(t, err)

		resp, err := p.Request(context.Background(), []byte("ping"))
		require.NoError(t, err)
		assert.Equal(t, []byte("pong"), resp.Data)
	})

	t.Run("uses default 10s timeout when no option is set", func(t *testing.T) {
		s, url := runServer()
		defer s.Shutdown()

		nc, err := nats.Connect(url)
		require.NoError(t, err)
		defer nc.Close()

		_, err = nc.Subscribe("test.default", func(msg *nats.Msg) {
			_ = msg.Respond([]byte("pong"))
		})
		require.NoError(t, err)

		strategy := producer.NewCoreStrategy(nc)
		p, err := producer.New(strategy, "test.default")
		require.NoError(t, err)

		resp, err := p.Request(context.Background(), []byte("ping"))
		require.NoError(t, err)
		assert.Equal(t, []byte("pong"), resp.Data)
	})
}
