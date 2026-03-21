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

	err = strategy.Publish(
		context.Background(),
		&nats.Msg{
			Subject: "test.core",
			Data:    []byte("data"),
		},
		producer.PublishOptions{},
	)

	assert.NoError(t, err)

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
