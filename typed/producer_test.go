package typed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/typed"
)

func TestNewProducer_EmptySubject(t *testing.T) {
	p, err := typed.NewProducer[order](&mockPublisher{}, "", typed.JSONCodec[order]{})
	assert.Nil(t, p)
	assert.ErrorIs(t, err, loafernatsx.ErrMissingSubject)
}

func TestNewProducer_NilPublisher(t *testing.T) {
	p, err := typed.NewProducer[order](nil, "orders.new", typed.JSONCodec[order]{})
	assert.Nil(t, p)
	assert.ErrorIs(t, err, loafernatsx.ErrUnsupportedType)
}

func TestNewProducer_Success(t *testing.T) {
	p, err := typed.NewProducer[order](&mockPublisher{}, "orders.new", typed.JSONCodec[order]{})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestProducer_Publish_Success(t *testing.T) {
	mp := &mockPublisher{}
	p, err := typed.NewProducer[order](mp, "orders.new", typed.JSONCodec[order]{})
	require.NoError(t, err)

	err = p.Publish(context.Background(), order{ID: "1", Amount: 9.99})
	assert.NoError(t, err)
	assert.True(t, mp.called)
	assert.Equal(t, "orders.new", mp.msg.Subject)
	assert.Contains(t, string(mp.msg.Data), `"id":"1"`)
}

func TestProducer_Publish_EncodeError(t *testing.T) {
	mp := &mockPublisher{}
	p, err := typed.NewProducer[order](mp, "orders.new", failCodec[order]{})
	require.NoError(t, err)

	err = p.Publish(context.Background(), order{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encode")
	assert.False(t, mp.called)
}

func TestProducer_Publish_PublisherError(t *testing.T) {
	mp := &mockPublisher{err: errors.New("nats down")}
	p, err := typed.NewProducer[order](mp, "orders.new", typed.JSONCodec[order]{})
	require.NoError(t, err)

	err = p.Publish(context.Background(), order{ID: "1", Amount: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nats down")
}
