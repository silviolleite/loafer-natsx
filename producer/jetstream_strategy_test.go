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

	result, err := strategy.Publish(
		context.Background(),
		&nats.Msg{
			Subject: "test.js",
			Data:    []byte("data"),
		},
		producer.PublishOptions{},
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TEST", result.Stream)
	assert.Greater(t, result.Sequence, uint64(0))
	assert.False(t, result.Duplicate)
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
	result, err := strategy.Publish(
		context.Background(),
		&nats.Msg{
			Subject: "test.js",
			Data:    []byte("data"),
		},
		opt,
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TEST", result.Stream)
}

func TestJetStreamStrategy_Publish_Duplicate(t *testing.T) {
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
		Name:     "DEDUP",
		Subjects: []string{"test.dedup"},
	})
	assert.NoError(t, err)

	strategy := producer.NewJetStreamStrategy(js, logger.NopLogger{})

	pubOpt := producer.PublishOptions{}
	producer.PublishWithMsgID("unique-1")(&pubOpt)

	msg := &nats.Msg{Subject: "test.dedup", Data: []byte("data")}

	first, err := strategy.Publish(context.Background(), msg, pubOpt)
	assert.NoError(t, err)
	assert.False(t, first.Duplicate)

	second, err := strategy.Publish(context.Background(), msg, pubOpt)
	assert.NoError(t, err)
	assert.True(t, second.Duplicate)
	assert.Equal(t, first.Sequence, second.Sequence)
}
