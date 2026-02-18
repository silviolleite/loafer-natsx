package producer_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

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
	s, url := runServer()
	defer s.Shutdown()

	nc, err := nats.Connect(url)
	assert.NoError(t, err)
	defer nc.Close()

	_, err = nc.Subscribe("test.req", func(msg *nats.Msg) {
		_ = msg.Respond([]byte("ok"))
	})
	assert.NoError(t, err)

	strategy := producer.NewCoreStrategy(nc)

	req := strategy.(producer.Requester)

	resp, err := req.Request(context.Background(), "test.req", []byte("ping"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), resp)
}
