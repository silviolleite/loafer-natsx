package typed_test

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/silviolleite/loafer-natsx/consumer"
	"github.com/silviolleite/loafer-natsx/typed"
)

func TestWrapHandler_Success(t *testing.T) {
	codec := typed.JSONCodec[order]{}
	called := false

	handler := typed.WrapHandler(codec, func(_ context.Context, msg order) (string, error) {
		called = true
		assert.Equal(t, "abc", msg.ID)
		return "ok", nil
	})

	data, _ := codec.Encode(order{ID: "abc", Amount: 10})
	result, err := handler(context.Background(), data)

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "ok", result)
}

func TestWrapHandler_DecodeError(t *testing.T) {
	codec := typed.JSONCodec[order]{}
	called := false

	handler := typed.WrapHandler(codec, func(_ context.Context, _ order) (any, error) {
		called = true
		return nil, nil
	})

	result, err := handler(context.Background(), []byte("bad json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
	assert.Nil(t, result)
	assert.False(t, called)
}

func TestWrapHandler_MetadataAccessibleFromTypedHandler(t *testing.T) {
	codec := typed.JSONCodec[order]{}

	headers := nats.Header{}
	headers.Set("X-Correlation-ID", "trace-99")

	md := &consumer.Metadata{
		Subject: "orders.created.user-42",
		Headers: headers,
	}

	ctx := consumer.WithMetadata(context.Background(), md)

	var (
		gotSubject       string
		gotCorrelationID string
	)

	handler := typed.WrapHandler(codec, func(ctx context.Context, msg order) (string, error) {
		meta, ok := consumer.MetadataFromContext(ctx)
		if ok {
			gotSubject = meta.Subject
			gotCorrelationID = meta.Headers.Get("X-Correlation-ID")
		}
		return "ok", nil
	})

	data, _ := codec.Encode(order{ID: "42", Amount: 100})
	result, err := handler(ctx, data)

	assert.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.Equal(t, "orders.created.user-42", gotSubject)
	assert.Equal(t, "trace-99", gotCorrelationID)
}
