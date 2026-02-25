package typed_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

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
