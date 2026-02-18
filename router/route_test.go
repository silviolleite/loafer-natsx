package router_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	loafernastx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/router"
)

func TestNew_MissingSubject(t *testing.T) {
	r, err := router.New(router.TypePubSub, "")
	assert.Nil(t, r)
	assert.ErrorIs(t, err, loafernastx.ErrMissingSubject)
}

func TestNew_UnsupportedType(t *testing.T) {
	r, err := router.New(router.Type(999), "subject")
	assert.Nil(t, r)
	assert.ErrorIs(t, err, loafernastx.ErrUnsupportedType)
}

func TestNew_PubSub_Success(t *testing.T) {
	r, err := router.New(router.TypePubSub, "orders.created")
	assert.NoError(t, err)
	assert.Equal(t, router.TypePubSub, r.Type())
	assert.Equal(t, "orders.created", r.Subject())
}

func TestNew_Queue_MissingGroup(t *testing.T) {
	r, err := router.New(router.TypeQueue, "orders.created")
	assert.Nil(t, r)
	assert.ErrorIs(t, err, loafernastx.ErrMissingQueueGroup)
}

func TestNew_Queue_Success(t *testing.T) {
	r, err := router.New(
		router.TypeQueue,
		"orders.created",
		router.WithQueueGroup("workers"),
	)
	assert.NoError(t, err)
	assert.Equal(t, router.TypeQueue, r.Type())
	assert.Equal(t, "workers", r.QueueGroup())
}

func TestNew_JetStream_MissingStream(t *testing.T) {
	r, err := router.New(
		router.TypeJetStream,
		"orders.created",
		router.WithDurable("d"),
	)
	assert.Nil(t, r)
	assert.ErrorIs(t, err, loafernastx.ErrMissingStream)
}

func TestNew_JetStream_MissingDurable(t *testing.T) {
	r, err := router.New(
		router.TypeJetStream,
		"orders.created",
		router.WithStream("ORDERS"),
	)
	assert.Nil(t, r)
	assert.ErrorIs(t, err, loafernastx.ErrMissingDurable)
}

func TestNew_JetStream_Defaults(t *testing.T) {
	r, err := router.New(
		router.TypeJetStream,
		"orders.created",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-durable"),
	)
	assert.NoError(t, err)
	assert.Equal(t, router.TypeJetStream, r.Type())
	assert.Equal(t, "ORDERS", r.Stream())
	assert.Equal(t, "orders-durable", r.Durable())
	assert.Equal(t, 10, r.MaxDeliver())
	assert.Equal(t, 30*time.Second, r.AckWait())
}

func TestNew_JetStream_CustomValues(t *testing.T) {
	r, err := router.New(
		router.TypeJetStream,
		"orders.created",
		router.WithStream("ORDERS"),
		router.WithDurable("orders-durable"),
		router.WithMaxDeliver(5),
		router.WithAckWait(10*time.Second),
		router.WithDeliveryPolicy(router.DeliverNewPolicy),
	)
	assert.NoError(t, err)
	assert.Equal(t, 5, r.MaxDeliver())
	assert.Equal(t, 10*time.Second, r.AckWait())
	assert.Equal(t, router.DeliverNewPolicy, r.DeliveryPolicy())
}

func TestNew_RequestReply_Success(t *testing.T) {
	replyCalled := false
	replyFunc := func(ctx context.Context, result any, err error) ([]byte, nats.Header, error) {
		replyCalled = true
		return []byte("ok"), nil, nil
	}

	r, err := router.New(
		router.TypeRequestReply,
		"orders.calculate",
		router.WithReply(replyFunc),
		router.WithQueueGroup("test-group"),
	)
	assert.NoError(t, err)
	assert.Equal(t, router.TypeRequestReply, r.Type())
	assert.Equal(t, "orders.calculate", r.Subject())
	assert.Equal(t, "test-group", r.QueueGroup())
	assert.NotNil(t, r.ReplyFunc())
	assert.False(t, replyCalled)
}

func TestNew_RequestReply_MissingQueueGroup(t *testing.T) {
	_, err := router.New(
		router.TypeRequestReply,
		"orders.calculate",
	)
	assert.Error(t, err)
	assert.ErrorIs(t, err, loafernastx.ErrMissingQueueGroup)
}

func TestHandlerTimeout(t *testing.T) {
	r, err := router.New(
		router.TypePubSub,
		"orders.created",
		router.WithHandlerTimeout(5*time.Second),
	)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Second, r.HandlerTimeout())
}

func TestDLQEnabled(t *testing.T) {
	r, err := router.New(
		router.TypeJetStream,
		"orders.failed",
		router.WithStream("ORDERS"),
		router.WithDurable("d"),
		router.WithEnableDLQ(),
	)
	assert.NoError(t, err)
	assert.True(t, r.DLQEnabled())
}
