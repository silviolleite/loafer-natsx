package producer_test

import (
	"context"
	"testing"

	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	"github.com/silviolleite/loafer-natsx/logger"
	"github.com/silviolleite/loafer-natsx/producer"
)

func TestJetStreamStrategy_Publish(t *testing.T) {
	opts := natstest.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true

	s := natstest.RunServer(&opts)
	defer s.Shutdown()

	nc, err := nats.Connect(s.ClientURL())
	assert.NoError(t, err)
	defer nc.Close()

	js, err := jetstream.New(nc)
	assert.NoError(t, err)

	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "TEST",
		Subjects: []string{"test.js"},
	})
	assert.NoError(t, err)

	strategy := producer.NewJetStreamStrategy(js, logger.NopLogger{})

	err = strategy.Publish(
		context.Background(),
		&nats.Msg{
			Subject: "test.js",
			Data:    []byte("data"),
		},
		producer.PublishOptions{},
	)

	assert.NoError(t, err)
}

func TestJetStreamStrategy_WithMsgID(t *testing.T) {
	opts := natstest.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true

	s := natstest.RunServer(&opts)
	defer s.Shutdown()

	nc, err := nats.Connect(s.ClientURL())
	assert.NoError(t, err)
	defer nc.Close()

	js, err := jetstream.New(nc)
	assert.NoError(t, err)

	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "TEST",
		Subjects: []string{"test.js"},
	})
	assert.NoError(t, err)

	strategy := producer.NewJetStreamStrategy(js, logger.NopLogger{})
	opt := producer.PublishOptions{}
	fn := producer.PublishWithMsgID("1")
	fn(&opt)
	err = strategy.Publish(
		context.Background(),
		&nats.Msg{
			Subject: "test.js",
			Data:    []byte("data"),
		},
		opt,
	)

	assert.NoError(t, err)
}
